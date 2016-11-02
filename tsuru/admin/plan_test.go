// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/router"
	"gopkg.in/check.v1"
)

func (s *S) TestPlanCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplan"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			name := req.FormValue("name") == "myplan"
			memory := req.FormValue("memory") == "0"
			swap := req.FormValue("swap") == "0"
			cpuShare := req.FormValue("cpushare") == "100"
			deflt := req.FormValue("default") == "false"
			router := req.FormValue("router") == ""
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && swap && cpuShare && deflt && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanCreate{}
	command.Flags().Parse(true, []string{"-c", "100"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Plan successfully created!\n")
}

func (s *S) TestPlanCreateFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplan"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			name := req.FormValue("name") == "myplan"
			memory := req.FormValue("memory") == "4194304"
			swap := req.FormValue("swap") == "512"
			cpuShare := req.FormValue("cpushare") == "100"
			deflt := req.FormValue("default") == "true"
			router := req.FormValue("router") == "myrouter"
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && swap && cpuShare && deflt && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanCreate{}
	command.Flags().Parse(true, []string{"-c", "100", "-m", "4194304", "-s", "512", "-d", "-r", "myrouter"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Plan successfully created!\n")
}

func (s *S) TestPlanCreateMemoryAndSwapUnits(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplan"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			name := req.FormValue("name") == "myplan"
			memory := req.FormValue("memory") == "100M"
			swap := req.FormValue("swap") == "512K"
			cpuShare := req.FormValue("cpushare") == "100"
			deflt := req.FormValue("default") == "true"
			router := req.FormValue("router") == "myrouter"
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && swap && cpuShare && deflt && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanCreate{}
	command.Flags().Parse(true, []string{"-c", "100", "-m", "100M", "-s", "512K", "-d", "-r", "myrouter"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Plan successfully created!\n")
}

func (s *S) TestPlanCreateError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplan"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusConflict},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/plans") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanCreate{}
	command.Flags().Parse(true, []string{"-c", "5"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "Failed to create plan!\n")
}

func (s *S) TestPlanCreateInvalidMemory(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplan"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusBadRequest},
		CondFunc: func(req *http.Request) bool {
			name := req.FormValue("name") == "myplan"
			memory := req.FormValue("memory") == "4"
			swap := req.FormValue("swap") == "0"
			cpuShare := req.FormValue("cpushare") == "100"
			deflt := req.FormValue("default") == "false"
			router := req.FormValue("router") == ""
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && swap && cpuShare && deflt && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanCreate{}
	command.Flags().Parse(true, []string{"-c", "100", "-m", "4"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "Failed to create plan!\n")
}

func (s *S) TestPlanCreateInvalidCpushare(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplan"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusBadRequest},
		CondFunc: func(req *http.Request) bool {
			name := req.FormValue("name") == "myplan"
			memory := req.FormValue("memory") == "4194304"
			swap := req.FormValue("swap") == "0"
			cpuShare := req.FormValue("cpushare") == "1"
			deflt := req.FormValue("default") == "false"
			router := req.FormValue("router") == ""
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && swap && cpuShare && deflt && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanCreate{}
	command.Flags().Parse(true, []string{"-c", "1", "-m", "4194304"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "Failed to create plan!\n")
}

func (s *S) TestPlanRemove(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplan"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/plans/myplan") && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Plan successfully removed!\n")
}

func (s *S) TestPlanRemoveError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplan"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusInternalServerError},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/plans/myplan") && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "Failed to remove plan!\n")
}

func (s *S) TestPlanRoutersListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   nil,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	r1 := router.PlanRouter{Name: "router1", Type: "foo"}
	r2 := router.PlanRouter{Name: "router2", Type: "bar"}
	data, err := json.Marshal([]router.PlanRouter{r1, r2})
	c.Assert(err, check.IsNil)
	expected := `+---------+------+
| Name    | Type |
+---------+------+
| router1 | foo  |
+---------+------+
| router2 | bar  |
+---------+------+
`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/plans/routers") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanRoutersList{}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
