// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
)

type ServiceList struct{}

func (s ServiceList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-list",
		Usage: "service-list",
		Desc: `Retrieves and shows a list of services the user has access. If there are
instances created for any service they will also be shown.`,
	}
}

func (s ServiceList) Run(ctx *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/services/instances")
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	rslt, err := cmd.ShowServicesInstancesList(b)
	if err != nil {
		return err
	}
	n, _ := ctx.Stdout.Write(rslt)
	if n != len(rslt) {
		return errors.New("Failed to write the output of the command")
	}
	return nil
}

type ServiceInstanceAdd struct {
	fs          *gnuflag.FlagSet
	teamOwner   string
	description string
	tags        cmd.StringSliceFlag
}

func (c *ServiceInstanceAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-add",
		Usage: "service-instance-add <service-name> <service-instance-name> [plan] [-t/--team-owner <team>] [-d/--description description] [-g/--tag tag]...",
		Desc: `Creates a service instance of a service. There can later be binded to
applications with [[tsuru service-bind]].

This example shows how to add a new instance of **mongodb** service, named
**tsuru_mongodb** with the plan **small**:

::

    $ tsuru service-instance-add mongodb tsuru_mongodb small -t myteam
`,
		MinArgs: 2,
		MaxArgs: 3,
	}
}

func (c *ServiceInstanceAdd) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName, instanceName := ctx.Args[0], ctx.Args[1]
	var plan string
	if len(ctx.Args) > 2 {
		plan = ctx.Args[2]
	}
	v := url.Values{}
	v.Set("name", instanceName)
	v.Set("plan", plan)
	v.Set("owner", c.teamOwner)
	v.Set("description", c.description)
	for _, tag := range c.tags {
		v.Add("tag", tag)
	}
	u, err := cmd.GetURL(fmt.Sprintf("/services/%s/instances", serviceName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Service successfully added.\n")
	return nil
}

func (c *ServiceInstanceAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		flagDesc := "the team that owns the service (mandatory if the user is member of more than one team)"
		c.fs = gnuflag.NewFlagSet("service-instance-add", gnuflag.ExitOnError)
		c.fs.StringVar(&c.teamOwner, "team-owner", "", flagDesc)
		c.fs.StringVar(&c.teamOwner, "t", "", flagDesc)
		descriptionMessage := "service instance description"
		c.fs.StringVar(&c.description, "description", "", descriptionMessage)
		c.fs.StringVar(&c.description, "d", "", descriptionMessage)
		tagMessage := "service instance tag"
		c.fs.Var(&c.tags, "tag", tagMessage)
		c.fs.Var(&c.tags, "g", tagMessage)
	}
	return c.fs
}

type ServiceInstanceUpdate struct {
	fs          *gnuflag.FlagSet
	description string
	tags        cmd.StringSliceFlag
}

func (c *ServiceInstanceUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-update",
		Usage: "service-instance-update <service-name> <service-instance-name> [-d/--description description] [-g/--tag tag]...",
		Desc: `Updates a service instance of a service.

The --description parameter sets a description for your service instance.

The --tag parameter adds a tag to your service instance. This parameter
may be used multiple times.`,
		MinArgs: 2,
	}
}

func (c *ServiceInstanceUpdate) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName, instanceName := ctx.Args[0], ctx.Args[1]
	u, err := cmd.GetURL(fmt.Sprintf("/services/%s/instances/%s", serviceName, instanceName))
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("description", c.description)
	for _, tag := range c.tags {
		v.Add("tag", tag)
	}
	request, err := http.NewRequest("PUT", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Service successfully updated.\n")
	return nil
}

func (c *ServiceInstanceUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("service-instance-update", gnuflag.ExitOnError)
		descriptionMessage := "service instance description"
		c.fs.StringVar(&c.description, "description", "", descriptionMessage)
		c.fs.StringVar(&c.description, "d", "", descriptionMessage)
		tagMessage := "service instance tag"
		c.fs.Var(&c.tags, "tag", tagMessage)
		c.fs.Var(&c.tags, "g", tagMessage)
	}
	return c.fs
}

