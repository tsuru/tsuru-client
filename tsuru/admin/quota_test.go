// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"net/http"
	"strings"

	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestUserQuotaViewInfo(c *check.C) {
	c.Assert((&UserQuotaView{}).Info(), check.NotNil)
}

func (s *S) TestUserQuotaViewRun(c *check.C) {
	result := `{"inuse":3,"limit":4}`
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"fss@corp.globo.com"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/users/fss@corp.globo.com/quota")
		},
	}
	s.setupFakeTransport(&trans)
	command := UserQuotaView{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	expected := `User: fss@corp.globo.com
Apps usage: 3/4
`
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestUserQuotaViewRunFailure(c *check.C) {
	context := cmd.Context{Args: []string{"fss@corp.globo.com"}}
	trans := cmdtest.Transport{Message: "user not found", Status: http.StatusNotFound}
	s.setupFakeTransport(&trans)
	command := UserQuotaView{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "user not found")
}

func (s *S) TestUserChangeQuotaInfo(c *check.C) {
	c.Assert((&UserChangeQuota{}).Info(), check.NotNil)
}

func (s *S) TestUserChangeQuotaRun(c *check.C) {
	var called bool
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"fss@corp.globo.com", "5"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			path := strings.HasSuffix(req.URL.Path, "/users/fss@corp.globo.com/quota")
			method := req.Method == "PUT"
			limit := req.FormValue("limit") == "5"
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
			return path && method && limit
		},
	}
	s.setupFakeTransport(&trans)
	command := UserChangeQuota{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Quota successfully updated.\n")
	c.Assert(called, check.Equals, true)
}

func (s *S) TestUserChangeQuotaRunUnlimited(c *check.C) {
	var called bool
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"fss@corp.globo.com", "unlimited"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			path := strings.HasSuffix(req.URL.Path, "/users/fss@corp.globo.com/quota")
			method := req.Method == "PUT"
			limit := req.FormValue("limit") == "-1"
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
			return path && method && limit
		},
	}
	s.setupFakeTransport(&trans)
	command := UserChangeQuota{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Quota successfully updated.\n")
	c.Assert(called, check.Equals, true)
}

func (s *S) TestUserChangeQuotaRunInvalidLimit(c *check.C) {
	context := cmd.Context{Args: []string{"fss@corp.globo.com", "unlimiteddd"}}
	command := UserChangeQuota{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, `invalid limit. It must be either an integer or "unlimited"`)
}

func (s *S) TestUserChangeQuotaFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	trans := &cmdtest.Transport{
		Message: "user not found",
		Status:  http.StatusNotFound,
	}
	context := cmd.Context{
		Args:   []string{"fss@corp.globo.com", "5"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(trans)
	command := UserChangeQuota{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "user not found")
}

func (s *S) TestAppQuotaViewInfo(c *check.C) {
	c.Assert((&AppQuotaView{}).Info(), check.NotNil)
}

func (s *S) TestAppQuotaViewRun(c *check.C) {
	result := `{"inuse":3,"limit":4}`
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/hibria/quota")
		},
	}
	s.setupFakeTransport(&trans)
	command := AppQuotaView{}
	command.Flags().Parse(true, []string{"--app", "hibria"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	expected := `App: hibria
Units usage: 3/4
`
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppQuotaViewRunFailure(c *check.C) {
	context := cmd.Context{Args: []string{"hybria"}}
	trans := cmdtest.Transport{Message: "app not found", Status: http.StatusNotFound}
	s.setupFakeTransport(&trans)
	command := AppQuotaView{}
	command.Flags().Parse(true, []string{})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "app not found")
}

func (s *S) TestAppQuotaChangeInfo(c *check.C) {
	c.Assert((&AppQuotaChange{}).Info(), check.NotNil)
}

func (s *S) TestAppQuotaChangeRun(c *check.C) {
	var called bool
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			url := strings.HasSuffix(req.URL.Path, "/apps/myapp/quota")
			method := req.Method == "PUT"
			limit := req.FormValue("limit") == "5"
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
			return url && method && limit
		},
	}
	s.setupFakeTransport(&trans)
	command := AppQuotaChange{}
	command.Flags().Parse(true, []string{"--app", "myapp", "5"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Quota successfully updated.\n")
	c.Assert(called, check.Equals, true)
}

func (s *S) TestAppQuotaChangeRunUnlimited(c *check.C) {
	var called bool
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			url := strings.HasSuffix(req.URL.Path, "/apps/myapp/quota")
			method := req.Method == "PUT"
			limit := req.FormValue("limit") == "-1"
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
			return url && method && limit
		},
	}
	s.setupFakeTransport(&trans)
	command := AppQuotaChange{}
	command.Flags().Parse(true, []string{"--app", "myapp", "unlimited"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Quota successfully updated.\n")
	c.Assert(called, check.Equals, true)
}

func (s *S) TestAppQuotaChangeRunInvalidLimit(c *check.C) {
	context := cmd.Context{}
	command := AppQuotaChange{}
	command.Flags().Parse(true, []string{"-a", "myapp", "unlimiteddd"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, `invalid limit. It must be either an integer or "unlimited"`)
}

func (s *S) TestAppQuotaChangeFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	trans := &cmdtest.Transport{
		Message: "app not found",
		Status:  http.StatusNotFound,
	}
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(trans)
	command := AppQuotaChange{}
	command.Flags().Parse(true, []string{"-a", "myapp", "5"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "app not found")
}

func (s *S) TestTeamQuotaViewRun(c *check.C) {
	result := `{"inuse":3,"limit":4}`
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myteam"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/teams/myteam/quota")
		},
	}
	s.setupFakeTransport(&trans)
	command := TeamQuotaView{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	expected := `Team: myteam
Apps usage: 3/4
`
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTeamQuotaViewRunFailure(c *check.C) {
	context := cmd.Context{Args: []string{"myteam"}}
	trans := cmdtest.Transport{Message: "team not found", Status: http.StatusNotFound}
	s.setupFakeTransport(trans)
	command := TeamQuotaView{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "team not found")
}

func (s *S) TestTeamChangeQuotaRun(c *check.C) {
	var called bool
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myteam", "5"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			path := strings.HasSuffix(req.URL.Path, "/teams/myteam/quota")
			method := req.Method == "PUT"
			limit := req.FormValue("limit") == "5"
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
			return path && method && limit
		},
	}
	s.setupFakeTransport(&trans)
	command := TeamChangeQuota{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Quota successfully updated.\n")
	c.Assert(called, check.Equals, true)
}

func (s *S) TestTeamChangeQuotaRunUnlimited(c *check.C) {
	var called bool
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myteam", "unlimited"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			path := strings.HasSuffix(req.URL.Path, "/teams/myteam/quota")
			method := req.Method == "PUT"
			limit := req.FormValue("limit") == "-1"
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
			return path && method && limit
		},
	}
	s.setupFakeTransport(&trans)
	command := TeamChangeQuota{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Quota successfully updated.\n")
	c.Assert(called, check.Equals, true)
}

func (s *S) TestTeamChangeQuotaRunInvalidLimit(c *check.C) {
	context := cmd.Context{Args: []string{"myteam", "unlimiteddd"}}
	command := TeamChangeQuota{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, `invalid limit. It must be either an integer or "unlimited"`)
}

func (s *S) TestTeamChangeQuotaFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myteam", "5"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.Transport{
		Message: "team not found",
		Status:  http.StatusNotFound,
	}
	s.setupFakeTransport(trans)
	command := TeamChangeQuota{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "team not found")
}
