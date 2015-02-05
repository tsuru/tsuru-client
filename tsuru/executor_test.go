// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/tsuru/tsuru/exec"
	"github.com/tsuru/tsuru/exec/exectest"
	"launchpad.net/gocheck"
)

func (s *S) TestExecutor(c *gocheck.C) {
	execut = &exectest.FakeExecutor{}
	c.Assert(executor(), gocheck.DeepEquals, execut)
	execut = nil
	c.Assert(executor(), gocheck.DeepEquals, exec.OsExecutor{})
}
