// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tablecli"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
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
		Usage:   "plan list [--bytes][--kubernetes-friendly][--show-max-cpu-burst-allowed]",
		Desc:    "List available plans that can be used when creating an app.",
		MinArgs: 0,
	}
}

type renderPlansOpts struct {
	isBytes, showDefaultColumn, showMaxBurstAllowed bool
}

func renderPlans(plans []apptypes.Plan, opts renderPlansOpts) string {
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

	if showBurstColumn && opts.showMaxBurstAllowed {
		table.Headers = append(table.Headers, "CPU Burst (max customizable)")
	}

	if opts.showDefaultColumn {
		table.Headers = append(table.Headers, "Default")
	}

	for _, p := range plans {
		var cpu, memory string
		if opts.isBytes {
			memory = fmt.Sprintf("%d", p.Memory)
		} else {
			memory = resource.NewQuantity(p.Memory, resource.BinarySI).String()
		}

		cpuMilli := p.CPUMilli

		override := p.Override
		if override != nil && override.CPUMilli != nil {
			cpuMilli = *p.Override.CPUMilli
			cpu = fmt.Sprintf("%g", float64(*p.Override.CPUMilli)/10) + "% (override)"
		} else if p.CPUMilli > 0 {
			cpu = fmt.Sprintf("%g", float64(p.CPUMilli)/10) + "%"
		}

		if p.Override != nil && p.Override.Memory != nil {
			memory = resource.NewQuantity(*p.Override.Memory, resource.BinarySI).String() + " (override)"
		}

		row := []string{
			p.Name,
			cpu,
			memory,
		}

		if showBurstColumn {
			var cpuBurst float64 = 1

			if p.CPUBurst != nil {
				cpuBurst = p.CPUBurst.Default
			}

			cpuBurstObservation := ""
			if p.Override != nil && p.Override.CPUBurst != nil {
				cpuBurst = *p.Override.CPUBurst
				cpuBurstObservation = " (override)"
			}

			row = append(row, displayCPUBurst(cpuMilli, cpuBurst)+cpuBurstObservation)
		}

		if showBurstColumn && opts.showMaxBurstAllowed {
			row = append(row, displayCPUBurst(cpuMilli, p.CPUBurst.MaxAllowed))
		}

		if opts.showDefaultColumn {
			row = append(row, strconv.FormatBool(p.Default))
		}
		table.AddRow(row)
	}
	return table.String()
}

func renderProcessPlan(appPlan apptypes.Plan, planByProcess map[string]string) string {
	table := tablecli.NewTable()
	table.Headers = []string{"Process", "Plan"}

	appProcessOverrides := []string{}
	override := appPlan.Override
	if override != nil {
		if override.CPUMilli != nil {
			appProcessOverrides = append(appProcessOverrides, fmt.Sprintf("CPU: %g%%", float64(*appPlan.Override.CPUMilli)/10))
		}

		if override.Memory != nil {
			memory := resource.NewQuantity(*appPlan.Override.Memory, resource.BinarySI).String()
			appProcessOverrides = append(appProcessOverrides, fmt.Sprintf("Memory: %s", memory))
		}
	}

	appRow := []string{
		"(default)",
		appPlan.Name,
	}

	if len(appProcessOverrides) > 0 {
		table.Headers = append(table.Headers, "Overrides")
		appRow = append(appRow, strings.Join(appProcessOverrides, ", "))
	}
	table.AddRow(appRow)

	processes := []string{}
	for process := range planByProcess {
		processes = append(processes, process)
	}

	sort.Strings(processes)

	for _, process := range processes {
		row := []string{
			process,
			planByProcess[process],
		}
		if len(appProcessOverrides) > 0 {
			row = append(row, "")
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
		maxAllowed := 1.0
		if p.CPUBurst != nil {
			maxAllowed = p.CPUBurst.MaxAllowed
		}
		maxCPULimit := resource.NewMilliQuantity(int64(float64(p.CPUMilli)*maxAllowed), resource.DecimalSI).String()

		row := []string{
			p.Name,
		}

		if showCPULimitsColumn {
			cpuBurst := p.GetCPUBurst()
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
	if p.CPUBurst != nil && p.CPUBurst.Default != 0 {
		return true
	}

	if p.Override != nil && p.Override.CPUBurst != nil {
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

func (c *PlanList) Run(context *cmd.Context) error {
	url, err := config.GetURL("/plans")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	var plans []apptypes.Plan
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
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
		fmt.Fprintf(context.Stdout, "%s", renderPlans(plans, renderPlansOpts{isBytes: c.bytes, showDefaultColumn: true, showMaxBurstAllowed: c.showMaxBurstAllowed}))
	}

	return nil
}
