// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	appTypes "github.com/tsuru/tsuru/types/app"
	"github.com/tsuru/tsuru/types/router"
	check "gopkg.in/check.v1"
)

func (s *S) TestRoutersListInfo(c *check.C) {
	c.Assert((&RoutersList{}).Info(), check.NotNil)
}

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
	s.setupFakeTransport(trans)
	command := RoutersList{}
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRoutersListInfo(c *check.C) {
	c.Assert((&AppRoutersList{}).Info(), check.NotNil)
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
	s.setupFakeTransport(trans)
	command := AppRoutersList{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err = command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := AppRoutersList{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRoutersAddInfo(c *check.C) {
	c.Assert((&AppRoutersAdd{}).Info(), check.NotNil)
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
				"Addresses":    []string{""},
				"Type":         []string{""},
				"Status":       []string{""},
				"StatusDetail": []string{""},
			})
			return strings.HasSuffix(req.URL.Path, "/1.5/apps/myapp/routers") && req.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)
	command := AppRoutersAdd{}
	command.Flags().Parse(true, []string{"-a", "myapp", "-o", "a=b", "-o", "x=y"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRoutersUpdateInfo(c *check.C) {
	c.Assert((&AppRoutersUpdate{}).Info(), check.NotNil)
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
				"Addresses":    []string{""},
				"Type":         []string{""},
				"Status":       []string{""},
				"StatusDetail": []string{""},
			})
			return strings.HasSuffix(req.URL.Path, "/1.5/apps/myapp/routers/myrouter") && req.Method == "PUT"
		},
	}
	s.setupFakeTransport(trans)
	command := AppRoutersUpdate{}
	command.Flags().Parse(true, []string{"-a", "myapp", "-o", "a=b", "-o", "x=y"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRoutersRemoveInfo(c *check.C) {
	c.Assert((&AppRoutersRemove{}).Info(), check.NotNil)
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
	s.setupFakeTransport(trans)
	command := AppRoutersRemove{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRouterAddInfo(c *check.C) {
	c.Assert((&RouterAdd{}).Info(), check.NotNil)
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
			data, err := io.ReadAll(r.Body)
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
	s.setupFakeTransport(&trans)
	command := AppVersionRouterAdd{}
	command.Info()
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestRouterRemoveInfo(c *check.C) {
	c.Assert((&RouterRemove{}).Info(), check.NotNil)
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
			data, err := io.ReadAll(r.Body)
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
	s.setupFakeTransport(&trans)
	command := AppVersionRouterRemove{}
	command.Info()
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
