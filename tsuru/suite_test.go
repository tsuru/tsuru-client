// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/tsuru/tsuru/cmd"
	tTesting "github.com/tsuru/tsuru/testing"
	"launchpad.net/gocheck"
)

type S struct {
	target []string
	token  []string
}

func (s *S) SetUpSuite(c *gocheck.C) {
	s.target = tTesting.SetTargetFile(c, []byte("http://localhost:8080"))
	s.token = tTesting.SetTokenFile(c, []byte("sometoken"))
}

func (s *S) TearDownSuite(c *gocheck.C) {
	tTesting.RollbackFile(s.target)
	tTesting.RollbackFile(s.token)
}

var _ = gocheck.Suite(&S{})
var manager *cmd.Manager

func Test(t *testing.T) { gocheck.TestingT(t) }

func (s *S) SetUpTest(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	manager = cmd.NewManager("glb", version, header, &stdout, &stderr, os.Stdin, nil)
}
