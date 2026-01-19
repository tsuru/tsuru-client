// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/tsuru/tsuru/fs"
	"github.com/tsuru/tsuru/fs/fstest"
	check "gopkg.in/check.v1"
)

type TopicCommand struct {
	name     string
	executed bool
	args     []string
}

func (c *TopicCommand) Info() *Info {
	return &Info{
		Name:  c.name,
		Desc:  "desc " + c.name,
		Usage: "usage",
	}
}

func (c *TopicCommand) Run(context *Context) error {
	c.executed = true
	c.args = context.Args
	return nil
}

type ArgCmd struct{}

func (c *ArgCmd) Info() *Info {
	return &Info{
		Name:    "arg",
		MinArgs: 1,
		MaxArgs: 2,
		Usage:   "arg [args]",
		Desc:    "some desc",
	}
}

func (cmd *ArgCmd) Run(ctx *Context) error {
	return nil
}

func (s *S) TestExtractProgramNameWithAbsolutePath(c *check.C) {
	got := ExtractProgramName("/usr/bin/tsuru")
	c.Assert(got, check.Equals, "tsuru")
}

func (s *S) TestExtractProgramNameWithRelativePath(c *check.C) {
	got := ExtractProgramName("./tsuru")
	c.Assert(got, check.Equals, "tsuru")
}

func (s *S) TestExtractProgramNameWithinThePATH(c *check.C) {
	got := ExtractProgramName("tsuru")
	c.Assert(got, check.Equals, "tsuru")
}

func (s *S) TestFileSystem(c *check.C) {
	fsystem = &fstest.RecordingFs{}
	c.Assert(filesystem(), check.DeepEquals, fsystem)
	fsystem = nil
	c.Assert(filesystem(), check.DeepEquals, fs.OsFs{})
}

func (s *S) TestValidateVersion(c *check.C) {
	var cases = []struct {
		current, support string
		expected         bool
	}{
		{
			current:  "0.2.1",
			support:  "0.3",
			expected: false,
		},
		{
			current:  "0.3.5",
			support:  "0.3",
			expected: true,
		},
		{
			current:  "0.2",
			support:  "0.3",
			expected: false,
		},
		{
			current:  "0.7.10",
			support:  "0.7.2",
			expected: true,
		},
		{
			current:  "beta",
			support:  "0.7.2",
			expected: false,
		},
		{
			current:  "0.7.10",
			support:  "beta",
			expected: false,
		},
		{
			current:  "0.7.10",
			support:  "",
			expected: true,
		},
		{
			current:  "0.8",
			support:  "0.7.15",
			expected: true,
		},
		{
			current:  "0.8",
			support:  "0.8",
			expected: true,
		},
		{
			current:  "1.0-rc2",
			support:  "1.0-rc1",
			expected: true,
		},
		{
			current:  "1.0-rc1",
			support:  "1.0-rc1",
			expected: true,
		},
		{
			current:  "1.0-rc1",
			support:  "1.0-rc2",
			expected: false,
		},
		{
			current:  "1.0-rc1",
			support:  "1.0",
			expected: false,
		},
		{
			current:  "1.0",
			support:  "1.0-rc1",
			expected: true,
		},
	}
	for i, cs := range cases {
		c.Check(validateVersion(cs.support, cs.current), check.Equals, cs.expected, check.Commentf("error on %d", i))
	}
}

var _ Cancelable = &CancelableCommand{}

type CancelableCommand struct {
	running  chan struct{}
	canceled chan struct{}
}

func (c *CancelableCommand) Info() *Info {
	return &Info{
		Name:  "foo",
		Desc:  "Foo do anything or nothing.",
		Usage: "foo",
	}
}

func (c *CancelableCommand) Run(context *Context) error {
	c.running <- struct{}{}
	select {
	case <-c.canceled:
	case <-time.After(time.Second * 5):
		return fmt.Errorf("timeout waiting for cancellation")
	}
	return nil
}

func (c *CancelableCommand) Cancel(context Context) error {
	fmt.Fprintln(context.Stdout, "Canceled.")
	c.canceled <- struct{}{}
	return nil
}

type TestCommand struct{}

func (c *TestCommand) Info() *Info {
	return &Info{
		Name:  "foo",
		Desc:  "Foo do anything or nothing.",
		Usage: "foo",
	}
}

func (c *TestCommand) Run(context *Context) error {
	io.WriteString(context.Stdout, "Running TestCommand")
	return nil
}

type ErrorCommand struct {
	msg string
}

func (c *ErrorCommand) Info() *Info {
	return &Info{Name: "error"}
}

