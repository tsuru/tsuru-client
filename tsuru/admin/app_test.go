// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestAppRoutesRebuildInfo(c *check.C) {
	c.Assert((&AppRoutesRebuild{}).Info(), check.NotNil)
}

func (s *S) TestAppRoutesRebuildRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/app1/routes") && req.Method == "POST"
		},
	}

	s.setupFakeTransport(trans)

	command := AppRoutesRebuild{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `routes was rebuilt successfully
`)
}

func (s *S) TestAppRoutesRebuildFailed(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Some error", Status: http.StatusBadGateway},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/app1/routes") && req.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)

	command := AppRoutesRebuild{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context)
	c.Assert(err, check.Not(check.IsNil))
	c.Assert(err.Error(), check.Matches, ".*: Some error")
}
