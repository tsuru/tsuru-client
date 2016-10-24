// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/docker/machine/drivers/fakedriver"
	"github.com/docker/machine/libmachine/host"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestParseConfigDefaultConfig(c *check.C) {
	dmConfig, err := parseConfigFile("")
	c.Assert(err, check.IsNil)
	c.Assert(dmConfig, check.DeepEquals, defaultTsuruInstallConfig)
}

func (s *S) TestParseConfigFileNotExists(c *check.C) {
	_, err := parseConfigFile("not-exist-conf.yml")
	c.Assert(err, check.NotNil)
}

func (s *S) TestParseConfigFile(c *check.C) {
	expected := &TsuruInstallConfig{
		DockerMachineConfig: &dm.DockerMachineConfig{
			DriverName: "amazonec2",
			DriverOpts: map[string]interface{}{
				"opt1": "option1-value",
			},
			CAPath: "/tmp/certs",
			Name:   "tsuru-test",
		},
		ComponentsConfig: &ComponentsConfig{
			TsuruAPIConfig: TsuruAPIConfig{
				TargetName:       "tsuru-test",
				RootUserEmail:    "admin@example.com",
				RootUserPassword: "admin123",
				IaaSConfig: map[string]interface{}{
					"dockermachine": map[string]interface{}{
						"ca-path": "/certs",
						"driver": map[string]interface{}{
							"name": "amazonec2",
							"options": map[string]interface{}{
								"opt1": "option1-value",
							},
						},
					},
				},
				ImageTag: "v1",
			},
			ComponentAddress: map[string]string{
				"mongo":    "",
				"redis":    "",
				"registry": "",
				"planb":    "",
			},
		},
		CoreHosts: 2,
		CoreDriversOpts: map[string][]interface{}{
			"amazonec2-region": {"us-east", "us-west"},
		},
		AppsHosts: 1,
		AppsDriversOpts: map[string][]interface{}{
			"amazonec2-tags": {"my-tag"},
		},
		DedicatedAppsHosts: true,
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

type FakeMachineProvisioner struct {
	hostsProvisioned int
}

func (p *FakeMachineProvisioner) ProvisionMachine(opts map[string]interface{}) (*dm.Machine, error) {
	p.hostsProvisioned = p.hostsProvisioned + 1
	return &dm.Machine{DriverOpts: dm.DriverOpts(opts)}, nil
}

func (s *S) TestProvisionPool(c *check.C) {
	opt1 := dm.DriverOpts{"variable-opt": "opt1"}
	opt2 := dm.DriverOpts{"variable-opt": "opt2"}
	tt := []struct {
		poolHosts           int
		dedicatedPool       bool
		machines            []*dm.Machine
		expectedProvisioned int
		expectedDriverOpts  []dm.DriverOpts
	}{
		{1, false, []*dm.Machine{{}}, 0, []dm.DriverOpts{}},
		{2, false, []*dm.Machine{{}}, 1, []dm.DriverOpts{opt1, {}}},
		{1, true, []*dm.Machine{{}}, 1, []dm.DriverOpts{opt1}},
		{2, true, []*dm.Machine{{}, {}}, 2, []dm.DriverOpts{opt1, opt2}},
		{3, true, []*dm.Machine{{}}, 3, []dm.DriverOpts{opt1, opt2, opt1}},
	}
	for ti, t := range tt {
		p := &FakeMachineProvisioner{}
		config := &TsuruInstallConfig{
			AppsHosts:          t.poolHosts,
			DedicatedAppsHosts: t.dedicatedPool,
			AppsDriversOpts: map[string][]interface{}{
				"variable-opt": {"opt1", "opt2"},
			},
		}
		machines, err := ProvisionPool(p, config, t.machines)
		c.Assert(err, check.IsNil)
		c.Assert(p.hostsProvisioned, check.Equals, t.expectedProvisioned)
		for i := 0; i < t.expectedProvisioned; i++ {
			if !reflect.DeepEqual(machines[i].DriverOpts, t.expectedDriverOpts[i]) {
				c.Errorf("Test case %d/%d failed. Expected %+v. Got %+v", ti, i, t.expectedDriverOpts[i], machines[i].DriverOpts)
			}
		}
	}
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
	machines := []*dm.Machine{
		{Host: &host.Host{DriverName: "amazonec2", Driver: &fakedriver.Driver{MockIP: "127.0.0.1"}}},
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
			Message: `[{"Name":"host1", "DriverName": "amazonec2", "Driver": {"IP": "127.0.0.1"}},
				{"Name":"host2", "DriverName":"amazonec2", "Driver": {"SSHPort": 22, "IP": "127.0.0.2"}}]`,
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
	expected := `+-------+-------------+------------------------------------------------+---------------------+
| Name  | Driver Name | State                                          | Driver              |
+-------+-------------+------------------------------------------------+---------------------+
| host1 | amazonec2   | EmptyStaticCreds: static credentials are empty | {                   |
|       |             |                                                |  "IP": "127.0.0.1"  |
|       |             |                                                | }                   |
+-------+-------------+------------------------------------------------+---------------------+
| host2 | amazonec2   | EmptyStaticCreds: static credentials are empty | {                   |
|       |             |                                                |  "IP": "127.0.0.2", |
|       |             |                                                |  "SSHPort": 22      |
|       |             |                                                | }                   |
+-------+-------------+------------------------------------------------+---------------------+
`
	c.Assert(buf.String(), check.Equals, expected)
}
