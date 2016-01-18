// Copyright 2015 tsuru-client authors. All rights reserved.
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

func (s *S) TestPoolListInfo(c *check.C) {
	c.Assert((&poolList{}).Info(), check.NotNil)
}

func (s *S) TestPoolListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"pools_by_team": [{"team": "test", "pools": ["pool"]}], "public_pools": [{"name": "public"}],"default_pool": [{"name": "default"}]}}`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := `+------+-------+
| Team | Pools |
+------+-------+
| test | pool  |
+------+-------+

+--------------+
| Public Pools |
+--------------+
| public       |
+--------------+

+--------------+
| Default Pool |
+--------------+
| default      |
+--------------+
`
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := poolList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
