// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"testing"

	"gopkg.in/check.v1"
)

type S struct{}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) TestNewDockerMachine(c *check.C) {
	dm, err := NewDockerMachine("virtualbox", nil)
	c.Assert(err, check.IsNil)
	c.Assert(dm, check.NotNil)
	c.Assert(dm.rawDriver, check.NotNil)
	c.Assert(dm.driverName, check.Equals, "virtualbox")
}

func (s *S) TestNewDockerMachineDriverOpts(c *check.C) {
	dm, err := NewDockerMachine("none", map[string]interface{}{"url": "localhost"})
	c.Assert(err, check.IsNil)
	c.Assert(dm, check.NotNil)
	c.Assert(dm.driverOpts.String("url"), check.Equals, "localhost")
}

func (s *S) TestCreateMachineNoneDriver(c *check.C) {
	dm, _ := NewDockerMachine("none", map[string]interface{}{"url": "http://1.2.3.4"})
	machine, err := dm.CreateMachine(nil)
	c.Assert(err, check.IsNil)
	c.Assert(machine, check.NotNil)
	c.Assert(machine.IP, check.Equals, "1.2.3.4")
	c.Assert(machine.Address, check.Equals, "http://1.2.3.4:2375")
}
