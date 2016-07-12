// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestPermissionListInfo(c *check.C) {
	c.Assert((&PermissionList{}).Info(), check.NotNil)
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
			return strings.HasSuffix(req.URL.Path, "/permissions") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := PermissionList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRoleAddInfo(c *check.C) {
	c.Assert((&RoleAdd{}).Info(), check.NotNil)
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
			return strings.HasSuffix(req.URL.Path, "/roles") && req.Method == "POST" &&
				req.FormValue("name") == "myrole" && req.FormValue("context") == "app"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleAdd{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully created!\n")
}

func (s *S) TestRoleAddFlags(c *check.C) {
	command := RoleAdd{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse(true, []string{"-d", "my description"})
	description := flagset.Lookup("description")
	usage := "Role description"
	c.Check(description, check.NotNil)
	c.Check(description.Name, check.Equals, "description")
	c.Check(description.Usage, check.Equals, usage)
	c.Check(description.Value.String(), check.Equals, "my description")
	c.Check(description.DefValue, check.Equals, "")
	sdescription := flagset.Lookup("d")
	c.Check(sdescription, check.NotNil)
	c.Check(sdescription.Name, check.Equals, "d")
	c.Check(sdescription.Usage, check.Equals, usage)
	c.Check(sdescription.Value.String(), check.Equals, "my description")
	c.Check(sdescription.DefValue, check.Equals, "")
}

func (s *S) TestRoleListInfo(c *check.C) {
	c.Assert((&RoleList{}).Info(), check.NotNil)
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
			return strings.HasSuffix(req.URL.Path, "/roles") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRoleInfoInfo(c *check.C) {
	c.Assert((&RoleInfo{}).Info(), check.NotNil)
}

func (s *S) TestRoleInfoRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `
    {"name": "role1",  "context": "a", "description":"my description", "scheme_names": ["app", "app.update"]}
`
	expected := `+-------+---------+-------------+----------------+
| Name  | Context | Permissions | Description    |
+-------+---------+-------------+----------------+
| role1 | a       | app         | my description |
|       |         | app.update  |                |
+-------+---------+-------------+----------------+
`
	context := cmd.Context{
		Args:   []string{"role1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/roles/role1") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleInfo{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRoleAssignInfo(c *check.C) {
	c.Assert((&RoleAssign{}).Info(), check.NotNil)
}

func (s *S) TestRoleAssignRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole", "me@me.com", "myapp"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/roles/myrole/user") && req.Method == "POST" &&
				req.FormValue("email") == "me@me.com" && req.FormValue("context") == "myapp"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleAssign{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully assigned!\n")
}

func (s *S) TestRoleDissociateInfo(c *check.C) {
	c.Assert((&RoleDissociate{}).Info(), check.NotNil)
}

func (s *S) TestRoleDissociateRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole", "me@me.com", "myapp"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/roles/myrole/user/me@me.com") && req.Method == "DELETE" &&
				req.URL.Query().Get("context") == "myapp"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleDissociate{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully dissociated!\n")
}

func (s *S) TestRolePermissionAddInfo(c *check.C) {
	c.Assert((&RolePermissionAdd{}).Info(), check.NotNil)
}

func (s *S) TestRolePermissionAddRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole", "app.create", "app.deploy"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			req.ParseForm()
			sort.Strings(req.Form["permission"])
			return strings.HasSuffix(req.URL.Path, "/roles/myrole/permissions") && req.Method == "POST" &&
				reflect.DeepEqual(req.Form["permission"], []string{"app.create", "app.deploy"})
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RolePermissionAdd{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Permission successfully added!\n")
}

func (s *S) TestRolePermissionRemoveInfo(c *check.C) {
	c.Assert((&RolePermissionRemove{}).Info(), check.NotNil)
}

func (s *S) TestRolePermissionRemoveRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole", "app.create"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/roles/myrole/permissions/app.create") && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RolePermissionRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Permission successfully removed!\n")
}

func (s *S) TestRoleRemoveInfo(c *check.C) {
	c.Assert((&RoleRemove{}).Info(), check.NotNil)
}

func (s *S) TestRoleRemoveRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/roles/myrole") && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Are you sure you want to remove role \"myrole\"? (y/n) Role successfully removed!\n")
}

func (s *S) TestRoleRemoveWithConfirmation(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/roles/myrole") && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleRemove{}
	command.Flags().Parse(true, []string{"-y"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully removed!\n")
}

func (s *S) TestRoleRemoveWithoutConfirmation(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Are you sure you want to remove role "myrole"? (y/n) Abort.` + "\n"
	context := cmd.Context{
		Args:   []string{"myrole"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("n\n"),
	}
	command := RoleRemove{}
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRoleDefaultAdd(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			req.ParseForm()
			sort.Strings(req.Form["user-create"])
			return strings.HasSuffix(req.URL.Path, "/role/default") && req.Method == "POST" &&
				reflect.DeepEqual(req.Form["user-create"], []string{"r1", "r2"}) &&
				reflect.DeepEqual(req.Form["team-create"], []string{"r3"})
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleDefaultAdd{}
	command.Flags().Parse(true, []string{"--user-create", "r1", "--user-create", "r2", "--team-create", "r3"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Roles successfully added as default!\n")
}

func (s *S) TestRoleDefaultRemove(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			req.ParseForm()
			sort.Strings(req.Form["user-create"])
			return strings.HasSuffix(req.URL.Path, "/role/default") && req.Method == "DELETE" &&
				reflect.DeepEqual(req.Form["user-create"], []string{"r1", "r2"}) &&
				reflect.DeepEqual(req.Form["team-create"], []string{"r3"})
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleDefaultRemove{}
	command.Flags().Parse(true, []string{"--user-create", "r1", "--user-create", "r2", "--team-create", "r3"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Roles successfully removed as default!\n")
}

func (s *S) TestRoleDefaultList(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	result := `[
	    {"name": "role1",  "context": "a", "events": ["team-create"]},
	    {"name": "role2",  "context": "b", "events": ["user-create"]}
	]`
	expected := `+-------------+-----------------------------------------------+-------+
| Event       | Description                                   | Roles |
+-------------+-----------------------------------------------+-------+
| team-create | role added to user when a new team is created | role1 |
+-------------+-----------------------------------------------+-------+
| user-create | role added to user when user is created       | role2 |
+-------------+-----------------------------------------------+-------+
`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/role/default") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoleDefaultList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
