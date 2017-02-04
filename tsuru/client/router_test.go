// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/router"
	check "gopkg.in/check.v1"
)

func (s *S) TestRoutersListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   nil,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	r1 := router.PlanRouter{Name: "router1", Type: "foo"}
	r2 := router.PlanRouter{Name: "router2", Type: "bar"}
	data, err := json.Marshal([]router.PlanRouter{r1, r2})
	c.Assert(err, check.IsNil)
	expected := `+---------+------+
| Name    | Type |
+---------+------+
| router1 | foo  |
+---------+------+
| router2 | bar  |
+---------+------+
`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/1.3/routers") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := RoutersList{}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
