// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"errors"

	check "gopkg.in/check.v1"
)

type cmdOutput struct {
	output string
	err    error
}

type fakeSSHTarget struct {
	ip        string
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

func (f *fakeSSHTarget) GetIP() string {
	if f.ip == "" {
		return "127.0.0.1"
	}
	return f.ip
}

func (f *fakeSSHTarget) GetSSHUsername() string {
	return "ubuntu"
}

func (f *fakeSSHTarget) GetSSHKeyPath() string {
	return "/mykey"
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
		"ip addr show dev eth1": {output: "", err: errors.New("failed to get ip")}}
	ip := getIp("eth2", target)
	c.Assert(ip, check.Equals, "127.0.0.1")
	ip = getIp("eth0", target)
	c.Assert(ip, check.Equals, "172.30.0.69")
	ip = getIp("eth1", target)
	c.Assert(ip, check.Equals, "127.0.0.1")
}
