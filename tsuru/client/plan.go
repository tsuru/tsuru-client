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
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
	apptypes "github.com/tsuru/tsuru/types/app"
	"k8s.io/apimachinery/pkg/api/resource"
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
		Usage:   "plan list [--bytes]",
		Desc:    "List available plans that can be used when creating an app.",
		MinArgs: 0,
	}
}

func renderPlans(plans []apptypes.Plan, isBytes, showDefaultColumn bool) string {
	table := tablecli.NewTable()
	table.Headers = []string{"Name", "CPU", "Memory"}
	hasSwap := false

	for _, p := range plans {
		if p.Swap > 0 {
			hasSwap = true
		}
	}

	if hasSwap {
		table.Headers = append(table.Headers, "Swap")
	}

	if showDefaultColumn {
		table.Headers = append(table.Headers, "Default")
	}

	for _, p := range plans {
		var cpu, memory, swap string
		if isBytes {
			memory = fmt.Sprintf("%d", p.Memory)
			swap = fmt.Sprintf("%d", p.Swap)
		} else {
			memory = resource.NewQuantity(p.Memory, resource.BinarySI).String()
			swap = resource.NewQuantity(p.Swap, resource.BinarySI).String()
		}

		if p.Override.CPUMilli != nil {
			cpu = fmt.Sprintf("%g", float64(*p.Override.CPUMilli)/10) + "% (override)"
		} else if p.CPUMilli > 0 {
			cpu = fmt.Sprintf("%g", float64(p.CPUMilli)/10) + "%"
		} else {
			cpu = fmt.Sprintf("%d (CPU share)", p.CpuShare)
		}

		if p.Override.Memory != nil {
			memory = resource.NewQuantity(*p.Override.Memory, resource.BinarySI).String() + " (override)"
		}

		row := []string{
			p.Name,
			cpu,
			memory,
		}

		if hasSwap {
			row = append(row, swap)
		}

		if showDefaultColumn {
			row = append(row, strconv.FormatBool(p.Default))
		}
		table.AddRow(row)
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
	fmt.Fprintf(context.Stdout, "%s", renderPlans(plans, c.bytes, true))
	return nil
}
