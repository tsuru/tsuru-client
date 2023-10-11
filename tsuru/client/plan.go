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
	bytes               bool
	k8sFriendly         bool
	showMaxBurstAllowed bool

	fs *gnuflag.FlagSet
}

func (c *PlanList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("plan-list", gnuflag.ExitOnError)
		bytes := "bytesized units for memory and swap."
		c.fs.BoolVar(&c.bytes, "bytes", false, bytes)
		c.fs.BoolVar(&c.showMaxBurstAllowed, "show-max-cpu-burst-allowed", false, "show column about max CPU burst allowed by plan")
		c.fs.BoolVar(&c.k8sFriendly, "kubernetes-friendly", false, "show values friendly for a kubernetes user")

		c.fs.BoolVar(&c.bytes, "b", false, bytes)
	}
	return c.fs
}

func (c *PlanList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plan-list",
		Usage:   "plan list [--bytes][--kubernetes-friendly]",
		Desc:    "List available plans that can be used when creating an app.",
		MinArgs: 0,
	}
}

func renderPlans(plans []apptypes.Plan, isBytes, showDefaultColumn bool, showMaxBurstAllowed bool) string {
	table := tablecli.NewTable()
	table.Headers = []string{"Name", "CPU", "Memory"}

	showBurstColumn := false

	for _, p := range plans {
		if hasBurst(p) {
			showBurstColumn = true
			break
		}
	}

	if showBurstColumn {
		table.Headers = append(table.Headers, "CPU Burst (default)")
	}

	if showBurstColumn && showMaxBurstAllowed {
		table.Headers = append(table.Headers, "CPU Burst (max customizable)")
	}

	if showDefaultColumn {
		table.Headers = append(table.Headers, "Default")
	}

	for _, p := range plans {
		var cpu, memory string
		if isBytes {
			memory = fmt.Sprintf("%d", p.Memory)
		} else {
			memory = resource.NewQuantity(p.Memory, resource.BinarySI).String()
		}

		if p.Override.CPUMilli != nil {
			cpu = fmt.Sprintf("%g", float64(*p.Override.CPUMilli)/10) + "% (override)"
		} else if p.CPUMilli > 0 {
			cpu = fmt.Sprintf("%g", float64(p.CPUMilli)/10) + "%"
		}

		if p.Override.Memory != nil {
			memory = resource.NewQuantity(*p.Override.Memory, resource.BinarySI).String() + " (override)"
		}

		row := []string{
			p.Name,
			cpu,
			memory,
		}

		if showBurstColumn {
			cpuBurst := p.CPUBurst.Default
			cpuBurstObservation := ""
			if p.Override.CPUBurst != nil {
				cpuBurst = *p.Override.CPUBurst
				cpuBurstObservation = " (override)"
			}

			row = append(row, displayCPUBurst(p.CPUMilli, cpuBurst)+cpuBurstObservation)
		}

		if showBurstColumn && showMaxBurstAllowed {
			row = append(row, displayCPUBurst(p.CPUMilli, p.CPUBurst.MaxAllowed))
		}

		if showDefaultColumn {
			row = append(row, strconv.FormatBool(p.Default))
		}
		table.AddRow(row)
	}
	return table.String()
}

func renderPlansK8SFriendly(plans []apptypes.Plan, showMaxBurstAllowed bool) string {
	table := tablecli.NewTable()
	table.Headers = []string{"Name"}

	showCPULimitsColumn := false
	for _, p := range plans {
		if hasBurst(p) {
			showCPULimitsColumn = true
			break
		}
	}

	if showCPULimitsColumn {
		table.Headers = append(table.Headers, "CPU requests", "CPU limits")
	} else {
		table.Headers = append(table.Headers, "CPU requests/limits")
	}

	if showMaxBurstAllowed {
		table.Headers = append(table.Headers, "CPU limits (max customizable)")
	}

	table.Headers = append(table.Headers, "Memory requests/limits", "Default")

	for _, p := range plans {
		memory := resource.NewQuantity(p.Memory, resource.BinarySI).String()
		cpuRequest := resource.NewMilliQuantity(int64(p.CPUMilli), resource.DecimalSI).String()
		maxCPULimit := resource.NewMilliQuantity(int64(float64(p.CPUMilli)*p.CPUBurst.MaxAllowed), resource.DecimalSI).String()

		row := []string{
			p.Name,
		}

		if showCPULimitsColumn {
			cpuBurst := p.CPUBurst.Default
			if cpuBurst < 1 {
				cpuBurst = 1
			}
			defaultCPULimit := resource.NewMilliQuantity(int64(float64(p.CPUMilli)*cpuBurst), resource.DecimalSI).String()
			row = append(row, cpuRequest, defaultCPULimit)
		} else {
			row = append(row, cpuRequest)
		}

		if showMaxBurstAllowed {
			row = append(row, maxCPULimit)
		}

		row = append(row, memory, strconv.FormatBool(p.Default))

		table.AddRow(row)
	}
	return table.String()
}

func hasBurst(p apptypes.Plan) bool {
	if p.CPUMilli == 0 {
		return false
	}
	if p.CPUBurst.Default != 0 {
		return true
	}

	if p.Override.CPUBurst != nil {
		return true
	}
	return false
}

func displayCPUBurst(currentCPU int, burst float64) string {
	if currentCPU == 0 || burst < 1 {
		return ""
	}

	cpu := int(float64(currentCPU) * burst / 10)
	return fmt.Sprintf("up to %g", float64(cpu)) + "%"
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

	if c.k8sFriendly {
		fmt.Fprintf(context.Stdout, "%s", renderPlansK8SFriendly(plans, c.showMaxBurstAllowed))
	} else {
		fmt.Fprintf(context.Stdout, "%s", renderPlans(plans, c.bytes, true, c.showMaxBurstAllowed))
	}

	return nil
}
