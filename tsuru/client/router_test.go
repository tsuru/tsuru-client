// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
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

	expected := `+------+------+-----------+--------------------------------+
| Name | Opts | Addresses | Status                         |
+------+------+-----------+--------------------------------+
| r1   | a: b | addr1     |                                |
|      | x: y |           |                                |
+------+------+-----------+--------------------------------+
| r2   |      | addr2     | ready                          |
+------+------+-----------+--------------------------------+
| r3   |      | addr3     | not ready: something happening |
+------+------+-----------+--------------------------------+
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
			var routerAdd tsuru.AppRouter
			data, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &routerAdd)
			c.Assert(err, check.IsNil)
			c.Assert(routerAdd, check.DeepEquals, tsuru.AppRouter{
				Opts: map[string]interface{}{"a": "b", "x": "y"},
				Name: "myrouter",
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
			var routerUpdt tsuru.AppRouter
			data, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &routerUpdt)
			c.Assert(err, check.IsNil)
			c.Assert(routerUpdt, check.DeepEquals, tsuru.AppRouter{
				Opts: map[string]interface{}{"a": "b", "x": "y"},
				Name: "myrouter",
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

func (s *S) TestAppVersionRouterAdd(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Version successfully updated.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"2"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.8/apps/myapp/routable")
			c.Assert(r.Method, check.Equals, "POST")
			var ret tsuru.SetRoutableArgs
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.SetRoutableArgs{
				Version:    "2",
				IsRoutable: true,
			})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppVersionRouterAdd{}
	command.Info()
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppVersionRouterRemove(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Version successfully updated.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"2"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.8/apps/myapp/routable")
			c.Assert(r.Method, check.Equals, "POST")
			var ret tsuru.SetRoutableArgs
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.SetRoutableArgs{
				Version:    "2",
				IsRoutable: false,
			})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppVersionRouterRemove{}
	command.Info()
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
