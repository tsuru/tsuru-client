// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestTeamAddUser(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `User "andorito" was added to the "cobrateam" team` + "\n"
	context := cmd.Context{
		Args:   []string{"cobrateam", "andorito"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}
	command := teamUserAdd{}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTeamAddUserInfo(c *check.C) {
	c.Assert((&teamUserAdd{}).Info(), check.NotNil)
}

func (s *S) TestTeamRemoveUser(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `User "andorito" was removed from the "cobrateam" team` + "\n"
	context := cmd.Context{
		Args:   []string{"cobrateam", "andorito"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}
	command := teamUserRemove{}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTeamRemoveUserInfo(c *check.C) {
	c.Assert((&teamUserRemove{}).Info(), check.NotNil)
}

func (s *S) TestTeamCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Team "core" successfully created!` + "\n"
	context := cmd.Context{
		Args:   []string{"core"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusCreated}}, nil, manager)
	command := teamCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTeamCreateInfo(c *check.C) {
	c.Assert((&teamCreate{}).Info(), check.NotNil)
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
			return req.URL.Path == "/teams/evergrey" && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := teamRemove{}
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
	command := teamRemove{}
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
	command := teamRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^Team evergrey not found.$")
}

func (s *S) TestTeamRemoveInfo(c *check.C) {
	c.Assert((&teamRemove{}).Info(), check.NotNil)
}

func (s *S) TestTeamRemoveIsACommand(c *check.C) {
	var _ cmd.Command = &teamRemove{}
}

func (s *S) TestTeamUserList(c *check.C) {
	var called bool
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"symfonia"}, Stdout: &buf}
	command := teamUserList{}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status:  http.StatusOK,
			Message: `{"name":"symfonia","users":["somebody@tsuru.io","otherbody@tsuru.io","me@tsuru.io"]}`,
		},
		CondFunc: func(r *http.Request) bool {
			called = true
			return r.Method == "GET" && r.URL.Path == "/teams/symfonia"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := `- me@tsuru.io
- otherbody@tsuru.io
- somebody@tsuru.io` + "\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestTeamUserListError(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"symfonia"}, Stdout: &buf}
	transport := cmdtest.Transport{Status: http.StatusNotFound, Message: "Team not found"}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := teamUserList{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "Team not found")
}

func (s *S) TestTeamUserListInfo(c *check.C) {
	c.Assert(teamUserList{}.Info(), check.NotNil)
}

func (s *S) TestTeamUserListIsACommand(c *check.C) {
	var _ cmd.Command = teamUserList{}
}

func (s *S) TestTeamListRun(c *check.C) {
	var called bool
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `[{"name":"timeredbull"},{"name":"cobrateam"}]`, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "GET" && req.URL.Path == "/teams"
		},
	}
	expected := `Teams:

  - timeredbull
  - cobrateam
`
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&teamList{}).Run(&cmd.Context{
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
	err := (&teamList{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestTeamListInfo(c *check.C) {
	c.Assert((&teamList{}).Info(), check.NotNil)
}

func (s *S) TestTeamListIsACommand(c *check.C) {
	var _ cmd.Command = &teamList{}
}

func (s *S) TestUserCreateShouldNotDependOnTsuruTokenFile(c *check.C) {
	rfs := &fstest.RecordingFs{}
	f, _ := rfs.Create(cmd.JoinWithUserDir(".tsuru_target"))
	f.Write([]byte("http://localhost"))
	f.Close()
	fsystem = rfs
	defer func() {
		fsystem = nil
	}()
	expected := "Password: \nConfirm: \n" + `User "foo@foo.com" successfully created!` + "\n"
	reader := strings.NewReader("foo123\nfoo123\n")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  reader,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusCreated}}, nil, manager)
	command := userCreate{}
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
	command := userCreate{}
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
	command := userCreate{}
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
	command := userCreate{}
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
	command := userCreate{}
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
	command := userCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "User creation is disabled.")
}

func (s *S) TestUserCreateInfo(c *check.C) {
	c.Assert((&userCreate{}).Info(), check.NotNil)
}

func (s *S) TestUserRemove(c *check.C) {
	rfs := &fstest.RecordingFs{}
	f, _ := rfs.Create(cmd.JoinWithUserDir(".tsuru_target"))
	f.Write([]byte("http://tsuru.io"))
	f.Close()
	fsystem = rfs
	defer func() {
		fsystem = nil
	}()
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && req.URL.Path == "/users"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := userRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(buf.String(), check.Equals, "Are you sure you want to remove your user from tsuru? (y/n) User successfully removed.\n")
	c.Assert(rfs.HasAction("remove "+cmd.JoinWithUserDir(".tsuru_token")), check.Equals, true)
}

func (s *S) TestUserRemoveWithArgs(c *check.C) {
	rfs := &fstest.RecordingFs{}
	f, _ := rfs.Create(cmd.JoinWithUserDir(".tsuru_target"))
	f.Write([]byte("http://tsuru.io"))
	f.Close()
	fsystem = rfs
	defer func() {
		fsystem = nil
	}()
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
		Args:   []string{"u@email.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && req.URL.Path == "/users" && req.URL.Query().Get("user") == context.Args[0]
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := userRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(buf.String(), check.Equals, "Are you sure you want to remove your user from tsuru? (y/n) User successfully removed.\n")
	c.Assert(rfs.HasAction("remove "+cmd.JoinWithUserDir(".tsuru_token")), check.Equals, true)
}

func (s *S) TestUserRemoveWithoutConfirmation(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{
		Stdout: &buf,
		Stdin:  strings.NewReader("n\n"),
	}
	command := userRemove{}
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "Are you sure you want to remove your user from tsuru? (y/n) Abort.\n")
}

func (s *S) TestUserRemoveWithRequestError(c *check.C) {
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "User not found.", Status: http.StatusNotFound}}, nil, manager)
	command := userRemove{}
	err := command.Run(&cmd.Context{Stdout: new(bytes.Buffer), Stdin: strings.NewReader("y\n")}, client)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^User not found.$")
}

func (s *S) TestUserRemoveInfo(c *check.C) {
	c.Assert((&userRemove{}).Info(), check.NotNil)
}

func (s *S) TestUserRemoveIsACommand(c *check.C) {
	var _ cmd.Command = &userRemove{}
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
		CondFunc: func(req *http.Request) bool {
			var got map[string]string
			called = true
			if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
				return false
			}
			cond := got["old"] == "gopher" && got["new"] == "bbrothers"
			return cond && req.Method == "PUT" && req.URL.Path == "/users/password"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := changePassword{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := "Current password: \nNew password: \nConfirm: \nPassword successfully updated!\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestChangePasswordWrongConfirmation(c *check.C) {
	var buf bytes.Buffer
	stdin := strings.NewReader("gopher\nblood\nsugar\n")
	context := cmd.Context{
		Stdin:  stdin,
		Stdout: &buf,
		Stderr: &buf,
	}
	command := changePassword{}
	err := command.Run(&context, nil)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "New password and password confirmation didn't match.")
}

func (s *S) TestChangePasswordInfo(c *check.C) {
	command := changePassword{}
	c.Assert(command.Info(), check.NotNil)
}

func (s *S) TestChangePasswordIsACommand(c *check.C) {
	var _ cmd.Command = &changePassword{}
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
			return r.Method == "POST" && r.URL.Path == "/users/user@tsuru.io/password" &&
				r.URL.Query().Get("token") == ""
		},
	}
	command := resetPassword{}
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
			return r.Method == "POST" && r.URL.Path == "/users/user@tsuru.io/password" &&
				r.URL.Query().Get("token") == "secret"
		},
	}
	command := resetPassword{}
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
	c.Assert((&resetPassword{}).Info(), check.NotNil)
}

func (s *S) TestResetPasswordFlags(c *check.C) {
	command := resetPassword{}
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
	var _ cmd.FlaggedCommand = &resetPassword{}
}

func (s *S) TestShowAPITokenRun(c *check.C) {
	var called bool
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `"23iou32nd3i2udnu23jd"`, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "GET" && req.URL.Path == "/users/api-key"
		},
	}
	expected := `API key: 23iou32nd3i2udnu23jd
`
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&showAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestShowAPITokenRunWithNoContent(c *check.C) {
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&showAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestShowAPITokenInfo(c *check.C) {
	c.Assert((&showAPIToken{}).Info(), check.NotNil)
}

func (s *S) TestTShowAPITokenIsACommand(c *check.C) {
	var _ cmd.Command = &showAPIToken{}
}

func (s *S) TestRegenerateAPITokenRun(c *check.C) {
	var called bool
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `"23iou32nd3i2udnu23jd"`, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "POST" && req.URL.Path == "/users/api-key"
		},
	}
	expected := `Your new API key is: 23iou32nd3i2udnu23jd
`
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&regenerateAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRegenerateAPITokenRunWithNoContent(c *check.C) {
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&regenerateAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestRegenerateAPITokenInfo(c *check.C) {
	c.Assert((&regenerateAPIToken{}).Info(), check.NotNil)
}

func (s *S) TestTRegenerateAPITokenIsACommand(c *check.C) {
	var _ cmd.Command = &regenerateAPIToken{}
}
