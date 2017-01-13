// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/router/rebuild"
	"gopkg.in/check.v1"
)

func (s *S) TestAppLockDeleteRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Lock successfully removed!\n"
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/app1/lock") && req.Method == http.MethodDelete
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := AppLockDelete{}
	command.Flags().Parse(true, []string{"--app", "app1", "-y"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLockDeleteRunAsksConfirmation(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("n\n"),
	}
	command := AppLockDelete{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Are you sure you want to remove the lock from app \"app1\"? (y/n) Abort.\n")
}

func (s *S) TestAppRoutesRebuildRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	rebuildResult := rebuild.RebuildRoutesResult{
		Added:   []string{"r1", "r2"},
		Removed: []string{"r9"},
	}
	data, err := json.Marshal(rebuildResult)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/app1/routes") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := AppRoutesRebuild{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `Added routes:
- r1
- r2
Removed routes:
- r9

Routes successfully rebuilt!
`)
}

func (s *S) TestAppRoutesRebuildRunNothingToDo(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	rebuildResult := rebuild.RebuildRoutesResult{}
	data, err := json.Marshal(rebuildResult)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/app1/routes") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := AppRoutesRebuild{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Nothing to do, routes already correct.\n")
}
