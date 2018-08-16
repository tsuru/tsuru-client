// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"gopkg.in/check.v1"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/fs/fstest"
)

func (s *S) TestTeamCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Team "core" successfully created!` + "\n"
	context := cmd.Context{
		Args:   []string{"core"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusCreated,
		},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			var ret tsuru.TeamData
			err = json.Unmarshal(data, &ret)
			c.Assert(ret, check.DeepEquals, tsuru.TeamData{Name: "core", Tags: []string{"tag1", "tag2"}})
			c.Assert(r.URL.Path, check.DeepEquals, "/1.0/teams")
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := TeamCreate{}
	command.Flags().Parse(true, []string{"-t", "tag1", "-t", "tag2"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTeamCreateInfo(c *check.C) {
	c.Assert((&TeamCreate{}).Info(), check.NotNil)
}

func (s *S) TestTeamUpdate(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"my-team"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			var ret tsuru.UpdateData
			err = json.Unmarshal(data, &ret)
			c.Assert(ret, check.DeepEquals, tsuru.UpdateData{Newname: "new-team", Tags: []string{"tag1", "tag2"}})
			c.Assert(strings.HasSuffix(r.URL.Path, "/teams/my-team"), check.Equals, true)
			c.Assert(r.Method, check.Equals, http.MethodPut)
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &TeamUpdate{}
	command.Flags().Parse(true, []string{"-n", "new-team", "-t", "tag1", "-t", "tag2"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Team successfully updated!\n")
}

func (s *S) TestTeamUpdateError(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"my-team"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	errMsg := "team not found"
	trans := &cmdtest.Transport{Message: errMsg, Status: http.StatusNotFound}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &TeamUpdate{}
	err := command.Run(&ctx, client)
	c.Assert(err, check.ErrorMatches, `team not found`)
}

func (s *S) TestTeamUpdateInfo(c *check.C) {
	c.Assert((&TeamUpdate{}).Info(), check.NotNil)
}

func (s *S) TestTeamRemove(c *check.C) {
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{
		Args:   []string{"evergrey"},
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return strings.HasSuffix(req.URL.Path, "/teams/evergrey") && req.Method == http.MethodDelete
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := TeamRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(buf.String(), check.Equals, `Are you sure you want to remove team "evergrey"? (y/n) Team "evergrey" successfully removed!`+"\n")
}

func (s *S) TestTeamRemoveWithouConfirmation(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"dream-theater"},
		Stdout: &buf,
		Stdin:  strings.NewReader("n\n"),
	}
	command := TeamRemove{}
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, `Are you sure you want to remove team "dream-theater"? (y/n) Abort.`+"\n")
}

func (s *S) TestTeamRemoveFailingRequest(c *check.C) {
	context := cmd.Context{
		Args:   []string{"evergrey"},
		Stdout: new(bytes.Buffer),
		Stdin:  strings.NewReader("y\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "Team evergrey not found.", Status: http.StatusNotFound}}, nil, manager)
	command := TeamRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^Team evergrey not found.$")
}

func (s *S) TestTeamRemoveInfo(c *check.C) {
	c.Assert((&TeamRemove{}).Info(), check.NotNil)
}

func (s *S) TestTeamRemoveIsACommand(c *check.C) {
	var _ cmd.Command = &TeamRemove{}
}

func (s *S) TestTeamListRun(c *check.C) {
	var called bool
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `[
{"name":"timeredbull", "permissions": ["app.deploy", "app.abc"]},
{"name":"cobrateam", "permissions": ["a", "b"]}
]`, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/teams")
		},
	}
	expected := `+-------------+-------------+------+
| Team        | Permissions | Tags |
+-------------+-------------+------+
| timeredbull | app.deploy  |      |
|             | app.abc     |      |
+-------------+-------------+------+
| cobrateam   | a           |      |
|             | b           |      |
+-------------+-------------+------+
`
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&TeamList{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTeamListRunNoPermissions(c *check.C) {
	var called bool
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `[{"name":"timeredbull"},{"name":"cobrateam"}]`, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/teams")
		},
	}
	expected := `+-------------+-------------+------+
