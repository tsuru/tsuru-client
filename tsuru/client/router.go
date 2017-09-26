// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/router"
	appTypes "github.com/tsuru/tsuru/types/app"
)

type RoutersList struct{}

func (c *RoutersList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "router-list",
		Usage:   "router-list",
		Desc:    "List all routers available for app creation.",
		MinArgs: 0,
	}
}

func (c *RoutersList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURLVersion("1.3", "/routers")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	var routers []router.PlanRouter
	if response.StatusCode == http.StatusOK {
		err = json.NewDecoder(response.Body).Decode(&routers)
		if err != nil {
			return err
		}
	}
	table := cmd.NewTable()
	table.Headers = cmd.Row([]string{"Name", "Type"})
	table.LineSeparator = true
	for _, router := range routers {
		table.AddRow(cmd.Row([]string{router.Name, router.Type}))
	}
	context.Stdout.Write(table.Bytes())
	return nil
}

type AppRoutersList struct {
	cmd.GuessingCommand
}

func (c *AppRoutersList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-router-list",
		Usage:   "app-router-list [-a/--app appname]",
		Desc:    "List all routers associated to an application.",
		MinArgs: 0,
	}
}

func (c *AppRoutersList) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURLVersion("1.5", fmt.Sprintf("/apps/%s/routers", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent {
		fmt.Fprintln(context.Stdout, "No routers available for app.")
		return nil
	}
	var routers []appTypes.AppRouter
	err = json.NewDecoder(response.Body).Decode(&routers)
	if err != nil {
		return err
	}
	renderRouters(routers, context.Stdout)
	return nil
}

func renderRouters(routers []appTypes.AppRouter, out io.Writer) {
	table := cmd.NewTable()
	table.Headers = cmd.Row([]string{"Name", "Opts", "Address"})
	table.LineSeparator = true
	for _, r := range routers {
		var optsStr []string
		for k, v := range r.Opts {
			optsStr = append(optsStr, fmt.Sprintf("%s: %s", k, v))
		}
		sort.Strings(optsStr)
		table.AddRow(cmd.Row([]string{r.Name, strings.Join(optsStr, "\n"), r.Address}))
	}
	out.Write(table.Bytes())
}

type AppRoutersAdd struct {
	cmd.GuessingCommand
	opts cmd.MapFlag
	fs   *gnuflag.FlagSet
}

func (c *AppRoutersAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-router-add",
		Usage:   "app-router-add <router name> [-a/--app appname] [-o/--opts key=value]...",
		Desc:    "Add a new router to an application.",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *AppRoutersAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		optsMessage := "Custom options sent directly to router implementation."
		c.fs.Var(&c.opts, "o", optsMessage)
		c.fs.Var(&c.opts, "opts", optsMessage)
	}
	return c.fs
}

func (c *AppRoutersAdd) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURLVersion("1.5", fmt.Sprintf("/apps/%s/routers", appName))
	if err != nil {
		return err
	}
	r := appTypes.AppRouter{
		Name: context.Args[0],
		Opts: c.opts,
	}
	val, err := form.EncodeToValues(r)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, strings.NewReader(val.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Router successfully added.")
	return nil
}

type AppRoutersUpdate struct {
	cmd.GuessingCommand
	opts cmd.MapFlag
	fs   *gnuflag.FlagSet
}

func (c *AppRoutersUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-router-update",
		Usage:   "app-router-update <router name> [-a/--app appname] [-o/--opts key=value]...",
		Desc:    "Update router opts in an application.",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *AppRoutersUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		optsMessage := "Custom options sent directly to router implementation."
		c.fs.Var(&c.opts, "o", optsMessage)
		c.fs.Var(&c.opts, "opts", optsMessage)
	}
	return c.fs
}

func (c *AppRoutersUpdate) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	routerName := context.Args[0]
	url, err := cmd.GetURLVersion("1.5", fmt.Sprintf("/apps/%s/routers/%s", appName, routerName))
	if err != nil {
		return err
	}
	r := appTypes.AppRouter{
		Name: routerName,
		Opts: c.opts,
	}
	val, err := form.EncodeToValues(r)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, strings.NewReader(val.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Router successfully updated.")
	return nil
}

type AppRoutersRemove struct {
	cmd.GuessingCommand
}

func (c *AppRoutersRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-router-remove",
		Usage:   "app-router-remove <router name> [-a/--app appname]",
		Desc:    "Remove a router from an application.",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *AppRoutersRemove) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURLVersion("1.5", fmt.Sprintf("/apps/%s/routers/%s", appName, context.Args[0]))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Router successfully removed.")
	return nil
}
