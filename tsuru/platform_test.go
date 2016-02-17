// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestPlatformList(c *check.C) {
	var buf bytes.Buffer
	var called bool
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status:  http.StatusOK,
			Message: `[{"Name":"ruby"},{"Name":"python"}]`,
		},
		CondFunc: func(r *http.Request) bool {
			called = true
			return r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/platforms")
		},
	}
	context := cmd.Context{Stdout: &buf}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	err := platformList{}.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := `- python
- ruby` + "\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestPlatformListWithDisabledPlatforms(c *check.C) {
	var buf bytes.Buffer
	var called bool
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status:  http.StatusOK,
			Message: `[{"Name":"ruby"},{"Name":"python"},{"Name":"ruby20", "Disabled":true}]`,
		},
		CondFunc: func(r *http.Request) bool {
			called = true
			return r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/platforms")
		},
	}
	context := cmd.Context{Stdout: &buf}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	err := platformList{}.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := `- python
- ruby
- ruby20 (disabled)` + "\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestPlatformListEmpty(c *check.C) {
	var buf bytes.Buffer
	transport := cmdtest.Transport{
		Status:  http.StatusOK,
		Message: `[]`,
	}
	context := cmd.Context{Stdout: &buf}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	err := platformList{}.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "No platforms available.\n")
}

func (s *S) TestPlatformListInfo(c *check.C) {
	c.Assert(platformList{}.Info(), check.NotNil)
}

func (s *S) TestPlatformListIsACommand(c *check.C) {
	var _ cmd.Command = platformList{}
}
