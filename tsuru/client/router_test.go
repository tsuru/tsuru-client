// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/router"
	appTypes "github.com/tsuru/tsuru/types/app"
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
	r2 := router.PlanRouter{Name: "router2", Type: "bar", Info: map[string]string{"i1": "v1", "i2": "v2"}}
	data, err := json.Marshal([]router.PlanRouter{r1, r2})
	c.Assert(err, check.IsNil)

	expected := `+---------+------+--------+
| Name    | Type | Info   |
+---------+------+--------+
| router1 | foo  |        |
+---------+------+--------+
| router2 | bar  | i1: v1 |
|         |      | i2: v2 |
+---------+------+--------+
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

func (s *S) TestAppRoutersListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   nil,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	routers := []appTypes.AppRouter{
		{Name: "r1", Type: "r", Opts: map[string]string{"a": "b", "x": "y"}, Address: "addr1"},
		{Name: "r2", Address: "addr2", Status: "ready"},
		{Name: "r3", Type: "r3", Address: "addr3", Status: "not ready", StatusDetail: "something happening"},
	}
	data, err := json.Marshal(routers)
	c.Assert(err, check.IsNil)

	expected := `+------+------+------+---------+--------------------------------+
| Name | Type | Opts | Address | Status                         |
+------+------+------+---------+--------------------------------+
| r1   | r    | a: b | addr1   |                                |
|      |      | x: y |         |                                |
+------+------+------+---------+--------------------------------+
| r2   |      |      | addr2   | ready                          |
+------+------+------+---------+--------------------------------+
| r3   | r3   |      | addr3   | not ready: something happening |
+------+------+------+---------+--------------------------------+
`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/1.5/apps/myapp/routers") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppRoutersList{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRoutersListRunEmpty(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   nil,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "No routers available for app.\n"
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/1.5/apps/myapp/routers") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppRoutersList{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRoutersAddRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrouter"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Router successfully added.\n"
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			c.Assert(req.Form, check.DeepEquals, url.Values{
				"Opts.a":       []string{"b"},
				"Opts.x":       []string{"y"},
				"Name":         []string{"myrouter"},
				"Address":      []string{""},
				"Type":         []string{""},
				"Status":       []string{""},
				"StatusDetail": []string{""},
			})
			return strings.HasSuffix(req.URL.Path, "/1.5/apps/myapp/routers") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppRoutersAdd{}
	command.Flags().Parse(true, []string{"-a", "myapp", "-o", "a=b", "-o", "x=y"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRoutersUpdateRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrouter"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Router successfully updated.\n"
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			c.Assert(req.Form, check.DeepEquals, url.Values{
				"Opts.a":       []string{"b"},
				"Opts.x":       []string{"y"},
				"Name":         []string{"myrouter"},
				"Address":      []string{""},
				"Type":         []string{""},
				"Status":       []string{""},
				"StatusDetail": []string{""},
			})
			return strings.HasSuffix(req.URL.Path, "/1.5/apps/myapp/routers/myrouter") && req.Method == "PUT"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppRoutersUpdate{}
	command.Flags().Parse(true, []string{"-a", "myapp", "-o", "a=b", "-o", "x=y"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRoutersRemoveRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myrouter"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Router successfully removed.\n"
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/1.5/apps/myapp/routers/myrouter") && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppRoutersRemove{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
