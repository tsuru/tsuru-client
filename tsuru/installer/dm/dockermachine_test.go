// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"os"
	"path/filepath"
	"strings"

	check "gopkg.in/check.v1"
)

func (s *S) TestNewDockerMachine(c *check.C) {
	config := &DockerMachineConfig{
		DriverName: "virtualbox",
	}
	dm, err := NewDockerMachine(config)
	c.Assert(err, check.IsNil)
	c.Assert(dm, check.NotNil)
	c.Assert(dm.driverName, check.Equals, "virtualbox")
}

func (s *S) TestNewDockerMachineDriverOpts(c *check.C) {
	config := &DockerMachineConfig{
		DriverName: "none",
		DriverOpts: map[string]interface{}{
			"url": "localhost",
		},
	}
	dm, err := NewDockerMachine(config)
	c.Assert(err, check.IsNil)
	c.Assert(dm, check.NotNil)
	c.Assert(dm.globalDriverOpts["url"].(string), check.Equals, "localhost")
}

func (s *S) TestUploadRegistryCertificate(c *check.C) {
	sshTarget := &fakeSSHTarget{}
	config := &DockerMachineConfig{}
	defer os.Remove(s.StoreBasePath)
	dm, err := NewDockerMachine(config)
	c.Assert(err, check.IsNil)
	err = dm.uploadRegistryCertificate("127.0.0.1", "ubuntu", sshTarget)
	c.Assert(err, check.IsNil)
	c.Assert(len(sshTarget.cmds), check.Equals, 11)
	c.Assert(sshTarget.cmds[0], check.Equals, "mkdir -p /home/ubuntu/certs/127.0.0.1:5000")
	c.Assert(sshTarget.cmds[1], check.Equals, "sudo mkdir -p /etc/docker/certs.d")
	s.containsWithSubstring(sshTarget.cmds, "sudo tee /etc/docker/certs.d/ca-key.pem", c)
	s.containsWithSubstring(sshTarget.cmds, "sudo tee /etc/docker/certs.d/ca.pem", c)
	s.containsWithSubstring(sshTarget.cmds, "sudo tee /etc/docker/certs.d/cert.pem", c)
	s.containsWithSubstring(sshTarget.cmds, "sudo tee /etc/docker/certs.d/key.pem", c)
	c.Assert(sshTarget.cmds[8], check.Equals, "sudo cp -r /home/ubuntu/certs/* /etc/docker/certs.d/")
	c.Assert(sshTarget.cmds[9], check.Equals, "sudo cat /etc/docker/certs.d/ca.pem | sudo tee -a /etc/ssl/certs/ca-certificates.crt")
	c.Assert(sshTarget.cmds[10], check.Equals, "sudo mkdir -p /var/lib/registry/")
	sshTarget2 := &fakeSSHTarget{}
	err = dm.uploadRegistryCertificate("127.0.0.2", "ubuntu", sshTarget2)
	c.Assert(err, check.IsNil)
	c.Assert(len(sshTarget2.cmds), check.Equals, 11)
	c.Assert(sshTarget2.cmds[0], check.Equals, "mkdir -p /home/ubuntu/certs/127.0.0.1:5000")
	c.Assert(sshTarget2.cmds[1], check.Equals, "sudo mkdir -p /etc/docker/certs.d")
	s.containsWithSubstring(sshTarget2.cmds, "sudo tee /home/ubuntu/certs/127.0.0.1:5000/registry-cert.pem", c)
	s.containsWithSubstring(sshTarget2.cmds, "sudo tee /home/ubuntu/certs/127.0.0.1:5000/registry-key.pem", c)
}

func (s *S) containsWithSubstring(l []string, subs string, c *check.C) {
	for _, v := range l {
		if strings.Contains(v, subs) {
			return
		}
	}
	c.Fatalf("substring %s not found in %#v", subs, l)
}

func (s *S) TestCreateRegistryCertificate(c *check.C) {
	config := &DockerMachineConfig{}
	defer os.Remove(s.StoreBasePath)
	dm, err := NewDockerMachine(config)
	c.Assert(err, check.IsNil)
	err = dm.createRegistryCertificate("127.0.0.1")
	c.Assert(err, check.IsNil)
	file, err := os.Stat(filepath.Join(dm.certsPath, "registry-cert.pem"))
	c.Assert(err, check.IsNil)
	c.Assert(file.Size() > 0, check.Equals, true)
	file, err = os.Stat(filepath.Join(dm.certsPath, "registry-key.pem"))
	c.Assert(err, check.IsNil)
	c.Assert(file.Size() > 0, check.Equals, true)
}
