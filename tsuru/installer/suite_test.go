// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	"github.com/tsuru/tsuru-client/tsuru/installer/testing"
	"github.com/tsuru/tsuru/cmd"
	check "gopkg.in/check.v1"
)

type S struct {
	TLSCertsPath installertest.CertsPath
}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

var manager *cmd.Manager

func (s *S) SetUpSuite(c *check.C) {
	tlsCertsPath, err := installertest.CreateTestCerts()
	c.Assert(err, check.IsNil)
	s.TLSCertsPath = tlsCertsPath
	var stdout, stderr bytes.Buffer
	manager = cmd.NewManager("glb", "1.0.0", "Supported-Tsuru-Version", &stdout, &stderr, os.Stdin, nil)
}

func TestMain(m *testing.M) {
	if os.Getenv(localbinary.PluginEnvKey) == localbinary.PluginEnvVal {
		driver := os.Getenv(localbinary.PluginEnvDriverName)
		err := dm.RunDriver(driver)
		if err != nil {
			fmt.Printf("Failed to run driver %s in test", driver)
			os.Exit(1)
		}
	} else {
		localbinary.CurrentBinaryIsDockerMachine = true
		os.Exit(m.Run())
	}
}
