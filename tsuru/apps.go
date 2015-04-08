// Copyright 2015 tsuru-client authors. All rights reserved.
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
	"strings"
	"text/template"
	"time"

	tsuruapp "github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
	"launchpad.net/gnuflag"
)

type appCreate struct {
	teamOwner string
	plan      string
	pool      string
	fs        *gnuflag.FlagSet
}

func (c *appCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-create",
		Usage: "app-create <appname> <platform> [--plan/-p plan_name] [--team/-t (team owner)] [-o/--pool pool_name]",
		Desc: `Creates a new app using the given name and platform. For tsuru,
a platform is provisioner dependent. To check the available platforms, use the
command [[tsuru platform-list]] and to add a platform use the command [[tsuru-admin platform-add]].

In order to create an app, you need to be member of at least one team. All
teams that you are member (see [[tsuru team-list]]) will be able to access the
app.

The [[--platform]] parameter is the name of the platform to be used when
creating the app. This will define how tsuru understands and executes your
app. The list of available platforms can be found running [[tsuru platform-list]].

The [[--plan]] parameter defines the plan to be used. The plan specifies how
computational resources are allocated to your application. Typically this
means limits for memory and swap usage, and how much cpu share is allocated.
The list of available plans can be found running [[tsuru plan-list]].

If this parameter is not informed, tsuru will choose the plan with the
[[default]] flag set to true.

The [[--team]] parameter describes which team is responsible for the created
app, this is only needed if the current user belongs to more than one team, in
which case this parameter will be mandatory.

The [[--pool]] parameter defines which pool your app will be deployed.
This is only needed if you have more than one pool associated with your teams.`,
		MinArgs: 2,
	}
}

func (c *appCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		infoMessage := "The plan used to create the app"
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.StringVar(&c.plan, "plan", "", infoMessage)
		c.fs.StringVar(&c.plan, "p", "", infoMessage)
		teamMessage := "Team owner app"
		c.fs.StringVar(&c.teamOwner, "team", "", teamMessage)
		c.fs.StringVar(&c.teamOwner, "t", "", teamMessage)
		poolMessage := "Pool to deploy your app"
		c.fs.StringVar(&c.pool, "pool", "", poolMessage)
		c.fs.StringVar(&c.pool, "o", "", poolMessage)
	}
	return c.fs
}

func (c *appCreate) Run(context *cmd.Context, client *cmd.Client) error {
	appName := context.Args[0]
	platform := context.Args[1]
	params := map[string]interface{}{
		"name":      appName,
		"platform":  platform,
		"plan":      map[string]interface{}{"name": c.plan},
		"teamOwner": c.teamOwner,
		"pool":      c.pool,
	}
	b, err := json.Marshal(params)
	if err != nil {
		return err
	}
	url, err := cmd.GetURL("/apps")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	out := make(map[string]string)
	err = json.Unmarshal(result, &out)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "App %q has been created!\n", appName)
	fmt.Fprintln(context.Stdout, "Use app-info to check the status of the app and its units.")
	if out["repository_url"] != "" {
		fmt.Fprintf(context.Stdout, "Your repository for %q project is %q\n", appName, out["repository_url"])
	}
	return nil
}

type appRemove struct {
	cmd.GuessingCommand
	cmd.ConfirmationCommand
	fs *gnuflag.FlagSet
}

func (c *appRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-remove",
		Usage: "app-remove [-a/--app appname] [-y/--assume-yes]",
		Desc: `Removes an application. If the app is bound to any service instance, all binds
will be removed before the app gets deleted (see [[tsuru service-unbind]]).

You need to be a member of a team that has access to the app to be able to
remove it (you are able to remove any app that you see in [[tsuru app-list]]).`,
		MinArgs: 0,
	}
}

func (c *appRemove) Run(context *cmd.Context, client *cmd.Client) error {
	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("Please use the -a/--app flag to specify which app you want to remove.")
	}
	if !c.Confirm(context, fmt.Sprintf(`Are you sure you want to remove app "%s"?`, appName)) {
		return nil
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s", appName))
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
	fmt.Fprintf(context.Stdout, `App "%s" successfully removed!`+"\n", appName)
	return nil
}

