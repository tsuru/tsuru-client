// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
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
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && swap && cpuShare && deflt
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
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && swap && cpuShare && deflt
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanCreate{}
	command.Flags().Parse(true, []string{"-c", "100", "-m", "4194304", "-s", "512", "-d"})
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
			if !strings.HasSuffix(req.URL.Path, "/plans") {
				return false
			}
			c.Assert(req.Method, check.Equals, "POST")
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")

			c.Check(req.FormValue("name"), check.Equals, "myplan")
			c.Check(req.FormValue("memory"), check.Equals, "104857600")
			c.Check(req.FormValue("swap"), check.Equals, "524288")
			c.Check(req.FormValue("cpushare"), check.Equals, "100")
			c.Check(req.FormValue("default"), check.Equals, "true")

			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanCreate{}
	command.Flags().Parse(true, []string{"-c", "100", "-m", "100Mi", "-s", "512Ki", "-d"})
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
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && swap && cpuShare && deflt
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
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && swap && cpuShare && deflt
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
			return strings.HasSuffix(req.URL.Path, "/plans/myplan") && req.Method == http.MethodDelete
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
			return strings.HasSuffix(req.URL.Path, "/plans/myplan") && req.Method == http.MethodDelete
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlanRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "Failed to remove plan!\n")
}
