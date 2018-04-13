// Copyright 2018 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestTokenCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"token_id": "mytokenid", "token": "mytokenvalue"}`
	expected := "Token \"mytokenid\" created: mytokenvalue"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			var ret map[string]interface{}
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(ret, check.DeepEquals, map[string]interface{}{})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := TokenCreateCmd{}
	command.Flags().Parse(true, []string{})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTokenCreateWithFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"token_id": "mytokenid", "token": "mytokenvalue"}`
	expected := "Token \"mytokenid\" created: mytokenvalue"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			var ret map[string]interface{}
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(ret, check.DeepEquals, map[string]interface{}{
				"token_id":    "myid",
				"description": "mydesc",
				"expires_in":  float64(180),
				"team":        "myteam",
			})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := TokenCreateCmd{}
	command.Flags().Parse(true, []string{
		"--id", "myid",
		"--team", "myteam",
		"--description", "mydesc",
		"--expires", "3m",
	})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
