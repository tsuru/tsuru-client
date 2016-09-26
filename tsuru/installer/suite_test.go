// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"os"
	"testing"

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
