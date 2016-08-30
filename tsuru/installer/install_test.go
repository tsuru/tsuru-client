// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/tsuru/tsuru/cmd"

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
	conf := `
name: tsuru-test
hosts: 2
ca-path: /tmp/certs
driver:
    name: amazonec2
    options:
        opt1: option1-value
`
	err := ioutil.WriteFile("/tmp/config.yml", []byte(conf), 0644)
	if err != nil {
		c.Fatal("Failed to write config file for test")
	}
	defer os.Remove("/tmp/config.yml")
	expected := &TsuruInstallConfig{
		DockerMachineConfig: &DockerMachineConfig{
			DriverName: "amazonec2",
			DriverOpts: map[string]interface{}{
				"opt1": "option1-value",
			},
			CAPath: "/tmp/certs",
			Name:   "tsuru-test",
		},
		NumHosts: 2,
	}
	dmConfig, err := parseConfigFile("/tmp/config.yml")
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
	expectedErr := "tsuru target \"test\" already exists"
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
