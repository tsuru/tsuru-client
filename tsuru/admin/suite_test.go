// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"os"
	"testing"

	"github.com/tsuru/tsuru/cmd"
	check "gopkg.in/check.v1"
)

type S struct {
	manager *cmd.Manager
}

func (s *S) SetUpSuite(c *check.C) {
	var stdout, stderr bytes.Buffer
	s.manager = cmd.NewManager("glb", "1.0.0", "Supported-Tsuru-Version", &stdout, &stderr, os.Stdin, nil)
	os.Setenv("TSURU_TARGET", "http://localhost")
}

func (s *S) TearDownSuite(c *check.C) {
	os.Unsetenv("TSURU_TARGET")
}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }
