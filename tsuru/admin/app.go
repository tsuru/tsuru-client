// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"fmt"
	"net/http"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru-client/tsuru/app"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
)

type AppRoutesRebuild struct {
	app.AppNameMixIn
}

func (c *AppRoutesRebuild) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-routes-rebuild",
		MinArgs: 0,
		Usage:   "app-routes-rebuild -a <app-name>",
		Desc: `Rebuild routes for an application.
This can be used to recover from some failure in the router that caused
existing routes to be lost.`,
	}
}

func (c *AppRoutesRebuild) Run(ctx *cmd.Context) error {
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	url, err := config.GetURL("/apps/" + appName + "/routes")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	rsp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode == http.StatusOK {
		fmt.Fprintln(ctx.Stdout, "routes was rebuilt successfully")
	}

	return nil
}
