// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"runtime"
	"strings"

	"github.com/tsuru/tsuru/exec/exectest"
	"gopkg.in/check.v1"
)

func (s *S) TestOpen(c *check.C) {
	fexec := exectest.FakeExecutor{}
	execut = &fexec
	defer func() {
		execut = nil
	}()
	url := "http://someurl"
	err := open(url)
	c.Assert(err, check.IsNil)
	if runtime.GOOS == "linux" {
		c.Assert(fexec.ExecutedCmd("xdg-open", []string{url}), check.Equals, true)
	} else if runtime.GOOS == "windows" {
		url = strings.ReplaceAll(url, "&", "^&")
		c.Assert(fexec.ExecutedCmd("cmd", []string{"/c", "start", "", url}), check.Equals, true)
	} else {
		c.Assert(fexec.ExecutedCmd("open", []string{url}), check.Equals, true)
	}
}
