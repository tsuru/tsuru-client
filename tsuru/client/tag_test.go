// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"net/http"

	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestTagListInfo(c *check.C) {
	c.Assert((&TagList{}).Info(), check.NotNil)
}

func (s *S) TestTagListWithApps(c *check.C) {
	var stdout, stderr bytes.Buffer
	appList := `[{"name":"app1","tags":["tag1"]},{"name":"app2","tags":["tag2","tag3"]},{"name":"app3","tags":[]},{"name":"app4","tags":["tag1","tag3"]}]`
	serviceList := "[]"
	expected := `+------+------+-------------------+
| Tag  | Apps | Service Instances |
+------+------+-------------------+
| tag1 | app1 |                   |
|      | app4 |                   |
+------+------+-------------------+
| tag2 | app2 |                   |
+------+------+-------------------+
| tag3 | app2 |                   |
|      | app4 |                   |
+------+------+-------------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := TagList{}
	s.setupFakeTransport(makeTransport([]string{appList, serviceList}))
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTagListWithServiceInstances(c *check.C) {
	var stdout, stderr bytes.Buffer
	appList := "[]"
	serviceList := `[{"service":"service1","service_instances":[{"name":"instance1","tags":["tag1"]},{"name":"instance2","tags":[]},{"name":"instance3","tags":["tag1","tag2"]}]},{"service":"service2","service_instances":[{"name":"instance4","tags":["tag1"]}]}]`
	expected := `+------+------+---------------------+
| Tag  | Apps | Service Instances   |
+------+------+---------------------+
| tag1 |      | service1: instance1 |
|      |      | service1: instance3 |
|      |      | service2: instance4 |
+------+------+---------------------+
| tag2 |      | service1: instance3 |
+------+------+---------------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := TagList{}
	s.setupFakeTransport(makeTransport([]string{appList, serviceList}))
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTagListWithAppsAndServiceInstances(c *check.C) {
	var stdout, stderr bytes.Buffer
	appList := `[{"name":"app1","tags":["tag1"]},{"name":"app2","tags":["tag2","tag3"]},{"name":"app3","tags":[]},{"name":"app4","tags":["tag1","tag3"]}]`
	serviceList := `[{"service":"service1","service_instances":[{"name":"instance1","tags":["tag1"]},{"name":"instance2","tags":[]},{"name":"instance3","tags":["tag1","tag2"]}]}]`
	expected := `+------+------+---------------------+
| Tag  | Apps | Service Instances   |
+------+------+---------------------+
| tag1 | app1 | service1: instance1 |
|      | app4 | service1: instance3 |
+------+------+---------------------+
| tag2 | app2 | service1: instance3 |
+------+------+---------------------+
| tag3 | app2 |                     |
|      | app4 |                     |
+------+------+---------------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := TagList{}
	s.setupFakeTransport(makeTransport([]string{appList, serviceList}))
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTagListWithEmptyResponse(c *check.C) {
	var stdout, stderr bytes.Buffer
	appList := `[{"name":"app1","tags":[]}]`
	serviceList := `[{"service_instances":[{"name":"service1","tags":[]}]}]`
	expected := ""
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := TagList{}
	s.setupFakeTransport(makeTransport([]string{appList, serviceList}))
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTagListRequestError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := TagList{}
	s.setupFakeTransport(&cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusBadGateway},
		CondFunc:  func(*http.Request) bool { return true },
	})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "502 Bad Gateway")
	c.Assert(stdout.String(), check.Equals, "")
}

func makeTransport(messages []string) http.RoundTripper {
	trueFunc := func(*http.Request) bool { return true }
	cts := make([]cmdtest.ConditionalTransport, len(messages))
	for i, message := range messages {
		cts[i] = cmdtest.ConditionalTransport{
			Transport: cmdtest.Transport{Message: message, Status: http.StatusOK},
			CondFunc:  trueFunc,
		}
	}

	return &cmdtest.MultiConditionalTransport{ConditionalTransports: cts}
}
