// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	apptypes "github.com/tsuru/tsuru/types/app"
)

type PlanList struct {
	bytes bool
	fs    *gnuflag.FlagSet
}

func (c *PlanList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("plan-list", gnuflag.ExitOnError)
		bytes := "bytesized units for memory and swap."
		c.fs.BoolVar(&c.bytes, "bytes", false, bytes)
		c.fs.BoolVar(&c.bytes, "b", false, bytes)
	}
	return c.fs
}

func (c *PlanList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plan-list",
		Usage:   "plan-list [--bytes]",
		Desc:    "List available plans that can be used when creating an app.",
		MinArgs: 0,
	}
}

func renderPlans(plans []apptypes.Plan, isBytes bool) string {
	table := cmd.NewTable()
	table.Headers = []string{"Name", "Memory", "Swap", "Cpu Share", "Default"}
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
			strconv.FormatBool(p.Default),
		})
	}
	return table.String()
}

func (c *PlanList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/plans")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	var plans []apptypes.Plan
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		fmt.Fprintln(context.Stdout, "No plans available.")
		return nil
	}
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "%s", renderPlans(plans, c.bytes))
	return nil
}
