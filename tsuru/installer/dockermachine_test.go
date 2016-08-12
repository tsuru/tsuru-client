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
	"github.com/docker/machine/drivers/fakedriver"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
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
	err := configureDriver(driver, opts)
	c.Assert(err, check.NotNil)
	opts["amazonec2-secret-key"] = "cde"
	err = configureDriver(driver, opts)
	c.Assert(err, check.IsNil)
	c.Assert(driver.SecurityGroupNames, check.DeepEquals, []string{"sg-123", "sg-456"})
	c.Assert(driver.SecretKey, check.Equals, "cde")
	c.Assert(driver.SubnetId, check.Equals, "net")
	c.Assert(driver.AccessKey, check.Equals, "abc")
	c.Assert(driver.RetryCount, check.Equals, 5)
}

type fakeSSHTarget struct {
	cmds   []string
	driver drivers.Driver
}

func (f *fakeSSHTarget) RunSSHCommand(cmd string) (string, error) {
	f.cmds = append(f.cmds, cmd)
	return "", nil
}

func (f *fakeSSHTarget) Driver() drivers.Driver {
	if f.driver == nil {
		driver := &fakedriver.Driver{MockIP: "127.0.0.1"}
		driver.Start()
		f.driver = driver
	}
	return f.driver
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
		fmt.Sprintf("%s/machines/%s/id_rsa", dm.storePath, dm.Name),
		"-r",
		fmt.Sprintf("%s/", dm.certsPath),
		fmt.Sprintf("%s@%s:/home/%s/", "", "127.0.0.1", ""),
	}
	c.Assert(fexec.ExecutedCmd("scp", expectedArgs), check.Equals, true)
	expectedCmds := []string{
		"mkdir -p /home//certs/127.0.0.1:5000",
		"cp /home//certs/*.pem /home//certs/127.0.0.1:5000/",
		"sudo mkdir /etc/docker/certs.d && sudo cp -r /home//certs/* /etc/docker/certs.d/",
		"cat /home//certs/127.0.0.1:5000/ca.pem | sudo tee -a /etc/ssl/certs/ca-certificates.crt",
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

//func (s *S) TestCreateMachineNoneDriver(c *check.C) {
//config := &DockerMachineConfig{
//DriverName: "none",
//DriverOpts: map[string]interface{}{
//"url": "http://1.2.3.4",
//},
//}
//dm, _ := NewDockerMachine(config)
//machine, err := dm.CreateMachine()
//c.Assert(err, check.IsNil)
//c.Assert(machine, check.NotNil)
//c.Assert(machine.IP, check.Equals, "1.2.3.4")
//}