func (c *appRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = cmd.MergeFlagSet(
			c.GuessingCommand.Flags(),
			c.ConfirmationCommand.Flags(),
		)
	}
	return c.fs
}

type appInfo struct {
	cmd.GuessingCommand
}

func (c *appInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-info",
		Usage: "app-info [-a/--app appname]",
		Desc: `Shows information about a specific app. Its state, platform, git repository,
etc. You need to be a member of a team that has access to the app to be able to
see information about it.`,
		MinArgs: 0,
	}
}

func (c *appInfo) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	defer response.Body.Close()
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	url, err = cmd.GetURL(fmt.Sprintf("/docker/node/apps/%s/containers", appName))
	if err != nil {
		return err
	}
	request, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err = client.Do(request)
	var adminResult []byte
	if err == nil {
		defer response.Body.Close()
		adminResult, err = ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
	}
	url, err = cmd.GetURL(fmt.Sprintf("/services/instances?app=%s", appName))
	if err != nil {
		return err
	}
	request, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err = client.Do(request)
	var servicesResult []byte
	if err == nil {
		defer response.Body.Close()
		servicesResult, err = ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
	}
	return c.Show(result, adminResult, servicesResult, context)
}

type unit struct {
	Name   string
	Ip     string
	Status string
}

func (u *unit) Available() bool {
	return u.Status == "started" || u.Status == "unreachable"
}

type app struct {
	Ip         string
	CName      []string
	Name       string
	Platform   string
	Repository string
	Teams      []string
	Units      []unit
	Owner      string
	TeamOwner  string
	Deploys    uint
	containers []container
	services   []serviceData
	Plan       tsuruapp.Plan
}

type serviceData struct {
	Service   string
	Instances []string
}

type container struct {
	ID               string
	Type             string
	IP               string
	HostAddr         string
	HostPort         string
	SSHHostPort      string
	Status           string
	Version          string
	Image            string
	LastStatusUpdate time.Time
}

func (a *app) Addr() string {
	cnames := strings.Join(a.CName, ", ")
	if cnames != "" {
		return fmt.Sprintf("%s, %s", cnames, a.Ip)
	}
	return a.Ip
}

func (a *app) GetTeams() string {
	return strings.Join(a.Teams, ", ")
}

func (a *app) String() string {
	format := `Application: {{.Name}}
Repository: {{.Repository}}
Platform: {{.Platform}}
Teams: {{.GetTeams}}
Address: {{.Addr}}
Owner: {{.Owner}}
Team owner: {{.TeamOwner}}
Deploys: {{.Deploys}}
`
	tmpl := template.Must(template.New("app").Parse(format))
	units := cmd.NewTable()
	titles := []string{"Unit", "State"}
	contMap := map[string]container{}
	if len(a.containers) > 0 {
		for _, cont := range a.containers {
			id := cont.ID
			if len(cont.ID) > 10 {
				id = id[:10]
			}
			contMap[id] = cont
		}
		titles = append(titles, []string{"Host", "Port", "IP"}...)
	}
	units.Headers = cmd.Row(titles)
	for _, unit := range a.Units {
		if unit.Name != "" {
			id := unit.Name
			if len(unit.Name) > 10 {
				id = id[:10]
			}
			row := []string{id, unit.Status}
			cont, ok := contMap[id]
			if ok {
				row = append(row, []string{cont.HostAddr, cont.HostPort, cont.IP}...)
			}
			units.AddRow(cmd.Row(row))
		}
	}
	if len(a.containers) > 0 {
		units.SortByColumn(2)
	}
	servicesTable := cmd.NewTable()
	servicesTable.Headers = []string{"Service", "Instance"}
	for _, service := range a.services {
		if len(service.Instances) == 0 {
			continue
		}
		servicesTable.AddRow([]string{service.Service, strings.Join(service.Instances, ", ")})
	}
	if len(a.containers) > 0 {
		units.SortByColumn(2)
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, a)
	var suffix string
	if units.Rows() > 0 {
		suffix = fmt.Sprintf("Units: %d\n%s", units.Rows(), units)
	}
	if servicesTable.Rows() > 0 {
		suffix = fmt.Sprintf("%s\nService instances: %d\n%s", suffix, servicesTable.Rows(), servicesTable)
	}
	if a.Plan.Name != "" {
		suffix = fmt.Sprintf("%s\nApp Plan:\n%s", suffix, renderPlans([]tsuruapp.Plan{a.Plan}, true))
	}
	return buf.String() + suffix
}

