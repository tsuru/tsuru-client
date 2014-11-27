// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
)

type autoScaleEnable struct {
	cmd.GuessingCommand
}

func (c *autoScaleEnable) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "autoscale-enable",
		Usage:   "autoscale-enable [-a/--app appname]",
		Desc:    "enable the app autoscale.",
		MinArgs: 1,
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
		Name:    "autoscale-disable",
		Usage:   "autoscale-disable [-a/--app appname]",
		Desc:    "disable the app autoscale.",
		MinArgs: 1,
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
