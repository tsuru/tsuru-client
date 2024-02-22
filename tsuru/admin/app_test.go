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

func (s *S) TestAppRoutesRebuildInfo(c *check.C) {
	c.Assert((&AppRoutesRebuild{}).Info(), check.NotNil)
}

func (s *S) TestAppRoutesRebuildRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	rebuildResult := `{
"r1": {
	"added": ["r1", "r2"],
	"removed": ["r9"]
},
"r2": {
	"removed": ["r9"]
},
"r3": {}
}`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: rebuildResult, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/app1/routes") && req.Method == "POST"
		},
	}

	s.setupFakeTransport(trans)

	command := AppRoutesRebuild{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `Router r1:
  * Added routes:
    - r1
    - r2
  * Removed routes:
    - r9
Router r2:
  * Removed routes:
    - r9
Router r3:
  * Nothing to do, routes already correct.
`)
}

func (s *S) TestAppRoutesRebuildRunWithPrefixes(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	rebuildResult := `{
"r1": {
	"PrefixResults": [
		{
			"prefix": "",
			"added": ["r1", "r2"],
			"removed": ["r9"]
		},
		{
			"prefix": "v1.version",
			"added": ["r3"],
			"removed": ["r8"]
		}
	],
	"added": ["r1", "r2"],
	"removed": ["r9"]
},
"r2": {
	"PrefixResults": [
		{
			"prefix": "",
			"removed": ["r9"]
		}
	],
	"removed": ["r9"]
},
"r3": {
	"PrefixResults": [
		{
			"prefix": ""
		}
	]
}
}`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: rebuildResult, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/app1/routes") && req.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)
	command := AppRoutesRebuild{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `Router r1:
 - Prefix "":
  * Added routes:
    - r1
    - r2
  * Removed routes:
    - r9
 - Prefix "v1.version":
  * Added routes:
    - r3
  * Removed routes:
    - r8
Router r2:
 - Prefix "":
  * Removed routes:
    - r9
Router r3:
 - Prefix "":
  * Nothing to do, routes already correct.
`)
}

func (s *S) TestAppRoutesRebuildRunNothingToDo(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	rebuildResult := map[string]rebuild.RebuildRoutesResult{
		"r1": {},
	}
	data, err := json.Marshal(rebuildResult)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/app1/routes") && req.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)

	command := AppRoutesRebuild{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `Router r1:
  * Nothing to do, routes already correct.
`)
}

func (s *S) TestAppRoutesRebuildRunNoRouters(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	rebuildResult := map[string]rebuild.RebuildRoutesResult{}
	data, err := json.Marshal(rebuildResult)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/app1/routes") && req.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)

	command := AppRoutesRebuild{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "App has no routers.\n")
}