type ServiceInstanceBind struct {
	cmd.GuessingCommand
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (sb *ServiceInstanceBind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	appName, err := sb.Guess()
	if err != nil {
		return err
	}
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	u, err := cmd.GetURL("/services/" + serviceName + "/instances/" + instanceName + "/" + appName)
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("noRestart", strconv.FormatBool(sb.noRestart))
	request, err := http.NewRequest("PUT", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	w := tsuruIo.NewStreamWriter(ctx.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, resp.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}

func (sb *ServiceInstanceBind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-bind",
		Usage: "service-instance-bind <service-name> <service-instance-name> [-a/--app appname] [--no-restart]",
		Desc: `Binds an application to a previously created service instance. See [[tsuru
service-instance-add]] for more details on how to create a service instance.

When binding an application to a service instance, tsuru will add new
environment variables to the application. All environment variables exported
by bind will be private (not accessible via [[tsuru env-get]]).`,
		MinArgs: 2,
	}
}

func (sb *ServiceInstanceBind) Flags() *gnuflag.FlagSet {
	if sb.fs == nil {
		sb.fs = sb.GuessingCommand.Flags()
		sb.fs.BoolVar(&sb.noRestart, "no-restart", false, "Binds an application to a service instance without restart the application")
	}
	return sb.fs
}

type ServiceInstanceUnbind struct {
	cmd.GuessingCommand
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (su *ServiceInstanceUnbind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	appName, err := su.Guess()
	if err != nil {
		return err
	}
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	url, err := cmd.GetURL("/services/" + serviceName + "/instances/" + instanceName + "/" + appName)
	if err != nil {
		return err
	}
	url += fmt.Sprintf("?noRestart=%t", su.noRestart)
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	w := tsuruIo.NewStreamWriter(ctx.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, resp.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}

func (su *ServiceInstanceUnbind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-unbind",
		Usage: "service-instance-unbind <service-name> <service-instance-name> [-a/--app appname] [--no-restart]",
		Desc: `Unbinds an application from a service instance. After unbinding, the instance
will not be available anymore. For example, when unbinding an application from
a MySQL service, the application would lose access to the database.`,
		MinArgs: 2,
	}
}

func (su *ServiceInstanceUnbind) Flags() *gnuflag.FlagSet {
	if su.fs == nil {
		su.fs = su.GuessingCommand.Flags()
		su.fs.BoolVar(&su.noRestart, "no-restart", false, "Unbinds an application from a service instance without restart the application")
	}
	return su.fs
}

type ServiceInstanceStatus struct{}

func (c ServiceInstanceStatus) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-status",
		Usage: "service-instance-status <service-name> <service-instance-name>",
		Desc: `Displays the status of the given service instance. For now, it checks only if
the instance is "up" (receiving connections) or "down" (refusing connections).`,
		MinArgs: 2,
	}
}

func (c ServiceInstanceStatus) Run(ctx *cmd.Context, client *cmd.Client) error {
	servName := ctx.Args[0]
	instName := ctx.Args[1]
	url, err := cmd.GetURL("/services/" + servName + "/instances/" + instName + "/status")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bMsg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	msg := string(bMsg) + "\n"
	n, err := fmt.Fprint(ctx.Stdout, msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return errors.New("Failed to write to standard output.\n")
	}
	return nil
}

type ServiceInstanceInfo struct{}

func (c ServiceInstanceInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-instance-info",
		Usage:   "service-instance-info <service-name> <instance-name>",
		Desc:    `Displays the information of the given service instance.`,
		MinArgs: 2,
	}
}

type ServiceInstanceInfoModel struct {
	ServiceName     string
	InstanceName    string
	Apps            []string
	Teams           []string
	TeamOwner       string
	Description     string
	PlanName        string
	PlanDescription string
	CustomInfo      map[string]string
	Tags            []string
}

