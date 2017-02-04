// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
)

type PlanCreate struct {
	memory     string
	swap       string
	cpushare   int
	setDefault bool
	router     string
	fs         *gnuflag.FlagSet
}

func (c *PlanCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("plan-create", gnuflag.ExitOnError)
		memory := `Amount of available memory for units in bytes or an integer value followed
by M, K or G for megabytes, kilobytes or gigabytes respectively.`
		c.fs.StringVar(&c.memory, "memory", "0", memory)
		c.fs.StringVar(&c.memory, "m", "0", memory)
		swap := `Amount of available swap space for units in bytes or an integer value followed
by M, K or G for megabytes, kilobytes or gigabytes respectively.`
		c.fs.StringVar(&c.swap, "swap", "0", swap)
		c.fs.StringVar(&c.swap, "s", "0", swap)
		cpushare := `Relative cpu share each unit will have available. This value is unitless and
relative, so specifying the same value for all plans means all units will
equally share processing power.`
		c.fs.IntVar(&c.cpushare, "cpushare", 0, cpushare)
		c.fs.IntVar(&c.cpushare, "c", 0, cpushare)
		setDefault := `Set plan as default, this will remove the default flag from any other plan.
The default plan will be used when creating an application without explicitly
setting a plan.`
		c.fs.BoolVar(&c.setDefault, "default", false, setDefault)
		c.fs.BoolVar(&c.setDefault, "d", false, setDefault)
		router := "The name of the router used by this plan."
		c.fs.StringVar(&c.router, "router", "", router)
		c.fs.StringVar(&c.router, "r", "", router)
	}
	return c.fs
}

func (c *PlanCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plan-create",
		Usage:   "plan-create <name> -c cpushare [-m memory] [-s swap] [-r router] [--default]",
		Desc:    `Creates a new plan for being used when creating apps.`,
		MinArgs: 1,
	}
}

func (c *PlanCreate) Run(context *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURL("/plans")
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("name", context.Args[0])
	v.Set("memory", c.memory)
	v.Set("swap", c.swap)
	v.Set("cpushare", strconv.Itoa(c.cpushare))
	v.Set("default", strconv.FormatBool(c.setDefault))
	v.Set("router", c.router)
	b := strings.NewReader(v.Encode())
	request, err := http.NewRequest("POST", u, b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		fmt.Fprintf(context.Stdout, "Failed to create plan!\n")
		return err
	}
	fmt.Fprintf(context.Stdout, "Plan successfully created!\n")
	return nil
}

type PlanRemove struct{}

func (c *PlanRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "plan-remove",
		Usage: "plan-remove <name>",
		Desc: `Removes an existing plan. It will no longer be available for newly created
apps. However, this won't change anything for existing apps that were created
using the removed plan. They will keep using the same value amount of
resources described by the plan.`,
		MinArgs: 1,
	}
}

func (c *PlanRemove) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/plans/" + context.Args[0])
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		fmt.Fprintf(context.Stdout, "Failed to remove plan!\n")
		return err
	}
	fmt.Fprintf(context.Stdout, "Plan successfully removed!\n")
	return nil
}
