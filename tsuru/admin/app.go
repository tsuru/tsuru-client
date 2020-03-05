// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/router/rebuild"
)

type AppLockDelete struct {
	cmd.GuessingCommand
	cmd.ConfirmationCommand
	fs *gnuflag.FlagSet
}

func (c *AppLockDelete) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-unlock",
		MinArgs: 0,
		Usage:   "app-unlock -a <app-name> [-y]",
		Desc: `Forces the removal of an application lock.
Use with caution, removing an active lock may cause inconsistencies.`,
	}
}

func (c *AppLockDelete) Run(ctx *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	if !c.Confirm(ctx, fmt.Sprintf(`Are you sure you want to remove the lock from app "%s"?`, appName)) {
		return nil
	}
	url, err := cmd.GetURL("/apps/" + appName + "/lock")
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Lock successfully removed!\n")
	return nil
}

func (c *AppLockDelete) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = cmd.MergeFlagSet(
			c.GuessingCommand.Flags(),
			c.ConfirmationCommand.Flags(),
		)
	}
	return c.fs
}

type AppRoutesRebuild struct {
	cmd.GuessingCommand
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

type compatibleRebuildResult struct {
	rebuild.RebuildRoutesResult
	rebuild.RebuildPrefixResult
}

func (c *AppRoutesRebuild) Run(ctx *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL("/apps/" + appName + "/routes")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	rsp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	var allRebuildResult map[string]compatibleRebuildResult
	err = json.NewDecoder(rsp.Body).Decode(&allRebuildResult)
	if err != nil {
		return err
	}
	if len(allRebuildResult) == 0 {
		fmt.Fprintf(ctx.Stdout, "App has no routers.\n")
		return nil
	}
	var routerNames []string
	for routerName := range allRebuildResult {
		routerNames = append(routerNames, routerName)
	}
	sort.Strings(routerNames)
	for _, routerName := range routerNames {
		rebuildResult := allRebuildResult[routerName]
		fmt.Fprintf(ctx.Stdout, "Router %v:\n", routerName)
		if len(rebuildResult.PrefixResults) == 0 {
			printRouterResult(ctx.Stdout, rebuildResult.Added, rebuildResult.Removed)
		}
		for _, prefixResult := range rebuildResult.PrefixResults {
			fmt.Fprintf(ctx.Stdout, " - Prefix %q:\n", prefixResult.Prefix)
			printRouterResult(ctx.Stdout, prefixResult.Added, prefixResult.Removed)
		}
	}
	return nil
}

func printRouterResult(w io.Writer, added, removed []string) {
	rebuilt := len(added) > 0 || len(removed) > 0
	if len(added) > 0 {
		fmt.Fprintf(w, "  * Added routes:\n")
		for _, added := range added {
			fmt.Fprintf(w, "    - %s\n", added)
		}
	}
	if len(removed) > 0 {
		fmt.Fprintf(w, "  * Removed routes:\n")
		for _, removed := range removed {
			fmt.Fprintf(w, "    - %s\n", removed)
		}
	}
	if !rebuilt {
		fmt.Fprintf(w, "  * Nothing to do, routes already correct.\n")
	}
}
