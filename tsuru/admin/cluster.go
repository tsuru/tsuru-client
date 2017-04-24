// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/ajg/form"
	"github.com/pkg/errors"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/provision/cluster"
)

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
		c.fs = gnuflag.NewFlagSet("with-flags", gnuflag.ContinueOnError)
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
		Desc:    `Creates or updates a provisioner cluster definition.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *ClusterUpdate) Run(context *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURLVersion("1.3", "/provisioner/clusters")
	if err != nil {
		return err
	}
	name := context.Args[0]
	provisioner := context.Args[1]
	clus := cluster.Cluster{
		Name:        name,
		Addresses:   c.addresses,
		Pools:       c.pools,
		CustomData:  c.customData,
		Default:     c.isDefault,
		Provisioner: provisioner,
	}
	var data []byte
	if c.cacert != "" {
		data, err = ioutil.ReadFile(c.cacert)
		if err != nil {
			return err
		}
		clus.CaCert = data
	}
	if c.clientcert != "" {
		data, err = ioutil.ReadFile(c.clientcert)
		if err != nil {
			return err
		}
		clus.ClientCert = data
	}
	if c.clientkey != "" {
		data, err = ioutil.ReadFile(c.clientkey)
		if err != nil {
			return err
		}
		clus.ClientKey = data
	}
	values, err := form.EncodeToValues(clus)
	if err != nil {
		return err
	}
	reader := strings.NewReader(values.Encode())
	request, err := http.NewRequest("POST", u, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Fprintln(context.Stdout, "Cluster successfully updated.")
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

func (c *ClusterList) Run(context *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURLVersion("1.3", "/provisioner/clusters")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent {
		fmt.Fprintln(context.Stdout, "No clusters registered.")
		return nil
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var clusters []cluster.Cluster
	err = json.Unmarshal(data, &clusters)
	if err != nil {
		return errors.Wrapf(err, "unable to parse data %q", string(data))
	}
	tbl := cmd.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = cmd.Row{"Name", "Provisioner", "Addresses", "Custom Data", "Default", "Pools"}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].Name < clusters[j].Name })
	for _, c := range clusters {
		var custom []string
		for k, v := range c.CustomData {
			custom = append(custom, fmt.Sprintf("%s=%s", k, v))
		}
		tbl.AddRow(cmd.Row{c.Name, c.Provisioner, strings.Join(c.Addresses, "\n"), strings.Join(custom, "\n"), strconv.FormatBool(c.Default), strings.Join(c.Pools, "\n")})
	}
	fmt.Fprint(context.Stdout, tbl.String())
	return nil
}

type ClusterRemove struct {
	cmd.ConfirmationCommand
}

func (c *ClusterRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "cluster-remove",
		Usage:   "cluster-remove <name>",
		Desc:    `Removes a provisioner cluster definition.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *ClusterRemove) Run(context *cmd.Context, client *cmd.Client) error {
	name := context.Args[0]
	u, err := cmd.GetURLVersion("1.3", "/provisioner/clusters/"+name)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Fprintln(context.Stdout, "Cluster successfully removed.")
	return nil
}
