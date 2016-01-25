// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tsuru/gnuflag"
	tsuruapp "github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
)

type planList struct {
	bytes bool
	fs    *gnuflag.FlagSet
}

func (c *planList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("plan-List", gnuflag.ExitOnError)
		bytes := "bytesized units for memory and swap."
		c.fs.BoolVar(&c.bytes, "bytes", false, bytes)
		c.fs.BoolVar(&c.bytes, "b", false, bytes)
	}
	return c.fs
}

func (c *planList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plan-list",
		Usage:   "plan-list [--bytes]",
		Desc:    "List available plans that can be used when creating an app.",
		MinArgs: 0,
	}
}

func renderPlans(plans []tsuruapp.Plan, isBytes bool) string {
	table := cmd.NewTable()
	table.Headers = []string{"Name", "Memory", "Swap", "Cpu Share", "Router", "Default"}
	for _, p := range plans {
		var memory, swap string
		if isBytes {
			memory = fmt.Sprintf("%d", p.Memory)
			swap = fmt.Sprintf("%d", p.Swap)
		} else {
			memory = fmt.Sprintf("%d MB", p.Memory/1024/1024)
			swap = fmt.Sprintf("%d MB", p.Swap/1024/1024)
		}
		table.AddRow([]string{
			p.Name, memory, swap,
			strconv.Itoa(p.CpuShare),
			p.Router,
			strconv.FormatBool(p.Default),
		})
	}
	return table.String()
}

func (c *planList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/plans")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	var plans []tsuruapp.Plan
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return err
	}
	if len(plans) == 0 {
		fmt.Fprintln(context.Stdout, "No plans available.")
		return nil
	}
	fmt.Fprintf(context.Stdout, "%s", renderPlans(plans, c.bytes))
	return nil
}
