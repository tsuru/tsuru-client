// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

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

// func (s *S) TestGetIP(c *check.C) {
// 	target := &fakeSSHTarget{}
// 	target.runOutput = map[string]*cmdOutput{
// 		"ip addr show dev eth0": {
// 			output: `2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc pfifo_fast state UP group default qlen 1000
// link/ether 12:d4:8c:93:e1:c5 brd ff:ff:ff:ff:ff:ff
// inet 172.30.0.69/24 brd 172.30.0.255 scope global eth0
// valid_lft forever preferred_lft forever
// inet6 fe80::10d4:8cff:fe93:e1c5/64 scope link
// valid_lft forever preferred_lft forever`},
// 		"ip addr show dev eth1": {output: "", err: errors.New("failed to get ip")}}
// 	ip := getIp("eth2", target)
// 	c.Assert(ip, check.Equals, "127.0.0.1")
// 	ip = getIp("eth0", target)
// 	c.Assert(ip, check.Equals, "172.30.0.69")
// 	ip = getIp("eth1", target)
// 	c.Assert(ip, check.Equals, "127.0.0.1")
// }
