// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
	"github.com/tsuru/tsuru/service"
)

type serviceList struct{}

func (s serviceList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-list",
		Usage: "service-list",
		Desc: `Retrieves and shows a list of services the user has access. If there are
instances created for any service they will also be shown.`,
	}
}

func (s serviceList) Run(ctx *cmd.Context, client *cmd.Client) error {
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
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	rslt, err := cmd.ShowServicesInstancesList(b)
	if err != nil {
		return err
	}
	n, err := ctx.Stdout.Write(rslt)
	if n != len(rslt) {
		return errors.New("Failed to write the output of the command")
	}
	return nil
}

type serviceAdd struct {
	fs        *gnuflag.FlagSet
	teamOwner string
}

func (c *serviceAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-add",
		Usage: "service-add <service-name> <service-instance-name> [plan] [-t/--team-owner <team>]",
		Desc: `Creates a service instance of a service. There can later be binded to
applications with [[tsuru service-bind]].

This example shows how to add a new instance of **mongodb** service, named
**tsuru_mongodb** with the plan **small**:

::

    $ tsuru service-add mongodb tsuru_mongodb small -t myteam
`,
		MinArgs: 2,
		MaxArgs: 3,
	}
}

func (c *serviceAdd) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName, instanceName := ctx.Args[0], ctx.Args[1]
	var plan string
	if len(ctx.Args) > 2 {
		plan = ctx.Args[2]
	}
	var b bytes.Buffer
	params := map[string]string{
		"name":         instanceName,
		"service_name": serviceName,
		"plan":         plan,
		"owner":        c.teamOwner,
	}
	err := json.NewEncoder(&b).Encode(params)
	if err != nil {
		return err
	}
	url, err := cmd.GetURL("/services/instances")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Service successfully added.\n")
	return nil
}

func (c *serviceAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		flagDesc := "the team that owns the service (mandatory if the user is member of more than one team)"
		c.fs = gnuflag.NewFlagSet("service-add", gnuflag.ExitOnError)
		c.fs.StringVar(&c.teamOwner, "team-owner", "", flagDesc)
		c.fs.StringVar(&c.teamOwner, "t", "", flagDesc)
	}
	return c.fs
}

type serviceBind struct {
	cmd.GuessingCommand
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (sb *serviceBind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	appName, err := sb.Guess()
	if err != nil {
		return err
	}
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	url, err := cmd.GetURL("/services/" + serviceName + "/instances/" + instanceName + "/" + appName)
	if err != nil {
		return err
	}
	url += fmt.Sprintf("?noRestart=%t", sb.noRestart)
	request, err := http.NewRequest("PUT", url, nil)
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

func (sb *serviceBind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-bind",
		Usage: "service-bind <service-name> <service-instance-name> [-a/--app appname] [--no-restart]",
		Desc: `Binds an application to a previously created service instance. See [[tsuru
service-add]] for more details on how to create a service instance.

When binding an application to a service instance, tsuru will add new
environment variables to the application. All environment variables exported
by bind will be private (not accessible via [[tsuru env-get]]).`,
		MinArgs: 2,
	}
}

func (sb *serviceBind) Flags() *gnuflag.FlagSet {
	if sb.fs == nil {
		sb.fs = sb.GuessingCommand.Flags()
		sb.fs.BoolVar(&sb.noRestart, "no-restart", false, "Binds an application to a service instance without restart the application")
	}
	return sb.fs
}

type serviceUnbind struct {
	cmd.GuessingCommand
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (su *serviceUnbind) Run(ctx *cmd.Context, client *cmd.Client) error {
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
	request, err := http.NewRequest("DELETE", url, nil)
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

func (su *serviceUnbind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-unbind",
		Usage: "service-unbind <service-name> <service-instance-name> [-a/--app appname] [--no-restart]",
		Desc: `Unbinds an application from a service instance. After unbinding, the instance
will not be available anymore. For example, when unbinding an application from
a MySQL service, the application would lose access to the database.`,
		MinArgs: 2,
	}
}

func (su *serviceUnbind) Flags() *gnuflag.FlagSet {
	if su.fs == nil {
		su.fs = su.GuessingCommand.Flags()
		su.fs.BoolVar(&su.noRestart, "no-restart", false, "Unbinds an application from a service instance without restart the application")
	}
	return su.fs
}

type serviceInstanceStatus struct{}

func (c serviceInstanceStatus) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-status",
		Usage: "service-status <service-name> <service-instance-name>",
		Desc: `Displays the status of the given service instance. For now, it checks only if
the instance is "up" (receiving connections) or "down" (refusing connections).`,
		MinArgs: 2,
	}
}

func (c serviceInstanceStatus) Run(ctx *cmd.Context, client *cmd.Client) error {
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

type serviceInfo struct{}

func (c serviceInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-info",
		Usage: "service-info <service-name>",
		Desc: `Displays a list of all instances of a given service (that the user has access
to), and apps bound to these instances.`,
		MinArgs: 1,
	}
}

