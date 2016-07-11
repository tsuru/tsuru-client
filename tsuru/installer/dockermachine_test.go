// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"

	"gopkg.in/check.v1"
)

type S struct{}

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

func (s *S) TestNewDockerMachine(c *check.C) {
	dm, err := NewDockerMachine("virtualbox", nil)
	c.Assert(err, check.IsNil)
	c.Assert(dm, check.NotNil)
	c.Assert(dm.driverName, check.Equals, "virtualbox")
	c.Assert(dm.tlsSupport, check.Equals, false)
}

func (s *S) TestNewDockerMachineSupportTLS(c *check.C) {
	dm, err := NewDockerMachine("amazonec2", nil)
	c.Assert(err, check.IsNil)
	c.Assert(dm, check.NotNil)
	c.Assert(dm.driverName, check.Equals, "amazonec2")
	c.Assert(dm.tlsSupport, check.Equals, true)
}

func (s *S) TestNewDockerMachineDriverOpts(c *check.C) {
	dm, err := NewDockerMachine("none", map[string]interface{}{"url": "localhost"})
	c.Assert(err, check.IsNil)
	c.Assert(dm, check.NotNil)
	c.Assert(dm.driverOpts.String("url"), check.Equals, "localhost")
}

func (s *S) TestCreateMachineNoneDriver(c *check.C) {
	dm, _ := NewDockerMachine("none", map[string]interface{}{"url": "http://1.2.3.4"})
	machine, err := dm.CreateMachine(nil)
	c.Assert(err, check.IsNil)
	c.Assert(machine, check.NotNil)
	c.Assert(machine.IP, check.Equals, "1.2.3.4")
	c.Assert(machine.TLS, check.Equals, false)
}
