// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/ajg/form"
	"github.com/pkg/errors"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
	appTypes "github.com/tsuru/tsuru/types/app"
)

type RouterAdd struct {
	rawConfig string
	fs        *gnuflag.FlagSet
}

func (c *RouterAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "router-add",
		Usage:   "router add <name> <type> [--config {json object}]",
		Desc:    "Adds a new dynamic router to tsuru.",
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *RouterAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("router-add", gnuflag.ExitOnError)
		c.fs.StringVar(&c.rawConfig, "config", "", "JSON object with router configuration")
	}
	return c.fs
}

func (c *RouterAdd) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	if len(ctx.Args) != 2 {
		return errors.New("invalid arguments")
	}
	dynRouter := tsuru.DynamicRouter{
		Name: ctx.Args[0],
		Type: ctx.Args[1],
	}
	if c.rawConfig != "" {
		err = json.Unmarshal([]byte(c.rawConfig), &dynRouter.Config)
		if err != nil {
			return errors.Wrap(err, "unable to parse config")
		}
	}
	_, err = apiClient.RouterApi.RouterCreate(context.TODO(), dynRouter)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Dynamic router successfully added.")
	return nil
}

type RouterUpdate struct {
	rawConfig string
	fs        *gnuflag.FlagSet
}

func (c *RouterUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "router-update",
		Usage:   "router update <name> <type> [--config {json object}]",
		Desc:    "Updates an existing dynamic router.",
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *RouterUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("router-add", gnuflag.ExitOnError)
		c.fs.StringVar(&c.rawConfig, "config", "", "JSON object with router configuration")
	}
	return c.fs
}

func (c *RouterUpdate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	if len(ctx.Args) != 2 {
		return errors.New("invalid arguments")
	}
	dynRouter := tsuru.DynamicRouter{
		Name: ctx.Args[0],
		Type: ctx.Args[1],
	}
	if c.rawConfig != "" {
		err = json.Unmarshal([]byte(c.rawConfig), &dynRouter.Config)
		if err != nil {
			return errors.Wrap(err, "unable to parse config")
		}
	}
	_, err = apiClient.RouterApi.RouterUpdate(context.TODO(), dynRouter.Name, dynRouter)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Dynamic router successfully updated.")
	return nil
}

type RouterRemove struct{}

func (c *RouterRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "router-remove",
		Usage:   "router remove <name>",
		Desc:    "Removes an existing dynamic router.",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *RouterRemove) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	if len(ctx.Args) != 1 {
		return errors.New("invalid arguments")
	}
	_, err = apiClient.RouterApi.RouterDelete(context.TODO(), ctx.Args[0])
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Dynamic router successfully removed.")
	return nil
}

type RoutersList struct{}

func (c *RoutersList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "router-list",
		Usage:   "router list",
		Desc:    "List all routers available for app creation.",
		MinArgs: 0,
	}
}

func (c *RoutersList) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	routers, _, err := apiClient.RouterApi.RouterList(context.TODO())
	if err != nil {
		return err
	}

	table := tablecli.NewTable()
	table.Headers = tablecli.Row([]string{"Name", "Type", "Info"})
	table.LineSeparator = true
	for _, router := range routers {
		var infos []string
		for k, v := range router.Info {
			infos = append(infos, fmt.Sprintf("%s: %s", k, v))
		}
		sort.Strings(infos)
		if router.Dynamic {
			router.Type += " (dynamic)"
		}
		table.AddRow(tablecli.Row([]string{router.Name, router.Type, strings.Join(infos, "\n")}))
	}
	ctx.Stdout.Write(table.Bytes())
	return nil
}

type RouterInfo struct{}

func (c *RouterInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "router-info",
		Usage:   "router info <name>",
		Desc:    "Show detailed information for router.",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *RouterInfo) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	routers, _, err := apiClient.RouterApi.RouterList(context.TODO())
	if err != nil {
		return err
	}
	name := ctx.Args[0]

	var router *tsuru.PlanRouter
	for _, r := range routers {
		if r.Name == name {
			router = &r
			break
		}
	}
	if router == nil {
		return errors.Errorf("router %q not found", name)
	}

	fmt.Fprintf(ctx.Stdout, "Name: %s\n", router.Name)
	fmt.Fprintf(ctx.Stdout, "Type: %s\n", router.Type)
	fmt.Fprintf(ctx.Stdout, "Dynamic: %v\n", router.Dynamic)
	fmt.Fprintf(ctx.Stdout, "Info:\n")
	for key, value := range router.Info {
		fmt.Fprintf(ctx.Stdout, "  %s: %s\n", key, value)
	}
	fmt.Fprintf(ctx.Stdout, "Config:\n")
	data, err := json.MarshalIndent(router.Config, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "%s\n", data)
	return nil
}

type AppRoutersList struct {
	cmd.GuessingCommand
}

func (c *AppRoutersList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-router-list",
		Usage:   "app router list [-a/--app appname]",
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
	renderRouters(routers, context.Stdout, "Name")
	return nil
}

func renderRouters(routers []appTypes.AppRouter, out io.Writer, idColumn string) {
	table := tablecli.NewTable()
	table.Headers = tablecli.Row([]string{idColumn, "Opts", "Addresses", "Status"})
	table.LineSeparator = true
	for _, r := range routers {
		var optsStr []string
		for k, v := range r.Opts {
			optsStr = append(optsStr, fmt.Sprintf("%s: %s", k, v))
		}
		sort.Strings(optsStr)
		statusStr := r.Status
		if r.StatusDetail != "" {
			statusStr = fmt.Sprintf("%s: %s", statusStr, r.StatusDetail)
		}
		addresses := r.Address
		if len(r.Addresses) > 0 {
			sort.Strings(r.Addresses)
			addresses = strings.Join(r.Addresses, "\n")
		}
		row := tablecli.Row([]string{
			r.Name,
			strings.Join(optsStr, "\n"),
			addresses,
			statusStr,
		})
		table.AddRow(row)
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
		Usage:   "app router add <router name> [-a/--app appname] [-o/--opts key=value]...",
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
		Usage:   "app router update <router name> [-a/--app appname] [-o/--opts key=value]...",
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
		Usage:   "app router remove <router name> [-a/--app appname]",
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

type appVersionRouterBase struct {
	cmd.GuessingCommand
	routable bool
}

func (c *appVersionRouterBase) Run(ctx *cmd.Context, cli *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}

	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	_, err = apiClient.AppApi.AppSetRoutable(context.TODO(), appName, tsuru.SetRoutableArgs{
		Version:    ctx.Args[0],
		IsRoutable: c.routable,
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Version successfully updated.")
	return nil
}

type AppVersionRouterAdd struct {
	appVersionRouterBase
}

func (c *AppVersionRouterAdd) Info() *cmd.Info {
	c.appVersionRouterBase.routable = true
	return &cmd.Info{
		Name:    "app-router-version-add",
		Usage:   "app router version add <version> [-a/--app appname]",
		Desc:    "Adds an app version as routable.",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

type AppVersionRouterRemove struct {
	appVersionRouterBase
}

func (c *AppVersionRouterRemove) Info() *cmd.Info {
	c.appVersionRouterBase.routable = false
	return &cmd.Info{
		Name:    "app-router-version-remove",
		Usage:   "app router version remove <version> [-a/--app appname]",
		Desc:    "Removes an app version from being routable.",
		MinArgs: 1,
		MaxArgs: 1,
	}
}