func (c ServiceInstanceInfo) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	url, err := cmd.GetURL("/services/" + serviceName + "/instances/" + instanceName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var si ServiceInstanceInfoModel
	err = json.Unmarshal(result, &si)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Service: %s\n", serviceName)
	fmt.Fprintf(ctx.Stdout, "Instance: %s\n", instanceName)
	fmt.Fprintf(ctx.Stdout, "Apps: %s\n", strings.Join(si.Apps, ", "))
	fmt.Fprintf(ctx.Stdout, "Teams: %s\n", strings.Join(si.Teams, ", "))
	fmt.Fprintf(ctx.Stdout, "Team Owner: %s\n", si.TeamOwner)
	fmt.Fprintf(ctx.Stdout, "Description: %s\n", si.Description)
	fmt.Fprintf(ctx.Stdout, "Tags: %s\n", strings.Join(si.Tags, ", "))
	fmt.Fprintf(ctx.Stdout, "Plan: %s\n", si.PlanName)
	fmt.Fprintf(ctx.Stdout, "Plan description: %s\n", si.PlanDescription)
	if len(si.CustomInfo) != 0 {
		ctx.Stdout.Write([]byte(fmt.Sprintf("\nCustom Info for \"%s\"\n", instanceName)))
		keyList := make([]string, 0)
		for key := range si.CustomInfo {
			keyList = append(keyList, key)
		}
		sort.Strings(keyList)
		for ind, key := range keyList {
			ctx.Stdout.Write([]byte(key + ":" + "\n"))
			ctx.Stdout.Write([]byte(si.CustomInfo[key] + "\n"))
			if ind != len(keyList)-1 {
				ctx.Stdout.Write([]byte("\n"))
			}
		}
	}
	return nil
}

type ServiceInfo struct{}

func (c ServiceInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-info",
		Usage: "service-info <service-name>",
		Desc: `Displays a list of all instances of a given service (that the user has access
to), and apps bound to these instances.`,
		MinArgs: 1,
	}
}

type ServiceInstanceModel struct {
	Name     string
	PlanName string
	Apps     []string
	Info     map[string]string
}

// in returns true if the list contains the value
func in(value string, list []string) bool {
	for _, item := range list {
		if value == item {
			return true
		}
	}
	return false
}

func (ServiceInfo) ExtraHeaders(instances []ServiceInstanceModel) []string {
	var headers []string
	for _, instance := range instances {
		for key := range instance.Info {
			if !in(key, headers) {
				headers = append(headers, key)
			}
		}
	}
	sort.Sort(sort.StringSlice(headers))
	return headers
}

func (c ServiceInfo) BuildInstancesTable(serviceName string, ctx *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/services/" + serviceName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var instances []ServiceInstanceModel
	err = json.Unmarshal(result, &instances)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte(fmt.Sprintf("Info for \"%s\"\n\n", serviceName)))
	if len(instances) > 0 {
		ctx.Stdout.Write([]byte("Instances\n"))
		table := cmd.NewTable()
		extraHeaders := c.ExtraHeaders(instances)
		hasPlan := false
		var data []string
		var headers []string
		for _, instance := range instances {
			if instance.PlanName != "" {
				hasPlan = true
			}
		}
		for _, instance := range instances {
			apps := strings.Join(instance.Apps, ", ")
			if hasPlan {
				data = []string{instance.Name, instance.PlanName, apps}
			} else {
				data = []string{instance.Name, apps}
			}
			for _, h := range extraHeaders {
				data = append(data, instance.Info[h])
			}
			table.AddRow(cmd.Row(data))
		}
		if hasPlan {
			headers = []string{"Instances", "Plan", "Apps"}
		} else {
			headers = []string{"Instances", "Apps"}
		}
		headers = append(headers, extraHeaders...)
		table.Headers = cmd.Row(headers)
		ctx.Stdout.Write(table.Bytes())
	}
	return nil
}

