// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestPoolListInfo(c *check.C) {
	c.Assert((&PoolList{}).Info(), check.NotNil)
}

func (s *S) TestPoolListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"Name":"theonepool","Teams":[],"Public":true,"Default":true,"Allowed":{"router":["hipache"]}},{"Name":"pool1","Teams":[],"Public":false,"Default":true},{"Name":"pool2","Teams":["admin"],"Public":false,"Default":false,"Allowed":{"team":["admin"]}},{"Name":"pool0","Teams":["admin"],"Public":false,"Default":false,"Allowed":{"team":["admin"]}},{"Name":"pool3","Teams":["admin"],"Public":false,"Default":false,"Provisioner":"swarm","Allowed":{"router":["hipache","planb"],"team":["admin"]}}]`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := `+------------+---------+-------------+-------+----------------+
| Pool       | Kind    | Provisioner | Teams | Routers        |
+------------+---------+-------------+-------+----------------+
| pool0      |         | default     | admin |                |
| pool2      |         | default     | admin |                |
| pool3      |         | swarm       | admin | hipache, planb |
| pool1      | default | default     |       |                |
| theonepool | public  | default     |       | hipache        |
+------------+---------+-------------+-------+----------------+
`
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := PoolList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