| Team        | Permissions | Tags |
+-------------+-------------+------+
| timeredbull |             |      |
+-------------+-------------+------+
| cobrateam   |             |      |
+-------------+-------------+------+
`
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&TeamList{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTeamListRunWithNoContent(c *check.C) {
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&TeamList{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestTeamListInfo(c *check.C) {
	c.Assert((&TeamList{}).Info(), check.NotNil)
}

func (s *S) TestTeamListIsACommand(c *check.C) {
	var _ cmd.Command = &TeamList{}
}

func (s *S) TestTeamInfoRun(c *check.C) {
	var called bool

	body := `
{
    "apps": [
        {
            "cname": [],
            "deploys": 0,
            "description": "",
            "ip": "hello-test.100.17.0.1.nip.io",
            "lock": {
                "Locked": false,
                "Reason": "",
                "Owner": "",
                "AcquireDate": "0001-01-01T00:00:00Z"
            },
            "name": "hello-test",
            "owner": "guiferpa@gmail.com",
            "plan": {
                "cpushare": 100,
                "memory": 0,
                "name": "autogenerated",
                "router": "hipache",
                "swap": 0
            },
            "platform": "static",
            "pool": "test",
            "repository": "",
            "router": "hipache",
            "routeropts": {},
            "tags": [],
            "teamowner": "admin",
            "teams": [
                "admin"
            ],
            "units": []
        }
    ],
    "name": "admin",
    "pools": [
        {
            "allowed": {
                "router": [
                    "hipache"
                ],
                "service": null,
                "team": [
                    "admin"
                ]
            },
            "default": false,
            "name": "test",
            "provisioner": "",
            "public": false,
            "teams": [
                "admin"
            ]
        }
    ],
    "users": [
        {
            "Email": "user@gmail.com",
            "Roles": [{"Name": "AllowAll","ContextType": "global","ContextValue": ""}],
            "Permissions": [{"Name": "","ContextType": "global", "ContextValue": ""}]
        }
    ]
}
`

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: body, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "GET"
		},
	}
	expected := `Team: admin

Users: 1
+----------------+------------------+
| User           | Roles            |
+----------------+------------------+
| user@gmail.com | AllowAll(global) |
+----------------+------------------+

Pools: 1
+------+------+-------------+---------+
| Pool | Kind | Provisioner | Routers |
+------+------+-------------+---------+
| test |      | default     | hipache |
+------+------+-------------+---------+

