// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	ttesting "github.com/tsuru/tsuru/cmd/testing"
	"github.com/tsuru/tsuru/fs/testing"
	"launchpad.net/gocheck"
)

func (s *S) TestTeamAddUser(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	expected := `User "andorito" was added to the "cobrateam" team` + "\n"
	context := cmd.Context{
		Args:   []string{"cobrateam", "andorito"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}
	command := teamUserAdd{}
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestTeamAddUserInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "team-user-add",
		Usage:   "team-user-add <teamname> <useremail>",
		Desc:    "adds a user to a team.",
		MinArgs: 2,
	}
	c.Assert((&teamUserAdd{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamRemoveUser(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	expected := `User "andorito" was removed from the "cobrateam" team` + "\n"
	context := cmd.Context{
		Args:   []string{"cobrateam", "andorito"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}
	command := teamUserRemove{}
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestTeamRemoveUserInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "team-user-remove",
		Usage:   "team-user-remove <teamname> <useremail>",
		Desc:    "removes a user from a team.",
		MinArgs: 2,
	}
	c.Assert((&teamUserRemove{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamCreate(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	expected := `Team "core" successfully created!` + "\n"
	context := cmd.Context{
		Args:   []string{"core"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "", Status: http.StatusCreated}}, nil, manager)
	command := teamCreate{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestTeamCreateInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "team-create",
		Usage:   "team-create <teamname>",
		Desc:    "creates a new team.",
		MinArgs: 1,
	}
	c.Assert((&teamCreate{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamRemove(c *gocheck.C) {
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{
		Args:   []string{"evergrey"},
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
	}
	trans := ttesting.ConditionalTransport{
		Transport: ttesting.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.URL.Path == "/teams/evergrey" && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := teamRemove{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(buf.String(), gocheck.Equals, `Are you sure you want to remove team "evergrey"? (y/n) Team "evergrey" successfully removed!`+"\n")
}

func (s *S) TestTeamRemoveWithouConfirmation(c *gocheck.C) {
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"dream-theater"},
		Stdout: &buf,
		Stdin:  strings.NewReader("n\n"),
	}
	command := teamRemove{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.IsNil)
	c.Assert(buf.String(), gocheck.Equals, `Are you sure you want to remove team "dream-theater"? (y/n) Abort.`+"\n")
}

func (s *S) TestTeamRemoveFailingRequest(c *gocheck.C) {
	context := cmd.Context{
		Args:   []string{"evergrey"},
		Stdout: new(bytes.Buffer),
		Stdin:  strings.NewReader("y\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "Team evergrey not found.", Status: http.StatusNotFound}}, nil, manager)
	command := teamRemove{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, "^Team evergrey not found.$")
}

func (s *S) TestTeamRemoveInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "team-remove",
		Usage:   "team-remove <team-name>",
		Desc:    "removes a team from tsuru server.",
		MinArgs: 1,
	}
	c.Assert((&teamRemove{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamRemoveIsACommand(c *gocheck.C) {
	var _ cmd.Command = &teamRemove{}
}

func (s *S) TestTeamUserList(c *gocheck.C) {
	var called bool
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"symfonia"}, Stdout: &buf}
	command := teamUserList{}
	transport := ttesting.ConditionalTransport{
		Transport: ttesting.Transport{
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
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	expected := `- me@tsuru.io
- otherbody@tsuru.io
- somebody@tsuru.io` + "\n"
	c.Assert(buf.String(), gocheck.Equals, expected)
}

func (s *S) TestTeamUserListError(c *gocheck.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"symfonia"}, Stdout: &buf}
	transport := ttesting.Transport{Status: http.StatusNotFound, Message: "Team not found"}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := teamUserList{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "Team not found")
}

func (s *S) TestTeamUserListInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "team-user-list",
		Usage:   "team-user-list <teamname>",
		Desc:    "List members of a team.",
		MinArgs: 1,
	}
	c.Assert(teamUserList{}.Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamUserListIsACommand(c *gocheck.C) {
	var _ cmd.Command = teamUserList{}
}

func (s *S) TestTeamListRun(c *gocheck.C) {
	var called bool
	trans := &ttesting.ConditionalTransport{
		Transport: ttesting.Transport{Message: `[{"name":"timeredbull"},{"name":"cobrateam"}]`, Status: http.StatusOK},
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
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestTeamListRunWithNoContent(c *gocheck.C) {
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&teamList{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, "")
}

func (s *S) TestTeamListInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "team-list",
		Usage:   "team-list",
		Desc:    "List all teams that you are member.",
		MinArgs: 0,
	}
	c.Assert((&teamList{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamListIsACommand(c *gocheck.C) {
	var _ cmd.Command = &teamList{}
}

func (s *S) TestUserCreateShouldNotDependOnTsuruTokenFile(c *gocheck.C) {
	rfs := &testing.RecordingFs{}
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
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "", Status: http.StatusCreated}}, nil, manager)
	command := userCreate{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestUserCreateReturnErrorIfPasswordsDontMatch(c *gocheck.C) {
	reader := strings.NewReader("foo123\nfoo1234\n")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  reader,
	}
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "", Status: http.StatusCreated}}, nil, manager)
	command := userCreate{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, "^Passwords didn't match.$")
}

func (s *S) TestUserCreate(c *gocheck.C) {
	expected := "Password: \nConfirm: \n" + `User "foo@foo.com" successfully created!` + "\n"
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("foo123\nfoo123\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "", Status: http.StatusCreated}}, nil, manager)
	command := userCreate{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestUserCreateShouldReturnErrorIfThePasswordIsNotGiven(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(""),
	}
	command := userCreate{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, "^You must provide the password!$")
}

func (s *S) TestUserCreateNotFound(c *gocheck.C) {
	transport := ttesting.Transport{
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
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "User creation is disabled.")
}

func (s *S) TestUserCreateMethodNotAllowed(c *gocheck.C) {
	transport := ttesting.Transport{
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
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "User creation is disabled.")
}

func (s *S) TestUserCreateInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "user-create",
		Usage:   "user-create <email>",
		Desc:    "creates a user.",
		MinArgs: 1,
	}
	c.Assert((&userCreate{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestUserRemove(c *gocheck.C) {
	rfs := &testing.RecordingFs{}
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
	trans := ttesting.ConditionalTransport{
		Transport: ttesting.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && req.URL.Path == "/users"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := userRemove{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(buf.String(), gocheck.Equals, "Are you sure you want to remove your user from tsuru? (y/n) User successfully removed.\n")
	c.Assert(rfs.HasAction("remove "+cmd.JoinWithUserDir(".tsuru_token")), gocheck.Equals, true)
}

func (s *S) TestUserRemoveWithoutConfirmation(c *gocheck.C) {
	var buf bytes.Buffer
	context := cmd.Context{
		Stdout: &buf,
		Stdin:  strings.NewReader("n\n"),
	}
	command := userRemove{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.IsNil)
	c.Assert(buf.String(), gocheck.Equals, "Are you sure you want to remove your user from tsuru? (y/n) Abort.\n")
}

func (s *S) TestUserRemoveWithRequestError(c *gocheck.C) {
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "User not found.", Status: http.StatusNotFound}}, nil, manager)
	command := userRemove{}
	err := command.Run(&cmd.Context{Stdout: new(bytes.Buffer), Stdin: strings.NewReader("y\n")}, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, "^User not found.$")
}

func (s *S) TestUserRemoveInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "user-remove",
		Usage:   "user-remove",
		Desc:    "removes your user from tsuru server.",
		MinArgs: 0,
	}
	c.Assert((&userRemove{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestUserRemoveIsACommand(c *gocheck.C) {
	var _ cmd.Command = &userRemove{}
}

func (s *S) TestChangePassword(c *gocheck.C) {
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
	trans := ttesting.ConditionalTransport{
		Transport: ttesting.Transport{Message: "", Status: http.StatusOK},
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
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	expected := "Current password: \nNew password: \nConfirm: \nPassword successfully updated!\n"
	c.Assert(buf.String(), gocheck.Equals, expected)
}

func (s *S) TestChangePasswordWrongConfirmation(c *gocheck.C) {
	var buf bytes.Buffer
	stdin := strings.NewReader("gopher\nblood\nsugar\n")
	context := cmd.Context{
		Stdin:  stdin,
		Stdout: &buf,
		Stderr: &buf,
	}
	command := changePassword{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "New password and password confirmation didn't match.")
}

func (s *S) TestChangePasswordInfo(c *gocheck.C) {
	expected := cmd.Info{
		Name:  "change-password",
		Usage: "change-password",
		Desc:  "Change your password.",
	}
	command := changePassword{}
	c.Assert(command.Info(), gocheck.DeepEquals, &expected)
}

func (s *S) TestChangePasswordIsACommand(c *gocheck.C) {
	var _ cmd.Command = &changePassword{}
}

func (s *S) TestPasswordFromReaderUsingFile(c *gocheck.C) {
	tmpdir, err := filepath.EvalSymlinks(os.TempDir())
	filename := path.Join(tmpdir, "password-reader.txt")
	c.Assert(err, gocheck.IsNil)
	file, err := os.Create(filename)
	c.Assert(err, gocheck.IsNil)
	defer os.Remove(filename)
	file.WriteString("hello")
	file.Seek(0, 0)
	password, err := passwordFromReader(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(password, gocheck.Equals, "hello")
}

func (s *S) TestPasswordFromReaderUsingStringsReader(c *gocheck.C) {
	reader := strings.NewReader("abcd\n")
	password, err := passwordFromReader(reader)
	c.Assert(err, gocheck.IsNil)
	c.Assert(password, gocheck.Equals, "abcd")
}

func (s *S) TestResetPassword(c *gocheck.C) {
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{Args: []string{"user@tsuru.io"}, Stdout: &buf}
	trans := ttesting.ConditionalTransport{
		Transport: ttesting.Transport{
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
	c.Assert(err, gocheck.IsNil)
	expected := `You've successfully started the password reset process.

Please check your email.` + "\n"
	c.Assert(buf.String(), gocheck.Equals, expected)
	c.Assert(called, gocheck.Equals, true)
}

func (s *S) TestResetPasswordStepTwo(c *gocheck.C) {
	var (
		buf    bytes.Buffer
		called bool
	)
	context := cmd.Context{Args: []string{"user@tsuru.io"}, Stdout: &buf}
	trans := ttesting.ConditionalTransport{
		Transport: ttesting.Transport{
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
	c.Assert(err, gocheck.IsNil)
	expected := `Your password has been reset and mailed to you.

Please check your email.` + "\n"
	c.Assert(buf.String(), gocheck.Equals, expected)
	c.Assert(called, gocheck.Equals, true)
}

func (s *S) TestResetPasswordInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:  "reset-password",
		Usage: "reset-password <email> [--token|-t <token>]",
		Desc: `Resets the user password.

This process is composed of two steps:

1. Generate a new token
2. Reset the password using the token

In order to generate the token, users should run this command without the --token flag.
The token will be mailed to the user.

With the token in hand, the user can finally reset the password using the --token flag.
The new password will also be mailed to the user.`,
		MinArgs: 1,
	}
	c.Assert((&resetPassword{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestResetPasswordFlags(c *gocheck.C) {
	command := resetPassword{}
	flagset := command.Flags()
	c.Assert(flagset, gocheck.NotNil)
	err := flagset.Parse(false, []string{"-t", "token123"})
	c.Assert(err, gocheck.IsNil)
	c.Assert(command.token, gocheck.Equals, "token123")
	token := flagset.Lookup("token")
	c.Assert(token, gocheck.NotNil)
	c.Check(token.Name, gocheck.Equals, "token")
	c.Check(token.Usage, gocheck.Equals, "Token to reset the password")
	c.Check(token.Value.String(), gocheck.Equals, "token123")
	c.Check(token.DefValue, gocheck.Equals, "")
	stoken := flagset.Lookup("t")
	c.Assert(stoken, gocheck.NotNil)
	c.Check(stoken.Name, gocheck.Equals, "t")
	c.Check(stoken.Usage, gocheck.Equals, "Token to reset the password")
	c.Check(stoken.Value.String(), gocheck.Equals, "token123")
	c.Check(stoken.DefValue, gocheck.Equals, "")
}

func (s *S) TestResetPasswordIsAFlaggedCommand(c *gocheck.C) {
	var _ cmd.FlaggedCommand = &resetPassword{}
}

func (s *S) TestShowAPITokenRun(c *gocheck.C) {
	var called bool
	trans := &ttesting.ConditionalTransport{
		Transport: ttesting.Transport{Message: `"23iou32nd3i2udnu23jd"`, Status: http.StatusOK},
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
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestShowAPITokenRunWithNoContent(c *gocheck.C) {
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&showAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, "")
}

func (s *S) TestShowAPITokenInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "token-show",
		Usage:   "token-show",
		Desc:    "Show API token user. If him does not have a key, it is generated.",
		MinArgs: 0,
	}
	c.Assert((&showAPIToken{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTShowAPITokenIsACommand(c *gocheck.C) {
	var _ cmd.Command = &showAPIToken{}
}

func (s *S) TestRegenerateAPITokenRun(c *gocheck.C) {
	var called bool
	trans := &ttesting.ConditionalTransport{
		Transport: ttesting.Transport{Message: `"23iou32nd3i2udnu23jd"`, Status: http.StatusOK},
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
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestRegenerateAPITokenRunWithNoContent(c *gocheck.C) {
	client := cmd.NewClient(&http.Client{Transport: &ttesting.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	var stdout, stderr bytes.Buffer
	err := (&regenerateAPIToken{}).Run(&cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  nil,
	}, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, "")
}

func (s *S) TestRegenerateAPITokenInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "token-regenerate",
		Usage:   "token-regenerate",
		Desc:    "Generates a new API key. If there is already a key, it is replaced.",
		MinArgs: 0,
	}
	c.Assert((&regenerateAPIToken{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTRegenerateAPITokenIsACommand(c *gocheck.C) {
	var _ cmd.Command = &regenerateAPIToken{}
}
