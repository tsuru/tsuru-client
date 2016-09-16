// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	check "gopkg.in/check.v1"

	"github.com/docker/machine/drivers/amazonec2"
	"github.com/docker/machine/drivers/fakedriver"
	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/persist/persisttest"
	"github.com/docker/machine/libmachine/state"
	dtesting "github.com/fsouza/go-dockerclient/testing"
	"github.com/tsuru/tsuru-client/tsuru/installer/testing"
)

type S struct {
	TLSCertsPath  installertest.CertsPath
	StoreBasePath string
}

var _ = check.Suite(&S{})

func TestMain(m *testing.M) {
	if os.Getenv(localbinary.PluginEnvKey) == localbinary.PluginEnvVal {
		driver := os.Getenv(localbinary.PluginEnvDriverName)
		err := RunDriver(driver)
		if err != nil {
			fmt.Printf("Failed to run driver %s in test", driver)
			os.Exit(1)
		}
	} else {
		localbinary.CurrentBinaryIsDockerMachine = true
		os.Exit(m.Run())
	}
}

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) SetUpSuite(c *check.C) {
	tlsCertsPath, err := installertest.CreateTestCerts()
	c.Assert(err, check.IsNil)
	s.StoreBasePath, _ = filepath.Split(tlsCertsPath.RootDir)
	storeBasePath = s.StoreBasePath
	s.TLSCertsPath = tlsCertsPath
}

func (s *S) TearDownSuite(c *check.C) {
	installertest.CleanCerts(s.TLSCertsPath.RootDir)
	os.Remove(s.StoreBasePath)
}

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
	println(server.URL())
	dm, err := NewDockerMachine(defaultDockerMachineConfig)
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
	dm, err := NewDockerMachine(defaultDockerMachineConfig)
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
	dm, err := NewDockerMachine(defaultDockerMachineConfig)
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
	dm, err := NewDockerMachine(defaultDockerMachineConfig)
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
	dm, err := NewDockerMachine(defaultDockerMachineConfig)
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
