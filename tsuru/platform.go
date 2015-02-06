// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/tsuru/tsuru/cmd"
)

type platform struct {
	Name string
}

type platformList struct{}

func (platformList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/platforms")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	var platforms []platform
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&platforms)
	if err != nil {
		return err
	}
	if len(platforms) == 0 {
		fmt.Fprintln(context.Stdout, "No platforms available.")
		return nil
	}
	platformNames := make([]string, len(platforms))
	for i, p := range platforms {
		platformNames[i] = p.Name
	}
	sort.Strings(platformNames)
	for _, p := range platformNames {
		fmt.Fprintf(context.Stdout, "- %s\n", p)
	}
	return nil
}

func (platformList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "platform-list",
		Usage:   "platform-list",
		Desc:    "Lists the available platforms. All platforms displayed in this list may be used to create new apps (see app-create).",
		MinArgs: 0,
	}
}
