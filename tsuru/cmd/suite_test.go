// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"os"
	"testing"

	check "gopkg.in/check.v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type S struct{}

var _ = check.Suite(&S{})
var globalManager *ManagerV2

func (s *S) SetUpTest(c *check.C) {
	globalManager = NewManagerV2()
	os.Setenv("TSURU_TARGET", "http://localhost")
	os.Setenv("TSURU_TOKEN", "abc123")
	if env := os.Getenv("TERM"); env == "" {
		os.Setenv("TERM", "tsuruterm")
	}
}

func (s *S) TearDownTest(c *check.C) {
	os.Unsetenv("TSURU_TARGET")
	os.Unsetenv("TSURU_TOKEN")
}
