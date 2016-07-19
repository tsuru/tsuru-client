// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"net/http"
	"os"

	docker "github.com/fsouza/go-dockerclient"
	dockertesting "github.com/fsouza/go-dockerclient/testing"
	"github.com/tsuru/config"
	"github.com/tsuru/tsuru/cmd"
	"gopkg.in/check.v1"
)

var manager *cmd.Manager

var realTsuruComponents = TsuruComponents

type TestComponent struct {
	iChan chan<- *InstallConfig
}

func (c *TestComponent) Name() string {
	return "test-component"
}

func (c *TestComponent) Install(m *Machine, i *InstallConfig) error {
	c.iChan <- i
	return nil
}

func (c *TestComponent) Status(m *Machine) (*ComponentStatus, error) {
	return &ComponentStatus{
		containerState: &docker.State{Running: true},
		addresses:      []string{"127.0.0.1:123"},
	}, nil
}

func (s *S) TestInstallInfo(c *check.C) {
	c.Assert((&Install{}).Info(), check.NotNil)
}

func (s *S) TestInstall(c *check.C) {
	config.Set("driver:name", "none")
	config.Set("driver:options:url", "http://127.0.0.1")
	err := config.WriteConfigFile("/tmp/config-test.yaml", 0644)
	if err != nil {
		c.Fatal(err)
	}
	defer os.Remove("/tmp/config-test.yaml")
	config.Unset("driver")
	server, _ := dockertesting.NewServer("127.0.0.1:2375", nil, nil)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := Install{config: "/tmp/config-test.yaml"}
	command.Run(&context, client)
	c.Assert(stdout.String(), check.Not(check.Equals), "")
	c.Assert(stderr.String(), check.Equals, "")
	server.Stop()
}

func (s *S) TestInstallCustomRegistry(c *check.C) {
	config.Set("driver:name", "none")
	config.Set("driver:options:url", "http://127.0.0.1")
	config.Set("registry", "myregistry.com")
	err := config.WriteConfigFile("/tmp/config-test.yaml", 0644)
	if err != nil {
		c.Fatal(err)
	}
	defer os.Remove("/tmp/config-test.yaml")
	config.Unset("driver")
	config.Unset("registry")
	iChan := make(chan *InstallConfig, 1)
	TsuruComponents = []TsuruComponent{&TestComponent{iChan: iChan}}
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"-r", "myregistry.com", "url=http://127.0.0.1"},
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := Install{config: "/tmp/config-test.yaml"}
	command.Run(&context, client)
	config := <-iChan
	c.Assert(config.Registry, check.Equals, "myregistry.com")
	TsuruComponents = realTsuruComponents
}

func (s *S) TestUninstallInfo(c *check.C) {
	c.Assert((&Uninstall{}).Info(), check.NotNil)
}

func (s *S) TestUninstall(c *check.C) {
	config.Set("driver:name", "none")
	config.Set("driver:options:url", "http://127.0.0.1")
	config.Set("registry", "myregistry.com")
	err := config.WriteConfigFile("/tmp/config-test.yaml", 0644)
	if err != nil {
		c.Fatal(err)
	}
	defer os.Remove("/tmp/config-test.yaml")
	config.Unset("driver")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := Uninstall{config: "/tmp/config-test.yaml"}
	command.Run(&context, client)
	c.Assert(stderr.String(), check.Equals, "")
	c.Assert(stdout.String(), check.Equals, "Machine successfully removed!\n")
}