Applications: 1
+-------------+-------+------------------------------+
| Application | Units | Address                      |
+-------------+-------+------------------------------+
| hello-test  |       | hello-test.100.17.0.1.nip.io |
+-------------+-------+------------------------------+
`
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&TeamInfo{}).Run(&cmd.Context{
		Args:   []string{"team1"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTeamInfoRunWithNoContent(c *check.C) {
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&TeamInfo{}).Run(&cmd.Context{
		Args:   []string{"team1"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestTeamInfoInfo(c *check.C) {
	c.Assert((&TeamInfo{}).Info(), check.NotNil)
}

func (s *S) TestTeamInfoIsACommand(c *check.C) {
	var _ cmd.Command = &TeamInfo{}
}

func (s *S) TestUserCreateShouldNotDependOnTsuruTokenFile(c *check.C) {
	rfs := &fstest.RecordingFs{}
	f, _ := rfs.Create(cmd.JoinWithUserDir(".tsuru_target"))
	f.Write([]byte("http://localhost"))
	f.Close()
	expected := "Password: \nConfirm: \n" + `User "foo@foo.com" successfully created!` + "\n"
	reader := strings.NewReader("foo123\nfoo123\n")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  reader,
	}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusCreated,
		},
		CondFunc: func(r *http.Request) bool {
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			password := r.FormValue("password") == "foo123"
			email := r.FormValue("email") == "foo@foo.com"
			url := r.URL.Path == "/1.0/users"
			return contentType && password && email && url
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := UserCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestUserCreateReturnErrorIfPasswordsDontMatch(c *check.C) {
	reader := strings.NewReader("foo123\nfoo1234\n")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  reader,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusCreated}}, nil, manager)
	command := UserCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^Passwords didn't match.$")
}

func (s *S) TestUserCreate(c *check.C) {
	expected := "Password: \nConfirm: \n" + `User "foo@foo.com" successfully created!` + "\n"
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("foo123\nfoo123\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusCreated}}, nil, manager)
	command := UserCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestUserCreateShouldReturnErrorIfThePasswordIsNotGiven(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
	}
	command := UserCreate{}
	err := command.Run(&context, nil)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^You must provide the password!$")
}

func (s *S) TestUserCreateNotFound(c *check.C) {
	transport := cmdtest.Transport{
		Message: "Not found",
		Status:  http.StatusNotFound,
	}
	reader := strings.NewReader("foo123\nfoo123\n")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  reader,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := UserCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "User creation is disabled.")
}

func (s *S) TestUserCreateMethodNotAllowed(c *check.C) {
	transport := cmdtest.Transport{
		Message: "Not found",
		Status:  http.StatusMethodNotAllowed,
	}
	reader := strings.NewReader("foo123\nfoo123\n")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  reader,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := UserCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "User creation is disabled.")
}

func (s *S) TestUserCreateInfo(c *check.C) {
	c.Assert((&UserCreate{}).Info(), check.NotNil)
}

func (s *S) TestUserRemove(c *check.C) {
	rfs := &fstest.RecordingFs{}
	f, _ := rfs.Create(cmd.JoinWithUserDir(".tsuru_target"))
	f.Write([]byte("http://tsuru.io"))
	f.Close()
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
	}
	transport := transportFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		if strings.HasSuffix(req.URL.Path, "/users/info") && req.Method == "GET" {
			body = `{"Email":"myuser@tsuru.io","Teams":[]}`
		} else if strings.HasSuffix(req.URL.Path, "/users") && req.Method == http.MethodDelete {
			called = true
		}
		return &http.Response{
			Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := UserRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(buf.String(), check.Equals, "Are you sure you want to remove the user \"myuser@tsuru.io\" from tsuru? (y/n) User \"myuser@tsuru.io\" successfully removed.\n")
}

func (s *S) TestUserRemoveWithArgs(c *check.C) {
	rfs := &fstest.RecordingFs{}
	f, _ := rfs.Create(cmd.JoinWithUserDir(".tsuru_target"))
	f.Write([]byte("http://tsuru.io"))
	f.Close()
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
		Args:   []string{"test+u@email.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == http.MethodDelete && strings.HasSuffix(req.URL.Path, "/users") && req.URL.Query().Get("user") == context.Args[0]
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := UserRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(buf.String(), check.Equals, "Are you sure you want to remove the user \"test+u@email.com\" from tsuru? (y/n) User \"test+u@email.com\" successfully removed.\n")
}

func (s *S) TestUserRemoveWithoutConfirmation(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{
		Stdout: &buf,
		Stdin:  strings.NewReader("n\n"),
	}
	trans := cmdtest.Transport{Message: `{"Email":"myself@email.com","Teams":["team1"]}`, Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := UserRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "Are you sure you want to remove the user \"myself@email.com\" from tsuru? (y/n) Abort.\n")
}

func (s *S) TestUserRemoveWithRequestError(c *check.C) {
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "User not found.", Status: http.StatusNotFound}}, nil, manager)
	command := UserRemove{}
	err := command.Run(&cmd.Context{Stdout: new(bytes.Buffer), Stdin: strings.NewReader("y\n")}, client)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^User not found.$")
}

func (s *S) TestUserRemoveInfo(c *check.C) {
	c.Assert((&UserRemove{}).Info(), check.NotNil)
}

func (s *S) TestUserRemoveIsACommand(c *check.C) {
	var _ cmd.Command = &UserRemove{}
}

func (s *S) TestChangePassword(c *check.C) {
	var (
		buf    bytes.Buffer
		called bool
		stdin  io.Reader
	)
	stdin = strings.NewReader("gopher\nbbrothers\nbbrothers\n")
	context := cmd.Context{
		Stdout: &buf,
		Stdin:  stdin,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			old := r.FormValue("old") == "gopher"
			new := r.FormValue("new") == "bbrothers"
			confirm := r.FormValue("confirm") == "bbrothers"
			method := r.Method == "PUT"
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(r.URL.Path, "/users/password")
			called = true
			return method && url && contentType && old && new && confirm
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ChangePassword{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := "Current password: \nNew password: \nConfirm: \nPassword successfully updated!\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestChangePasswordWrongConfirmation(c *check.C) {
	var (
		buf   bytes.Buffer
		stdin io.Reader
	)
	stdin = strings.NewReader("gopher\nbbrothers\nbrothers\n")
	context := cmd.Context{
		Stdout: &buf,
		Stdin:  stdin,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "New password and password confirmation didn't match.", Status: http.StatusBadRequest},
		CondFunc: func(r *http.Request) bool {
			old := r.FormValue("old") == "gopher"
			new := r.FormValue("new") == "bbrothers"
			confirm := r.FormValue("confirm") == "brothers"
			method := r.Method == "PUT"
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(r.URL.Path, "/users/password")
			return method && url && contentType && old && new && confirm
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ChangePassword{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "New password and password confirmation didn't match.")
}

func (s *S) TestChangePasswordInfo(c *check.C) {
	command := ChangePassword{}
	c.Assert(command.Info(), check.NotNil)
}

func (s *S) TestChangePasswordIsACommand(c *check.C) {
	var _ cmd.Command = &ChangePassword{}
}

func (s *S) TestResetPassword(c *check.C) {
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{Args: []string{"user@tsuru.io"}, Stdout: &buf}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status:  http.StatusOK,
			Message: "",
		},
		CondFunc: func(r *http.Request) bool {
			called = true
			return r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/users/user@tsuru.io/password") &&
				r.URL.Query().Get("token") == ""
		},
	}
	command := ResetPassword{}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `You've successfully started the password reset process.

