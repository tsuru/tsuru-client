// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"net/http"

	docker "github.com/fsouza/go-dockerclient"
	dockertesting "github.com/fsouza/go-dockerclient/testing"
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
	server, _ := dockertesting.NewServer("127.0.0.1:2375", nil, nil)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"url=http://127.0.0.1"},
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := Install{driverName: "none"}
	command.Run(&context, client)
	c.Assert(stdout.String(), check.Not(check.Equals), "")
	c.Assert(stderr.String(), check.Equals, "")
	server.Stop()
}

func (s *S) TestInstallCustomRegistry(c *check.C) {
	iChan := make(chan *InstallConfig, 1)
	TsuruComponents = []TsuruComponent{&TestComponent{iChan: iChan}}
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"-r", "myregistry.com", "url=http://127.0.0.1"},
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := Install{driverName: "none", registry: "myregistry.com"}
	command.Run(&context, client)
	config := <-iChan
	c.Assert(config.Registry, check.Equals, "myregistry.com")
	TsuruComponents = realTsuruComponents
}

func (s *S) TestUninstallInfo(c *check.C) {
	c.Assert((&Uninstall{}).Info(), check.NotNil)
}

func (s *S) TestUninstall(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"url=http://1.2.3.4"},
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := Uninstall{driverName: "none"}
	command.Run(&context, client)
	c.Assert(stderr.String(), check.Equals, "")
	c.Assert(stdout.String(), check.Equals, "Machine successfully removed!\n")
}
