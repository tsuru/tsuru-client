// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/net"
	"github.com/tsuru/tsuru/provision"
)

type AddNodeCmd struct {
	fs       *gnuflag.FlagSet
	register bool
}

func (AddNodeCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-add",
		Usage: "node-add [param_name=param_value]... [--register]",
		Desc: `Creates or registers a new node in the cluster.
By default, this command will call the configured IaaS to create a new
machine. Every param will be sent to the IaaS implementation.

IaaS providers should have been previously configured in the [[tsuru.conf]]
file. See tsuru.conf reference docs for more information.

If using an IaaS to create a node is not wanted it's possible to simply
register an existing node with the [[--register]] flag.

Parameters with special meaning:
  iaas=<iaas name>
    Which iaas provider should be used, if not set tsuru will use the default
    iaas specified in tsuru.conf file.

  template=<template name>
    A machine template with predefined parameters, additional parameters will
    override template ones. See 'machine-template-add' command.

  address=<api url>
    Only used if [[--register]] flag is used. Should point to the endpoint of
    a working server.

  pool=<pool name>
    Mandatory parameter specifying to which pool the added node will belong.
    Available pools can be lister with the [[pool-list]] command.
`,
		MinArgs: 1,
	}
}

func (a *AddNodeCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	opts := provision.AddNodeOptions{
		Register: a.register,
		Metadata: map[string]string{},
	}
	for _, param := range ctx.Args {
		if strings.Contains(param, "=") {
			keyValue := strings.SplitN(param, "=", 2)
			opts.Metadata[keyValue[0]] = keyValue[1]
		}
	}
	v, err := form.EncodeToValues(&opts)
	if err != nil {
		return err
	}
	u, err := cmd.GetURLVersion("1.2", "/node")
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", u, bytes.NewBufferString(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	err = cmd.StreamJSONResponse(ctx.Stdout, resp)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte("Node successfully registered.\n"))
	return nil
}

func (a *AddNodeCmd) Flags() *gnuflag.FlagSet {
	if a.fs == nil {
		a.fs = gnuflag.NewFlagSet("with-flags", gnuflag.ContinueOnError)
		a.fs.BoolVar(&a.register, "register", false, "Registers an existing docker endpoint, the IaaS won't be called.")
	}
	return a.fs
}

type UpdateNodeCmd struct {
	fs      *gnuflag.FlagSet
	disable bool
	enable  bool
}

func (UpdateNodeCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-update",
		Usage: "node-update <address> [param_name=param_value...] [--disable] [--enable]",
		Desc: `Modifies metadata associated to a node. If a parameter is set to an
empty value, it will be removed from the node's metadata.

If the [[--disable]] flag is used, the node will be marked as disabled and the
scheduler won't consider it when selecting a node to receive containers.`,
		MinArgs: 1,
	}
}

func (a *UpdateNodeCmd) Flags() *gnuflag.FlagSet {
	if a.fs == nil {
		a.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		a.fs.BoolVar(&a.disable, "disable", false, "Disable node in scheduler.")
		a.fs.BoolVar(&a.enable, "enable", false, "Enable node in scheduler.")
	}
	return a.fs
}

func (a *UpdateNodeCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	opts := provision.UpdateNodeOptions{
		Address:  ctx.Args[0],
		Disable:  a.disable,
		Enable:   a.enable,
		Metadata: map[string]string{},
	}
	for _, param := range ctx.Args[1:] {
		if strings.Contains(param, "=") {
			keyValue := strings.SplitN(param, "=", 2)
			opts.Metadata[keyValue[0]] = keyValue[1]
		}
	}
	u, err := cmd.GetURLVersion("1.2", "/node")
	if err != nil {
		return err
	}
	v, err := form.EncodeToValues(&opts)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", u, bytes.NewBufferString(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte("Node successfully updated.\n"))
	return nil
}

type RemoveNodeCmd struct {
	cmd.ConfirmationCommand
	fs          *gnuflag.FlagSet
	destroy     bool
	noRebalance bool
}

func (RemoveNodeCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-remove",
		Usage: "node-remove <address> [--no-rebalance] [--destroy] [-y]",
		Desc: `Removes a node from the cluster.

By default tsuru will redistribute all containers present on the removed node
among other nodes. This behavior can be inhibited using the [[--no-rebalance]]
flag.

If the node being removed was created using a IaaS provider tsuru will NOT
destroy the machine on the IaaS, unless the [[--destroy]] flag is used.`,
		MinArgs: 1,
	}
}

