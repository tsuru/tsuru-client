// Copyright 2018 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestTokenCreateCmdInfo(c *check.C) {
	c.Assert((&TokenCreateCmd{}).Info(), check.NotNil)
}

func (s *S) TestTokenCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"token_id": "mytokenid", "token": "mytokenvalue"}`
	expected := "Token \"mytokenid\" created: mytokenvalue\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/tokens")
			c.Assert(r.Method, check.Equals, "POST")
			var ret map[string]interface{}
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, map[string]interface{}{})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := TokenCreateCmd{}
	command.Flags().Parse([]string{})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTokenCreateWithFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"token_id": "mytokenid", "token": "mytokenvalue"}`
	expected := "Token \"mytokenid\" created: mytokenvalue\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/tokens")
			c.Assert(r.Method, check.Equals, "POST")
			var ret map[string]interface{}
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, map[string]interface{}{
				"token_id":    "myid",
				"description": "mydesc",
				"expires_in":  float64(180),
				"team":        "myteam",
			})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := TokenCreateCmd{}
	command.Flags().Parse([]string{
		"--id", "myid",
		"--team", "myteam",
		"--description", "mydesc",
		"--expires", "3m",
	})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTokenUpdateCmdInfo(c *check.C) {
	c.Assert((&TokenUpdateCmd{}).Info(), check.NotNil)
}

func (s *S) TestTokenUpdate(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"token_id": "mytokenid", "token": "mytokenvalue"}`
	expected := "Token \"mytokenid\" updated: mytokenvalue\n"
	context := cmd.Context{
		Args:   []string{"mytokenid"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/tokens/mytokenid")
			c.Assert(r.Method, check.Equals, "PUT")
			var ret map[string]interface{}
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, map[string]interface{}{})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := TokenUpdateCmd{}
	command.Flags().Parse([]string{})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTokenUpdateWithFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"token_id": "mytokenid", "token": "mytokenvalue"}`
	expected := "Token \"mytokenid\" updated: mytokenvalue\n"
	context := cmd.Context{
		Args:   []string{"mytokenid"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/tokens/mytokenid")
			c.Assert(r.Method, check.Equals, "PUT")
			var ret map[string]interface{}
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, map[string]interface{}{
				"description": "mydesc",
				"expires_in":  float64(180),
				"regenerate":  true,
			})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := TokenUpdateCmd{}
	command.Flags().Parse([]string{
		"--description", "mydesc",
		"--expires", "3m",
		"--regenerate",
	})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTokenListCmdInfo(c *check.C) {
	c.Assert((&TokenListCmd{}).Info(), check.NotNil)
}

func (s *S) TestTokenList(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[
		{
			"token_id": "mytokenid", "token": "mytokenvalue", "team": "myteam",
			"description": "desc", "creator_email": "me@me",
			"created_at": "2018-02-20T20:20:20.00-03:00",
			"roles": [
				{"name": "r1", "contextvalue": "v1"},
				{"name": "r2", "contextvalue": "v2"}
			]
		},
		{
			"token_id": "othertoken", "token": "", "team": "myteam",
			"description": "desc", "creator_email": "me@me",
			"created_at": "2018-02-20T20:20:20.00-03:00",
			"expires_at": "2018-02-20T20:20:20.00-03:00",
			"last_access": "2018-02-20T20:20:20.00-03:00",
			"roles": [
				{"name": "r1", "contextvalue": "v1"},
				{"name": "r2", "contextvalue": "v2"}
			]
		}
	]`
	expected := `+------------+--------+---------------------+---------------------+---------------------+--------+
| Token ID   | Team   | Created At          | Expires At          | Last Access         | Roles  |
+------------+--------+---------------------+---------------------+---------------------+--------+
| mytokenid  | myteam | 20 Feb 18 17:20 CST | -                   | -                   | r1(v1) |
|            |        |                     |                     |                     | r2(v2) |
+------------+--------+---------------------+---------------------+---------------------+--------+
| othertoken | myteam | 20 Feb 18 17:20 CST | 20 Feb 18 17:20 CST | 20 Feb 18 17:20 CST | r1(v1) |
|            |        |                     |                     |                     | r2(v2) |
+------------+--------+---------------------+---------------------+---------------------+--------+
`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/tokens")
			c.Assert(r.Method, check.Equals, "GET")
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := TokenListCmd{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTokenDeleteCmdInfo(c *check.C) {
	c.Assert((&TokenDeleteCmd{}).Info(), check.NotNil)
}

func (s *S) TestTokenDelete(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Token successfully deleted.\n"
	context := cmd.Context{
		Args:   []string{"mytokenid"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/tokens/mytokenid")
			c.Assert(r.Method, check.Equals, "DELETE")
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := TokenDeleteCmd{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTokenInfoCmdInfo(c *check.C) {
	c.Assert((&TokenInfoCmd{}).Info(), check.NotNil)
}

func (s *S) TestTokenInfo(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `
		{
			"token_id": "mytokenid",
			"token": "mytokenvalue",
			"description": "desc",
			"created_at": "2018-02-20T20:20:20.00-03:00",
			"expires_at": "2018-02-20T20:20:20.00-03:00",
			"last_access": "2018-02-20T20:20:20.00-03:00",
			"creator_email": "me@me",
			"team": "myteam",
			"roles": [
				{"name": "r1", "contextvalue": "v1"}
			]
		}`
	expected := `Token: mytokenvalue
Token Id: mytokenid
Description: desc
Created at: 20 Feb 18 17:20 CST
Expires at: 20 Feb 18 17:20 CST
Last Acess: 20 Feb 18 17:20 CST
Creator: me@me
Team: myteam
Roles: r1(v1)
`
	context := cmd.Context{
		Args:   []string{"mytokenid"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.7/tokens/mytokenid")
			c.Assert(r.Method, check.Equals, "GET")
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := TokenInfoCmd{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
