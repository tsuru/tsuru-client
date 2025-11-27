// Copyright 2020 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	tsuruClientApp "github.com/tsuru/tsuru-client/tsuru/app"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
)

type int32Value int32

func (i *int32Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 32)
	*i = int32Value(v)
	return err
}
func (i *int32Value) Get() interface{} { return int32(*i) }
func (i *int32Value) String() string   { return fmt.Sprintf("%v", *i) }

type int32PointerValue struct {
	value **int32
}

func (i *int32PointerValue) Set(s string) error {
	if s == "" {
		*i.value = nil
		return nil
	}
	v, err := strconv.ParseInt(s, 0, 32)
	v32 := int32(v)

	*i.value = &v32
	return err
}
func (i *int32PointerValue) Get() interface{} { return *i.value }
func (i *int32PointerValue) String() string {
	if *i.value == nil {
		return ""
	}
	return fmt.Sprintf("%v", *i.value)
}

type AutoScaleSet struct {
	tsuruClientApp.AppNameMixIn
	fs         *gnuflag.FlagSet
	autoscale  tsuru.AutoScaleSpec
	schedules  cmd.StringSliceFlag
	prometheus cmd.StringSliceFlag
}

func (c *AutoScaleSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-autoscale-set",
		Usage: "unit autoscale set [-a/--app appname] [-p/--process processname] [--cpu targetCPU] [--min minUnits] [--max maxUnits] [--schedule scheduleWindow] [--prometheus prometheusSettings]",
		Desc: `
# Sets an autoscale configuration:
# Based on 50% of CPU utilization with min units 1 and max units 3
unit autoscale set -a my-app --cpu 50% --min 1 --max 3

# Based on a schedule window everyday from 6AM to 6PM UTC
unit autoscale set -a my-app --min 1 --max 3 --schedule '{"minReplicas": 2, "start": "0 6 * * *", "end": "0 18 * * *"}'

# Based on a prometheus metric

unit autoscale set -a my-app --min 1 --max 3 --prometheus '{"name": "my_metric_identification", "threshold": 10, "query":"sum(my_metric{tsuru_app=\"my_app\"})"}'

# Combining
unit autoscale set -a my-app --cpu 50% --min 1 --max 3 --schedule '{"minReplicas": 2, "start": "0 6 * * *", "end": "0 18 * * *"}'

# When using more than one trigger (CPU + Schedule as an example), the number of units will be determined by the highest value
`,
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *AutoScaleSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()

		c.fs.StringVar(&c.autoscale.Process, "process", "", "Process name")
		c.fs.StringVar(&c.autoscale.Process, "p", "", "Process name")

		c.fs.StringVar(&c.autoscale.AverageCPU, "cpu", "", "Target CPU value in percent of app cpu plan. Example: 50%")

		c.autoscale.MinUnits = 1
		c.fs.Var((*int32Value)(&c.autoscale.MinUnits), "min", "Minimum Units")

		c.fs.Var((*int32Value)(&c.autoscale.MaxUnits), "max", "Maximum Units")

		c.fs.Var(&c.schedules, "schedule", "Schedule window to up/down scale. Example: {\"minReplicas\": 2, \"start\": \"0 6 * * *\", \"end\": \"0 18 * * *\"}")
		c.fs.Var(&c.prometheus, "prometheus", "Prometheus settings to up/down scale. Example: {\"name\": \"my_metric_identification\", \"threshold\": 10, \"query\":\"sum(my_metric{tsuru_app=\\\"my_app\\\"})\"}")

		c.fs.Var(&int32PointerValue{&c.autoscale.Behavior.ScaleDown.PercentagePolicyValue}, "scale-down-percentage", "Percentage of units to downscale when the metric is below the threshold")
		c.fs.Var(&int32PointerValue{&c.autoscale.Behavior.ScaleDown.PercentagePolicyValue}, "sdp", "Percentage of units to downscale when the metric is below the threshold")

		c.fs.Var(&int32PointerValue{&c.autoscale.Behavior.ScaleDown.StabilizationWindow}, "scale-down-stabilization-window", "Stabilization window in seconds to avoid scale down")
		c.fs.Var(&int32PointerValue{&c.autoscale.Behavior.ScaleDown.StabilizationWindow}, "sdsw", "Stabilization window in seconds to avoid scale down")

		c.fs.Var(&int32PointerValue{&c.autoscale.Behavior.ScaleDown.UnitsPolicyValue}, "scale-down-units", "Number of units to downscale when the metric is below the threshold")
		c.fs.Var(&int32PointerValue{&c.autoscale.Behavior.ScaleDown.UnitsPolicyValue}, "sdu", "Number of units to downscale when the metric is below the threshold")
	}
	return c.fs
}

func (c *AutoScaleSet) Run(ctx *cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	schedules := []tsuru.AutoScaleSchedule{}
	for _, scheduleString := range c.schedules {
		var autoScaleSchedule tsuru.AutoScaleSchedule
		if err = json.Unmarshal([]byte(scheduleString), &autoScaleSchedule); err != nil {
			return err
		}
		schedules = append(schedules, autoScaleSchedule)
	}
	c.autoscale.Schedules = schedules
	prometheus := []tsuru.AutoScalePrometheus{}
	for _, prometheusString := range c.prometheus {
		var autoScalePrometheus tsuru.AutoScalePrometheus
		if err = json.Unmarshal([]byte(prometheusString), &autoScalePrometheus); err != nil {
			return err
		}
		prometheus = append(prometheus, autoScalePrometheus)
	}
	c.autoscale.Prometheus = prometheus
	_, err = apiClient.AppApi.AutoScaleAdd(context.TODO(), appName, c.autoscale)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Unit auto scale successfully set.")
	return nil
}

type AutoScaleUnset struct {
	tsuruClientApp.AppNameMixIn
	fs      *gnuflag.FlagSet
	process string
}

func (c *AutoScaleUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "unit-autoscale-unset",
		Usage:   "unit autoscale unset [-a/--app appname] [-p/--process processname]",
		Desc:    `Unsets a unit auto scale configuration.`,
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *AutoScaleUnset) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

func (c *AutoScaleUnset) Run(ctx *cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	_, err = apiClient.AppApi.AutoScaleRemove(context.TODO(), appName, c.process)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Unit auto scale successfully unset.")
	return nil
}

type AutoScaleSwap struct {
	tsuruClientApp.AppNameMixIn
	fs      *gnuflag.FlagSet
	version string
}

func (c *AutoScaleSwap) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "unit-autoscale-swap",
		Usage:   "unit autoscale swap [-a/--app appname] [--version version]",
		Desc:    `Swap a unit auto scale configuration to another version.`,
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *AutoScaleSwap) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

func (c *AutoScaleSwap) Run(ctx *cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}

	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}

	if c.version == "" {
		return errors.Errorf(`The version is required.

Use the --version flag to specify it.

`)
	}

	swapAutoScaleSpec := tsuru.SwapAutoScaleSpec{Version: c.version}
	_, err = apiClient.AppApi.AutoScaleSwap(context.TODO(), appName, swapAutoScaleSpec)
	if err != nil {
		return err
	}

	fmt.Fprintln(ctx.Stdout, "Unit auto scale successfully swapped.")
	return nil
}
