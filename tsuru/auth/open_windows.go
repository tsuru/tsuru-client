// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"strings"

	"github.com/tsuru/tsuru/exec"
)

func open(url string) error {
	var opts exec.ExecuteOptions
	url = strings.Replace(url, "&", "^&", -1)
	opts = exec.ExecuteOptions{
		Cmd:  "cmd",
		Args: []string{"/c", "start", "", url},
	}
	return executor().Execute(opts)
}

var execut exec.Executor

func executor() exec.Executor {
	if execut == nil {
		execut = exec.OsExecutor{}
	}
	return execut
}
