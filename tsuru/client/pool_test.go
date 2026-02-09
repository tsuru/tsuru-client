// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestPoolListInfo(c *check.C) {
	c.Assert((&PoolList{}).Info(), check.NotNil)
}

func (s *S) TestPoolListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"Name":"theonepool","Public":true,"Default":true,"Allowed":{"router":["hipache"]}},{"Name":"pool1","Public":false,"Default":true},{"Name":"pool2","Public":false,"Default":false,"Allowed":{"team":["admin"]}},{"Name":"pool0","Public":false,"Default":false,"Allowed":{"team":["admin"]}},{"Name":"pool3","Public":false,"Default":false,"Provisioner":"swarm","Allowed":{"router":["hipache","planb"],"team":["admin","team1","team2","team3","team4","team5"]}}]`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := `+------------+---------+-------------+-----------------------------+----------------+
| Pool       | Kind    | Provisioner | Teams                       | Routers        |
+------------+---------+-------------+-----------------------------+----------------+
| pool0      |         | default     | admin                       |                |
+------------+---------+-------------+-----------------------------+----------------+
| pool2      |         | default     | admin                       |                |
+------------+---------+-------------+-----------------------------+----------------+
| pool3      |         | swarm       | admin, team1, team2, team3, | hipache, planb |
|            |         |             | team4, team5                |                |
+------------+---------+-------------+-----------------------------+----------------+
| pool1      | default | default     |                             |                |
+------------+---------+-------------+-----------------------------+----------------+
| theonepool | public  | default     |                             | hipache        |
+------------+---------+-------------+-----------------------------+----------------+
`
	s.setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})
	command := PoolList{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPoolListRunNoContent(c *check.C) {
	var stdout bytes.Buffer
	context := cmd.Context{Args: []string{}, Stdout: &stdout}
	s.setupFakeTransport(&cmdtest.Transport{Status: http.StatusNoContent})
	command := PoolList{}
	err := command.Run(&context)
	expected := `+------+------+-------------+-------+---------+
| Pool | Kind | Provisioner | Teams | Routers |
+------+------+-------------+-------+---------+
`
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPoolInfoInfo(c *check.C) {
	c.Assert((&PoolInfo{}).Info(), check.NotNil)
}

func (s *S) TestPoolInfoRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"pool1"},
	}
	pool := tsuru.Pool{
		Name:        "pool1",
		Provisioner: "kubernetes",
		Public:      false,
		Default:     false,
		Teams:       []string{"team1", "team2"},
		Allowed: map[string][]string{
			"router":  {"router1", "router2"},
			"service": {"service1"},
		},
		Labels: map[string]string{
			"env":    "production",
			"region": "us-east-1",
		},
	}
	data, err := json.Marshal(pool)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.URL.Path, check.Equals, "/pools/pool1")
			c.Assert(req.Method, check.Equals, http.MethodGet)
			return true
		},
	}
	s.setupFakeTransport(trans)
	command := PoolInfo{}
	err = command.Run(&ctx)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `Name:         pool1
Provisioner:  kubernetes
Teams:        team1
              team2

Allowed:
+---------+----------+
| Type    | Value    |
+---------+----------+
| router  | router1  |
|         | router2  |
+---------+----------+
| service | service1 |
+---------+----------+

Labels:
+--------+------------+
| Key    | Value      |
+--------+------------+
| env    | production |
| region | us-east-1  |
+--------+------------+
`)
}

func (s *S) TestPoolInfoRunPublic(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"pool1"},
	}
	pool := tsuru.Pool{
		Name:        "pool1",
		Provisioner: "",
		Public:      true,
		Default:     false,
	}
	data, err := json.Marshal(pool)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return true
		},
	}
	s.setupFakeTransport(trans)
	command := PoolInfo{}
	err = command.Run(&ctx)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `Name:         pool1
Kind:         public
Provisioner:  default
`)
}

func (s *S) TestPoolInfoRunDefault(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"pool1"},
	}
	pool := tsuru.Pool{
		Name:        "pool1",
		Provisioner: "swarm",
		Public:      false,
		Default:     true,
	}
	data, err := json.Marshal(pool)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return true
		},
	}
	s.setupFakeTransport(trans)
	command := PoolInfo{}
	err = command.Run(&ctx)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `Name:         pool1
Kind:         default
Provisioner:  swarm
`)
}

func (s *S) TestPoolInfoRunNoContent(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"pool1"},
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return true
		},
	}
	s.setupFakeTransport(trans)
	command := PoolInfo{}
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestPoolInfoRunJSON(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"pool1"},
	}
	pool := tsuru.Pool{
		Name:        "pool1",
		Provisioner: "kubernetes",
		Public:      false,
		Default:     false,
		Teams:       []string{"team1", "team2"},
		Labels: map[string]string{
			"env": "production",
		},
	}
	data, err := json.Marshal(pool)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return true
		},
	}
	s.setupFakeTransport(trans)
	command := PoolInfo{}
	command.Flags().Parse([]string{"--json"})
	err = command.Run(&ctx)
	c.Assert(err, check.IsNil)
	var result tsuru.Pool
	err = json.Unmarshal(stdout.Bytes(), &result)
	c.Assert(err, check.IsNil)
	c.Assert(result.Name, check.Equals, "pool1")
	c.Assert(result.Provisioner, check.Equals, "kubernetes")
	c.Assert(result.Teams, check.DeepEquals, []string{"team1", "team2"})
}
