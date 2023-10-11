// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestPlanListInfo(c *check.C) {
	c.Assert((&PlanList{}).Info(), check.NotNil)
}

func (s *S) TestPlanListBytes(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test",  "memory": 536870912, "cpumilli": 100, "default": false},
	{"name": "test2", "memory": 536870912, "cpumilli": 200, "default": true}
]`
	expected := `+-------+-----+-----------+---------+
| Name  | CPU | Memory    | Default |
+-------+-----+-----------+---------+
| test  | 10% | 536870912 | false   |
| test2 | 20% | 536870912 | true    |
+-------+-----+-----------+---------+
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
	command := PlanList{}
	command.Flags().Parse(true, []string{"-b"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListHuman(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test",  "memory": 536870912, "cpumilli": 100, "default": false},
	{"name": "test2", "memory": 536870912, "cpumilli": 200, "default": true}
]`
	expected := `+-------+-----+--------+---------+
| Name  | CPU | Memory | Default |
+-------+-----+--------+---------+
| test  | 10% | 512Mi  | false   |
| test2 | 20% | 512Mi  | true    |
+-------+-----+--------+---------+
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
	command := PlanList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListKubernetesFriendly(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test", "cpumilli": 300, "memory": 536870912, "default": false}
]`
	expected := `+------+---------------------+------------------------+---------+
| Name | CPU requests/limits | Memory requests/limits | Default |
+------+---------------------+------------------------+---------+
| test | 300m                | 512Mi                  | false   |
+------+---------------------+------------------------+---------+
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
	command := PlanList{k8sFriendly: true}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListOverride(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test",  "memory": 536870912, "default": false, "override": {"cpumilli": 300, "memory": 268435456}}
]`
	expected := `+------+----------------+------------------+---------+
| Name | CPU            | Memory           | Default |
+------+----------------+------------------+---------+
| test | 30% (override) | 256Mi (override) | false   |
+------+----------------+------------------+---------+
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
	command := PlanList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListWithBurst(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test", "cpumilli": 300, "memory": 536870912, "default": false, "cpuBurst": {"default": 1.1}}
]`
	expected := `+------+-----+--------+---------------------+---------+
| Name | CPU | Memory | CPU Burst (default) | Default |
+------+-----+--------+---------------------+---------+
| test | 30% | 512Mi  | up to 33%           | false   |
+------+-----+--------+---------------------+---------+
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
	command := PlanList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListWithBurstKubernetesFriendly(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test", "cpumilli": 300, "memory": 536870912, "default": false, "cpuBurst": {"default": 1.1}}
]`
	expected := `+------+--------------+------------+------------------------+---------+
| Name | CPU requests | CPU limits | Memory requests/limits | Default |
+------+--------------+------------+------------------------+---------+
| test | 300m         | 330m       | 512Mi                  | false   |
+------+--------------+------------+------------------------+---------+
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
	command := PlanList{k8sFriendly: true}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListWithBurstAndMaxAllowed(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test", "cpumilli": 300, "memory": 536870912, "default": false, "cpuBurst": {"default": 1.1, "maxAllowed": 2}}
]`
	expected := `+------+-----+--------+---------------------+------------------------------+---------+
| Name | CPU | Memory | CPU Burst (default) | CPU Burst (max customizable) | Default |
+------+-----+--------+---------------------+------------------------------+---------+
| test | 30% | 512Mi  | up to 33%           | up to 60%                    | false   |
+------+-----+--------+---------------------+------------------------------+---------+
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
	command := PlanList{
		showMaxBurstAllowed: true,
	}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListWithBurstOverride(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test", "cpumilli": 300, "memory": 536870912, "default": false, "cpuBurst": {"default": 1.1}, "override": {"cpuBurst": 1.2}}
]`
	expected := `+------+-----+--------+----------------------+---------+
| Name | CPU | Memory | CPU Burst (default)  | Default |
+------+-----+--------+----------------------+---------+
| test | 30% | 512Mi  | up to 36% (override) | false   |
+------+-----+--------+----------------------+---------+
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
	command := PlanList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListCPUMilli(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test",  "memory": 536870912, "cpumilli": 500, "default": false, "override": {"cpumilli": null, "memory": null}}
]`
	expected := `+------+-----+--------+---------+
| Name | CPU | Memory | Default |
+------+-----+--------+---------+
| test | 50% | 512Mi  | false   |
+------+-----+--------+---------+
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
	command := PlanList{}
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
	command := PlanList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "No plans available.\n")
}