func (c *RemoveNodeCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	msg := "Are you sure you sure you want to remove \"%s\" from cluster"
	if c.destroy {
		msg += " and DESTROY the machine from IaaS"
	}
	address := ctx.Args[0]
	if !c.Confirm(ctx, fmt.Sprintf(msg+"?", address)) {
		return nil
	}
	v := url.Values{}
	if c.destroy {
		v.Set("remove-iaas", "true")
	}
	v.Set("no-rebalance", strconv.FormatBool(c.noRebalance))
	u, err := cmd.GetURLVersion("1.2", fmt.Sprintf("/node/%s?%s", address, v.Encode()))
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte("Node successfully removed.\n"))
	return nil
}

func (c *RemoveNodeCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.ConfirmationCommand.Flags()
		c.fs.BoolVar(&c.destroy, "destroy", false, "Destroy node from IaaS")
		c.fs.BoolVar(&c.noRebalance, "no-rebalance", false, "Do not rebalance containers from removed node.")
	}
	return c.fs
}

type ListNodesCmd struct {
	fs         *gnuflag.FlagSet
	filter     cmd.MapFlag
	simplified bool
}

func (c *ListNodesCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-list",
		Usage: "node-list [--filter/-f <metadata>=<value>]...",
		Desc: `Lists nodes in the cluster. It will also show you metadata associated to each
node and the IaaS ID if the node was added using tsuru IaaS providers.

Using the [[-f/--filter]] flag, the user is able to filter the nodes that
appear in the list based on the key pairs displayed in the metadata column.
Users can also combine filters using [[-f]] multiple times.`,
		MinArgs: 0,
	}
}

func (c *ListNodesCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("with-flags", gnuflag.ContinueOnError)
		filter := "Filter by metadata name and value"
		c.fs.Var(&c.filter, "filter", filter)
		c.fs.Var(&c.filter, "f", filter)
		c.fs.BoolVar(&c.simplified, "q", false, "Display only nodes IP address")
	}
	return c.fs
}

func (c *ListNodesCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURLVersion("1.2", "/node")
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	t := cmd.Table{Headers: cmd.Row([]string{"Address", "IaaS ID", "Status", "Metadata"}), LineSeparator: true}
	if resp.StatusCode == http.StatusNoContent {
		ctx.Stdout.Write(t.Bytes())
		return nil
	}
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}
	machineMap := map[string]map[string]interface{}{}
	if result["machines"] != nil {
		machines := result["machines"].([]interface{})
		for _, m := range machines {
			machine := m.(map[string]interface{})
			machineMap[machine["Address"].(string)] = m.(map[string]interface{})
		}
	}
	var nodes []map[string]interface{}
	if result["nodes"] != nil {
		nodes = c.filterNodes(result["nodes"].([]interface{}))
	}
	if c.simplified {
		for _, node := range nodes {
			fmt.Fprintln(ctx.Stdout, node["Address"].(string))
		}
		return nil
	}
	for _, node := range nodes {
		addr := node["Address"].(string)
		status := node["Status"].(string)
		result := []string{}
		metadataField, _ := node["Metadata"]
		if metadataField != nil {
			metadata := metadataField.(map[string]interface{})
			for key, value := range metadata {
				result = append(result, fmt.Sprintf("%s=%s", key, value.(string)))
			}
		}
		sort.Strings(result)
		m, ok := machineMap[net.URLToHost(addr)]
		var iaasId string
		if ok {
			iaasId = m["Id"].(string)
		}
		t.AddRow(cmd.Row([]string{addr, iaasId, status, strings.Join(result, "\n")}))
	}
	t.Sort()
	ctx.Stdout.Write(t.Bytes())
	return nil
}

func (c *ListNodesCmd) filterNodes(nodes []interface{}) []map[string]interface{} {
	filteredNodes := make([]map[string]interface{}, 0)
	for _, n := range nodes {
		node := n.(map[string]interface{})
		if c.nodeMetadataMatchesFilters(node) {
			filteredNodes = append(filteredNodes, node)
		}
	}
	return filteredNodes
}

func (c *ListNodesCmd) nodeMetadataMatchesFilters(node map[string]interface{}) bool {
	metadataField, _ := node["Metadata"]
	if c.filter != nil && metadataField == nil {
		return false
	}
	if metadataField != nil {
		metadata := metadataField.(map[string]interface{})
		for key, value := range c.filter {
			metaVal, _ := metadata[key]
			if metaVal != value {
				return false
			}
		}
	}
	return true
}
