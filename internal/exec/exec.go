// Copyright 2013 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package exec provides a interface to run external commands as an
// abstraction layer.
package exec

import (
	"fmt"
	"io"
	"os/exec"
)

// ExecuteOptions specify parameters to the Execute method.
type ExecuteOptions struct {
	Cmd    string
	Args   []string
	Envs   []string
	Dir    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

var _ Executor = &OsExec{}

type Executor interface {
	// Command executes the specified command.
	Command(opts ExecuteOptions) error
}

type OsExec struct{}

func (*OsExec) Command(opts ExecuteOptions) error {
	c := exec.Command(opts.Cmd, opts.Args...)
	c.Stdin = opts.Stdin
	c.Stdout = opts.Stdout
	c.Stderr = opts.Stderr
	c.Env = opts.Envs
	c.Dir = opts.Dir
	return c.Run()
}

var _ Executor = &FakeExec{}

type FakeExec struct {
	OutStderr  string
	OutStdout  string
	OutErr     error
	CalledOpts ExecuteOptions
}

func (e *FakeExec) Command(opts ExecuteOptions) error {
	if opts.Stdout != nil {
		fmt.Fprint(opts.Stdout, e.OutStdout)
	}
	if opts.Stderr != nil {
		fmt.Fprint(opts.Stderr, e.OutStderr)
	}
	e.CalledOpts = opts
	return e.OutErr
}