type ServiceInstanceModel struct {
	Name string
	Apps []string
	Info map[string]string
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

func (serviceInfo) ExtraHeaders(instances []ServiceInstanceModel) []string {
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

func (c serviceInfo) BuildInstancesTable(serviceName string, ctx *cmd.Context, client *cmd.Client) error {
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
		for _, instance := range instances {
			apps := strings.Join(instance.Apps, ", ")
			data := []string{instance.Name, apps}
			for _, h := range extraHeaders {
				data = append(data, instance.Info[h])
			}
			table.AddRow(cmd.Row(data))
		}
		headers := []string{"Instances", "Apps"}
		headers = append(headers, extraHeaders...)
		table.Headers = cmd.Row(headers)
		ctx.Stdout.Write(table.Bytes())
	}
	return nil
}

func (c serviceInfo) BuildPlansTable(serviceName string, ctx *cmd.Context, client *cmd.Client) error {
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

func (c serviceInfo) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]
	err := c.BuildInstancesTable(serviceName, ctx, client)
	if err != nil {
		return err
	}
	return c.BuildPlansTable(serviceName, ctx, client)
}

type serviceDoc struct{}

func (serviceDoc) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-doc",
		Usage:   "service-doc <service-name>",
		Desc:    `Shows the documentation of a service.`,
		MinArgs: 1,
	}
}

func (serviceDoc) Run(ctx *cmd.Context, client *cmd.Client) error {
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
	ctx.Stdout.Write(result)
	return nil
}

type serviceRemove struct {
	yes       bool
	yesUnbind bool
	fs        *gnuflag.FlagSet
}

func (c *serviceRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-remove",
		Usage: "service-remove <service-name> <service-instance-name> [--assume-yes] [--unbind]",
		Desc: `Destroys a service instance. It can't remove a service instance that is bound
to an app, so before remove a service instance, make sure there is no apps
bound to it (see [[tsuru service-info]] command).`,
		MinArgs: 2,
	}
}

func removeServiceInstanceWithUnbind(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	url := fmt.Sprintf("/services/%s/instances/%s?unbindall=%s", serviceName, instanceName, "true")
	url, err := cmd.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	defer resp.Body.Close()
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

func (c *serviceRemove) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	var answer string
	if !c.yes {
		fmt.Fprintf(ctx.Stdout, `Are you sure you want to remove service "%s"? (y/n) `, instanceName)
		fmt.Fscanf(ctx.Stdin, "%s", &answer)
		if answer != "y" {
			fmt.Fprintln(ctx.Stdout, "Abort.")
			return nil
		}
	}
	var url string
	if c.yesUnbind {
		return removeServiceInstanceWithUnbind(ctx, client)
	}
	url = fmt.Sprintf("/services/%s/instances/%s", serviceName, instanceName)
	url, err := cmd.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, _ := client.Do(request)
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var jsonMsg tsuruIo.SimpleJsonMessage
	json.Unmarshal(result, &jsonMsg)
	var msgError error
	if jsonMsg.Error != "" {
		msgError = errors.New(jsonMsg.Error)
	}
	if msgError != nil {
		if msgError.Error() == service.ErrServiceInstanceBound.Error() {
			fmt.Fprintf(ctx.Stdout, `Applications bound to the service "%s": "%s"`+"\n", instanceName, jsonMsg.Message)
			fmt.Fprintf(ctx.Stdout, `Do you want unbind all apps? (y/n) `)
			fmt.Fscanf(ctx.Stdin, "%s", &answer)
			if answer != "y" {
				fmt.Fprintln(ctx.Stdout, "Abort.")
				return nil
			}
			msgError = removeServiceInstanceWithUnbind(ctx, client)
		}
		return msgError
	}
	fmt.Fprintf(ctx.Stdout, `Service "%s" successfully removed!`+"\n", instanceName)
	return nil
}

func (c *serviceRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("service-remove", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.yes, "assume-yes", false, "Don't ask for confirmation, just remove the service.")
		c.fs.BoolVar(&c.yes, "y", false, "Don't ask for confirmation, just remove the service.")
		c.fs.BoolVar(&c.yesUnbind, "unbind", false, "Don't ask for confirmation, just remove all applications bound.")
		c.fs.BoolVar(&c.yesUnbind, "u", false, "Don't ask for confirmation, just remove all applications bound.")
	}
	return c.fs
}

type serviceInstanceGrant struct{}

func (c *serviceInstanceGrant) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-instance-grant",
		Usage:   "service-instance-grant <service-name> <service-instance-name> <team-name>",
		Desc:    `Grant access to team in a service instance.`,
		MinArgs: 3,
	}
}

func (c *serviceInstanceGrant) Run(ctx *cmd.Context, client *cmd.Client) error {
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

type serviceInstanceRevoke struct{}

func (c *serviceInstanceRevoke) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-instance-revoke",
		Usage:   "service-instance-revoke <service-name> <service-instance-name> <team-name>",
		Desc:    `Revoke access to team in a service instance.`,
		MinArgs: 3,
	}
}

func (c *serviceInstanceRevoke) Run(ctx *cmd.Context, client *cmd.Client) error {
	sName := ctx.Args[0]
	siName := ctx.Args[1]
	teamName := ctx.Args[2]
	url := fmt.Sprintf("/services/%s/instances/permission/%s/%s", sName, siName, teamName)
	url, err := cmd.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
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
