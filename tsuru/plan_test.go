// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	tsuruapp "github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
)

func (s *S) TestPlanListInfo(c *check.C) {
	c.Assert((&planList{}).Info(), check.NotNil)
}

func (s *S) TestPlanList(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
    {"name": "test",  "memory": 536870912, "swap": 268435456, "cpushare": 100, "router": "r1", "default": false},
    {"name": "test2", "memory": 536870912, "swap": 268435456, "cpushare": 200, "router": "r2", "default": true}
]`
	expected := `+-------+-----------+-----------+-----------+--------+---------+
| Name  | Memory    | Swap      | Cpu Share | Router | Default |
+-------+-----------+-----------+-----------+--------+---------+
| test  | 536870912 | 268435456 | 100       | r1     | false   |
| test2 | 536870912 | 268435456 | 200       | r2     | true    |
+-------+-----------+-----------+-----------+--------+---------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/plans" && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := planList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListHuman(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
    {"name": "test",  "memory": 536870912, "swap": 268435456, "cpushare": 100, "default": false},
    {"name": "test2", "memory": 536870912, "swap": 268435456, "cpushare": 200, "default": true}
]`
	expected := `+-------+--------+--------+-----------+--------+---------+
| Name  | Memory | Swap   | Cpu Share | Router | Default |
+-------+--------+--------+-----------+--------+---------+
| test  | 512 MB | 256 MB | 100       |        | false   |
| test2 | 512 MB | 256 MB | 200       |        | true    |
+-------+--------+--------+-----------+--------+---------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/plans" && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := planList{}
	command.Flags().Parse(true, []string{"-h"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppPlanChange(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"hiperplan"},
	}
	expectedOut := "-- plan changed --"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			var plan tsuruapp.Plan
			json.NewDecoder(req.Body).Decode(&plan)
			c.Assert(plan.Name, check.Equals, "hiperplan")
			return req.URL.Path == "/apps/handful_of_nothing/plan" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var command appPlanChange
	command.Flags().Parse(true, []string{"--app", "handful_of_nothing", "-y"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestAppPlanChangeAsk(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
		Args:   []string{"hiperplan"},
	}
	expectedOut := "-- plan changed --"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			var plan tsuruapp.Plan
			json.NewDecoder(req.Body).Decode(&plan)
			c.Assert(plan.Name, check.Equals, "hiperplan")
			return req.URL.Path == "/apps/handful_of_nothing/plan" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var command appPlanChange
	command.Flags().Parse(true, []string{"--app", "handful_of_nothing"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expectedOut = `Are you sure you want to change the plan of the application "handful_of_nothing" to "hiperplan"? (y/n) -- plan changed --`
	c.Assert(stdout.String(), check.Equals, expectedOut)
}
