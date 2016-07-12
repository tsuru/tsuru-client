// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"net/http"

	dockertesting "github.com/fsouza/go-dockerclient/testing"
	"github.com/tsuru/tsuru/cmd"
	"gopkg.in/check.v1"
)

var manager *cmd.Manager

func (s *S) TestInstallInfo(c *check.C) {
	c.Assert((&Install{}).Info(), check.NotNil)
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
	command := Install{driverName: "none"}
	command.Run(&context, client)
	c.Assert(stdout.String(), check.Not(check.Equals), "")
	c.Assert(stderr.String(), check.Equals, "")
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
