// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
)

type ClusterAdd struct {
	fs         *gnuflag.FlagSet
	cacert     string
	clientcert string
	clientkey  string
	addresses  cmd.StringSliceFlag
	pools      cmd.StringSliceFlag
	customData cmd.MapFlag
	createData cmd.MapFlag
	isDefault  bool
}

func (c *ClusterAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ContinueOnError)
		desc := "Path to CA cert file."
		c.fs.StringVar(&c.cacert, "cacert", "", desc)
		desc = "Path to client cert file."
		c.fs.StringVar(&c.clientcert, "clientcert", "", desc)
		desc = "Path to client key file."
		c.fs.StringVar(&c.clientkey, "clientkey", "", desc)
		desc = "Whether this is the default cluster."
		c.fs.BoolVar(&c.isDefault, "default", false, desc)
		desc = "Address to be used in cluster."
		c.fs.Var(&c.addresses, "addr", desc)
		desc = "Pool which will use this cluster."
		c.fs.Var(&c.pools, "pool", desc)
		desc = "Custom provisioner specific data."
		c.fs.Var(&c.customData, "custom", desc)
		desc = "Create data, if set a iaas will be called with this data to create a new machine."
		c.fs.Var(&c.createData, "create-data", desc)
	}
	return c.fs
}

func (c *ClusterAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "cluster-add",
		Usage:   "cluster-add <name> <provisioner> [--addr address...] [--pool poolname]... [--cacert cacertfile] [--clientcert clientcertfile] [--clientkey clientkeyfile] [--custom key=value]... [--create-data key=value]... [--default]",
		Desc:    `Creates a provisioner cluster definition.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *ClusterAdd) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	name := ctx.Args[0]
	provisioner := ctx.Args[1]
	clus := tsuru.Cluster{
		Name:        name,
		Addresses:   c.addresses,
		Pools:       c.pools,
		CustomData:  c.customData,
		Default_:    c.isDefault,
		Provisioner: provisioner,
		CreateData:  c.createData,
	}
	var data []byte
	if c.cacert != "" {
		data, err = ioutil.ReadFile(c.cacert)
		if err != nil {
			return err
		}
		clus.Cacert = string(data)
	}
	if c.clientcert != "" {
		data, err = ioutil.ReadFile(c.clientcert)
		if err != nil {
			return err
		}
		clus.Clientcert = string(data)
	}
	if c.clientkey != "" {
		data, err = ioutil.ReadFile(c.clientkey)
		if err != nil {
			return err
		}
		clus.Clientkey = string(data)
	}
	response, err := apiClient.ClusterApi.ClusterCreate(context.TODO(), clus)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Fprintln(ctx.Stdout, "Cluster successfully added.")
	return nil
}

type ClusterUpdate struct {
	fs         *gnuflag.FlagSet
	cacert     string
	clientcert string
	clientkey  string
	addresses  cmd.StringSliceFlag
	pools      cmd.StringSliceFlag
	customData cmd.MapFlag
	isDefault  bool
}

func (c *ClusterUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ContinueOnError)
		desc := "Path to CA cert file."
		c.fs.StringVar(&c.cacert, "cacert", "", desc)
		desc = "Path to client cert file."
		c.fs.StringVar(&c.clientcert, "clientcert", "", desc)
		desc = "Path to client key file."
		c.fs.StringVar(&c.clientkey, "clientkey", "", desc)
		desc = "Whether this is the default cluster."
		c.fs.BoolVar(&c.isDefault, "default", false, desc)
		desc = "Address to be used in cluster."
		c.fs.Var(&c.addresses, "addr", desc)
		desc = "Pool which will use this cluster."
		c.fs.Var(&c.pools, "pool", desc)
		desc = "Custom provisioner specific data."
		c.fs.Var(&c.customData, "custom", desc)
	}
	return c.fs
}

func (c *ClusterUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "cluster-update",
		Usage:   "cluster-update <name> <provisioner> --addr address... [--pool poolname]... [--cacert cacertfile] [--clientcert clientcertfile] [--clientkey clientkeyfile] [--custom key=value]... [--default]",
		Desc:    `Updates a provisioner cluster definition.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *ClusterUpdate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	ctx.RawOutput()
	name := ctx.Args[0]
	provisioner := ctx.Args[1]
	clus := tsuru.Cluster{
		Name:        name,
		Addresses:   c.addresses,
		Pools:       c.pools,
		CustomData:  c.customData,
		Default_:    c.isDefault,
		Provisioner: provisioner,
	}
	var data []byte
	if c.cacert != "" {
		data, err = ioutil.ReadFile(c.cacert)
		if err != nil {
			return err
		}
		clus.Cacert = string(data)
	}
	if c.clientcert != "" {
		data, err = ioutil.ReadFile(c.clientcert)
		if err != nil {
			return err
		}
		clus.Clientcert = string(data)
	}
	if c.clientkey != "" {
		data, err = ioutil.ReadFile(c.clientkey)
		if err != nil {
			return err
		}
		clus.Clientkey = string(data)
	}
	resp, err := apiClient.ClusterApi.ClusterUpdate(context.TODO(), name, clus)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Fprintln(ctx.Stdout, "Cluster successfully updated.")
	return nil
}

type ClusterList struct{}

func (c *ClusterList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "cluster-list",
		Usage: "cluster-list",
		Desc:  `List registered provisioner cluster definitions.`,
	}
}

func (c *ClusterList) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	ctx.RawOutput()
	clusters, resp, err := apiClient.ClusterApi.ClusterList(context.TODO())
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNoContent {
		fmt.Fprintln(ctx.Stdout, "No clusters registered.")
		return nil
	}
	defer resp.Body.Close()

	tbl := tablecli.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = tablecli.Row{"Name", "Provisioner", "Addresses", "Custom Data", "Default", "Pools"}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].Name < clusters[j].Name })
	for _, c := range clusters {
		var custom []string
		for k, v := range c.CustomData {
			custom = append(custom, fmt.Sprintf("%s=%s", k, v))
		}
		tbl.AddRow(tablecli.Row{c.Name, c.Provisioner, strings.Join(c.Addresses, "\n"), strings.Join(custom, "\n"), strconv.FormatBool(c.Default_), strings.Join(c.Pools, "\n")})
	}
	fmt.Fprint(ctx.Stdout, tbl.String())
	return nil
}

type ClusterRemove struct {
	cmd.ConfirmationCommand
}

func (c *ClusterRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "cluster-remove",
		Usage:   "cluster-remove <name> [-y]",
		Desc:    `Removes a provisioner cluster definition.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *ClusterRemove) Run(ctx *cmd.Context, cli *cmd.Client) error {
	name := ctx.Args[0]

	if !c.Confirm(ctx, fmt.Sprintf("Are you sure you want to remove cluster \"%s\"?", name)) {
		return nil
	}

	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	ctx.RawOutput()
	response, err := apiClient.ClusterApi.ClusterDelete(context.TODO(), name)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Fprintln(ctx.Stdout, "Cluster successfully removed.")
	return nil
}