func (c *ErrorCommand) Run(context *Context) error {
	if c.msg == "abort" {
		return ErrAbortCommand
	}
	return fmt.Errorf("%s", c.msg)
}

type FailAndWorkCommand struct {
	calls int
}

func (c *FailAndWorkCommand) Info() *Info {
	return &Info{Name: "fail-and-work"}
}

func (c *FailAndWorkCommand) Run(context *Context) error {
	c.calls++
	if c.calls == 1 {
		return errors.New("FailAndWorkCommand more than one call")
	}
	fmt.Fprintln(context.Stdout, "worked nicely!")
	return nil
}

type SuccessLoginCommand struct{}

func (c *SuccessLoginCommand) Info() *Info {
	return &Info{Name: "login"}
}

func (c *SuccessLoginCommand) Run(context *Context) error {
	fmt.Fprintln(context.Stdout, "logged in!")
	return nil
}

type UnauthorizedErrorCommand struct{}

func (c *UnauthorizedErrorCommand) Info() *Info {
	return &Info{Name: "unauthorized-error"}
}

func (c *UnauthorizedErrorCommand) Run(context *Context) error {
	return errors.New("unauthorized")
}

type UnauthorizedLoginErrorCommand struct {
	UnauthorizedErrorCommand
}

func (c *UnauthorizedLoginErrorCommand) Info() *Info {
	return &Info{Name: "login"}
}

type CommandWithFlags struct {
	fs      *pflag.FlagSet
	age     int
	minArgs int
	args    []string
	multi   bool
}

func (c *CommandWithFlags) Info() *Info {
	return &Info{
		Name:    "with-flags",
		Desc:    "with-flags doesn't do anything, really.",
		Usage:   "with-flags",
		MinArgs: c.minArgs,
	}
}

func (c *CommandWithFlags) Run(context *Context) error {
	c.args = context.Args
	return nil
}

func (c *CommandWithFlags) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("with-flags", pflag.ContinueOnError)
		desc := "your age"
		if c.multi {
			desc = "velvet darkness\nthey fear"
		}
		c.fs.IntVarP(&c.age, "age", "a", 0, desc)
	}
	return c.fs
}

type HelpCommandWithFlags struct {
	fs *pflag.FlagSet
	h  bool
}

func (c *HelpCommandWithFlags) Info() *Info {
	return &Info{
		Name:  "hflags",
		Desc:  "hflags doesn't do anything, really.",
		Usage: "hflags",
	}
}

func (c *HelpCommandWithFlags) Run(context *Context) error {
	fmt.Fprintf(context.Stdout, "help called? %v", c.h)
	return nil
}

func (c *HelpCommandWithFlags) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("with-flags", pflag.ContinueOnError)
		c.fs.BoolVarP(&c.h, "help", "h", false, "help?")
	}
	return c.fs
}

type FailAndWorkCommandCustom struct {
	calls int
	err   error
}

func (c *FailAndWorkCommandCustom) Info() *Info {
	return &Info{Name: "fail-and-work"}
}

func (c *FailAndWorkCommandCustom) Run(context *Context) error {
	c.calls++
	if c.calls == 1 {
		return c.err
	}
	fmt.Fprintln(context.Stdout, "worked nicely!")
	return nil
}

type FailCommandCustom struct {
	err error
}

func (c *FailCommandCustom) Info() *Info {
	return &Info{Name: "failcmd"}
}

func (c *FailCommandCustom) Run(context *Context) error {
	return c.err
}

func (s *S) TestShorthandCommandInfo(c *check.C) {
	originalCmd := &TestCommand{}
	shorthandCmd := &ShorthandCommand{Command: originalCmd, shorthand: "f"}

	info := shorthandCmd.Info()
	c.Assert(info.Name, check.Equals, "f")
	c.Assert(info.GroupID, check.Equals, "shorthands")
	c.Assert(info.OnlyAppendOnRoot, check.Equals, true)
}

func (s *S) TestShorthandCommandInfoWithUsage(c *check.C) {
	cmd := &commandWithUsage{name: "app-deploy", usage: "app-deploy <file> [options]"}
	shorthandCmd := &ShorthandCommand{Command: cmd, shorthand: "deploy"}

	info := shorthandCmd.Info()
	c.Assert(info.Name, check.Equals, "deploy")
	c.Assert(info.Usage, check.Equals, "deploy <file> [options]")
	c.Assert(info.GroupID, check.Equals, "shorthands")
	c.Assert(info.OnlyAppendOnRoot, check.Equals, true)
}

func (s *S) TestShorthandCommandRun(c *check.C) {
	originalCmd := &TestCommand{}
	shorthandCmd := &ShorthandCommand{Command: originalCmd, shorthand: "f"}

	var stdout bytes.Buffer
	context := &Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Stdin:  os.Stdin,
	}
	err := shorthandCmd.Run(context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Running TestCommand")
}

