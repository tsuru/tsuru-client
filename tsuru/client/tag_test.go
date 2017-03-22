// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestTagListWithApps(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"name":"app1","tags":["tag1"]},{"name":"app2","tags":["tag2","tag3"]},{"name":"app3","tags":[]},{"name":"app4","tags":["tag1","tag3"]}]`
	expected := `+------+------------+
| Tag  | Apps       |
+------+------------+
| tag1 | app1, app4 |
+------+------------+
| tag2 | app2       |
+------+------------+
| tag3 | app2, app4 |
+------+------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := TagList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTagListWithEmptyResponse(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"name":"app1","tags":[]}]`
	expected := ""
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := TagList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