func (c *appInfo) Show(result []byte, adminResult []byte, servicesResult []byte, context *cmd.Context) error {
	var a app
	err := json.Unmarshal(result, &a)
	if err != nil {
		return err
	}
	json.Unmarshal(adminResult, &a.containers)
	json.Unmarshal(servicesResult, &a.services)
	fmt.Fprintln(context.Stdout, &a)
	return nil
}

type appGrant struct {
	cmd.GuessingCommand
}

func (c *appGrant) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-grant",
		Usage: "app-grant <teamname> [-a/--app appname]",
		Desc: `Allows a team to access an application. You need to be a member of a team that
has access to the app to allow another team to access it. grants access to an
app to a team.`,
		MinArgs: 1,
	}
}

func (c *appGrant) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	teamName := context.Args[0]
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/teams/%s", appName, teamName))
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
	fmt.Fprintf(context.Stdout, `Team "%s" was added to the "%s" app`+"\n", teamName, appName)
	return nil
}

type appRevoke struct {
	cmd.GuessingCommand
}

func (c *appRevoke) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-revoke",
		Usage: "app-revoke <teamname> [-a/--app appname]",
		Desc: `Revokes the permission to access an application from a team. You need to have
access to the application to revoke access from a team.

An application cannot be orphaned, so it will always have at least one
authorized team.`,
		MinArgs: 1,
	}
}

func (c *appRevoke) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	teamName := context.Args[0]
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/teams/%s", appName, teamName))
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
	fmt.Fprintf(context.Stdout, `Team "%s" was removed from the "%s" app`+"\n", teamName, appName)
	return nil
}

type appList struct{}

func (c appList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/apps")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	defer response.Body.Close()
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return c.Show(result, context)
}

func (c appList) Show(result []byte, context *cmd.Context) error {
	var apps []app
	err := json.Unmarshal(result, &apps)
	if err != nil {
		return err
	}
	table := cmd.NewTable()
	table.Headers = cmd.Row([]string{"Application", "Units State Summary", "Address"})
	for _, app := range apps {
		var available int
		var total int
		for _, unit := range app.Units {
			if unit.Name != "" {
				total++
				if unit.Available() {
					available += 1
				}
			}
		}
		summary := fmt.Sprintf("%d of %d units in-service", available, total)
		addrs := strings.Replace(app.Addr(), ", ", "\n", -1)
		table.AddRow(cmd.Row([]string{app.Name, summary, addrs}))
	}
	table.LineSeparator = true
	table.Sort()
	context.Stdout.Write(table.Bytes())
	return nil
}

func (c appList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-list",
		Usage: "app-list",
		Desc: `Lists all apps that you have access to. App access is controlled by teams. If
your team has access to an app, then you have access to it.`,
	}
}

type appStop struct {
	cmd.GuessingCommand
}

func (c *appStop) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-stop",
		Usage:   "app-stop [-a/--app appname]",
		Desc:    `Stops an application.`,
		MinArgs: 0,
	}
}

func (c *appStop) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/stop", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, err = io.Copy(context.Stdout, response.Body)
	if err != nil {
		return err
	}
	return nil
}

type appStart struct {
	cmd.GuessingCommand
}

func (c *appStart) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-start",
		Usage:   "app-start [-a/--app appname]",
		Desc:    `Starts an application.`,
		MinArgs: 0,
	}
}

func (c *appStart) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/start", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, err = io.Copy(context.Stdout, response.Body)
	if err != nil {
		return err
	}
	return nil
}

