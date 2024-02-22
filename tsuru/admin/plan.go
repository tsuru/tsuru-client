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
	"github.com/tsuru/tsuru-client/tsuru/config"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	"k8s.io/apimachinery/pkg/api/resource"
)

type PlanCreate struct {
	memory     string
	cpu        string
	setDefault bool
	fs         *gnuflag.FlagSet
}

func (c *PlanCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("plan-create", gnuflag.ExitOnError)
		memory := `Amount of available memory for units in bytes or an integer value followed
by M, K or G for megabytes, kilobytes or gigabytes respectively.`
		c.fs.StringVar(&c.memory, "memory", "0", memory)
		c.fs.StringVar(&c.memory, "m", "0", memory)

		cpu := `Relative cpu each unit will have available.`
		c.fs.StringVar(&c.cpu, "cpu", "0", cpu)
		c.fs.StringVar(&c.cpu, "c", "0", cpu)
		setDefault := `Set plan as default, this will remove the default flag from any other plan.
The default plan will be used when creating an application without explicitly
setting a plan.`
		c.fs.BoolVar(&c.setDefault, "default", false, setDefault)
		c.fs.BoolVar(&c.setDefault, "d", false, setDefault)
	}
	return c.fs
}

func (c *PlanCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plan-create",
		Usage:   "plan create <name> -c cpu [-m memory] [--default]",
		Desc:    `Creates a new plan for being used when creating apps.`,
		MinArgs: 1,
	}
}

func (c *PlanCreate) Run(context *cmd.Context) error {
	u, err := config.GetURL("/plans")
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("name", context.Args[0])

	memoryValue, err := parseMemoryQuantity(c.memory)
	if err != nil {
		return err
	}
	v.Set("memory", fmt.Sprintf("%d", memoryValue))

	cpuValue, err := parseCPUQuantity(c.cpu)
	if err != nil {
		return err
	}
	v.Set("cpumilli", fmt.Sprintf("%d", cpuValue))

	v.Set("default", strconv.FormatBool(c.setDefault))
	b := strings.NewReader(v.Encode())
	request, err := http.NewRequest("POST", u, b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = tsuruHTTP.DefaultClient.Do(request)
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
		Usage: "plan remove <name>",
		Desc: `Removes an existing plan. It will no longer be available for newly created
apps. However, this won't change anything for existing apps that were created
using the removed plan. They will keep using the same value amount of
resources described by the plan.`,
		MinArgs: 1,
	}
}

func (c *PlanRemove) Run(context *cmd.Context) error {
	url, err := config.GetURL("/plans/" + context.Args[0])
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = tsuruHTTP.DefaultClient.Do(request)
	if err != nil {
		fmt.Fprintf(context.Stdout, "Failed to remove plan!\n")
		return err
	}
	fmt.Fprintf(context.Stdout, "Plan successfully removed!\n")
	return nil
}

func parseMemoryQuantity(userQuantity string) (numBytes int64, err error) {
	if v, parseErr := strconv.Atoi(userQuantity); parseErr == nil {
		return int64(v), nil
	}
	memoryQuantity, err := resource.ParseQuantity(userQuantity)
	if err != nil {
		return 0, err
	}

	numBytes, _ = memoryQuantity.AsInt64()
	return numBytes, nil
}

func parseCPUQuantity(userQuantity string) (numMillis int64, err error) {
	var v int
	if v, err = strconv.Atoi(userQuantity); err == nil {
		return int64(v) * 1000, nil
	}

	if strings.HasSuffix(userQuantity, "%") {
		v, err = strconv.Atoi(userQuantity[0 : len(userQuantity)-1])
		if err != nil {
			return 0, err
		}
		return int64(v) * 10, nil
	}

	var cpuQuantity resource.Quantity
	cpuQuantity, err = resource.ParseQuantity(userQuantity)

	if err != nil {
		return 0, err
	}

	cpu := cpuQuantity.AsApproximateFloat64()

	return int64(cpu * 1000), nil
}
