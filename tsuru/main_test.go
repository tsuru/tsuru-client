// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gopkg.in/check.v1"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
)

type S struct{}

func (s *S) SetUpSuite(c *check.C) {
	os.Setenv("TSURU_TARGET", "http://localhost:8080")
	os.Setenv("TSURU_TOKEN", "sometoken")
}

func (s *S) TearDownSuite(c *check.C) {
	os.Unsetenv("TSURU_TARGET")
	os.Unsetenv("TSURU_TOKEN")
}

var _ = check.Suite(&S{})
var manager *cmd.ManagerV2

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) SetUpTest(c *check.C) {
	manager = buildManager(os.Stdout, os.Stderr)
}

func (s *S) TestCommandsRegistered(c *check.C) {
	commands := []string{
		"app-create",
		"app-remove",
		"app-list",
		"app-grant",
		"app-revoke",
		"app-log",
		"app-run",
		"app-restart",
		"env-get",
		"env-set",
		"env-unset",
		"service-list",
		"service-update",
		"service-instance-update",
		"service-instance-add",
		"service-instance-remove",
		"service-instance-bind",
		"service-instance-unbind",
		"service-instance-info",
		"service-info",
		"app-info",
		"unit-add",
		"unit-remove",
		"unit-set",
		"cname-add",
		"cname-remove",
		"platform-list",
		"platform-add",
		"app-start",
		"plugin-install",
		"plugin-remove",
		"plugin-list",
		"plugin-bundle",
		"app-stop",
		"app-deploy",
		"app-deploy-rollback",
		"app-deploy-rollback-update",
		"plan-list",
		"change-password",
		"reset-password",
		"user-remove",
		"user-list",
		"team-remove",
		"user-create",
		"team-create",
		"team-list",
		"team-info",
		"pool-list",
		"app-update",
		"service-create",
		"service-destroy",
		"service-doc-get",
	}

	cobraCmd := manager.Cobra()
	cobraCommands := cobraCmd.Commands()

	foundCommands := make(map[string]bool)
	for _, c := range cobraCommands {
		parts := strings.Split(c.Use, " ")
		foundCommands[parts[0]] = true
	}

	for _, name := range commands {
		_, found := foundCommands[name]
		c.Assert(found, check.Equals, true, check.Commentf("command %q not found", name))
	}
}

func (s *S) TestVersion(c *check.C) {
	command := versionCmd{}
	context := cmd.Context{
		Args:   []string{},
		Stdout: &bytes.Buffer{},
	}
	command.Run(&context)
	c.Assert(context.Stdout.(*bytes.Buffer).String(), check.Matches, "Client version: dev.\n.*")
}

func (s *S) TestVersionInfo(c *check.C) {
	expected := &cmd.Info{
		Name: "version",
		Desc: "display the current version",
	}
	c.Assert((&versionCmd{}).Info(), check.DeepEquals, expected)
}

func (s *S) TestVersionWithAPI(c *check.C) {

	command := versionCmd{}
	context := cmd.Context{
		Args:   []string{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{"version":"1.7.4"}`))
	}))
	defer ts.Close()

	os.Setenv("TSURU_TARGET", ts.URL)
	defer os.Unsetenv("TSURU_TARGET")
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(context.Stdout.(*bytes.Buffer).String(),
		check.Equals, "Client version: dev.\nServer version: 1.7.4.\n")
}

func (s *S) TestVersionAPIInvalidURL(c *check.C) {
	tsuruHTTP.AuthenticatedClient = &http.Client{}
	command := versionCmd{}
	context := cmd.Context{
		Args:   []string{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	URL := "notvalid.test"
	os.Setenv("TSURU_TARGET", URL)
	defer os.Unsetenv("TSURU_TARGET")
	err := command.Run(&context)
	c.Assert(true, check.Equals, strings.Contains(err.Error(), "Unable to retrieve server version"))
	c.Assert(true, check.Equals, strings.Contains(err.Error(), "no such host"))

	stdout := context.Stdout.(*bytes.Buffer).String()

	c.Assert(stdout, check.Matches, "Client version: dev.\n")
}
