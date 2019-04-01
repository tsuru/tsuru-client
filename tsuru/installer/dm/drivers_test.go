// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"errors"

	"github.com/tsuru/config"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	check "gopkg.in/check.v1"
)

type cmdOutput struct {
	output string
	err    error
}

type fakeSSHTarget struct {
	cmds      []string
	runOutput map[string]*cmdOutput
}

func (f *fakeSSHTarget) RunSSHCommand(cmd string) (string, error) {
	f.cmds = append(f.cmds, cmd)
	if f.runOutput != nil && f.runOutput[cmd] != nil {
		return f.runOutput[cmd].output, f.runOutput[cmd].err
	}
	return "", nil
}

func (s *S) TestGetPrivateIPInterfaceFromConfig(c *check.C) {
	config.Set("driver:private-ip-interface", "eth1")
	defer config.Unset("driver:private-ip-interface")
	iface := getPrivateIPInterface()
	c.Assert(iface, check.Equals, "eth1")
	config.Set("driver:private-ip-interface", "")
	iface = getPrivateIPInterface()
	c.Assert(iface, check.Equals, "")
	config.Unset("driver:private-ip-interface")
	iface = getPrivateIPInterface()
	c.Assert(iface, check.Equals, "eth0")
}

func (s *S) TestGetPrivateIP(c *check.C) {
	defer config.Unset("driver:private-ip-interface")
	m := &dockermachine.Machine{
		Base: &iaas.Machine{
			Address: "base-addr",
		},
	}
	addr := GetPrivateIP(m)
	c.Assert(addr, check.Equals, "base-addr")
}

func (s *S) TestGetIP(c *check.C) {
	target := &fakeSSHTarget{}
	target.runOutput = map[string]*cmdOutput{
		"ip addr show dev eth0": {
			output: `2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc pfifo_fast state UP group default qlen 1000
link/ether 12:d4:8c:93:e1:c5 brd ff:ff:ff:ff:ff:ff
inet 172.30.0.69/24 brd 172.30.0.255 scope global eth0
valid_lft forever preferred_lft forever
inet6 fe80::10d4:8cff:fe93:e1c5/64 scope link
valid_lft forever preferred_lft forever`},
		"ip addr show dev eth1": {output: "", err: errors.New("failed to get ip")},
		"ip addr show dev xyz":  {output: "Device \"xyz\" does not exist."},
	}
	ip, err := getIP(target, "eth2")
	c.Assert(err, check.NotNil)
	c.Assert(ip, check.Equals, "")
	ip, err = getIP(target, "eth0")
	c.Assert(err, check.IsNil)
	c.Assert(ip, check.Equals, "172.30.0.69")
	ip, err = getIP(target, "eth1")
	c.Assert(err, check.NotNil)
	c.Assert(ip, check.Equals, "")
	ip, err = getIP(target, "xyz")
	c.Assert(err, check.ErrorMatches, `failed to parse private ip from interface`)
	c.Assert(ip, check.Equals, "")
}

func (s *S) TestDefaultDriverConfig(c *check.C) {
	tt := []struct {
		driverName   string
		expectedOpts map[string]interface{}
	}{
		{"virtualbox", map[string]interface{}{"driver:options:virtualbox-memory": 2048, "driver:options:virtualbox-nat-nictype": "virtio"}},
	}
	for _, t := range tt {
		opts := DefaultDriverConfig(t.driverName)
		c.Check(opts, check.DeepEquals, t.expectedOpts)
	}
}