func (s *S) TestShorthandCommandFlags(c *check.C) {
	originalCmd := &CommandWithFlags{}
	shorthandCmd := &ShorthandCommand{Command: originalCmd, shorthand: "wf"}

	flags := shorthandCmd.Flags()
	c.Assert(flags, check.NotNil)

	// Verify the flags from the original command are available
	ageFlag := flags.Lookup("age")
	c.Assert(ageFlag, check.NotNil)
	c.Assert(ageFlag.Shorthand, check.Equals, "a")
}

func (s *S) TestShorthandCommandFlagsWithNonFlaggedCommand(c *check.C) {
	originalCmd := &TestCommand{}
	shorthandCmd := &ShorthandCommand{Command: originalCmd, shorthand: "f"}

	flags := shorthandCmd.Flags()
	c.Assert(flags, check.NotNil)
	// Should return an empty flagset for non-flagged commands
	c.Assert(flags.HasFlags(), check.Equals, false)
}

func (s *S) TestRegisterShorthand(c *check.C) {
	mngr := NewManagerV2()

	originalCmd := &TestCommand{}
	mngr.RegisterShorthand(originalCmd, "f")

	// Shorthand should be registered in v2 manager (root) with the shorthand name
	rootCommands := mngr.rootCmd.Commands()
	var foundShorthand bool
	for _, v2cmd := range rootCommands {
		if v2cmd.Use == "f" {
			foundShorthand = true
			c.Assert(v2cmd.GroupID, check.Equals, "shorthands")
			c.Assert(v2cmd.Hidden, check.Equals, false) // OnlyAppendOnRoot makes it visible
			break
		}
	}
	c.Assert(foundShorthand, check.Equals, true)
}

type commandWithUsage struct {
	name  string
	usage string
}

func (c *commandWithUsage) Info() *Info {
	return &Info{
		Name:  c.name,
		Desc:  "A command with custom usage.",
		Usage: c.usage,
	}
}

func (c *commandWithUsage) Run(context *Context) error {
	io.WriteString(context.Stdout, "Running CommandWithUsage")
	return nil
}

func (s *S) TestHumanizeCommand(c *check.C) {
	program := ExtractProgramName(os.Args[0])
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", program + " foo"},
		{"app-info", program + " app info"},
		{"app-log-set", program + " app log set"},
		{"single", program + " single"},
	}
	for _, tt := range tests {
		result := humanizeCommand(tt.input)
		c.Assert(result, check.Equals, tt.expected)
	}
}

func (s *S) TestDeprecatedCommandInfo(c *check.C) {
	originalCmd := &commandWithUsage{
		name:  "app-info",
		usage: "app-info [flags]",
	}
	deprecatedCmd := &DeprecatedCommand{
		Command: originalCmd,
		oldName: "app-show",
	}

	info := deprecatedCmd.Info()

	c.Assert(info.Name, check.Equals, "app-show")
	c.Assert(info.Usage, check.Equals, "app-show [flags]")
	c.Assert(info.Desc, check.Matches, `(?s)DEPRECATED: For better usability, this command has been replaced by ".*app info"\..*`)
}

func (s *S) TestDeprecatedCommandInfoPreservesOriginalDesc(c *check.C) {
	originalCmd := &commandWithUsage{
		name:  "new-cmd",
		usage: "new-cmd",
	}
	deprecatedCmd := &DeprecatedCommand{
		Command: originalCmd,
		oldName: "old-cmd",
	}

	info := deprecatedCmd.Info()

	c.Assert(info.Desc, check.Matches, `(?s)DEPRECATED:.*A command with custom usage\.`)
}

func (s *S) TestDeprecatedCommandRunOutputsColoredWarning(c *check.C) {
	program := ExtractProgramName(os.Args[0])
	var stdout, stderr bytes.Buffer
	originalCmd := &commandWithUsage{
		name:  "app-info",
		usage: "app-info",
	}
	deprecatedCmd := &DeprecatedCommand{
		Command: originalCmd,
		oldName: "app-show",
	}

	ctx := &Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	err := deprecatedCmd.Run(ctx)
	c.Assert(err, check.IsNil)

	expectedOldCmd := program + " app show"
	expectedNewCmd := program + " app info"
	c.Assert(stderr.String(), check.Matches, `(?s).*WARNING:.*`+expectedOldCmd+`.*has been deprecated.*`+expectedNewCmd+`.*`)
	c.Assert(stdout.String(), check.Equals, "Running CommandWithUsage")
}
