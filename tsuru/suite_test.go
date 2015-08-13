// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/tsuru/tsuru/cmd"
	"gopkg.in/check.v1"
)

type S struct{}

func (s *S) SetUpSuite(c *check.C) {
	os.Setenv("TSURU_TARGET", "http://localhost:8080")
	os.Setenv("TSURU_TOKEN", "sometoken")
}

func (s *S) TearDownSuite(c *check.C) {
	os.Unsetenv("TSURU_TARGET")
	os.Unsetenv("TSURU_TOKEN")
}

var _ = check.Suite(&S{})
var manager *cmd.Manager

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) SetUpTest(c *check.C) {
	var stdout, stderr bytes.Buffer
	manager = cmd.NewManager("glb", version, header, &stdout, &stderr, os.Stdin, nil)
}
