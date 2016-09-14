// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/errors"
)

type appSwap struct {
	cmd.Command
	force     bool
	cnameOnly bool
	fs        *gnuflag.FlagSet
}

func (s *appSwap) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-swap",
		Usage: "app-swap <app1-name> <app2-name> [-c/--cname-only]",
		Desc: `Swaps routing between two apps. This allows zero downtime and makes rollback
as simple as swapping the applications back.

Use [[--cname-only]] if you want to swap all cnames except the default
cname of application`,
		MinArgs: 2,
	}
}

func (s *appSwap) Flags() *gnuflag.FlagSet {
	if s.fs == nil {
		s.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		s.fs.BoolVar(&s.force, "force", false, "Force Swap among apps with different number of units or different platform.")
		s.fs.BoolVar(&s.force, "f", false, "Force Swap among apps with different number of units or different platform.")
		s.fs.BoolVar(&s.cnameOnly, "cname-only", false, "Swap all cnames except the default cname.")
		s.fs.BoolVar(&s.cnameOnly, "c", false, "Swap all cnames except the default cname.")
	}
	return s.fs
}

func (s *appSwap) Run(context *cmd.Context, client *cmd.Client) error {
	v := url.Values{}
	v.Set("app1", context.Args[0])
	v.Set("app2", context.Args[1])
	v.Set("force", strconv.FormatBool(s.force))
	v.Set("cnameOnly", strconv.FormatBool(s.cnameOnly))
	u, err := cmd.GetURL("/swap")
	if err != nil {
		return err
	}
	err = makeSwap(client, u, strings.NewReader(v.Encode()))
	if err != nil {
		if e, ok := err.(*errors.HTTP); ok && e.Code == http.StatusPreconditionFailed {
			var answer string
			fmt.Fprintf(context.Stdout, "WARNING: %s.\nSwap anyway? (y/n) ", strings.TrimRight(e.Message, "\n"))
			fmt.Fscanf(context.Stdin, "%s", &answer)
			if answer == "y" || answer == "yes" {
				v = url.Values{}
				v.Set("app1", context.Args[0])
				v.Set("app2", context.Args[1])
				v.Set("force", "true")
				v.Set("cnameOnly", strconv.FormatBool(s.cnameOnly))
				u, err = cmd.GetURL("/swap")
				if err != nil {
					return err
				}
				return makeSwap(client, u, strings.NewReader(v.Encode()))
			}
			fmt.Fprintln(context.Stdout, "swap aborted.")
			return nil
		}
		return err
	}
	fmt.Fprintln(context.Stdout, "Apps successfully swapped!")
	return err
}

func makeSwap(client *cmd.Client, url string, body io.Reader) error {
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	return err
}
