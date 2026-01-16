// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	goVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/tsuru/tsuru/fs"
)

var (
	ErrAbortCommand = errors.New("")

	// ErrLookup is the error that should be returned by lookup functions when it
	// cannot find a matching command for the given parameters.
	ErrLookup = errors.New("lookup error - command not found")
)

// Cancelable are implemented by commands that support cancellation
type Cancelable interface {
	// Cancel handles the command cancellation and is required to be thread safe as
	// this method is called by a different goroutine.
	// Cancel should return an error if the operation is not cancelable yet/anymore or there
	// was any error during the cancellation.
	// Cancel may be called multiple times.
	Cancel(context Context) error
}

type Command interface {
	Info() *Info
	Run(context *Context) error
}

type FlaggedCommand interface {
	Command
	Flags() *pflag.FlagSet
}

type AutoCompleteCommand interface {
	Command
	Complete(args []string, toComplete string) ([]string, error)
}

type DeprecatedCommand struct {
	Command
	oldName string
}

func (c *DeprecatedCommand) Info() *Info {
	info := c.Command.Info()
	newCommand := humanizeCommand(info.Name)

	info.Desc = fmt.Sprintf("DEPRECATED: For better usability, this command has been replaced by %q.\n\n%s", newCommand, info.Desc)

	info.Usage = c.oldName + stripUsage(info.Name, info.Usage)
	info.Name = c.oldName
	return info
}

func humanizeCommand(s string) string {
	program := ExtractProgramName(os.Args[0])
	return program + " " + strings.ReplaceAll(s, "-", " ")
}

func (c *DeprecatedCommand) Run(context *Context) error {
	oldCommand := humanizeCommand(c.oldName)
	newCommand := humanizeCommand(c.Command.Info().Name)

	warningText := fmt.Sprintf("WARNING: %q has been deprecated, please use %q instead.\n\n", oldCommand, newCommand)

	fmt.Fprint(context.Stderr, Colorfy(warningText, "yellow", "", "bold"))

	return c.Command.Run(context)
}

func (c *DeprecatedCommand) Flags() *pflag.FlagSet {
	if cmd, ok := c.Command.(FlaggedCommand); ok {
		return cmd.Flags()
	}
	return pflag.NewFlagSet("", pflag.ContinueOnError)
}

type ShorthandCommand struct {
	Command
	shorthand string
}

func (c *ShorthandCommand) Info() *Info {
	info := c.Command.Info()

	info.Usage = c.shorthand + stripUsage(info.Name, info.Usage)

	info.Name = c.shorthand
	info.V2.GroupID = "shorthands"
	info.V2.OnlyAppendOnRoot = true
	return info
}

func (c *ShorthandCommand) Run(context *Context) error {
	return c.Command.Run(context)
}

func (c *ShorthandCommand) Flags() *pflag.FlagSet {
	if cmd, ok := c.Command.(FlaggedCommand); ok {
		return cmd.Flags()
	}
	return pflag.NewFlagSet("", pflag.ContinueOnError)
}

func (c *ShorthandCommand) Complete(args []string, toComplete string) ([]string, error) {
	if autoCompleteCmd, ok := c.Command.(AutoCompleteCommand); ok {
		return autoCompleteCmd.Complete(args, toComplete)
	}
	return nil, nil
}

type Context struct {
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

func (c *Context) RawOutput() {
	if pager, ok := c.Stdout.(*pagerWriter); ok {
		c.Stdout = pager.baseWriter
	}
	if sync, ok := c.Stdin.(*syncReader); ok {
		c.Stdin = sync.baseReader
	}
}

var ArbitraryArgs = -1

type Info struct {
	Name    string
	MinArgs int
	MaxArgs int
	Usage   string
	Desc    string
	V2      InfoV2
	fail    bool
}

var flagFormatRegexp = regexp.MustCompile(`(?m)^([^-\s])`)

func ExtractProgramName(path string) string {
	return filepath.Base(path)
}

var (
	fsystem   fs.Fs
	fsystemMu sync.Mutex
)

func filesystem() fs.Fs {
	fsystemMu.Lock()
	defer fsystemMu.Unlock()
	if fsystem == nil {
		fsystem = fs.OsFs{}
	}
	return fsystem
}

// validateVersion checks whether current version is greater or equal to
// supported version.
func validateVersion(supported, current string) bool {
	if current == "dev" {
		return true
	}
	if supported == "" {
		return true
	}
	vSupported, err := goVersion.NewVersion(supported)
	if err != nil {
		return false
	}
	vCurrent, err := goVersion.NewVersion(current)
	if err != nil {
		return false
	}
	return vCurrent.Compare(vSupported) >= 0
}
