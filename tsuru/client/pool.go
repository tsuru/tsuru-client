// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/mitchellh/go-wordwrap"
	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
)

type poolFilter struct {
	name string
	team string
}
type Pool tsuru.Pool

func (p Pool) Kind() string {
	return poolKind(tsuru.Pool(p))
}

func (p Pool) GetProvisioner() string {
	return poolProvisioner(tsuru.Pool(p))
}

type PoolList struct {
	fs         *pflag.FlagSet
	filter     poolFilter
	simplified bool
	json       bool
}

func poolKind(p tsuru.Pool) string {
	if p.Public {
		return "public"
	}
	if p.Default {
		return "default"
	}
	return ""
}

func poolProvisioner(p tsuru.Pool) string {
	if p.Provisioner == "" {
		return "default"
	}
	return p.Provisioner
}

func (c *PoolList) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)
		c.fs.StringVarP(&c.filter.name, standards.FlagName, standards.ShortFlagName, "", "Filter pools by name")
		c.fs.StringVarP(&c.filter.team, standards.FlagTeam, standards.ShortFlagTeam, "", "Filter pools by team ")

		c.fs.BoolVarP(&c.simplified, standards.FlagOnlyName, standards.ShortFlagOnlyName, false, "Display only pools name")
		c.fs.BoolVar(&c.json, standards.FlagJSON, false, "Display in JSON format")

	}
	return c.fs
}

func (pl *PoolList) Run(ctx *cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	pools, resp, err := apiClient.PoolApi.PoolList(context.TODO())
	t := tablecli.Table{Headers: tablecli.Row([]string{"Pool", "Kind", "Provisioner", "Teams", "Routers"}), LineSeparator: true}
	t.TableWriterTruncate = true
	if resp != nil && resp.StatusCode == http.StatusNoContent {
		if pl.json {
			fmt.Fprintln(ctx.Stdout, "[]")
			return nil
		}
		ctx.Stdout.Write(t.Bytes())
		return nil
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	sort.Slice(pools, func(i, j int) bool {
		cmp := strings.Compare(poolKind(pools[i]), poolKind(pools[j]))
		if cmp == 0 {
			return pools[i].Name < pools[j].Name
		}
		return cmp < 0
	})

	pools = pl.clientSideFilter(pools)

	if pl.simplified {
		for _, v := range pools {
			fmt.Fprintln(ctx.Stdout, v.Name)
		}
		return nil
	}

	if pl.json {
		return formatter.JSON(ctx.Stdout, pools)
	}

	for _, pool := range pools {
		teams := ""
		if !pool.Public && !pool.Default {
			teams = strings.Join(pool.Allowed["team"], ", ")
		}
		routers := strings.Join(pool.Allowed["router"], ", ")
		t.AddRow(tablecli.Row([]string{
			pool.Name,
			poolKind(pool),
			poolProvisioner(pool),
			wordwrap.WrapString(teams, 30),
			wordwrap.WrapString(routers, 30),
		}))
	}
	ctx.Stdout.Write(t.Bytes())
	return nil
}

func (c *PoolList) clientSideFilter(pools []tsuru.Pool) []tsuru.Pool {
	result := make([]tsuru.Pool, 0, len(pools))

	for _, pool := range pools {
		insert := true
		if c.filter.name != "" && !strings.Contains(pool.Name, c.filter.name) {
			insert = false
		}

		if c.filter.team != "" && !sliceContains(pool.Allowed["team"], c.filter.team) {
			insert = false
		}

		if insert {
			result = append(result, pool)
		}
	}

	return result
}
func sliceContains(s []string, d string) bool {
	for _, i := range s {
		if i == d {
			return true
		}
	}

	return false
}

func (PoolList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "pool-list",
		Usage: "pool-list",
		Desc:  "List all pools available for deploy.",
	}
}

type PoolInfo struct {
	fs   *pflag.FlagSet
	json bool
}

func (c *PoolInfo) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("pool-info", pflag.ExitOnError)
		c.fs.BoolVar(&c.json, standards.FlagJSON, false, "Display in JSON format")
	}
	return c.fs
}

func (c *PoolInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "pool-info",
		Usage:   "<pool>",
		Desc:    `Shows information about a specific pool.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *PoolInfo) Run(ctx *cmd.Context) error {
	poolName := ctx.Args[0]
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	pool, resp, err := apiClient.PoolApi.PoolGet(context.TODO(), poolName)
	if resp != nil && resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c.json {
		return formatter.JSON(ctx.Stdout, pool)
	}

	tabWriter := tabwriter.NewWriter(ctx.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(tabWriter, "Name:\t%s\n", pool.Name)

	kind := ""
	if pool.Public {
		kind = "public"
	} else if pool.Default {
		kind = "default"
	}
	if kind != "" {
		fmt.Fprintf(tabWriter, "Kind:\t%s\n", kind)
	}

	provisioner := pool.Provisioner
	if provisioner == "" {
		provisioner = "default"
	}
	fmt.Fprintf(tabWriter, "Provisioner:\t%s\n", provisioner)

	if len(pool.Teams) > 0 {
		fmt.Fprintf(tabWriter, "Teams:")

		for _, team := range pool.Teams {
			fmt.Fprintf(tabWriter, "\t%s\n", team)
		}
	}

	tabWriter.Flush()

	if len(pool.Allowed) > 0 {
		fmt.Fprintf(ctx.Stdout, "\nAllowed:\n")
		allowedTable := tablecli.NewTable()
		allowedTable.Headers = tablecli.Row{"Type", "Value"}
		allowedTable.LineSeparator = true
		allowedTable.TableWriterPadding = standards.SubTableWriterPadding
		var keys []string
		for k := range pool.Allowed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			values := pool.Allowed[k]
			allowedTable.AddRow(tablecli.Row{k, strings.Join(values, "\n")})
		}
		fmt.Fprint(ctx.Stdout, allowedTable.String())
	}

	if len(pool.Labels) > 0 {
		fmt.Fprintf(ctx.Stdout, "\nLabels:\n")
		labelsTable := tablecli.NewTable()
		labelsTable.Headers = tablecli.Row{"Key", "Value"}
		labelsTable.TableWriterPadding = standards.SubTableWriterPadding
		var keys []string
		for k := range pool.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			labelsTable.AddRow(tablecli.Row{k, pool.Labels[k]})
		}
		fmt.Fprint(ctx.Stdout, labelsTable.String())
	}

	return nil
}
