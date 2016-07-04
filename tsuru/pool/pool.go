// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/errors"
)

type AddPoolToSchedulerCmd struct {
	public       bool
	defaultPool  bool
	forceDefault bool
	fs           *gnuflag.FlagSet
}

func (AddPoolToSchedulerCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "pool-add",
		Usage: "pool-add <pool> [-p/--public] [-d/--default] [-f/--force]",
		Desc: `Adds a new pool.

Each docker node added using [[docker-node-add]] command belongs to one pool.
Also, when creating a new application a pool must be chosen and this means
that all units of the created application will be spawned in nodes belonging
to the chosen pool.`,
		MinArgs: 1,
	}
}

func (c *AddPoolToSchedulerCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		msg := "Make pool public (all teams can use it)"
		c.fs.BoolVar(&c.public, "public", false, msg)
		c.fs.BoolVar(&c.public, "p", false, msg)
		msg = "Make pool default (when none is specified during [[app-create]] this pool will be used)"
		c.fs.BoolVar(&c.defaultPool, "default", false, msg)
		c.fs.BoolVar(&c.defaultPool, "d", false, msg)
		msg = "Force overwrite default pool"
		c.fs.BoolVar(&c.forceDefault, "force", false, msg)
		c.fs.BoolVar(&c.forceDefault, "f", false, msg)
	}
	return c.fs
}

func (c *AddPoolToSchedulerCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	v := url.Values{}
	v.Set("name", ctx.Args[0])
	v.Set("public", strconv.FormatBool(c.public))
	v.Set("default", strconv.FormatBool(c.defaultPool))
	v.Set("force", strconv.FormatBool(c.forceDefault))
	u, err := cmd.GetURL("/pools")
	err = doRequest(client, u, "POST", v.Encode())
	if err != nil {
		if e, ok := err.(*errors.HTTP); ok && e.Code == http.StatusPreconditionFailed {
			retryMessage := "WARNING: Default pool already exist. Do you want change to %s pool? (y/n) "
			v.Set("force", "true")
			url, _ := cmd.GetURL("/pools")
			successMessage := "Pool successfully registered.\n"
			failMessage := "Pool add aborted.\n"
			return confirmAction(ctx, client, url, "POST", v.Encode(), retryMessage, failMessage, successMessage)
		}
		return err
	}
	ctx.Stdout.Write([]byte("Pool successfully registered.\n"))
	return nil
}

func doRequest(client *cmd.Client, url, method, body string) error {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	return nil
}

func confirmAction(ctx *cmd.Context, client *cmd.Client, url, method, body string, retryMessage, failMessage, successMessage string) error {
	var answer string
	fmt.Fprintf(ctx.Stdout, retryMessage, ctx.Args[0])
	fmt.Fscanf(ctx.Stdin, "%s", &answer)
	if answer == "y" || answer == "yes" {
		err := doRequest(client, url, method, body)
		if err != nil {
			return err
		}
		ctx.Stdout.Write([]byte(successMessage))
		return nil

	}
	ctx.Stdout.Write([]byte(failMessage))
	return nil
}
