// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/tsuru/tsuru/exec"
	"github.com/tsuru/tsuru/exec/exectest"
	"gopkg.in/check.v1"
)

func (s *S) TestExecutor(c *check.C) {
	execut = &exectest.FakeExecutor{}
	c.Assert(executor(), check.DeepEquals, execut)
	execut = nil
	c.Assert(executor(), check.DeepEquals, exec.OsExecutor{})
}
