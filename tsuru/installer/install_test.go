// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"

	"gopkg.in/check.v1"
)

func (s *S) TestParseConfigDefaultConfig(c *check.C) {
	dmConfig, err := parseConfigFile("")
	c.Assert(err, check.IsNil)
	c.Assert(dmConfig, check.DeepEquals, defaultDockerMachineConfig)
}

func (s *S) TestParseConfigFileNotExists(c *check.C) {
	_, err := parseConfigFile("not-exist-conf.yml")
	c.Assert(err, check.NotNil)
}

func (s *S) TestParseConfigFile(c *check.C) {
	conf := `
name: tsuru-test
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
	expected := &DockerMachineConfig{
		DriverName: "amazonec2",
		DriverOpts: map[string]interface{}{
			"opt1": "option1-value",
		},
		CAPath: "/tmp/certs",
		Name:   "tsuru-test",
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
	manager := cmd.NewManager("test", "1.0.0", "Supported-Tsuru", &stdout, &stderr, os.Stdin, nil)
	command := Install{}
	command.Flags().Parse(true, []string{"-c", "./testdata/wrong-conf.yml"})
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Ok", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	expectedErr := "tsuru target \"test\" already exists"
	err := command.Run(&context, client)
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
