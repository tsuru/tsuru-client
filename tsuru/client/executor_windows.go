// Copyright 2018 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	osexec "os/exec"

	"github.com/tsuru/tsuru/exec"
)

var Execut exec.Executor

func Executor() exec.Executor {
	if Execut == nil {
		Execut = windowsCmdExecutor{}
	}
	return Execut
}

type windowsCmdExecutor struct{}

func (windowsCmdExecutor) Execute(opts exec.ExecuteOptions) error {
	args := append([]string{
		"/c",
		opts.Cmd,
	}, opts.Args...)
	c := osexec.Command("cmd", args...)
	c.Stdin = opts.Stdin
	c.Stdout = opts.Stdout
	c.Stderr = opts.Stderr
	c.Env = opts.Envs
	c.Dir = opts.Dir
	return c.Run()
}
