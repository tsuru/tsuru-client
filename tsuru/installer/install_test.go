// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/machine/drivers/fakedriver"
	"github.com/docker/machine/libmachine/host"
	"github.com/tsuru/tsuru-client/tsuru/installer/defaultconfig"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	"gopkg.in/check.v1"
)

func (s *S) TestParseConfigDefaultConfig(c *check.C) {
	dmConfig, err := parseConfigFile("")
	c.Assert(err, check.IsNil)
	c.Assert(dmConfig, check.DeepEquals, defaultInstallOpts)
}

func (s *S) TestParseConfigFileNotExists(c *check.C) {
	_, err := parseConfigFile("not-exist-conf.yml")
	c.Assert(err, check.NotNil)
}

func (s *S) TestParseConfigFile(c *check.C) {
	expected := &InstallOpts{
		Name: "tsuru-test",
		DockerMachineConfig: dm.DockerMachineConfig{
			DriverOpts: &dm.DriverOpts{
				Name: "amazonec2",
				Options: map[string]interface{}{
					"opt1": "option1-value",
				},
			},
			CAPath:      "/tmp/certs",
			DockerFlags: []string{"experimental"},
		},
		ComponentsConfig: &ComponentsConfig{
			InstallDashboard: true,
			TargetName:       "tsuru-test",
			RootUserEmail:    "admin@example.com",
			RootUserPassword: "admin123",
			IaaSConfig: iaasConfig{
				Dockermachine: iaasConfigInternal{
					CaPath: "/certs",
					Driver: iaasConfigDriver{
						Name: "amazonec2",
						Options: map[string]interface{}{
							"opt1": "option1-value",
						},
					},
				},
			},
		},
		Hosts: hostGroups{
			Apps: hostGroupConfig{
				Size:      1,
				Dedicated: true,
				Driver: multiOptionsDriver{Options: map[string][]interface{}{
					"amazonec2-tags": {"my-tag"},
				}}},
			Core: hostGroupConfig{
				Size: 2,
				Driver: multiOptionsDriver{Options: map[string][]interface{}{
					"amazonec2-region": {"us-east", "us-west"},
				}},
			},
		},
	}
	dmConfig, err := parseConfigFile("./testdata/hosts.yml")
	c.Assert(err, check.IsNil)
	c.Assert(dmConfig, check.DeepEquals, expected)
}

func (s *S) TestInstallInfo(c *check.C) {
	c.Assert((&Install{}).Info(), check.NotNil)
}

func (s *S) TestInstallCommandFlags(c *check.C) {
	command := Install{}
	flags := command.Flags()
	c.Assert(flags, check.NotNil)
	flags.Parse(true, []string{"-c", "my-conf.yml"})
	config := flags.Lookup("c")
	usage := "Configuration file"
	c.Check(config, check.NotNil)
	c.Check(config.Name, check.Equals, "c")
	c.Check(config.Usage, check.Equals, usage)
	c.Check(config.Value.String(), check.Equals, "my-conf.yml")
	c.Check(config.DefValue, check.Equals, "")
	config = flags.Lookup("config")
	c.Check(config, check.NotNil)
	c.Check(config.Name, check.Equals, "config")
	c.Check(config.Usage, check.Equals, usage)
	c.Check(config.Value.String(), check.Equals, "my-conf.yml")
	c.Check(config.DefValue, check.Equals, "")
}

