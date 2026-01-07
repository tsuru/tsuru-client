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

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
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
			return strings.HasSuffix(req.URL.Path, "/permissions") && req.Method == http.MethodGet
		},
	}
	s.setupFakeTransport(trans)
	command := PermissionList{}
	err := command.Run(&context)
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
			return strings.HasSuffix(req.URL.Path, "/roles") && req.Method == http.MethodPost &&
				req.FormValue("name") == "myrole" && req.FormValue("context") == "app"
		},
	}
	s.setupFakeTransport(trans)
	command := RoleAdd{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully created!\n")
}

func (s *S) TestRoleAddFlags(c *check.C) {
	command := RoleAdd{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse([]string{"-d", "my description"})
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
			return strings.HasSuffix(req.URL.Path, "/roles") && req.Method == http.MethodGet
		},
	}
	s.setupFakeTransport(trans)
	command := RoleList{}
	err := command.Run(&context)
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
			return strings.HasSuffix(req.URL.Path, "/roles/role1") && req.Method == http.MethodGet
		},
	}
	s.setupFakeTransport(trans)
	command := RoleInfo{}
	err := command.Run(&context)
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
			return strings.HasSuffix(req.URL.Path, "/roles/myrole/user") && req.Method == http.MethodPost &&
				req.FormValue("email") == "me@me.com" && req.FormValue("context") == "myapp"
		},
	}
	s.setupFakeTransport(trans)
	command := RoleAssign{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully assigned!\n")
}

func (s *S) TestRoleAssignRunWithToken(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole", "mytoken", "myapp"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.URL.Path, check.Equals, "/1.6/roles/myrole/token")
			c.Assert(req.Method, check.Equals, http.MethodPost)
			c.Assert(req.FormValue("token_id"), check.Equals, "mytoken")
			c.Assert(req.FormValue("context"), check.Equals, "myapp")
			return true
		},
	}
	s.setupFakeTransport(trans)
	command := RoleAssign{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully assigned!\n")
}

func (s *S) TestRoleAssignRunWithGroup(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole", "group:grp1", "myapp"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.URL.Path, check.Equals, "/1.9/roles/myrole/group")
			c.Assert(req.Method, check.Equals, http.MethodPost)
			c.Assert(req.FormValue("group_name"), check.Equals, "grp1")
			c.Assert(req.FormValue("context"), check.Equals, "myapp")
			return true
		},
	}
	s.setupFakeTransport(trans)
	command := RoleAssign{}
	err := command.Run(&context)
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
			return strings.HasSuffix(req.URL.Path, "/roles/myrole/user/me@me.com") && req.Method == http.MethodDelete &&
				req.URL.Query().Get("context") == "myapp"
		},
	}
	s.setupFakeTransport(trans)
	command := RoleDissociate{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully dissociated!\n")
}

