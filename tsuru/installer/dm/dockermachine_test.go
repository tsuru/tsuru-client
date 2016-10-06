// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	check "gopkg.in/check.v1"

	"github.com/docker/machine/drivers/amazonec2"
	"github.com/docker/machine/drivers/fakedriver"
	"github.com/docker/machine/libmachine/engine"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/persist/persisttest"
	"github.com/docker/machine/libmachine/state"
	dtesting "github.com/fsouza/go-dockerclient/testing"
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

func (s *S) TestNewDockerMachineCopyProvidedCa(c *check.C) {
	config := &DockerMachineConfig{
		CAPath: s.TLSCertsPath.RootDir,
	}
	defer os.Remove(s.StoreBasePath)
	dm, err := NewDockerMachine(config)
	c.Assert(err, check.IsNil)
	c.Assert(dm, check.NotNil)
	expected, err := ioutil.ReadFile(s.TLSCertsPath.RootCert)
	c.Assert(err, check.IsNil)
	contents, err := ioutil.ReadFile(filepath.Join(dm.certsPath, "ca.pem"))
	c.Assert(err, check.IsNil)
	c.Assert(contents, check.DeepEquals, expected)
	expected, err = ioutil.ReadFile(s.TLSCertsPath.RootKey)
	c.Assert(err, check.IsNil)
	contents, err = ioutil.ReadFile(filepath.Join(dm.certsPath, "ca-key.pem"))
	c.Assert(err, check.IsNil)
	c.Assert(contents, check.DeepEquals, expected)
}

func (s *S) TestConfigureDriver(c *check.C) {
	dm := &DockerMachine{globalDriverOpts: DriverOpts{
		"amazonec2-access-key":     "abc",
		"amazonec2-subnet-id":      "net",
		"amazonec2-security-group": []string{"sg-123", "sg-456"},
	}}
	driver := amazonec2.NewDriver("", "")
	opts := map[string]interface{}{
		"amazonec2-tags": "my-tag1",
	}
	err := dm.configureDriver(driver, opts)
	c.Assert(err, check.NotNil)
	opts["amazonec2-secret-key"] = "cde"
	err = dm.configureDriver(driver, opts)
	c.Assert(err, check.IsNil)
	c.Assert(driver.SecurityGroupNames, check.DeepEquals, []string{"sg-123", "sg-456"})
	c.Assert(driver.SecretKey, check.Equals, "cde")
	c.Assert(driver.SubnetId, check.Equals, "net")
	c.Assert(driver.AccessKey, check.Equals, "abc")
	c.Assert(driver.RetryCount, check.Equals, 5)
	c.Assert(driver.Tags, check.Equals, "my-tag1")
}

func (s *S) TestUploadRegistryCertificate(c *check.C) {
	sshTarget := &fakeSSHTarget{ip: "127.0.0.1"}
	config := &DockerMachineConfig{
		CAPath: s.TLSCertsPath.RootDir,
	}
	defer os.Remove(s.StoreBasePath)
	dm, err := NewDockerMachine(config)
	c.Assert(err, check.IsNil)
	err = dm.uploadRegistryCertificate(sshTarget)
	c.Assert(err, check.IsNil)
	c.Assert(len(sshTarget.cmds), check.Equals, 10)
	c.Assert(sshTarget.cmds[0], check.Equals, "mkdir -p /home/ubuntu/certs/127.0.0.1:5000")
	c.Assert(sshTarget.cmds[1], check.Equals, "sudo mkdir /etc/docker/certs.d")
	c.Assert(strings.Contains(sshTarget.cmds[2], "sudo tee /home/ubuntu/certs/127.0.0.1:5000/registry-cert.pem"), check.Equals, true)
	c.Assert(strings.Contains(sshTarget.cmds[3], "sudo tee /home/ubuntu/certs/127.0.0.1:5000/registry-key.pem"), check.Equals, true)
	c.Assert(strings.Contains(sshTarget.cmds[4], "sudo tee /etc/docker/certs.d/ca.pem"), check.Equals, true)
	c.Assert(strings.Contains(sshTarget.cmds[5], "sudo tee /etc/docker/certs.d/cert.pem"), check.Equals, true)
	c.Assert(strings.Contains(sshTarget.cmds[6], "sudo tee /etc/docker/certs.d/key.pem"), check.Equals, true)
	c.Assert(sshTarget.cmds[7], check.Equals, "sudo cp -r /home/ubuntu/certs/* /etc/docker/certs.d/")
	c.Assert(sshTarget.cmds[8], check.Equals, "sudo cat /etc/docker/certs.d/ca.pem | sudo tee -a /etc/ssl/certs/ca-certificates.crt")
	c.Assert(sshTarget.cmds[9], check.Equals, "sudo mkdir -p /var/lib/registry/")
	sshTarget2 := &fakeSSHTarget{ip: "127.0.0.2"}
	err = dm.uploadRegistryCertificate(sshTarget2)
	c.Assert(err, check.IsNil)
	c.Assert(len(sshTarget2.cmds), check.Equals, 10)
	c.Assert(sshTarget2.cmds[0], check.Equals, "mkdir -p /home/ubuntu/certs/127.0.0.1:5000")
	c.Assert(sshTarget2.cmds[1], check.Equals, "sudo mkdir /etc/docker/certs.d")
	c.Assert(strings.Contains(sshTarget2.cmds[2], "sudo tee /home/ubuntu/certs/127.0.0.1:5000/registry-cert.pem"), check.Equals, true)
	c.Assert(strings.Contains(sshTarget2.cmds[3], "sudo tee /home/ubuntu/certs/127.0.0.1:5000/registry-key.pem"), check.Equals, true)
}

