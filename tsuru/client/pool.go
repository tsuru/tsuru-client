// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/mitchellh/go-wordwrap"
	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
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

type PoolList struct {
	fs         *pflag.FlagSet
	filter     poolFilter
	simplified bool
	json       bool
}

type Pool struct {
	Name        string
	Public      bool
	Default     bool
	Provisioner string
	Allowed     map[string][]string
}

func (p *Pool) Kind() string {
	if p.Public {
		return "public"
	}
	if p.Default {
		return "default"
	}
	return ""
}

func (p *Pool) GetProvisioner() string {
	if p.Provisioner == "" {
		return "default"
	}
	return p.Provisioner
}

type poolEntriesList []Pool

func (l poolEntriesList) Len() int      { return len(l) }
func (l poolEntriesList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l poolEntriesList) Less(i, j int) bool {
	cmp := strings.Compare(l[i].Kind(), l[j].Kind())
	if cmp == 0 {
		return l[i].Name < l[j].Name
	}
	return cmp < 0
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

func (pl *PoolList) Run(context *cmd.Context) error {
	url, err := config.GetURL("/pools")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	t := tablecli.Table{Headers: tablecli.Row([]string{"Pool", "Kind", "Provisioner", "Teams", "Routers"}), LineSeparator: true}
	if resp.StatusCode == http.StatusNoContent {
		context.Stdout.Write(t.Bytes())
		return nil
	}
	defer resp.Body.Close()
	var pools []Pool
	err = json.NewDecoder(resp.Body).Decode(&pools)
	if err != nil {
		return err
	}
	sort.Sort(poolEntriesList(pools))

	pools = pl.clientSideFilter(pools)

	if pl.simplified {
		for _, v := range pools {
			fmt.Fprintln(context.Stdout, v.Name)
		}
		return nil
	}

	if pl.json {
		return formatter.JSON(context.Stdout, pools)
	}

	for _, pool := range pools {
		teams := ""
		if !pool.Public && !pool.Default {
			teams = strings.Join(pool.Allowed["team"], ", ")
		}
		routers := strings.Join(pool.Allowed["router"], ", ")
		t.AddRow(tablecli.Row([]string{
			pool.Name,
			pool.Kind(),
			pool.GetProvisioner(),
			wordwrap.WrapString(teams, 30),
			wordwrap.WrapString(routers, 30),
		}))
	}
	context.Stdout.Write(t.Bytes())
	return nil
}

func (c *PoolList) clientSideFilter(pools []Pool) []Pool {
	result := make([]Pool, 0, len(pools))

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
		Name:    "pool-list",
		Usage:   "pool-list",
		Desc:    "List all pools available for deploy.",
		MinArgs: 0,
	}
}
