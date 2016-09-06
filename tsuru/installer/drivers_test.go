// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"github.com/tsuru/config"
	check "gopkg.in/check.v1"
)

func (s *S) TestGetPrivateIPInterfaceFromConfig(c *check.C) {
	config.Set("driver:private-ip-interface", "eth1")
	defer config.Unset("driver:private-ip-interface")
	iface, err := GetPrivateIPInterface("")
	c.Assert(err, check.IsNil)
	c.Assert(iface, check.Equals, "eth1")
}

func (s *S) TestGetPrivateIPInterfaceForDriver(c *check.C) {
	iface, err := GetPrivateIPInterface("amazonec2")
	c.Assert(err, check.IsNil)
	c.Assert(iface, check.Equals, "eth0")
}
