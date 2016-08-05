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

	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
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
	err = os.Remove(s.StoreBasePath)
	c.Assert(err, check.IsNil)
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
