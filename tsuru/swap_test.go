// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/testing"
	"launchpad.net/gocheck"
)

func (s *S) TestSwapInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "app-swap",
		Usage:   "app-swap <app1-name> <app2-name> [-f/--force]",
		Desc:    "Swap routes between two apps. Use force if you want to swap apps with different numbers of units or diferent platform without confirmation",
		MinArgs: 2,
	}
	command := Swap{}
	c.Assert(command.Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestSwap(c *gocheck.C) {
	var buf bytes.Buffer
	var called bool
	transport := testing.ConditionalTransport{
		Transport: testing.Transport{Status: http.StatusOK, Message: ""},
		CondFunc: func(r *http.Request) bool {
			called = true
			return r.Method == "PUT" && r.URL.Path == "/swap"
		},
	}
	context := cmd.Context{
		Args:   []string{"app1", "app2"},
		Stdout: &buf,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := Swap{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	expected := "Apps successfully swapped!\n"
	c.Assert(buf.String(), gocheck.Equals, expected)
}

func (s *S) TestSwapWhenAppsAreNotEqual(c *gocheck.C) {
	var buf bytes.Buffer
	var called int
	stdin := bytes.NewBufferString("yes")
	transportError := testing.ConditionalTransport{
		Transport: testing.Transport{Status: http.StatusUnauthorized, Message: "Apps are not equal."},
		CondFunc: func(r *http.Request) bool {
			called += 1
			return r.URL.RawQuery == "app1=app1&app2=app2&force=false"
		},
	}
	transportOk := testing.ConditionalTransport{
		Transport: testing.Transport{Status: http.StatusOK, Message: ""},
		CondFunc: func(r *http.Request) bool {
			called += 1
			return r.URL.RawQuery == "app1=app1&app2=app2&force=true"
		},
	}
	multiTransport := testing.MultiConditionalTransport{
		ConditionalTransports: []testing.ConditionalTransport{transportError, transportOk},
	}
	context := cmd.Context{
		Args:   []string{"app1", "app2"},
		Stdout: &buf,
		Stdin:  stdin,
	}
	client := cmd.NewClient(&http.Client{Transport: &multiTransport}, nil, manager)
	command := Swap{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, 2)
}

func (s *S) TestAnswerAcceptable(c *gocheck.C) {
	answersOptions := []string{"y", "yes"}
	isAcceptable := answerAcceptable("y", answersOptions)
	c.Assert(isAcceptable, gocheck.Equals, true)
	isAcceptable = answerAcceptable("no", answersOptions)
	c.Assert(isAcceptable, gocheck.Equals, false)
}

func (s *S) TestSwapIsACommand(c *gocheck.C) {
	var _ cmd.Command = &Swap{}
}
