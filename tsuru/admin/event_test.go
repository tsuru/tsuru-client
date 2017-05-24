// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"net/http"
	"os"

	"github.com/ajg/form"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/event"
	check "gopkg.in/check.v1"
)

func (s *S) TestEventBlockList(c *check.C) {
	os.Setenv("TSURU_DISABLE_COLORS", "1")
	defer os.Unsetenv("TSURU_DISABLE_COLORS")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	blocksData := `
[{
    "ID": "58c6db0b0640fd2fec413cc6",
    "StartTime": "2017-03-13T17:46:51.326Z",
    "EndTime": "0001-01-01T00:00:00Z",
    "KindName": "app.create",
    "OwnerName": "user@email.com",
    "Target": {
      "Type": "",
      "Value": ""
    },
    "Reason": "Problems",
    "Active": true
  }, {
	"ID": "58c1d29ac47369e95c5520c8",
	"StartTime": "2017-03-13T16:43:09.888Z",
	"EndTime": "2017-03-13T17:27:25.149Z",
	"KindName": "app.deploy",
	"OwnerName": "",
	"Target": {
		"Type": "",
		"Value": ""
	},
	"Reason": "Maintenance.",
	"Active": false
}]`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: blocksData, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.3/events/blocks"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := EventBlockList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `+--------------------------+------------------------------------+------------+----------------+----------------------+--------------+
| ID                       | Start (duration)                   | Kind       | Owner          | Target (Type: Value) | Reason       |
+--------------------------+------------------------------------+------------+----------------+----------------------+--------------+
| 58c6db0b0640fd2fec413cc6 | 13 Mar 17 17:46 +0000              | app.create | user@email.com | all: all             | Problems     |
| 58c1d29ac47369e95c5520c8 | 13 Mar 17 16:43 +0000 (44m15.261s) | app.deploy | all            | all: all             | Maintenance. |
+--------------------------+------------------------------------+------------+----------------+----------------------+--------------+
`
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestEventBlockListNoEvents(c *check.C) {
	os.Setenv("TSURU_DISABLE_COLORS", "1")
	defer os.Unsetenv("TSURU_DISABLE_COLORS")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.3/events/blocks"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := EventBlockList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `+----+------------------+------+-------+----------------------+--------+
| ID | Start (duration) | Kind | Owner | Target (Type: Value) | Reason |
+----+------------------+------+-------+----------------------+--------+
+----+------------------+------+-------+----------------------+--------+
`
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestEventBlockAdd(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"Reason"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			var block event.Block
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.IgnoreCase(true)
			err = dec.DecodeValues(&block, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(block, check.DeepEquals, event.Block{Reason: "Reason", Active: false})
			return req.URL.Path == "/1.3/events/blocks" && req.Method == http.MethodPost
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := EventBlockAdd{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Block successfully added.\n")
}

func (s *S) TestEventBlockAddAllFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"Reason"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			var block event.Block
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.IgnoreCase(true)
			err = dec.DecodeValues(&block, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(block, check.DeepEquals, event.Block{
				KindName:  "app.deploy",
				OwnerName: "user@email.com",
				Target:    event.Target{Type: event.TargetTypeApp, Value: "myapp"},
				Reason:    "Reason",
				Active:    false,
			})
			return req.URL.Path == "/1.3/events/blocks" && req.Method == http.MethodPost
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := EventBlockAdd{}
	command.Flags().Parse(true, []string{"-k", "app.deploy", "-o", "user@email.com", "-t", "app", "-v", "myapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Block successfully added.\n")
}

func (s *S) TestEventBlockRemove(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"ABC123K12"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.3/events/blocks/ABC123K12" && req.Method == http.MethodDelete
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := EventBlockRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Block ABC123K12 successfully removed.\n")
}
