// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"

	"github.com/spf13/pflag"
)

type ConfirmationCommand struct {
	yes bool
	fs  *pflag.FlagSet
}

func (cmd *ConfirmationCommand) Flags() *pflag.FlagSet {
	if cmd.fs == nil {
		cmd.fs = pflag.NewFlagSet("", pflag.ExitOnError)
		cmd.fs.BoolVarP(&cmd.yes, "assume-yes", "y", false, "Don't ask for confirmation.")
	}
	return cmd.fs
}

func (cmd *ConfirmationCommand) Confirm(context *Context, question string) bool {
	if cmd.yes {
		return true
	}
	context.RawOutput()
	fmt.Fprintf(context.Stdout, `%s (y/n) `, question)
	var answer string
	if context.Stdin != nil {
		fmt.Fscanf(context.Stdin, "%s", &answer)
	}
	if answer != "y" {
		fmt.Fprintln(context.Stdout, "Abort.")
		return false
	}
	return true
}