func (s *S) TestInstallTargetAlreadyExists(c *check.C) {
	var stdout, stderr bytes.Buffer
	manager := cmd.BuildBaseManager("uninstall-client", "0.0.0", "", nil)
	client := cmd.NewClient(&http.Client{}, nil, manager)
	context := cmd.Context{
		Args:   []string{"test", fmt.Sprintf("%s:8080", "1.2.3.4")},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	targetadd := manager.Commands["target-add"]
	t, ok := targetadd.(cmd.FlaggedCommand)
	c.Assert(ok, check.Equals, true)
	err := t.Flags().Parse(true, []string{"-s"})
	c.Assert(err, check.IsNil)
	err = t.Run(&context, client)
	c.Assert(err, check.IsNil)
	defer func(manager *cmd.Manager) {
		c := cmd.NewClient(&http.Client{}, nil, manager)
		cont := cmd.Context{
			Args:   []string{"test"},
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		targetrm := manager.Commands["target-remove"]
		targetrm.Run(&cont, c)
	}(manager)
	command := Install{}
	command.Flags().Parse(true, []string{"-c", "./testdata/wrong-conf.yml"})
	context = cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedErr := "pre-install checks failed: tsuru target \"test\" already exists"
	err = command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(expectedErr, check.Equals, err.Error())
}

func (s *S) TestUninstallInfo(c *check.C) {
	c.Assert((&Uninstall{}).Info(), check.NotNil)
}

func (s *S) TestUninstallCommandFlags(c *check.C) {
	command := Uninstall{}
	flags := command.Flags()
	c.Assert(flags, check.NotNil)
	flags.Parse(true, []string{"-c", "my-conf.yml"})
	config := flags.Lookup("c")
	usage := "Configuration file"
	c.Check(config, check.NotNil)
	c.Check(config.Name, check.Equals, "c")
	c.Check(config.Usage, check.Equals, usage)
	c.Check(config.Value.String(), check.Equals, "my-conf.yml")
	c.Check(config.DefValue, check.Equals, "")
	config = flags.Lookup("config")
	c.Check(config, check.NotNil)
	c.Check(config.Name, check.Equals, "config")
	c.Check(config.Usage, check.Equals, usage)
	c.Check(config.Value.String(), check.Equals, "my-conf.yml")
	c.Check(config.DefValue, check.Equals, "")
}

func (s *S) TestAddInstallHosts(c *check.C) {
	os.Setenv("TSURU_TARGET", "http://localhost")
	defer os.Unsetenv("TSURU_TARGET")
	var called bool
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status: http.StatusCreated,
		},
		CondFunc: func(r *http.Request) bool {
			called = true
			var driver map[string]interface{}
			err := json.Unmarshal([]byte(r.FormValue("driver")), &driver)
			c.Assert(err, check.IsNil)
			c.Assert(driver["MockIP"], check.Equals, "127.0.0.1")
			return r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/install/hosts")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	machines := []*dockermachine.Machine{
		{Base: &iaas.Machine{}, Host: &host.Host{DriverName: "amazonec2", Driver: &fakedriver.Driver{MockIP: "127.0.0.1"}}},
	}
	err := addInstallHosts(machines, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
}

func (s *S) TestInstallHostList(c *check.C) {
	os.Setenv("TSURU_TARGET", "http://localhost")
	defer os.Unsetenv("TSURU_TARGET")
	var buf bytes.Buffer
	var called bool
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status: http.StatusOK,
			Message: `[{"Name":"host1", "DriverName": "generic", "Driver": {"IP": "127.0.0.1"}},
				{"Name":"host2", "DriverName":"generic", "Driver": {"SSHPort": 9999, "IP": "127.0.0.2"}}]`,
		},
		CondFunc: func(r *http.Request) bool {
			called = true
			return r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/install/hosts")
		},
	}
	context := cmd.Context{Stdout: &buf}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	cmd := &InstallHostList{}
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := `+-------+-------------+---------+---------------------+
| Name  | Driver Name | State   | Driver              |
+-------+-------------+---------+---------------------+
| host1 | generic     | Stopped | {                   |
|       |             |         |  "IP": "127.0.0.1"  |
|       |             |         | }                   |
+-------+-------------+---------+---------------------+
| host2 | generic     | Stopped | {                   |
|       |             |         |  "IP": "127.0.0.2", |
|       |             |         |  "SSHPort": 9999    |
|       |             |         | }                   |
+-------+-------------+---------+---------------------+
`
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestInstallConfigInit(c *check.C) {
	var buf bytes.Buffer
	d, err := ioutil.TempDir("", "installer")
	c.Assert(err, check.IsNil)
	context := cmd.Context{
		Stdout: &buf,
		Args:   []string{filepath.Join(d, "config.yml"), filepath.Join(d, "compose.yml")},
	}
	command := InstallConfigInit{}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	compose, err := ioutil.ReadFile(filepath.Join(d, "compose.yml"))
	c.Assert(err, check.IsNil)
	c.Assert(string(compose), check.DeepEquals, defaultconfig.Compose)
	opts, err := parseConfigFile(filepath.Join(d, "config.yml"))
	c.Assert(err, check.IsNil)
	c.Assert(opts, check.DeepEquals, defaultInstallOpts)
}