func (s *S) TestCreateRegistryCertificate(c *check.C) {
	config := &DockerMachineConfig{
		CAPath: s.TLSCertsPath.RootDir,
	}
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

type fakeMachineAPI struct {
	*persisttest.FakeStore
	driverName string
	hostName   string
	closed     bool
}

func (f *fakeMachineAPI) NewHost(driverName string, rawDriver []byte) (*host.Host, error) {
	f.driverName = driverName
	return &host.Host{
		Name: "machine",
		Driver: &fakedriver.Driver{
			MockState: state.Running,
			MockIP:    "127.0.0.1",
		},
		HostOptions: &host.Options{
			EngineOptions: &engine.Options{},
		},
	}, nil
}

func (f *fakeMachineAPI) Create(h *host.Host) error {
	f.hostName = h.Name
	return nil
}

func (f *fakeMachineAPI) Close() error {
	f.closed = true
	return nil
}

func (f *fakeMachineAPI) GetMachinesDir() string {
	return ""
}

func (s *S) TestCreateMachine(c *check.C) {
	tlsConfig := dtesting.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	server, err := dtesting.NewTLSServer("127.0.0.1:2376", nil, nil, tlsConfig)
	c.Assert(err, check.IsNil)
	defer server.Stop()
	dm, err := NewDockerMachine(DefaultDockerMachineConfig)
	c.Assert(err, check.IsNil)
	fakeAPI := &fakeMachineAPI{}
	dm.client = fakeAPI
	dm.certsPath = s.TLSCertsPath.RootDir
	machine, err := dm.CreateMachine(map[string]interface{}{})
	c.Assert(err, check.IsNil)
	c.Assert(machine, check.NotNil)
	c.Assert(machine.IP, check.Equals, "127.0.0.1")
	c.Assert(machine.Address, check.Equals, "https://127.0.0.1:2376")
	c.Assert(fakeAPI.driverName, check.Equals, "virtualbox")
	c.Assert(fakeAPI.hostName, check.Equals, "machine")
}

func (s *S) TestDeleteMachine(c *check.C) {
	dm, err := NewDockerMachine(DefaultDockerMachineConfig)
	c.Assert(err, check.IsNil)
	dm.client = &fakeMachineAPI{
		FakeStore: &persisttest.FakeStore{
			Hosts: []*host.Host{{
				Name: "test-machine",
				Driver: &fakedriver.Driver{
					MockState: state.Running,
					MockIP:    "1.2.3.4",
				},
			}},
		},
	}
	err = dm.DeleteMachine("test-machine")
	c.Assert(err, check.IsNil)
}

func (s *S) TestDeleteAll(c *check.C) {
	dm, err := NewDockerMachine(DefaultDockerMachineConfig)
	c.Assert(err, check.IsNil)
	dm.client = &fakeMachineAPI{
		FakeStore: &persisttest.FakeStore{
			Hosts: []*host.Host{{
				Name: "test-machine2",
				Driver: &fakedriver.Driver{
					MockState: state.Running,
					MockIP:    "1.2.3.4",
				},
			},
				{
					Name: "test-machine",
					Driver: &fakedriver.Driver{
						MockState: state.Running,
						MockIP:    "1.2.3.5",
					},
				},
			},
		},
	}
	err = dm.DeleteAll()
	c.Assert(err, check.IsNil)
}

func (s *S) TestDeleteMachineLoadError(c *check.C) {
	dm, err := NewDockerMachine(DefaultDockerMachineConfig)
	c.Assert(err, check.IsNil)
	expectedErr := fmt.Errorf("failed to load")
	dm.client = &fakeMachineAPI{
		FakeStore: &persisttest.FakeStore{
			LoadErr: expectedErr,
		},
	}
	err = dm.DeleteMachine("test-machine")
	c.Assert(err, check.Equals, expectedErr)
}

func (s *S) TestClose(c *check.C) {
	dm, err := NewDockerMachine(DefaultDockerMachineConfig)
	c.Assert(err, check.IsNil)
	fakeAPI := &fakeMachineAPI{
		FakeStore: &persisttest.FakeStore{},
	}
	dm.client = fakeAPI
	dm.CreateMachine(map[string]interface{}{})
	c.Assert(fakeAPI.closed, check.Equals, false)
	dm.Close()
	c.Assert(fakeAPI.closed, check.Equals, true)
}

func (s *S) TestNewTempDockerMachine(c *check.C) {
	dm, err := NewTempDockerMachine()
	c.Assert(err, check.IsNil)
	defer dm.Close()
	f, err := os.Stat(dm.certsPath)
	c.Assert(err, check.IsNil)
	c.Assert(f.IsDir(), check.Equals, true)
	f, err = os.Stat(dm.storePath)
	c.Assert(err, check.IsNil)
	c.Assert(f.IsDir(), check.Equals, true)
}

func (s *S) TestTempDockerMachineNewHost(c *check.C) {
	dm, err := NewTempDockerMachine()
	c.Assert(err, check.IsNil)
	h, err := dm.NewHost("amazonec2", "my-ssh-key", map[string]interface{}{})
	c.Assert(err, check.IsNil)
	c.Assert(h.DriverName, check.Equals, "amazonec2")
	b, err := ioutil.ReadFile(h.Driver.GetSSHKeyPath())
	c.Assert(err, check.IsNil)
	c.Assert(string(b), check.Equals, "my-ssh-key")
}
