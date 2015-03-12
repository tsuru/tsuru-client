// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
)

func (s *S) TestAppRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "http.go		http_test.go"
	context := cmd.Context{
		Args:   []string{"ls"},
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
			b := make([]byte, 2)
			req.Body.Read(b)
			return req.URL.Path == "/apps/ble/run" && string(b) == "ls"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := appRun{}
	command.Flags().Parse(true, []string{"--app", "ble"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRunShouldUseAllSubsequentArgumentsAsArgumentsToTheGivenCommand(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "-rw-r--r--  1 f  staff  119 Apr 26 18:23 http.go\n"
	context := cmd.Context{
		Args:   []string{"ls", "-l"},
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
			b := make([]byte, 5)
			req.Body.Read(b)
			return req.URL.Path == "/apps/ble/run" && string(b) == "ls -l"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := appRun{}
	command.Flags().Parse(true, []string{"--app", "ble"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected+expected)
}

func (s *S) TestAppRunWithoutTheFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "-rw-r--r--  1 f  staff  119 Apr 26 18:23 http.go"
	context := cmd.Context{
		Args:   []string{"ls", "-lh"},
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
			b := make([]byte, 6)
			req.Body.Read(b)
			return req.URL.Path == "/apps/bla/run" && string(b) == "ls -lh"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "bla"}
	command := appRun{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, nil)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRunShouldReturnErrorWhenCommandGoWrong(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"cmd_error"},
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
			return req.URL.Path == "/apps/bla/run"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "bla"}
	command := appRun{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, nil)
	err = command.Run(&context, client)
	c.Assert(err, check.ErrorMatches, "command doesn't exist.")
}

func (s *S) TestAppRunInfo(c *check.C) {
	command := appRun{}
	c.Assert(command.Info(), check.NotNil)
}