func (c ServiceInfo) BuildPlansTable(serviceName string, ctx *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL(fmt.Sprintf("/services/%s/plans", serviceName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var plans []map[string]string
	err = json.Unmarshal(result, &plans)
	if err != nil {
		return err
	}
	if len(plans) > 0 {
		fmt.Fprint(ctx.Stdout, "\nPlans\n")
		table := cmd.NewTable()
		for _, plan := range plans {
			data := []string{plan["Name"], plan["Description"]}
			table.AddRow(cmd.Row(data))
		}
		table.Headers = cmd.Row([]string{"Name", "Description"})
		ctx.Stdout.Write(table.Bytes())
	}
	return nil
}

func (c ServiceInfo) WriteDoc(ctx *cmd.Context, client *cmd.Client) error {
	sName := ctx.Args[0]
	url := fmt.Sprintf("/services/%s/doc", sName)
	url, err := cmd.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(result) != 0 {
		fmt.Fprint(ctx.Stdout, "\nDocumentation:\n")
		ctx.Stdout.Write(result)
	}
	return nil
}

func (c ServiceInfo) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]
	err := c.BuildInstancesTable(serviceName, ctx, client)
	if err != nil {
		return err
	}
	err = c.BuildPlansTable(serviceName, ctx, client)
	if err != nil {
		return err
	}
	return c.WriteDoc(ctx, client)
}

type ServiceInstanceRemove struct {
	cmd.ConfirmationCommand
	fs    *gnuflag.FlagSet
	force bool
}

func (c *ServiceInstanceRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-remove",
		Usage: "service-instance-remove <service-name> <service-instance-name> [-f/--force] [-y/--assume-yes]",
		Desc: `Destroys a service instance. It can't remove a service instance that is bound
to an app, so before remove a service instance, make sure there is no apps
bound to it (see [[tsuru service-instance-info]] command).`,
		MinArgs: 2,
	}
}

func (c *ServiceInstanceRemove) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	msg := fmt.Sprintf("Are you sure you want to remove the instance %q", instanceName)
	if c.force {
		msg += " and all binds"
	}
	if !c.Confirm(ctx, msg+"?") {
		return nil
	}
	url := fmt.Sprintf("/services/%s/instances/%s?unbindall=%v", serviceName, instanceName, c.force)
	url, err := cmd.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	err = cmd.StreamJSONResponse(ctx.Stdout, resp)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, `Service "%s" successfully removed!`+"\n", instanceName)
	return nil
}

func (c *ServiceInstanceRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.ConfirmationCommand.Flags()
		c.fs.BoolVar(&c.force, "f", false, "Forces the removal of a service instance binded to apps.")
		c.fs.BoolVar(&c.force, "force", false, "Forces the removal of a service instance binded to apps.")
	}
	return c.fs
}

type ServiceInstanceGrant struct{}

func (c *ServiceInstanceGrant) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-instance-grant",
		Usage:   "service-instance-grant <service-name> <service-instance-name> <team-name>",
		Desc:    `Grant access to team in a service instance.`,
		MinArgs: 3,
	}
}

func (c *ServiceInstanceGrant) Run(ctx *cmd.Context, client *cmd.Client) error {
	sName := ctx.Args[0]
	siName := ctx.Args[1]
	teamName := ctx.Args[2]
	url := fmt.Sprintf("/services/%s/instances/permission/%s/%s", sName, siName, teamName)
	url, err := cmd.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, `Granted access to team %s in %s service instance.`+"\n", teamName, siName)
	return nil
}

type ServiceInstanceRevoke struct{}

func (c *ServiceInstanceRevoke) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-instance-revoke",
		Usage:   "service-instance-revoke <service-name> <service-instance-name> <team-name>",
		Desc:    `Revoke access to team in a service instance.`,
		MinArgs: 3,
	}
}

func (c *ServiceInstanceRevoke) Run(ctx *cmd.Context, client *cmd.Client) error {
	sName := ctx.Args[0]
	siName := ctx.Args[1]
	teamName := ctx.Args[2]
	url := fmt.Sprintf("/services/%s/instances/permission/%s/%s", sName, siName, teamName)
	url, err := cmd.GetURL(url)
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
	fmt.Fprintf(ctx.Stdout, `Revoked access to team %s in %s service instance.`+"\n", teamName, siName)
	return nil
}
