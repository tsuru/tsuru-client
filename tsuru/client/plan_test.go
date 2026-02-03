// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	appTypes "github.com/tsuru/tsuru/types/app"
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
	s.setupFakeTransport(trans)
	command := PlanList{}
	command.Flags().Parse([]string{"-b"})
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanList{}
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanList{k8sFriendly: true}
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanList{}
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanList{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlanListWithBurstAndCPUOverrided(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
	{"name": "test", "cpumilli": 300, "memory": 536870912, "default": false, "cpuBurst": {"default": 1.1}, "override": {"cpumilli": 1000}}
]`
	expected := `+------+-----------------+--------+---------------------+---------+
| Name | CPU             | Memory | CPU Burst (default) | Default |
+------+-----------------+--------+---------------------+---------+
| test | 100% (override) | 512Mi  | up to 110%          | false   |
+------+-----------------+--------+---------------------+---------+
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
	s.setupFakeTransport(trans)
	command := PlanList{}
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanList{k8sFriendly: true}
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanList{
		showMaxBurstAllowed: true,
	}
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanList{}
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanList{}
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanList{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "No plans available.\n")
}

func (s *S) TestRenderProcessPlan(c *check.C) {
	cases := []struct {
		appPlan        appTypes.Plan
		planByProcess  map[string]string
		expectedResult string
	}{
		{
			appPlan: appTypes.Plan{
				Name: "c1m2",
			},
			planByProcess: map[string]string{
				"worker": "c1m8",
				"web":    "c1m4",
			},
			expectedResult: "" +
				"+-----------+------+\n" +
				"| Process   | Plan |\n" +
				"+-----------+------+\n" +
				"| (default) | c1m2 |\n" +
				"| web       | c1m4 |\n" +
				"| worker    | c1m8 |\n" +
				"+-----------+------+\n",
		},
		{
			appPlan: appTypes.Plan{
				Name: "c1m2",
				Override: &appTypes.PlanOverride{
					CPUMilli: func(d int) *int { return &d }(1000),
					Memory:   func(d int64) *int64 { return &d }(1024 * 1024 * 1024),
				},
			},
			planByProcess: map[string]string{
				"worker": "c1m8",
				"web":    "c1m4",
			},
			expectedResult: "" +
				"+-----------+------+------------------------+\n" +
				"| Process   | Plan | Overrides              |\n" +
				"+-----------+------+------------------------+\n" +
				"| (default) | c1m2 | CPU: 100%, Memory: 1Gi |\n" +
				"| web       | c1m4 |                        |\n" +
				"| worker    | c1m8 |                        |\n" +
				"+-----------+------+------------------------+\n",
		},
	}

	for _, cc := range cases {
		output := renderProcessPlan(cc.appPlan, cc.planByProcess)
		c.Assert(cc.expectedResult, check.Equals, output)
	}
}

func (s *S) TestRenderPlansWithoutDefaultColumn(c *check.C) {
	plans := []appTypes.Plan{
		{
			Name:     "test",
			CPUMilli: 300,
			Memory:   536870912,
			CPUBurst: &appTypes.CPUBurst{Default: 1.1},
		},
	}
	expected := `+------+-----+--------+-----------+
| Name | CPU | Memory | CPU Burst |
+------+-----+--------+-----------+
| test | 30% | 512Mi  | up to 33% |
+------+-----+--------+-----------+
`
	result := renderPlans(plans, renderPlansOpts{showDefaultColumn: false})
	c.Assert(result, check.Equals, expected)
}

func (s *S) TestRenderPlansWithDefaultColumn(c *check.C) {
	plans := []appTypes.Plan{
		{
			Name:     "test",
			CPUMilli: 300,
			Memory:   536870912,
			CPUBurst: &appTypes.CPUBurst{Default: 1.1},
		},
	}
	expected := `+------+-----+--------+---------------------+---------+
| Name | CPU | Memory | CPU Burst (default) | Default |
+------+-----+--------+---------------------+---------+
| test | 30% | 512Mi  | up to 33%           | false   |
+------+-----+--------+---------------------+---------+
`
	result := renderPlans(plans, renderPlansOpts{showDefaultColumn: true})
	c.Assert(result, check.Equals, expected)
}
