// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestPlanListInfo(c *check.C) {
	c.Assert((&planList{}).Info(), check.NotNil)
}

func (s *S) TestPlanListBytes(c *check.C) {
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
			return strings.HasSuffix(req.URL.Path, "/plans") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := planList{}
	command.Flags().Parse(true, []string{"-b"})
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
			return strings.HasSuffix(req.URL.Path, "/plans") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := planList{}
	// command.Flags().Parse(true, []string{"-h"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListEmpty(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/plans") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := planList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "No plans available.\n")
}
