// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tablecli"
	tsuruClient "github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/healer"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/net"
	"github.com/tsuru/tsuru/provision"
	apiTypes "github.com/tsuru/tsuru/types/api"
)

type AddNodeCmd struct {
	fs         *gnuflag.FlagSet
	register   bool
	caCert     string
	clientCert string
	clientKey  string
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
    Available pools can be listed with the [[pool-list]] command.
`,
		MinArgs: 1,
	}
}

func (a *AddNodeCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	opts := provision.AddNodeOptions{
		Register:   a.register,
		CaCert:     []byte(a.caCert),
		ClientCert: []byte(a.clientCert),
		ClientKey:  []byte(a.clientKey),
		Metadata:   map[string]string{},
	}
	var err error
	if a.caCert != "" {
		opts.CaCert, err = ioutil.ReadFile(a.caCert)
		if err != nil {
			return err
		}
	}
	if a.clientCert != "" {
		opts.ClientCert, err = ioutil.ReadFile(a.clientCert)
		if err != nil {
			return err
		}
	}
	if a.clientKey != "" {
		opts.ClientKey, err = ioutil.ReadFile(a.clientKey)
		if err != nil {
			return err
		}
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
		a.fs = gnuflag.NewFlagSet("", gnuflag.ContinueOnError)
		a.fs.BoolVar(&a.register, "register", false, "Registers an existing docker endpoint, the IaaS won't be called.")
		a.fs.StringVar(&a.caCert, "cacert", "", "Path to CA file tsuru should trust.")
		a.fs.StringVar(&a.clientCert, "clientcert", "", "Path to client TLS certificate file.")
		a.fs.StringVar(&a.clientKey, "clientkey", "", "Path to client TLS key file.")
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
	req, err := http.NewRequest(http.MethodDelete, u, nil)
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
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
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
	t := tablecli.Table{Headers: tablecli.Row([]string{"Address", "IaaS ID", "Status", "Metadata"}), LineSeparator: true}
	if resp.StatusCode == http.StatusNoContent {
		ctx.Stdout.Write(t.Bytes())
		return nil
	}
	var result apiTypes.ListNodeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}
	machineMap := map[string]iaas.Machine{}
	if len(result.Machines) > 0 {
		for _, m := range result.Machines {
			machineMap[m.Address] = m
		}
	}
	var nodes []provision.NodeSpec
	if len(result.Nodes) > 0 {
		nodes = c.filterNodes(result.Nodes)
	}
	if c.simplified {
		for _, node := range nodes {
			fmt.Fprintln(ctx.Stdout, node.Address)
		}
		return nil
	}
	for _, node := range nodes {
		addr := node.Address
		status := node.Status
		result := []string{}
		for key, value := range node.Metadata {
			result = append(result, fmt.Sprintf("%s=%s", key, value))
		}
		sort.Strings(result)
		m, ok := machineMap[net.URLToHost(addr)]
		var iaasID string
		if ok {
			iaasID = m.Id
		}
		t.AddRow(tablecli.Row([]string{addr, iaasID, status, strings.Join(result, "\n")}))
	}
	t.Sort()
	ctx.Stdout.Write(t.Bytes())
	return nil
}

func (c *ListNodesCmd) filterNodes(nodes []provision.NodeSpec) []provision.NodeSpec {
	filteredNodes := make([]provision.NodeSpec, 0)
	for _, n := range nodes {
		if c.nodeMetadataMatchesFilters(n) {
			filteredNodes = append(filteredNodes, n)
		}
	}
	return filteredNodes
}

func (c *ListNodesCmd) nodeMetadataMatchesFilters(node provision.NodeSpec) bool {
	for key, value := range c.filter {
		if key == provision.PoolMetadataName {
			if value != node.Pool {
				return false
			}
			continue
		}
		metaVal := node.Metadata[key]
		if metaVal != value {
			return false
		}
	}
	return true
}

type GetNodeHealingConfigCmd struct{}

func (c *GetNodeHealingConfigCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-healing-info",
		Usage: "node-healing-info",
		Desc:  "Show the current configuration for active healing nodes.",
	}
}

func (c *GetNodeHealingConfigCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURLVersion("1.2", "/healing/node")
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
	defer resp.Body.Close()
	var conf map[string]healer.NodeHealerConfig
	err = json.NewDecoder(resp.Body).Decode(&conf)
	if err != nil {
		return err
	}
	v := func(v *int) string {
		if v == nil || *v == 0 {
			return "disabled"
		}
		return fmt.Sprintf("%ds", *v)
	}
	baseConf := conf[""]
	delete(conf, "")
	fmt.Fprint(ctx.Stdout, "Default:\n")
	tbl := tablecli.NewTable()
	tbl.Headers = tablecli.Row{"Config", "Value"}
	tbl.AddRow(tablecli.Row{"Enabled", fmt.Sprintf("%v", baseConf.Enabled != nil && *baseConf.Enabled)})
	tbl.AddRow(tablecli.Row{"Max unresponsive time", v(baseConf.MaxUnresponsiveTime)})
	tbl.AddRow(tablecli.Row{"Max time since success", v(baseConf.MaxTimeSinceSuccess)})
	fmt.Fprint(ctx.Stdout, tbl.String())
	if len(conf) > 0 {
		fmt.Fprintln(ctx.Stdout)
	}
	poolNames := make([]string, 0, len(conf))
	for pool := range conf {
		poolNames = append(poolNames, pool)
	}
	sort.Strings(poolNames)
	for i, name := range poolNames {
		poolConf := conf[name]
		fmt.Fprintf(ctx.Stdout, "Pool %q:\n", name)
		tbl := tablecli.NewTable()
		tbl.Headers = tablecli.Row{"Config", "Value", "Inherited"}
		tbl.AddRow(tablecli.Row{"Enabled", fmt.Sprintf("%v", poolConf.Enabled != nil && *poolConf.Enabled), strconv.FormatBool(poolConf.EnabledInherited)})
		tbl.AddRow(tablecli.Row{"Max unresponsive time", v(poolConf.MaxUnresponsiveTime), strconv.FormatBool(poolConf.MaxUnresponsiveTimeInherited)})
		tbl.AddRow(tablecli.Row{"Max time since success", v(poolConf.MaxTimeSinceSuccess), strconv.FormatBool(poolConf.MaxTimeSinceSuccessInherited)})
		fmt.Fprint(ctx.Stdout, tbl.String())
		if i < len(poolNames)-1 {
			fmt.Fprintln(ctx.Stdout)
		}
	}
	return nil
}

type SetNodeHealingConfigCmd struct {
	fs              *gnuflag.FlagSet
	enable          bool
	disable         bool
	pool            string
	maxUnresponsive int
	maxUnsuccessful int
}

func (c *SetNodeHealingConfigCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-healing-update",
		Usage: "node-healing-update [-p/--pool pool] [--enable] [--disable] [--max-unresponsive <seconds>] [--max-unsuccessful <seconds>]",
		Desc:  "Update node healing configuration",
	}
}

func (c *SetNodeHealingConfigCmd) Flags() *gnuflag.FlagSet {
	msg := "The pool name to which the configuration will apply. If unset it'll be set as default for all pools."
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ContinueOnError)
		c.fs.StringVar(&c.pool, "p", "", msg)
		c.fs.StringVar(&c.pool, "pool", "", msg)
		c.fs.BoolVar(&c.enable, "enable", false, "Enable active node healing")
		c.fs.BoolVar(&c.disable, "disable", false, "Disable active node healing")
		c.fs.IntVar(&c.maxUnresponsive, "max-unresponsive", -1, "Number of seconds tsuru will wait for the node to notify it's alive")
		c.fs.IntVar(&c.maxUnsuccessful, "max-unsuccessful", -1, "Number of seconds tsuru will wait for the node to run successul checks")
	}
	return c.fs
}

func (c *SetNodeHealingConfigCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	if c.enable && c.disable {
		return errors.New("conflicting flags --enable and --disable")
	}
	v := url.Values{}
	v.Set("pool", c.pool)
	if c.maxUnresponsive >= 0 {
		v.Set("MaxUnresponsiveTime", strconv.Itoa(c.maxUnresponsive))
	}
	if c.maxUnsuccessful >= 0 {
		v.Set("MaxTimeSinceSuccess", strconv.Itoa(c.maxUnsuccessful))
	}
	if c.enable {
		v.Set("Enabled", strconv.FormatBool(true))
	}
	if c.disable {
		v.Set("Enabled", strconv.FormatBool(false))
	}
	body := strings.NewReader(v.Encode())
	u, err := cmd.GetURLVersion("1.2", "/healing/node")
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(req)
	if err == nil {
		fmt.Fprintln(ctx.Stdout, "Node healing configuration successfully updated.")
	}
	return err
}

type DeleteNodeHealingConfigCmd struct {
	cmd.ConfirmationCommand
	fs              *gnuflag.FlagSet
	pool            string
	enabled         bool
	maxUnresponsive bool
	maxUnsuccessful bool
}

func (c *DeleteNodeHealingConfigCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-healing-delete",
		Usage: "node-healing-delete [-p/--pool pool] [--enabled] [--max-unresponsive] [--max-unsuccessful]",
		Desc: `Delete a node healing configuration entry.

If [[--pool]] is provided the configuration entries from the specified pool
will be removed and the default value will be used.

If [[--pool]] is not provided the configuration entry will be removed from the
default configuration.`,
	}
}

func (c *DeleteNodeHealingConfigCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.ConfirmationCommand.Flags()
		msg := "The pool name from where the configuration will be removed. If unset it'll delete the default healing configuration."
		c.fs.StringVar(&c.pool, "p", "", msg)
		c.fs.StringVar(&c.pool, "pool", "", msg)
		c.fs.BoolVar(&c.enabled, "enabled", false, "Remove the 'enabled' configuration option")
		c.fs.BoolVar(&c.maxUnresponsive, "max-unresponsive", false, "Remove the 'max-unresponsive' configuration option")
		c.fs.BoolVar(&c.maxUnsuccessful, "max-unsuccessful", false, "Remove the 'max-unsuccessful' configuration option")
	}
	return c.fs
}

func (c *DeleteNodeHealingConfigCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	msg := "Are you sure you want to remove %snode healing configuration%s?"
	if c.pool == "" {
		msg = fmt.Sprintf(msg, "the default ", "")
	} else {
		msg = fmt.Sprintf(msg, "", " for pool "+c.pool)
	}
	if !c.Confirm(ctx, msg) {
		return errors.New("command aborted by user")
	}
	v := url.Values{}
	v.Set("pool", c.pool)
	if c.enabled {
		v.Add("name", "Enabled")
	}
	if c.maxUnresponsive {
		v.Add("name", "MaxUnresponsiveTime")
	}
	if c.maxUnsuccessful {
		v.Add("name", "MaxTimeSinceSuccess")
	}
	u, err := cmd.GetURLVersion("1.2", "/healing/node?"+v.Encode())
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	if err == nil {
		fmt.Fprintln(ctx.Stdout, "Node healing configuration successfully removed.")
	}
	return err
}

type RebalanceNodeCmd struct {
	cmd.ConfirmationCommand
	fs             *gnuflag.FlagSet
	dry            bool
	metadataFilter cmd.MapFlag
	appFilter      cmd.StringSliceFlag
}

func (c *RebalanceNodeCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-rebalance",
		Usage: "node-rebalance [--dry] [-y/--assume-yes] [-m/--metadata <metadata>=<value>]... [-a/--app <appname>]...",
		Desc: `Move units among nodes trying to create a more even distribution. This command
will automatically choose to which node each unit should be moved, trying to
distribute the units as evenly as possible.

The --dry flag runs the balancing algorithm without doing any real
modification. It will only print which units would be moved and where they
would be created.`,
		MinArgs: 0,
	}
}

func (c *RebalanceNodeCmd) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	if !c.dry && !c.Confirm(context, "Are you sure you want to rebalance containers?") {
		return nil
	}
	u, err := cmd.GetURLVersion("1.3", "/node/rebalance")
	if err != nil {
		return err
	}
	opts := provision.RebalanceNodesOptions{
		Dry: c.dry,
	}
	if len(c.metadataFilter) > 0 {
		opts.MetadataFilter = c.metadataFilter
	}
	if len(c.appFilter) > 0 {
		opts.AppFilter = c.appFilter
	}
	v, err := form.EncodeToValues(&opts)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, bytes.NewBufferString(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(context.Stdout, response)
}

func (c *RebalanceNodeCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.ConfirmationCommand.Flags()
		c.fs.BoolVar(&c.dry, "dry", false, "Dry run, only shows what would be done")
		desc := "Filter by host metadata"
		c.fs.Var(&c.metadataFilter, "metadata", desc)
		c.fs.Var(&c.metadataFilter, "m", desc)
		desc = "Filter by app name"
		c.fs.Var(&c.appFilter, "app", desc)
		c.fs.Var(&c.appFilter, "a", desc)
	}
	return c.fs
}

type InfoNodeCmd struct{}

func (InfoNodeCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-info",
		Usage: "node-info <address>",
		Desc: `Get info about a node from the cluster.
`,
		MinArgs: 1,
	}
}

func (c *InfoNodeCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	address := ctx.Args[0]
	u, err := cmd.GetURLVersion("1.6", fmt.Sprintf("/node/%s", address))
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
	var result apiTypes.InfoNodeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Address: %s\n", result.Node.Address))
	buf.WriteString(fmt.Sprintf("Status: %s\n", result.Node.Status))
	buf.WriteString(fmt.Sprintf("Pool: %s\n", result.Node.Pool))
	buf.WriteString(fmt.Sprintf("Provisioner: %s\n", result.Node.Provisioner))
	buf.WriteString("Metadata:\n")
	nodeTable := tablecli.Table{Headers: tablecli.Row([]string{"Key", "Value"}), LineSeparator: true}
	for key, value := range result.Node.Metadata {
		nodeTable.AddRow(tablecli.Row([]string{key, value}))
	}
	nodeTable.Sort()
	if result.Node.IaaSID != "" {
		nodeTable.AddRow(tablecli.Row([]string{"iaasID", result.Node.IaaSID}))
	}
	buf.WriteString(nodeTable.String())
	buf.WriteString("\n")
	unitsTable := tablecli.Table{Headers: tablecli.Row([]string{"Unit", "Status", "Type", "App", "ProcessName"}), LineSeparator: true}
	for _, unit := range result.Units {
		if unit.ID == "" {
			continue
		}
		row := []string{tsuruClient.ShortID(unit.ID), string(unit.Status), unit.Type, unit.AppName, unit.ProcessName}
		unitsTable.AddRow(tablecli.Row(row))
	}
	buf.WriteString(fmt.Sprintf("Units: %d\n", unitsTable.Rows()))
	if unitsTable.Rows() > 0 {
		buf.WriteString(unitsTable.String())
	}
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("Node Status:\n"))
	if !result.Status.LastSuccess.IsZero() {
		buf.WriteString(fmt.Sprintf("Last Success: %s\n", result.Status.LastSuccess.Local().Format(time.Stamp)))
	}
	if !result.Status.LastUpdate.IsZero() {
		buf.WriteString(fmt.Sprintf("Last Update: %s\n", result.Status.LastUpdate.Local().Format(time.Stamp)))
	}
	statusTable := tablecli.Table{Headers: tablecli.Row([]string{"Time", "Name", "Success", "Error"}), LineSeparator: true}
	for _, check := range result.Status.Checks {
		for _, cc := range check.Checks {
			statusTable.AddRow(tablecli.Row([]string{check.Time.Local().Format(time.Stamp), cc.Name, fmt.Sprintf("%t", cc.Successful), cc.Err}))
		}
	}
	if statusTable.Rows() > 0 {
		statusTable.Reverse()
		buf.WriteString(statusTable.String())
	} else {
		buf.WriteString("Missing check information")
	}
	ctx.Stdout.Write(buf.Bytes())
	return nil
}
