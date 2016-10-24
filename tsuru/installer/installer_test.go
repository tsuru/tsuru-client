// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import check "gopkg.in/check.v1"

func (s *S) TestBuildClusterTable(c *check.C) {
	i := &Installation{CoreCluster: &FakeServiceCluster{}}
	table := i.buildClusterTable()
	expected := `+-----------+---------+---------+
| IP        | State   | Manager |
+-----------+---------+---------+
| 127.0.0.1 | running | true    |
+-----------+---------+---------+
`
	c.Assert(table.String(), check.Equals, expected)
}

func (s *S) TestBuildComponentsTable(c *check.C) {
	i := &Installation{CoreCluster: &FakeServiceCluster{}, Components: []TsuruComponent{&MongoDB{}}}
	table := i.buildComponentsTable()
	expected := `+-----------+-------+----------+
| Component | Ports | Replicas |
+-----------+-------+----------+
| MongoDB   | 8080  | 1        |
+-----------+-------+----------+
`
	c.Assert(table.String(), check.Equals, expected)
}
