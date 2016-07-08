// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
	dockertesting "github.com/fsouza/go-dockerclient/testing"
	"github.com/tsuru/tsuru-client/tsuru/installer"
	"github.com/tsuru/tsuru/cmd"
	"gopkg.in/check.v1"
)

func TestMain(m *testing.M) {
	if os.Getenv(localbinary.PluginEnvKey) == localbinary.PluginEnvVal {
		driver := os.Getenv(localbinary.PluginEnvDriverName)
		err := installer.RunDriver(driver)
		if err != nil {
			fmt.Printf("Failed to run driver %s in test", driver)
			os.Exit(1)
		}
	} else {
		localbinary.CurrentBinaryIsDockerMachine = true
		os.Exit(m.Run())
	}
}

func (s *S) TestInstallInfo(c *check.C) {
	c.Assert((&install{}).Info(), check.NotNil)
}

func (s *S) TestInstall(c *check.C) {
	dockertesting.NewServer("127.0.0.1:2375", nil, nil)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"url=http://127.0.0.1"},
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := install{driverName: "none"}
	command.Run(&context, client)
	c.Assert(stdout.String(), check.Not(check.Equals), "")
	c.Assert(stderr.String(), check.Equals, "")
}

func (s *S) TestUninstallInfo(c *check.C) {
	c.Assert((&uninstall{}).Info(), check.NotNil)
}

func (s *S) TestUninstall(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"url=http://1.2.3.4"},
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := uninstall{driverName: "none"}
	command.Run(&context, client)
	c.Assert(stderr.String(), check.Equals, "")
	c.Assert(stdout.String(), check.Equals, "Machine successfully removed!\n")
}