Please check your email.` + "\n"
	c.Assert(buf.String(), check.Equals, expected)
	c.Assert(called, check.Equals, true)
}

func (s *S) TestResetPasswordStepTwo(c *check.C) {
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{Args: []string{"user@tsuru.io"}, Stdout: &buf}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status:  http.StatusOK,
			Message: "",
		},
		CondFunc: func(r *http.Request) bool {
			called = true
			return r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/users/user@tsuru.io/password") &&
				r.URL.Query().Get("token") == "secret"
		},
	}
	command := ResetPassword{}
	command.Flags().Parse(true, []string{"-t", "secret"})
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `Your password has been reset and mailed to you.

Please check your email.` + "\n"
	c.Assert(buf.String(), check.Equals, expected)
	c.Assert(called, check.Equals, true)
}

func (s *S) TestResetPasswordInfo(c *check.C) {
	c.Assert((&ResetPassword{}).Info(), check.NotNil)
}

func (s *S) TestResetPasswordFlags(c *check.C) {
	command := ResetPassword{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	err := flagset.Parse(false, []string{"-t", "token123"})
	c.Assert(err, check.IsNil)
	c.Assert(command.token, check.Equals, "token123")
	token := flagset.Lookup("token")
	c.Assert(token, check.NotNil)
	c.Check(token.Name, check.Equals, "token")
	c.Check(token.Usage, check.Equals, "Token to reset the password")
	c.Check(token.Value.String(), check.Equals, "token123")
	c.Check(token.DefValue, check.Equals, "")
	stoken := flagset.Lookup("t")
	c.Assert(stoken, check.NotNil)
	c.Check(stoken.Name, check.Equals, "t")
	c.Check(stoken.Usage, check.Equals, "Token to reset the password")
	c.Check(stoken.Value.String(), check.Equals, "token123")
	c.Check(stoken.DefValue, check.Equals, "")
}

func (s *S) TestResetPasswordIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &ResetPassword{}
}

func (s *S) TestShowAPITokenRun(c *check.C) {
	var called bool
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `"23iou32nd3i2udnu23jd"`, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/users/api-key")
		},
	}
	expected := `API key: 23iou32nd3i2udnu23jd