func (s *S) TestRoleDissociateRunWithToken(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrole", "mytoken", "myapp"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(""), Status: http.StatusCreated},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.URL.Path, check.Equals, "/1.6/roles/myrole/token/mytoken")
			c.Assert(req.Method, check.Equals, http.MethodDelete)
			c.Assert(req.FormValue("context"), check.Equals, "myapp")
			return true
		},
	}
	s.setupFakeTransport(trans)
	command := RoleDissociate{}
	err := command.Run(&context)
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
			return strings.HasSuffix(req.URL.Path, "/roles/myrole/permissions") && req.Method == http.MethodPost &&
				reflect.DeepEqual(req.Form["permission"], []string{"app.create", "app.deploy"})
		},
	}
	s.setupFakeTransport(trans)
	command := RolePermissionAdd{}
	err := command.Run(&context)
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
			return strings.HasSuffix(req.URL.Path, "/roles/myrole/permissions/app.create") && req.Method == http.MethodDelete
		},
	}
	s.setupFakeTransport(trans)
	command := RolePermissionRemove{}
	err := command.Run(&context)
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
			return strings.HasSuffix(req.URL.Path, "/roles/myrole") && req.Method == http.MethodDelete
		},
	}
	s.setupFakeTransport(trans)
	command := RoleRemove{}
	err := command.Run(&context)
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
			return strings.HasSuffix(req.URL.Path, "/roles/myrole") && req.Method == http.MethodDelete
		},
	}
	s.setupFakeTransport(trans)
	command := RoleRemove{}
	command.Flags().Parse([]string{"-y"})
	err := command.Run(&context)
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
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRoleDefaultAddInfo(c *check.C) {
	c.Assert((&RoleDefaultAdd{}).Info(), check.NotNil)
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
			return strings.HasSuffix(req.URL.Path, "/role/default") && req.Method == http.MethodPost &&
				reflect.DeepEqual(req.Form["user-create"], []string{"r1", "r2"}) &&
				reflect.DeepEqual(req.Form["team-create"], []string{"r3"})
		},
	}
	s.setupFakeTransport(trans)
	command := RoleDefaultAdd{}
	command.Flags().Parse([]string{"--user-create", "r1", "--user-create", "r2", "--team-create", "r3"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Roles successfully added as default!\n")
}

func (s *S) TestRoleDefaultRemoveInfo(c *check.C) {
	c.Assert((&RoleDefaultRemove{}).Info(), check.NotNil)
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
			return strings.HasSuffix(req.URL.Path, "/role/default") && req.Method == http.MethodDelete &&
				reflect.DeepEqual(req.Form["user-create"], []string{"r1", "r2"}) &&
				reflect.DeepEqual(req.Form["team-create"], []string{"r3"})
		},
	}
	s.setupFakeTransport(trans)
	command := RoleDefaultRemove{}
	command.Flags().Parse([]string{"--user-create", "r1", "--user-create", "r2", "--team-create", "r3"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Roles successfully removed as default!\n")
}

func (s *S) TestRoleDefaultListInfo(c *check.C) {
	c.Assert((&RoleDefaultList{}).Info(), check.NotNil)
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
			return strings.HasSuffix(req.URL.Path, "/role/default") && req.Method == http.MethodGet
		},
	}
	s.setupFakeTransport(trans)
	command := RoleDefaultList{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRoleUpdateInfo(c *check.C) {
	c.Assert((&RoleUpdate{}).Info(), check.NotNil)
}

func (s *S) TestRoleUpdate(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"team-member"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			path := req.URL.Path == "/1.4/roles"
			method := req.Method == http.MethodPut
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && method && contentType && req.FormValue("name") == "team-member" && req.FormValue("description") == "a developer"
		},
	}
	s.setupFakeTransport(trans)
	cmd := RoleUpdate{}
	cmd.Flags().Parse([]string{"-d", "a developer"})
	err := cmd.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully updated\n")
}

func (s *S) TestRoleUpdateWithoutFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "neither the description, context or new name were set. You must define at least one"
	context := cmd.Context{
		Args:   []string{"team-member"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			path := strings.HasSuffix(req.URL.Path, "/roles")
			method := req.Method == http.MethodPut
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && method && contentType && req.FormValue("name") == "team-member" && req.FormValue("description") == "a developer"
		},
	}
	s.setupFakeTransport(trans)
	cmd := RoleUpdate{}
	err := cmd.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, expected)
}

func (s *S) TestRoleUpdateMultipleFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"team-member"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			path := strings.HasSuffix(req.URL.Path, "/roles")
			method := req.Method == http.MethodPut
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && method && contentType && req.FormValue("name") == "team-member" && req.FormValue("description") == "a developer" && req.FormValue("contextType") == "team" && req.FormValue("newName") == "newName"
		},
	}
	s.setupFakeTransport(trans)
	cmd := RoleUpdate{}
	cmd.Flags().Parse([]string{"-d", "a developer", "-c", "team", "-n", "newName"})
	err := cmd.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Role successfully updated\n")
}

func (s *S) TestRoleUpdateWithInvalidContent(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"invalid-role"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusConflict},
		CondFunc: func(req *http.Request) bool {
			path := strings.HasSuffix(req.URL.Path, "/roles")
			method := req.Method == http.MethodPut
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && method && contentType && req.FormValue("name") == "invalid-role" && req.FormValue("description") == "a developer"
		},
	}
	s.setupFakeTransport(trans)
	cmd := RoleUpdate{}
	cmd.Flags().Parse([]string{"-d", "a developer"})
	err := cmd.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "")
	c.Assert(stderr.String(), check.Equals, "Failed to update role\n")
}
