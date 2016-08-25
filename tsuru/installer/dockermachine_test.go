// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	check "gopkg.in/check.v1"

	"github.com/docker/machine/drivers/amazonec2"
	"github.com/docker/machine/drivers/azure"
	"github.com/docker/machine/drivers/fakedriver"
	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/persist/persisttest"
	"github.com/docker/machine/libmachine/state"
	dtesting "github.com/fsouza/go-dockerclient/testing"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru-client/tsuru/installer/testing"
	"github.com/tsuru/tsuru/exec/exectest"
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
	storeBasePath, err := ioutil.TempDir("", "tests-store")
	c.Assert(err, check.IsNil)
	s.StoreBasePath = storeBasePath
	tlsCertsPath, err := installertest.CreateTestCerts()
	c.Assert(err, check.IsNil)
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
	c.Assert(dm.driverOpts["url"].(string), check.Equals, "localhost")
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
	driver := amazonec2.NewDriver("", "")
	opts := map[string]interface{}{
		"amazonec2-access-key":     "abc",
		"amazonec2-subnet-id":      "net",
		"amazonec2-security-group": []string{"sg-123", "sg-456"},
	}
	err := configureDriver(driver, opts, []string{})
	c.Assert(err, check.NotNil)
	opts["amazonec2-secret-key"] = "cde"
	err = configureDriver(driver, opts, []string{})
	c.Assert(err, check.IsNil)
	c.Assert(driver.SecurityGroupNames, check.DeepEquals, []string{"sg-123", "sg-456"})
	c.Assert(driver.SecretKey, check.Equals, "cde")
	c.Assert(driver.SubnetId, check.Equals, "net")
	c.Assert(driver.AccessKey, check.Equals, "abc")
	c.Assert(driver.RetryCount, check.Equals, 5)
}

func (s *S) TestConfigureDriverOpenPorts(c *check.C) {
	driver := azure.NewDriver("", "")
	opts := map[string]interface{}{
		"azure-subscription-id": "abc",
	}
	err := configureDriver(driver, opts, []string{"8080"})
	c.Assert(err, check.IsNil)
	c.Assert(driver.(*azure.Driver).OpenPorts, check.DeepEquals, []string{"8080"})
}

type fakeSSHTarget struct {
	cmds []string
}

func (f *fakeSSHTarget) RunSSHCommand(cmd string) (string, error) {
	f.cmds = append(f.cmds, cmd)
	return "", nil
}

func (f *fakeSSHTarget) GetIP() string {
	return "127.0.0.1"
}

func (f *fakeSSHTarget) GetSSHUsername() string {
	return "ubuntu"
}

func (f *fakeSSHTarget) GetSSHKeyPath() string {
	return "/mykey"
}

func (s *S) TestUploadRegistryCertificate(c *check.C) {
	fakeSSHTarget := &fakeSSHTarget{}
	fexec := exectest.FakeExecutor{}
	client.Execut = &fexec
	defer func() {
		client.Execut = nil
	}()
	config := &DockerMachineConfig{
		CAPath: s.TLSCertsPath.RootDir,
	}
	defer os.Remove(s.StoreBasePath)
	dm, err := NewDockerMachine(config)
	c.Assert(err, check.IsNil)
	err = dm.uploadRegistryCertificate(fakeSSHTarget)
	c.Assert(err, check.IsNil)
	expectedArgs := []string{"-o StrictHostKeyChecking=no",
		"-i",
		"/mykey",
		"-r",
		fmt.Sprintf("%s/", dm.certsPath),
		fmt.Sprintf("%s@%s:/home/%s/", "ubuntu", "127.0.0.1", "ubuntu"),
	}
	c.Assert(fexec.ExecutedCmd("scp", expectedArgs), check.Equals, true)
	expectedCmds := []string{
		"mkdir -p /home/ubuntu/certs/127.0.0.1:5000",
		"cp /home/ubuntu/certs/*.pem /home/ubuntu/certs/127.0.0.1:5000/",
		"sudo mkdir /etc/docker/certs.d && sudo cp -r /home/ubuntu/certs/* /etc/docker/certs.d/",
		"cat /home/ubuntu/certs/127.0.0.1:5000/ca.pem | sudo tee -a /etc/ssl/certs/ca-certificates.crt",
		"mkdir -p /var/lib/registry/",
		"sudo /usr/local/sbin/iptables -D DOCKER-ISOLATION -i docker_gwbridge -o docker0 -j DROP",
		"sudo /usr/local/sbin/iptables -D DOCKER-ISOLATION -i docker0 -o docker_gwbridge -j DROP",
	}
	c.Assert(fakeSSHTarget.cmds, check.DeepEquals, expectedCmds)
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
	machine, err := dm.CreateMachine([]string{"8080"})
	c.Assert(err, check.IsNil)
	c.Assert(machine, check.NotNil)
	c.Assert(machine.IP, check.Equals, "127.0.0.1")
	c.Assert(machine.Address, check.Equals, "https://127.0.0.1:2376")
	c.Assert(machine.OpenPorts, check.DeepEquals, []string{"8080"})
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
	dm.CreateMachine([]string{})
	c.Assert(fakeAPI.closed, check.Equals, false)
	dm.Close()
	c.Assert(fakeAPI.closed, check.Equals, true)
}
