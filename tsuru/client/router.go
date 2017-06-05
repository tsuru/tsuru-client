// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/router"
)

type RoutersList struct{}

func (c *RoutersList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "router-list",
		Usage:   "router-list",
		Desc:    "List all routers available for app creation.",
		MinArgs: 0,
	}
}

func (c *RoutersList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURLVersion("1.3", "/routers")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	var routers []router.PlanRouter
	if response.StatusCode == http.StatusOK {
		err = json.NewDecoder(response.Body).Decode(&routers)
		if err != nil {
			return err
		}
	}
	table := cmd.NewTable()
	table.Headers = cmd.Row([]string{"Name", "Type"})
	table.LineSeparator = true
	for _, router := range routers {
		table.AddRow(cmd.Row([]string{router.Name, router.Type}))
	}
	context.Stdout.Write(table.Bytes())
	return nil
}
