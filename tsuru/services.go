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

	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
	"launchpad.net/gnuflag"
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
		Usage: "service-add <servicename> <serviceinstancename> [plan] [-t/--team-owner <team>]",
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
}

func (sb *serviceBind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	appName, err := sb.Guess()
	if err != nil {
		return err
	}
	instanceName := ctx.Args[0]
	url, err := cmd.GetURL("/services/instances/" + instanceName + "/" + appName)
	if err != nil {
		return err
	}
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
		Usage: "service-bind <service-instance-name> [-a/--app appname]",
		Desc: `Binds an application to a previously created service instance. See [[tsuru
service-add]] for more details on how to create a service instance.

When binding an application to a service instance, tsuru will add new
environment variables to the application. All environment variables exported
by bind will be private (not accessible via [[tsuru env-get]]).`,
		MinArgs: 1,
	}
}

type serviceUnbind struct {
	cmd.GuessingCommand
}

func (su *serviceUnbind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	appName, err := su.Guess()
	if err != nil {
		return err
	}
	instanceName := ctx.Args[0]
	url, err := cmd.GetURL("/services/instances/" + instanceName + "/" + appName)
	if err != nil {
		return err
	}
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
		Usage: "service-unbind <instancename> [-a/--app appname]",
		Desc: `Unbinds an application from a service instance. After unbinding, the instance
will not be available anymore. For example, when unbinding an application from
a MySQL service, the application would lose access to the database.`,
		MinArgs: 1,
	}
}

type serviceInstanceStatus struct{}

func (c serviceInstanceStatus) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-status",
		Usage: "service-status <service-instance-name>",
		Desc: `Displays the status of the given service instance. For now, it checks only if
the instance is "up" (receiving connections) or "down" (refusing connections).`,
		MinArgs: 1,
	}
}

func (c serviceInstanceStatus) Run(ctx *cmd.Context, client *cmd.Client) error {
	instName := ctx.Args[0]
	url, err := cmd.GetURL("/services/instances/" + instName + "/status")
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
	ctx.Stdout.Write([]byte("\nPlans\n"))
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
		Usage: "service-remove <serviceinstancename> [--assume-yes] [--unbind]",
		Desc: `Destroys a service instance. It can't remove a service instance that is bound
to an app, so before remove a service instance, make sure there is no apps
bound to it (see [[tsuru service-info]] command).`,
		MinArgs: 1,
	}
}

func removeServiceInstanceWithUnbind(ctx *cmd.Context, client *cmd.Client) error {
	name := ctx.Args[0]
	url := fmt.Sprintf("/services/instances/%s?unbindall=%s", name, "true")
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
	fmt.Fprintf(ctx.Stdout, `Service "%s" successfully removed!`+"\n", name)
	return nil
}

func (c *serviceRemove) Run(ctx *cmd.Context, client *cmd.Client) error {
	name := ctx.Args[0]
	var answer string
	if !c.yes {
		fmt.Fprintf(ctx.Stdout, `Are you sure you want to remove service "%s"? (y/n) `, name)
		fmt.Fscanf(ctx.Stdin, "%s", &answer)
		if answer != "y" {
			fmt.Fprintln(ctx.Stdout, "Abort.")
			return nil
		}
	}
	var url string
	if c.yesUnbind {
		err := removeServiceInstanceWithUnbind(ctx, client)
		if err != nil {
			fmt.Fprintf(ctx.Stdout, err.Error())
		}
		return err
	}
	url = fmt.Sprintf("/services/instances/%s", name)
	url, err := cmd.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		if resp.StatusCode == http.StatusConflict {
			fmt.Fprintf(ctx.Stdout, `Do you want unbind all apps? (y/n) `)
			fmt.Fscanf(ctx.Stdin, "%s", &answer)
			if answer != "y" {
				fmt.Fprintln(ctx.Stdout, err.Error())
				return err
			}
			err = removeServiceInstanceWithUnbind(ctx, client)
		}
		return err
	}
	fmt.Fprintf(ctx.Stdout, `Service "%s" successfully removed!`+"\n", name)
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
		Usage:   "service-instance-grant <service_instance_name> <team_name>",
		Desc:    `Grant access to team in a service instance.`,
		MinArgs: 2,
	}
}

func (c *serviceInstanceGrant) Run(ctx *cmd.Context, client *cmd.Client) error {
	siName := ctx.Args[0]
	teamName := ctx.Args[1]
	url := fmt.Sprintf("/services/instances/permission/%s/%s", siName, teamName)
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
		Usage:   "service-instance-revoke <service_instance_name> <team_name>",
		Desc:    `Revoke access to team in a service instance.`,
		MinArgs: 2,
	}
}

func (c *serviceInstanceRevoke) Run(ctx *cmd.Context, client *cmd.Client) error {
	siName := ctx.Args[0]
	teamName := ctx.Args[1]
	url := fmt.Sprintf("/services/instances/permission/%s/%s", siName, teamName)
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
