// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"github.com/tsuru/tsuru/exec"
	"github.com/tsuru/tsuru/exec/exectest"
	"gopkg.in/check.v1"
)

func (s *S) TestExecutor(c *check.C) {
	Execut = &exectest.FakeExecutor{}
	c.Assert(Executor(), check.DeepEquals, Execut)
	Execut = nil
	c.Assert(Executor(), check.DeepEquals, exec.OsExecutor{})
}
