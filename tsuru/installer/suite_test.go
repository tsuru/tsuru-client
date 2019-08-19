// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
	installertest "github.com/tsuru/tsuru-client/tsuru/installer/testing"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	check "gopkg.in/check.v1"
)

type S struct {
	TLSCertsPath installertest.CertsPath
	tmpDir       string
}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

var manager *cmd.Manager

func (s *S) SetUpSuite(c *check.C) {
	var err error
	s.tmpDir, err = ioutil.TempDir("", "tsuru-client-test")
	c.Assert(err, check.IsNil)
	err = os.Setenv("HOME", s.tmpDir)
	c.Assert(err, check.IsNil)
	tlsCertsPath, err := installertest.CreateTestCerts()
	c.Assert(err, check.IsNil)
	s.TLSCertsPath = tlsCertsPath
	var stdout, stderr bytes.Buffer
	manager = cmd.NewManager("glb", "1.0.0", "Supported-Tsuru-Version", &stdout, &stderr, os.Stdin, nil)
	swarmPort = 0
}

func (s *S) TearDownSuite(c *check.C) {
	err := os.RemoveAll(s.tmpDir)
	c.Assert(err, check.IsNil)
}

func TestMain(m *testing.M) {
	if os.Getenv(localbinary.PluginEnvKey) == localbinary.PluginEnvVal {
		driver := os.Getenv(localbinary.PluginEnvDriverName)
		err := dockermachine.RunDriver(driver)
		if err != nil {
			fmt.Printf("Failed to run driver %s in test", driver)
			os.Exit(1)
		}
	} else {
		localbinary.CurrentBinaryIsDockerMachine = true
		os.Exit(m.Run())
	}
}
