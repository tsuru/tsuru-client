// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
	"launchpad.net/gnuflag"
	"net/http"
)

type Swap struct {
	cmd.Command
	force bool
	fs    *gnuflag.FlagSet
}

func (s *Swap) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "swap",
		Usage:   "swap app1-name app2-name [--force]",
		Desc:    "Swap router between two apps. Use force if you want to Swap apps with different numbers of units or diferent platform.",
		MinArgs: 2,
	}
}

func (s *Swap) Flags() *gnuflag.FlagSet {
	if s.fs == nil {
		s.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		s.fs.BoolVar(&s.force, "force", false, "Force Swap among apps with different number of units or different platform.")
	}
	return s.fs
}

func (s *Swap) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL(fmt.Sprintf("/swap?app1=%s&app2=%s&force=%t", context.Args[0], context.Args[1], s.force))
	if err != nil {
		return err
	}
	err = makeSwap(client, url)
	if err != nil {
		if err.Error() == app.ErrAppNotEqual.Error() {
			var answer string
			answersOptions := []string{"y", "yes"}
			fmt.Fprint(context.Stdout, "We can't Swap your apps because they are not compatible. Do you want to do it anyway? (y/n)")
			fmt.Fscanf(context.Stdin, "%s", &answer)
			if answerAcceptable(answer, answersOptions) {
				url, _ = cmd.GetURL(fmt.Sprintf("/swap?app1=%s&app2=%s&force=%t", context.Args[0], context.Args[1], true))
				err = makeSwap(client, url)
				if err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}
	fmt.Fprintln(context.Stdout, "Apps successfully swapped!")
	return err
}

func makeSwap(client *cmd.Client, url string) error {
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	return err
}

func answerAcceptable(answer string, answersOptions []string) bool {
	for _, answerOption := range answersOptions {
		if answer == answerOption {
			return true
		}
	}
	return false
}
