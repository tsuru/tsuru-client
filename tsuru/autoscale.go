// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tsuru/tsuru/cmd"
	"launchpad.net/gnuflag"
)

type autoScaleEnable struct {
	cmd.GuessingCommand
}

func (c *autoScaleEnable) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "autoscale-enable",
		Usage: "autoscale-enable [-a/--app appname]",
		Desc:  "enable the app autoscale.",
	}
}

func (c *autoScaleEnable) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/autoscale/%s/enable", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "Autoscale enabled!\n")
	return nil
}

type autoScaleDisable struct {
	cmd.GuessingCommand
}

func (c *autoScaleDisable) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "autoscale-disable",
		Usage: "autoscale-disable [-a/--app appname]",
		Desc:  "disable the app autoscale.",
	}
}

func (c *autoScaleDisable) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/autoscale/%s/disable", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "Autoscale disabled!\n")
	return nil
}

type autoScaleConfig struct {
	cmd.GuessingCommand
	fs                 *gnuflag.FlagSet
	maxUnits           int
	minUnits           int
	increaseStep       int
	increaseWaitTime   int
	increaseExpression string
	decreaseStep       int
	decreaseWaitTime   int
	decreaseExpression string
	enabled            bool
}

func (c *autoScaleConfig) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.IntVar(&c.maxUnits, "max-units", 10, "Maximum number of units.")
		c.fs.IntVar(&c.minUnits, "min-units", 1, "Minimum number of units.")
		c.fs.IntVar(&c.increaseStep, "increase-step", 1, "Number of units that will be increased on scale.")
		c.fs.IntVar(&c.increaseWaitTime, "increase-wait-time", 300, "Seconds before allowing another scaling activity.")
		c.fs.StringVar(&c.increaseExpression, "increase-expression", "{cpu_max} > 90", "Expression used to scale.")
		c.fs.StringVar(&c.decreaseExpression, "decrease-expression", "{cpu_max} < 10", "Expression used to scale.")
		c.fs.IntVar(&c.decreaseWaitTime, "decrease-wait-time", 300, "Seconds before allowing another scaling activity.")
		c.fs.IntVar(&c.decreaseStep, "decrease-step", 1, "Number of units that will be decrease on scale.")
		c.fs.BoolVar(&c.enabled, "enabled", false, "Enable auto scale.")
	}
	return c.fs
}

func (c *autoScaleConfig) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "autoscale-config",
		Usage: "autoscale-config [-a/--app appname] --max-units unitsnumber --min-units unitsnumber --increase-step unitsnumber --increase-wait-time seconds --increase-expression expression --decrease-wait-time seconds --decrease-expression expression --enabled",
		Desc:  "config app autoscale.",
	}
}

type Action struct {
	Wait       int
	Expression string
	Units      int
}

// AutoScaleConfig represents the App configuration for the auto scale.
type AutoScaleConfig struct {
	Increase Action
	Decrease Action
	MinUnits int
	MaxUnits int
	Enabled  bool
}

func (c *autoScaleConfig) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	config := AutoScaleConfig{
		MinUnits: c.minUnits,
		MaxUnits: c.maxUnits,
		Enabled:  c.enabled,
		Increase: Action{
			Wait:       int(time.Duration(c.increaseWaitTime) * time.Second),
			Expression: c.increaseExpression,
			Units:      c.increaseStep,
		},
		Decrease: Action{
			Wait:       int(time.Duration(c.decreaseWaitTime) * time.Second),
			Expression: c.decreaseExpression,
			Units:      c.decreaseStep,
		},
	}
	url, err := cmd.GetURL(fmt.Sprintf("/autoscale/%s", appName))
	if err != nil {
		return err
	}
	body, err := json.Marshal(&config)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	request.Header.Add("Content-Type", "application/json")
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "Autoscale successfully configured!\n")
	return nil
}
