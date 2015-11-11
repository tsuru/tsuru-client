// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestPermissionListInfo(c *check.C) {
	c.Assert((&permissionList{}).Info(), check.NotNil)
}

func (s *S) TestPermissionListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
    {"name": "*",  "contexts": ["a"]},
    {"name": "app",  "contexts": ["a", "b"]},
    {"name": "app.deploy",  "contexts": ["b"]},
    {"name": "other",  "contexts": ["zzz"]}
]`
	expected := `+------------+----------+
| Name       | Contexts |
+------------+----------+
| *          | a        |
| app        | a, b     |
| app.deploy | b        |
| other      | zzz      |
+------------+----------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/permissions" && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := permissionList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRoleAddInfo(c *check.C) {
	c.Assert((&roleAdd{}).Info(), check.NotNil)
}

func (s *S) TestRoleAddRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole", "app"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/roles" && req.Method == "POST" &&
				req.FormValue("name") == "myrole" && req.FormValue("context") == "app"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := roleAdd{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully created!\n")
}

func (s *S) TestRoleListInfo(c *check.C) {
	c.Assert((&roleList{}).Info(), check.NotNil)
}

func (s *S) TestRoleListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
    {"name": "role1",  "context": "a", "scheme_names": ["app", "app.update"]},
    {"name": "role2",  "context": "b", "scheme_names": ["service_instance"]}
]`
	expected := `+-------+---------+------------------+
| Role  | Context | Permissions      |
+-------+---------+------------------+
| role1 | a       | app              |
|       |         | app.update       |
+-------+---------+------------------+
| role2 | b       | service_instance |
+-------+---------+------------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/roles" && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := roleList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
