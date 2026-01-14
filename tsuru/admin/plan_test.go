// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestPlanCreateInfo(c *check.C) {
	c.Assert((&PlanCreate{}).Info(), check.NotNil)
}
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
			cpuMilli := req.FormValue("cpumilli") == "100"
			deflt := req.FormValue("default") == "false"
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && cpuMilli && deflt
		},
	}
	s.setupFakeTransport(trans)
	command := PlanCreate{}
	command.Flags().Parse([]string{"-c", "100m"})
	err := command.Run(&context)
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
			cpuMilli := req.FormValue("cpumilli") == "100"
			deflt := req.FormValue("default") == "true"
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && cpuMilli && deflt
		},
	}
	s.setupFakeTransport(trans)
	command := PlanCreate{}
	command.Flags().Parse([]string{"-c", "10%", "-m", "4194304", "-d"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Plan successfully created!\n")
}

func (s *S) TestPlanCreateMemoryUnits(c *check.C) {
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
			c.Check(req.FormValue("cpumilli"), check.Equals, "100")
			c.Check(req.FormValue("default"), check.Equals, "true")

			return true
		},
	}
	s.setupFakeTransport(trans)
	command := PlanCreate{}
	command.Flags().Parse([]string{"-c", "100m", "-m", "100Mi", "-d"})
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanCreate{}
	command.Flags().Parse([]string{"-c", "5"})
	err := command.Run(&context)
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
			cpuMilli := req.FormValue("cpumilli") == "100"
			deflt := req.FormValue("default") == "false"
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && cpuMilli && deflt
		},
	}
	s.setupFakeTransport(trans)
	command := PlanCreate{}
	command.Flags().Parse([]string{"-c", "100", "-m", "4"})
	err := command.Run(&context)
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
			cpuMilli := req.FormValue("cpumilli") == "1000"
			deflt := req.FormValue("default") == "false"
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(req.URL.Path, "/plans")
			return method && url && contentType && name && memory && cpuMilli && deflt
		},
	}
	s.setupFakeTransport(trans)
	command := PlanCreate{}
	command.Flags().Parse([]string{"-c", "1", "-m", "4194304"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "Failed to create plan!\n")
}

func (s *S) TestPlanRemoveInfo(c *check.C) {
	c.Assert((&PlanRemove{}).Info(), check.NotNil)
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
	s.setupFakeTransport(trans)
	command := PlanRemove{}
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := PlanRemove{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "Failed to remove plan!\n")
}