`
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&ShowAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestShowAPITokenRunWithFlag(c *check.C) {
	var called bool
	var stdout, stderr bytes.Buffer
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: `"23iou32nd3i2udnu23jd"`,
			Status:  http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/users/api-key") &&
				req.URL.RawQuery == "user="+url.QueryEscape("admin@example.com")
		},
	}
	expected := `API key: 23iou32nd3i2udnu23jd
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}
	command := ShowAPIToken{}
	command.Flags().Parse(true, []string{"-u", "admin@example.com"})
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestShowAPITokenRunWithNoContent(c *check.C) {
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&ShowAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestShowAPITokenInfo(c *check.C) {
	c.Assert((&ShowAPIToken{}).Info(), check.NotNil)
}

func (s *S) TestTShowAPITokenIsACommand(c *check.C) {
	var _ cmd.Command = &ShowAPIToken{}
}

func (s *S) TestRegenerateAPITokenRun(c *check.C) {
	var called bool
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `"23iou32nd3i2udnu23jd"`, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/users/api-key")
		},
	}
	expected := `Your new API key is: 23iou32nd3i2udnu23jd
`
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&RegenerateAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRegenerateAPITokenRunWithFlag(c *check.C) {
	var called bool
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `"23iou32nd3i2udnu23jd"`, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/users/api-key") &&
				req.URL.RawQuery == "user=admin@example.com"
		},
	}
	expected := `Your new API key is: 23iou32nd3i2udnu23jd
`
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}
	command := RegenerateAPIToken{}
	command.Flags().Parse(true, []string{"-u", "admin@example.com"})
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRegenerateAPITokenRunWithNoContent(c *check.C) {
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&RegenerateAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestRegenerateAPITokenInfo(c *check.C) {
	c.Assert((&RegenerateAPIToken{}).Info(), check.NotNil)
}

func (s *S) TestTRegenerateAPITokenIsACommand(c *check.C) {
	var _ cmd.Command = &RegenerateAPIToken{}
}

func (s *S) TestListUsersInfo(c *check.C) {
	expected := &cmd.Info{
		Name:    "user-list",
		MinArgs: 0,
		Usage:   "user-list [--user/-u useremail] [--role/-r role [-c/--context-value value]]",
		Desc:    "List all users in tsuru. It may also filter users by user email or role name with context value.",
	}
	c.Assert((&ListUsers{}).Info(), check.DeepEquals, expected)
}

func (s *S) TestListUsersRunWithoutFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	manager = cmd.NewManager("glb", "0.2", "ad-ver", &stdout, &stderr, nil, nil)
	result := `[{"email": "test@test.com",
"roles":[
	{"name": "role1", "contexttype": "team", "contextvalue": "a"},
	{"name": "role2", "contexttype": "app", "contextvalue": "x"}
]
}]`
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/users")
		},
	}
	expected := `+---------------+---------------+
| User          | Roles         |
+---------------+---------------+
| test@test.com | role1(team a) |
|               | role2(app x)  |
+---------------+---------------+
`
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ListUsers{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestListUsersRunFilterByUserEmail(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	manager = cmd.NewManager("glb", "0.2", "ad-ver", &stdout, &stderr, nil, nil)
	result := `[{"email": "test@test.com",
"roles":[
	{"name": "role1", "contexttype": "team", "contextvalue": "a"},
	{"name": "role2", "contexttype": "app", "contextvalue": "x"}
],
"permissions":[
	{"name": "app.create", "contexttype": "team", "contextvalue": "a"},
	{"name": "app.deploy", "contexttype": "app", "contextvalue": "x"}
]
}]`
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/users") &&
				req.URL.RawQuery == "userEmail=test2@test.com&role=&context="
		},
	}
	expected := `+---------------+---------------+
| User          | Roles         |
+---------------+---------------+
| test@test.com | role1(team a) |
|               | role2(app x)  |
+---------------+---------------+
`
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ListUsers{}
	command.Flags().Parse(true, []string{"-u", "test2@test.com"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestListUsersRunFilterByRole(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	manager = cmd.NewManager("glb", "0.2", "ad-ver", &stdout, &stderr, nil, nil)
	result := `[{"email": "test@test.com",
	"roles":[
		{"name": "role1", "contexttype": "team", "contextvalue": "a"},
		{"name": "role2", "contexttype": "app", "contextvalue": "x"}
	],
	"permissions":[
		{"name": "app.create", "contexttype": "team", "contextvalue": "a"},
		{"name": "app.deploy", "contexttype": "app", "contextvalue": "x"}
	]
	}]`
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/users") &&
				req.URL.RawQuery == "userEmail=&role=role2&context="
		},
	}
	expected := `+---------------+---------------+
| User          | Roles         |
+---------------+---------------+
| test@test.com | role1(team a) |
|               | role2(app x)  |
+---------------+---------------+
`
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ListUsers{}
	command.Flags().Parse(true, []string{"-r", "role2"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestListUsersRunFilterByRoleWithContext(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	manager = cmd.NewManager("glb", "0.2", "ad-ver", &stdout, &stderr, nil, nil)
	result := `[{"email": "test@test.com",
	"roles":[
		{"name": "role1", "contexttype": "team", "contextvalue": "a"},
		{"name": "role2", "contexttype": "app", "contextvalue": "x"}
	],
	"permissions":[
		{"name": "app.create", "contexttype": "team", "contextvalue": "a"},
		{"name": "app.deploy", "contexttype": "app", "contextvalue": "x"}
	]
	}]`
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/users") &&
				req.URL.RawQuery == "userEmail=&role=role2&context=x"
		},
	}
	expected := `+---------------+---------------+
| User          | Roles         |
+---------------+---------------+
| test@test.com | role1(team a) |
|               | role2(app x)  |
+---------------+---------------+
`
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ListUsers{}
	command.Flags().Parse(true, []string{"-r", "role2", "-c", "x"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestListUsersRunWithMoreThanOneFlagReturnsError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	manager = cmd.NewManager("glb", "0.2", "ad-ver", &stdout, &stderr, nil, nil)
	result := `[{"email": "test@test.com",
		"roles":[
			{"name": "role1", "contexttype": "team", "contextvalue": "a"},
			{"name": "role2", "contexttype": "app", "contextvalue": "x"}
		],
		"permissions":[
			{"name": "app.create", "contexttype": "team", "contextvalue": "a"},
			{"name": "app.deploy", "contexttype": "app", "contextvalue": "x"}
		]
		}]`
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/users") &&
				req.URL.RawQuery == "userEmail=test@test.com&role=role2"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ListUsers{}
	command.Flags().Parse(true, []string{"-u", "test@test.com", "-r", "role2"})
	err := command.Run(&context, client)
	c.Assert(err, check.ErrorMatches, "You cannot filter by user email and role at same time. Enter <tsuru user-list --help> for more information.")
}

func (s *S) TestListUsersRunWithContextFlagAndNotRolaFlagError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	manager = cmd.NewManager("glb", "0.2", "ad-ver", &stdout, &stderr, nil, nil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/users") &&
				req.URL.RawQuery == "userEmail=test@test.com&role=&context=team"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ListUsers{}
	command.Flags().Parse(true, []string{"-c", "team"})
	err := command.Run(&context, client)
	c.Assert(err, check.ErrorMatches, "You should provide a role to filter by context value.")
}

func (s *S) TestListUsersFlags(c *check.C) {
	command := ListUsers{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	err := flagset.Parse(false, []string{"-u", "test@test.com"})
	c.Assert(err, check.IsNil)
	c.Assert(command.userEmail, check.Equals, "test@test.com")
	user := flagset.Lookup("user")
	c.Assert(user, check.NotNil)
	c.Check(user.Name, check.Equals, "user")
	c.Check(user.Usage, check.Equals, "Filter user by user email")
	c.Check(user.Value.String(), check.Equals, "test@test.com")
	c.Check(user.DefValue, check.Equals, "")
	suser := flagset.Lookup("u")
	c.Assert(suser, check.NotNil)
	c.Check(suser.Name, check.Equals, "u")
	c.Check(suser.Usage, check.Equals, "Filter user by user email")
	c.Check(suser.Value.String(), check.Equals, "test@test.com")
	c.Check(suser.DefValue, check.Equals, "")
	err = flagset.Parse(false, []string{"-r", "role1"})
	c.Assert(err, check.IsNil)
	c.Assert(command.role, check.Equals, "role1")
	role := flagset.Lookup("role")
	c.Assert(user, check.NotNil)
	c.Check(role.Name, check.Equals, "role")
	c.Check(role.Usage, check.Equals, "Filter user by role")
	c.Check(role.Value.String(), check.Equals, "role1")
	c.Check(role.DefValue, check.Equals, "")
	srole := flagset.Lookup("r")
	c.Assert(srole, check.NotNil)
	c.Check(srole.Name, check.Equals, "r")
	c.Check(srole.Usage, check.Equals, "Filter user by role")
	c.Check(srole.Value.String(), check.Equals, "role1")
	c.Check(srole.DefValue, check.Equals, "")
}
