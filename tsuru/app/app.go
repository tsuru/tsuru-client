// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"github.com/pkg/errors"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/standards"
)

type AppNameMixIn struct {
	fs      *gnuflag.FlagSet
	appName string
}

func (cmd *AppNameMixIn) AppNameByArgsAndFlag(args []string) (string, error) {
	if len(args) > 0 {
		if cmd.appName != "" {
			return "", errors.New("You can't use the app flag and specify the app name as an argument at the same time.")
		}

		return args[0], nil
	}

	return cmd.AppNameByFlag()
}

func (cmd *AppNameMixIn) AppNameByFlag() (string, error) {
	if cmd.appName == "" {
		return "", errors.Errorf(`The name of the app is required.

Use the --app flag to specify it.

`)
	}
	return cmd.appName, nil
}

func (cmd *AppNameMixIn) Flags() *gnuflag.FlagSet {
	if cmd.fs == nil {
		cmd.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		cmd.fs.StringVar(&cmd.appName, "app", "", "The name of the app.")
		cmd.fs.StringVar(&cmd.appName, standards.ShortFlagApp, "", "The name of the app.")
	}
	return cmd.fs
}
