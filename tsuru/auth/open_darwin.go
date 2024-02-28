// Copyright 2023 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"github.com/tsuru/tsuru/exec"
)

func open(url string) error {
	opts := exec.ExecuteOptions{
		Cmd:  "open",
		Args: []string{url},
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
