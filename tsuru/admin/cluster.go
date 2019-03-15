// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
	ctx.RawOutput()
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
		Default:     c.isDefault,
		Provisioner: provisioner,
		CreateData:  c.createData,
	}
	var data []byte
	if c.cacert != "" {
		data, err = ioutil.ReadFile(c.cacert)
		if err != nil {
			return err
		}
		clus.Cacert = data
	}
	if c.clientcert != "" {
		data, err = ioutil.ReadFile(c.clientcert)
		if err != nil {
			return err
		}
		clus.Clientcert = data
	}
	if c.clientkey != "" {
		data, err = ioutil.ReadFile(c.clientkey)
		if err != nil {
			return err
		}
		clus.Clientkey = data
	}
	response, err := apiClient.ClusterApi.ClusterCreate(context.TODO(), clus)
	if err != nil {
		return err
	}
	err = optionalStreamResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
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
	createData cmd.MapFlag
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
		desc = "Create data, if set a iaas will be called with this data to re-create the machine."
		c.fs.Var(&c.createData, "create-data", desc)
	}
	return c.fs
}

func (c *ClusterUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "cluster-update",
		Usage:   "cluster-update <name> <provisioner> --addr address... [--pool poolname]... [--cacert cacertfile] [--clientcert clientcertfile] [--clientkey clientkeyfile] [--custom key=value]... [--create-data key=value]... [--default]",
		Desc:    `Updates a provisioner cluster definition.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *ClusterUpdate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()
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
		Default:     c.isDefault,
		Provisioner: provisioner,
		CreateData:  c.createData,
	}
	var data []byte
	if c.cacert != "" {
		data, err = ioutil.ReadFile(c.cacert)
		if err != nil {
			return err
		}
		clus.Cacert = data
	}
	if c.clientcert != "" {
		data, err = ioutil.ReadFile(c.clientcert)
		if err != nil {
			return err
		}
		clus.Clientcert = data
	}
	if c.clientkey != "" {
		data, err = ioutil.ReadFile(c.clientkey)
		if err != nil {
			return err
		}
		clus.Clientkey = data
	}
	resp, err := apiClient.ClusterApi.ClusterUpdate(context.TODO(), name, clus)
	if err != nil {
		return err
	}
	err = optionalStreamResponse(ctx.Stdout, resp)
	if err != nil {
		return err
	}
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
	clusters, resp, err := apiClient.ClusterApi.ClusterList(context.TODO())
	if resp != nil && resp.StatusCode == http.StatusNoContent {
		fmt.Fprintln(ctx.Stdout, "No clusters registered.")
		return nil
	}
	if err != nil {
		return err
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
		tbl.AddRow(tablecli.Row{c.Name, c.Provisioner, strings.Join(c.Addresses, "\n"), strings.Join(custom, "\n"), strconv.FormatBool(c.Default), strings.Join(c.Pools, "\n")})
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
	ctx.RawOutput()
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
	response, err := apiClient.ClusterApi.ClusterDelete(context.TODO(), name)
	if err != nil {
		return err
	}
	err = optionalStreamResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Cluster successfully removed.")
	return nil
}

type ProvisionerList struct{}

func (c *ProvisionerList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "provisioner-list",
		Usage: "provisioner-list",
		Desc:  `List registered provisioners and their cluster options.`,
	}
}

func (c *ProvisionerList) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	provisioners, resp, err := apiClient.ClusterApi.ProvisionerList(context.TODO())
	if resp != nil && resp.StatusCode == http.StatusNoContent {
		fmt.Fprintln(ctx.Stdout, "No provisioners registered.")
		return nil
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	tbl := tablecli.NewTable()
	tbl.Headers = tablecli.Row{"Name", "Cluster Usage"}
	sort.Slice(provisioners, func(i, j int) bool {
		return provisioners[i].Name < provisioners[j].Name
	})
	for _, p := range provisioners {
		tbl.AddRow(tablecli.Row{p.Name, p.ClusterHelp.ProvisionerHelp})
	}
	fmt.Fprint(ctx.Stdout, tbl.String())
	return nil
}

type ProvisionerInfo struct{}

func (c *ProvisionerInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "provisioner-info",
		Usage:   "provisioner-info <provisioner name>",
		Desc:    `Detailed information about provisioner.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *ProvisionerInfo) Run(ctx *cmd.Context, cli *cmd.Client) error {
	provisionerName := ctx.Args[0]
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	provisioners, resp, err := apiClient.ClusterApi.ProvisionerList(context.TODO())
	if resp != nil && resp.StatusCode == http.StatusNoContent {
		return errors.New("provisioner not found")
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var provisioner *tsuru.Provisioner
	for _, p := range provisioners {
		if p.Name == provisionerName {
			provisioner = &p
			break
		}
	}
	if provisioner == nil {
		return errors.New("provisioner not found")
	}
	fmt.Fprintf(ctx.Stdout, "Name: %v\n", provisioner.Name)
	fmt.Fprintf(ctx.Stdout, "Cluster usage: %v\n", provisioner.ClusterHelp.ProvisionerHelp)
	fmt.Fprintf(ctx.Stdout, "\nCustom Data:\n")
	tbl := tablecli.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = tablecli.Row{"Name", "Usage"}
	for key, value := range provisioner.ClusterHelp.CustomDataHelp {
		tbl.AddRow(tablecli.Row{key, value})
	}
	tbl.Sort()
	fmt.Fprint(ctx.Stdout, tbl.String())

	fmt.Fprintf(ctx.Stdout, "\nCreate Data:\n")
	tbl = tablecli.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = tablecli.Row{"Name", "Usage"}
	for key, value := range provisioner.ClusterHelp.CreateDataHelp {
		tbl.AddRow(tablecli.Row{key, value})
	}
	tbl.Sort()
	fmt.Fprint(ctx.Stdout, tbl.String())
	return nil
}

func optionalStreamResponse(w io.Writer, resp *http.Response) error {
	if resp.Header.Get("Content-Type") == "application/x-json-stream" {
		return cmd.StreamJSONResponse(w, resp)
	}
	return nil
}
