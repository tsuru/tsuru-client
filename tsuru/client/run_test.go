// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
)

func (s *S) TestAppRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "http.go		http_test.go"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	msg := io.SimpleJsonMessage{Message: expected}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: string(result),
			Status:  http.StatusOK,
		},
		CondFunc: func(req *http.Request) bool {
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			cmd := req.FormValue("command") == "ls"
			path := strings.HasSuffix(req.URL.Path, "/apps/ble/run")
			return path && cmd && contentType
		},
	}
	s.setupFakeTransport(trans)
	command := AppRun{}
	err = command.Flags().Parse(true, []string{"--app", "ble", "ls"})
	c.Assert(err, check.IsNil)

	context.Args = command.Flags().Args()
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRunFlagIsolated(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "http.go		http_test.go"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	msg := io.SimpleJsonMessage{Message: expected}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: string(result),
			Status:  http.StatusOK,
		},
		CondFunc: func(req *http.Request) bool {
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			cmd := req.FormValue("isolated") == "true"
			path := strings.HasSuffix(req.URL.Path, "/apps/ble/run")
			return path && cmd && contentType
		},
	}
	s.setupFakeTransport(trans)
	command := AppRun{}
	err = command.Flags().Parse(true, []string{"--app", "ble", "--isolated", "ls"})
	c.Assert(err, check.IsNil)

	context.Args = command.Flags().Args()
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRunShouldUseAllSubsequentArgumentsAsArgumentsToTheGivenCommand(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "-rw-r--r--  1 f  staff  119 Apr 26 18:23 http.go\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	msg := io.SimpleJsonMessage{Message: expected}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: string(result) + "\n" + string(result),
			Status:  http.StatusOK,
		},
		CondFunc: func(req *http.Request) bool {
			cmd := req.FormValue("command") == "ls -l"
			path := strings.HasSuffix(req.URL.Path, "/apps/ble/run")
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return cmd && path && contentType
		},
	}
	s.setupFakeTransport(trans)
	command := AppRun{}
	err = command.Flags().Parse(true, []string{"--app", "ble", "ls -l"})

	c.Assert(err, check.IsNil)

	context.Args = command.Flags().Args()

	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected+expected)
}

func (s *S) TestAppRunWithoutTheFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "-rw-r--r--  1 f  staff  119 Apr 26 18:23 http.go"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	msg := io.SimpleJsonMessage{Message: expected}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: string(result),
			Status:  http.StatusOK,
		},
		CondFunc: func(req *http.Request) bool {
			path := strings.HasSuffix(req.URL.Path, "/apps/bla/run")
			cmd := req.FormValue("command") == "ls -lh"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && cmd && contentType
		},
	}
	s.setupFakeTransport(trans)
	command := AppRun{}
	err = command.Flags().Parse(true, []string{"-a", "bla", "ls -lh"})
	c.Assert(err, check.IsNil)

	context.Args = command.Flags().Args()
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRunShouldReturnErrorWhenCommandGoWrong(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	msg := io.SimpleJsonMessage{Error: "command doesn't exist."}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: string(result),
			Status:  http.StatusOK,
		},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/bla/run")
		},
	}
	s.setupFakeTransport(trans)
	command := AppRun{}
	err = command.Flags().Parse(true, []string{"-a", "bla", "cmd_error"})
	c.Assert(err, check.IsNil)

	context.Args = command.Flags().Args()

	err = command.Run(&context)
	c.Assert(err, check.ErrorMatches, "command doesn't exist.")
}

func (s *S) TestAppRunInfo(c *check.C) {
	command := AppRun{}
	c.Assert(command.Info(), check.NotNil)
}