type appRestart struct {
	cmd.GuessingCommand
}

func (c *appRestart) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/restart", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	w := tsuruIo.NewStreamWriter(context.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, response.Body) {
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

func (c *appRestart) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-restart",
		Usage:   "app-restart [-a/--app appname]",
		Desc:    `Restarts an application.`,
		MinArgs: 0,
	}
}

type cnameAdd struct {
	cmd.GuessingCommand
}

func (c *cnameAdd) Run(context *cmd.Context, client *cmd.Client) error {
	err := addCName(context.Args, c.GuessingCommand, client)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully defined.")
	return nil
}

func (c *cnameAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "cname-add",
		Usage: "cname-add <cname> [<cname> ...] [-a/--app appname]",
		Desc: `Adds a new CNAME to the application.

It will not manage any DNS register, it's up to the user to create the DNS
register. Once the app contains a custom CNAME, it will be displayed by "app-
list" and "app-info".`,
		MinArgs: 1,
	}
}

type cnameRemove struct {
	cmd.GuessingCommand
}

func (c *cnameRemove) Run(context *cmd.Context, client *cmd.Client) error {
	err := unsetCName(context.Args, c.GuessingCommand, client)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully undefined.")
	return nil
}

func (c *cnameRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "cname-remove",
		Usage: "cname-remove <cname> [<cname> ...] [-a/--app appname]",
		Desc: `Removes a CNAME from the application. This undoes the change that cname-add
does.

After unsetting the CNAME from the app, [[tsuru app-list]] and [[tsuru app-
info]] will display the internal, unfriendly address that tsuru uses.`,
		MinArgs: 1,
	}
}

func unsetCName(v []string, g cmd.GuessingCommand, client *cmd.Client) error {
	appName, err := g.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/cname", appName))
	if err != nil {
		return err
	}
	cnames := make(map[string][]string)
	cnames["cname"] = v
	c, err := json.Marshal(cnames)
	if err != nil {
		return err
	}
	body := bytes.NewReader(c)
	request, err := http.NewRequest("DELETE", url, body)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	return nil
}

func addCName(v []string, g cmd.GuessingCommand, client *cmd.Client) error {
	appName, err := g.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/cname", appName))
	if err != nil {
		return err
	}
	cnames := make(map[string][]string)
	cnames["cname"] = v
	c, err := json.Marshal(cnames)
	if err != nil {
		return err
	}
	body := bytes.NewReader(c)
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	return nil
}

type SetTeamOwner struct {
	cmd.GuessingCommand
}

func (c *SetTeamOwner) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.GuessingCommand.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/team-owner", appName))
	if err != nil {
		return err
	}
	body := strings.NewReader(context.Args[0])
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "app's owner team successfully changed.")
	return nil
}

func (c *SetTeamOwner) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-set-team-owner",
		Usage:   "app-set-team-owner <new-team-owner> [-a/--app appname]",
		Desc:    "Sets owner team for an application.",
		MinArgs: 1,
	}
}

type unitAdd struct {
	cmd.GuessingCommand
}

func (c *unitAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-add",
		Usage: "unit-add <# of units> [-a/--app appname]",
		Desc: `Adds new units (instances) to an application. You need to have access to the
app to be able to add new units to it.`,
		MinArgs: 1,
	}
}

func (c *unitAdd) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/units", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, bytes.NewBufferString(context.Args[0]))
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	w := tsuruIo.NewStreamWriter(context.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, response.Body) {
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

type unitRemove struct {
	cmd.GuessingCommand
}

func (c *unitRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-remove",
		Usage: "unit-remove <# of units> [-a/--app appname]",
		Desc: `Removes units (instances) from an application. You need to have access to the
app to be able to remove units from it.`,
		MinArgs: 1,
	}
}

func (c *unitRemove) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/units", appName))
	if err != nil {
		return err
	}
	body := bytes.NewBufferString(context.Args[0])
	request, err := http.NewRequest("DELETE", url, body)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Units successfully removed!")
	return nil
}
