// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	tsuruapp "github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
	tsuruerr "github.com/tsuru/tsuru/errors"
)

type AppCreate struct {
	teamOwner   string
	plan        string
	router      string
	pool        string
	description string
	routerOpts  cmd.MapFlag
	fs          *gnuflag.FlagSet
}

func (c *AppCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-create",
		Usage: "app-create <appname> <platform> [--plan/-p plan_name] [--router/-r router_name] [--team/-t (team owner)] [--pool/-o pool_name] [--description/-d description] [--router-opts key=value]...",
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

The [[--router]] parameter defines the router to be used. The list of available
routers can be found running [[tsuru router-list]].

If this parameter is not informed, tsuru will choose the router with the
[[default]] flag set to true.

The [[--team]] parameter describes which team is responsible for the created
app, this is only needed if the current user belongs to more than one team, in
which case this parameter will be mandatory.

The [[--pool]] parameter defines which pool your app will be deployed.
This is only needed if you have more than one pool associated with your teams.

The [[--description]] parameter sets a description for your app.
It is an optional parameter, and if its not set the app will only not have a
description associated.

The [[--router-opts]] parameter allow passing custom parameters to the router
used by the application's plan. The key and values used depends on the router
implementation.`,
		MinArgs: 2,
	}
}

func (c *AppCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		infoMessage := "The plan used to create the app"
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.StringVar(&c.plan, "plan", "", infoMessage)
		c.fs.StringVar(&c.plan, "p", "", infoMessage)
		routerMessage := "The router used by the app"
		c.fs.StringVar(&c.router, "router", "", routerMessage)
		c.fs.StringVar(&c.router, "r", "", routerMessage)
		teamMessage := "Team owner app"
		c.fs.StringVar(&c.teamOwner, "team", "", teamMessage)
		c.fs.StringVar(&c.teamOwner, "t", "", teamMessage)
		poolMessage := "Pool to deploy your app"
		c.fs.StringVar(&c.pool, "pool", "", poolMessage)
		c.fs.StringVar(&c.pool, "o", "", poolMessage)
		descriptionMessage := "App description"
		c.fs.StringVar(&c.description, "description", "", descriptionMessage)
		c.fs.StringVar(&c.description, "d", "", descriptionMessage)
		c.fs.Var(&c.routerOpts, "router-opts", "Router options")
	}
	return c.fs
}

func (c *AppCreate) Run(context *cmd.Context, client *cmd.Client) error {
	appName := context.Args[0]
	platform := context.Args[1]
	v, err := form.EncodeToValues(map[string]interface{}{"routeropts": c.routerOpts})
	if err != nil {
		return err
	}
	v.Set("name", appName)
	v.Set("platform", platform)
	v.Set("plan", c.plan)
	v.Set("teamOwner", c.teamOwner)
	v.Set("pool", c.pool)
	v.Set("description", c.description)
	v.Set("router", c.router)
	b := strings.NewReader(v.Encode())
	u, err := cmd.GetURL("/apps")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

type AppUpdate struct {
	description string
	plan        string
	router      string
	pool        string
	teamOwner   string
	fs          *gnuflag.FlagSet
	cmd.GuessingCommand
	cmd.ConfirmationCommand
}

func (c *AppUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-update",
		Usage: "app-update [-a/--app appname] [--description/-d description] [--plan/-p plan_name] [--router/-r router_name] [--pool/-o pool] [--team-owner/-t team-owner]",
		Desc: `Updates an app, changing its description, plan or pool information.

The [[--description]] parameter sets a description for your app.

The [[--plan]] parameter changes the plan of your app.

The [[--router]] parameter changes the router of your app.

The [[--pool]] parameter changes the pool of your app.

The [[--team-owner]] parameter sets owner team for an application.`,
	}
}

func (c *AppUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		flagSet := gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		descriptionMessage := "App description"
		planMessage := "App plan"
		routerMessage := "App router"
		poolMessage := "App pool"
		teamOwnerMessage := "App team owner"
		flagSet.StringVar(&c.description, "description", "", descriptionMessage)
		flagSet.StringVar(&c.description, "d", "", descriptionMessage)
		flagSet.StringVar(&c.plan, "plan", "", planMessage)
		flagSet.StringVar(&c.plan, "p", "", planMessage)
		flagSet.StringVar(&c.router, "router", "", routerMessage)
		flagSet.StringVar(&c.router, "r", "", routerMessage)
		flagSet.StringVar(&c.pool, "o", "", poolMessage)
		flagSet.StringVar(&c.pool, "pool", "", poolMessage)
		flagSet.StringVar(&c.teamOwner, "t", "", teamOwnerMessage)
		flagSet.StringVar(&c.teamOwner, "team-owner", "", teamOwnerMessage)
		c.fs = cmd.MergeFlagSet(
			c.GuessingCommand.Flags(),
			flagSet,
		)
	}
	return c.fs
}

func (c *AppUpdate) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("Please use the -a/--app flag to specify which app you want to update.")
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s", appName))
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("plan", c.plan)
	v.Set("router", c.router)
	v.Set("description", c.description)
	v.Set("pool", c.pool)
	v.Set("teamOwner", c.teamOwner)
	request, err := http.NewRequest("PUT", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		e := err.(*tsuruerr.HTTP)
		if e.Code == http.StatusBadRequest {
			return errors.New("You must set a flag. Use the 'app-update --help' command for more information.")
		}
		return err
	}
	err = cmd.StreamJSONResponse(context.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "App %q has been updated!\n", appName)
	return nil
}

type AppRemove struct {
	cmd.GuessingCommand
	cmd.ConfirmationCommand
	fs *gnuflag.FlagSet
}

func (c *AppRemove) Info() *cmd.Info {
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

func (c *AppRemove) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("Please use the -a/--app flag to specify which app you want to remove.")
	}
	if !c.Confirm(context, fmt.Sprintf(`Are you sure you want to remove app "%s"?`, appName)) {
		return nil
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(context.Stdout, response)
}

func (c *AppRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = cmd.MergeFlagSet(
			c.GuessingCommand.Flags(),
			c.ConfirmationCommand.Flags(),
		)
	}
	return c.fs
}

type AppInfo struct {
	cmd.GuessingCommand
}

func (c *AppInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-info",
		Usage: "app-info [-a/--app appname]",
		Desc: `Shows information about a specific app. Its state, platform, git repository,
etc. You need to be a member of a team that has access to the app to be able to
see information about it.`,
		MinArgs: 0,
	}
}

func (c *AppInfo) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s", appName))
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
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	defer response.Body.Close()
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	u, err = cmd.GetURL(fmt.Sprintf("/services/instances?app=%s", appName))
	if err != nil {
		return err
	}
	request, err = http.NewRequest("GET", u, nil)
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
	var quota []byte
	u, err = cmd.GetURL("/apps/" + appName + "/quota")
	if err != nil {
		return err
	}
	request, _ = http.NewRequest("GET", u, nil)
	response, err = client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	quota, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return c.Show(result, servicesResult, quota, context)
}

type unit struct {
	ID          string
	IP          string
	Status      string
	ProcessName string
	Address     *url.URL
}

func (u *unit) Host() string {
	if u.Address == nil {
		return ""
	}
	host, _, _ := net.SplitHostPort(u.Address.Host)
	return host
}

func (u *unit) Port() string {
	if u.Address == nil {
		return ""
	}
	_, port, _ := net.SplitHostPort(u.Address.Host)
	return port
}

func (u *unit) Available() bool {
	return u.Status == "started"
}

type lock struct {
	Locked      bool
	Reason      string
	Owner       string
	AcquireDate time.Time
}

func (l *lock) String() string {
	format := `Lock:
 Acquired in: %s
 Owner: %s
 Running: %s`
	return fmt.Sprintf(format, l.AcquireDate, l.Owner, l.Reason)
}

type app struct {
	IP          string
	CName       []string
	Name        string
	Platform    string
	Repository  string
	Teams       []string
	Units       []unit
	Owner       string
	TeamOwner   string
	Deploys     uint
	Pool        string
	Description string
	Lock        lock
	services    []serviceData
	Quota       quota
	Plan        tsuruapp.Plan
	Router      string
}

type serviceData struct {
	Service   string
	Instances []string
	Plans     []string
}

type quota struct {
	Limit int
	InUse int
}

func (a *app) Addr() string {
	cnames := strings.Join(a.CName, ", ")
	if cnames != "" {
		return fmt.Sprintf("%s, %s", cnames, a.IP)
	}
	return a.IP
}

func (a *app) GetTeams() string {
	return strings.Join(a.Teams, ", ")
}

func (a *app) String() string {
	format := `Application: {{.Name}}
Description:{{if .Description}} {{.Description}}{{end}}
Repository: {{.Repository}}
Platform: {{.Platform}}
Router: {{.Router}}
Teams: {{.GetTeams}}
Address: {{.Addr}}
Owner: {{.Owner}}
Team owner: {{.TeamOwner}}
Deploys: {{.Deploys}}
Pool:{{if .Pool}} {{.Pool}}{{end}}{{if .Lock.Locked}}
{{.Lock.String}}{{end}}
Quota: {{.Quota.InUse}}/{{if .Quota.Limit}}{{.Quota.Limit}} units{{else}}unlimited{{end}}
`
	var buf bytes.Buffer
	tmpl := template.Must(template.New("app").Parse(format))
	unitsByProcess := map[string][]unit{}
	for _, u := range a.Units {
		units := unitsByProcess[u.ProcessName]
		unitsByProcess[u.ProcessName] = append(units, u)
	}
	processes := make([]string, 0, len(unitsByProcess))
	for process := range unitsByProcess {
		processes = append(processes, process)
	}
	sort.Strings(processes)
	titles := []string{"Unit", "State", "Host", "Port"}
	for _, process := range processes {
		units := unitsByProcess[process]
		unitsTable := cmd.NewTable()
		unitsTable.Headers = cmd.Row(titles)
		for _, unit := range units {
			if unit.ID == "" {
				continue
			}
			id := unit.ID
			if len(unit.ID) > 12 {
				id = id[:12]
			}
			row := []string{id, unit.Status, unit.Host(), unit.Port()}
			unitsTable.AddRow(cmd.Row(row))
		}
		if unitsTable.Rows() > 0 {
			unitsTable.SortByColumn(2)
			buf.WriteString("\n")
			processStr := ""
			if process != "" {
				processStr = fmt.Sprintf(" [%s]", process)
			}
			buf.WriteString(fmt.Sprintf("Units%s: %d\n", processStr, unitsTable.Rows()))
			buf.WriteString(unitsTable.String())
		}
	}
	servicesTable := cmd.NewTable()
	servicesTable.Headers = []string{"Service", "Instance (Plan)"}
	for _, service := range a.services {
		if len(service.Instances) == 0 {
			continue
		}
		var instancePlan []string
		for i, instance := range service.Instances {
			value := instance
			if i < len(service.Plans) && service.Plans[i] != "" {
				value = fmt.Sprintf("%s (%s)", instance, service.Plans[i])
			}
			instancePlan = append(instancePlan, value)
		}
		instancePlanString := strings.Join(instancePlan, "\n")
		servicesTable.AddRow([]string{service.Service, instancePlanString})
	}
	if servicesTable.Rows() > 0 {
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("Service instances: %d\n", servicesTable.Rows()))
		buf.WriteString(servicesTable.String())
	}
	if a.Plan.Name != "" {
		buf.WriteString("\n")
		buf.WriteString("App Plan:\n")
		buf.WriteString(renderPlans([]tsuruapp.Plan{a.Plan}, true))
	}
	var tplBuffer bytes.Buffer
	tmpl.Execute(&tplBuffer, a)
	return tplBuffer.String() + buf.String()
}

func (c *AppInfo) Show(result []byte, servicesResult []byte, quota []byte, context *cmd.Context) error {
	var a app
	err := json.Unmarshal(result, &a)
	if err != nil {
		return err
	}
	json.Unmarshal(servicesResult, &a.services)
	json.Unmarshal(quota, &a.Quota)
	fmt.Fprintln(context.Stdout, &a)
	return nil
}

type AppGrant struct {
	cmd.GuessingCommand
}

func (c *AppGrant) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-grant",
		Usage: "app-grant <teamname> [-a/--app appname]",
		Desc: `Allows a team to access an application. You need to be a member of a team that
has access to the app to allow another team to access it. grants access to an
app to a team.`,
		MinArgs: 1,
	}
}

func (c *AppGrant) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	teamName := context.Args[0]
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/teams/%s", appName, teamName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", u, nil)
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

type AppRevoke struct {
	cmd.GuessingCommand
}

func (c *AppRevoke) Info() *cmd.Info {
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

func (c *AppRevoke) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	teamName := context.Args[0]
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/teams/%s", appName, teamName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
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

type appFilter struct {
	name      string
	platform  string
	teamOwner string
	owner     string
	pool      string
	locked    bool
	status    string
}

func (f *appFilter) queryString(client *cmd.Client) (url.Values, error) {
	result := make(url.Values)
	if f.name != "" {
		result.Set("name", f.name)
	}
	if f.platform != "" {
		result.Set("platform", f.platform)
	}
	if f.teamOwner != "" {
		result.Set("teamOwner", f.teamOwner)
	}
	if f.owner != "" {
		owner := f.owner
		if owner == "me" {
			user, err := cmd.GetUser(client)
			if err != nil {
				return nil, err
			}
			owner = user.Email
		}
		result.Set("owner", owner)
	}
	if f.locked {
		result.Set("locked", "true")
	}
	if f.pool != "" {
		result.Set("pool", f.pool)
	}
	if f.status != "" {
		result.Set("status", f.status)
	}
	return result, nil
}

type AppList struct {
	fs         *gnuflag.FlagSet
	filter     appFilter
	simplified bool
}

func (c *AppList) Run(context *cmd.Context, client *cmd.Client) error {
	qs, err := c.filter.queryString(client)
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps?%s", qs.Encode()))
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

func (c *AppList) Show(result []byte, context *cmd.Context) error {
	var apps []app
	err := json.Unmarshal(result, &apps)
	if err != nil {
		return err
	}
	table := cmd.NewTable()
	if c.simplified {
		for _, app := range apps {
			fmt.Fprintln(context.Stdout, app.Name)
		}
		return nil
	}
	table.Headers = cmd.Row([]string{"Application", "Units State Summary", "Address"})
	for _, app := range apps {
		var available int
		var total int
		for _, unit := range app.Units {
			if unit.ID != "" {
				total++
				if unit.Available() {
					available++
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

func (c *AppList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("app-list", gnuflag.ExitOnError)
		c.fs.StringVar(&c.filter.name, "name", "", "Filter applications by name")
		c.fs.StringVar(&c.filter.name, "n", "", "Filter applications by name")
		c.fs.StringVar(&c.filter.pool, "pool", "", "Filter applications by pool")
		c.fs.StringVar(&c.filter.pool, "o", "", "Filter applications by pool")
		c.fs.StringVar(&c.filter.status, "status", "", "Filter applications by unit status. Accepts multiple values separated by commas. Possible values can be: building, created, starting, error, started, stopped, asleep")
		c.fs.StringVar(&c.filter.status, "s", "", "Filter applications by unit status. Accepts multiple values separated by commas. Possible values can be: building, created, starting, error, started, stopped, asleep")
		c.fs.StringVar(&c.filter.platform, "platform", "", "Filter applications by platform")
		c.fs.StringVar(&c.filter.platform, "p", "", "Filter applications by platform")
		c.fs.StringVar(&c.filter.teamOwner, "team", "", "Filter applications by team owner")
		c.fs.StringVar(&c.filter.teamOwner, "t", "", "Filter applications by team owner")
		c.fs.StringVar(&c.filter.owner, "user", "", "Filter applications by owner")
		c.fs.StringVar(&c.filter.owner, "u", "", "Filter applications by owner")
		c.fs.BoolVar(&c.filter.locked, "locked", false, "Filter applications by lock status")
		c.fs.BoolVar(&c.filter.locked, "l", false, "Filter applications by lock status")
		c.fs.BoolVar(&c.simplified, "q", false, "Display only applications name")
	}
	return c.fs
}

func (c *AppList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-list",
		Usage: "app-list",
		Desc: `Lists all apps that you have access to. App access is controlled by teams. If
your team has access to an app, then you have access to it.

Flags can be used to filter the list of applications.`,
	}
}

type AppStop struct {
	cmd.GuessingCommand
	process string
	fs      *gnuflag.FlagSet
}

func (c *AppStop) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-stop",
		Usage:   "app-stop [-a/--app appname] [-p/--process processname]",
		Desc:    "Stops an application, or one of the processes of the application.",
		MinArgs: 0,
	}
}

func (c *AppStop) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/stop", appName))
	if err != nil {
		return err
	}
	body := strings.NewReader("process=" + c.process)
	request, err := http.NewRequest("POST", u, body)
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

func (c *AppStop) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

type AppStart struct {
	cmd.GuessingCommand
	process string
	fs      *gnuflag.FlagSet
}

func (c *AppStart) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-start",
		Usage:   "app-start [-a/--app appname] [-p/--process processname]",
		Desc:    "Starts an application, or one of the processes of the application.",
		MinArgs: 0,
	}
}

func (c *AppStart) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/start", appName))
	if err != nil {
		return err
	}
	body := strings.NewReader("process=" + c.process)
	request, err := http.NewRequest("POST", u, body)
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

func (c *AppStart) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

type AppRestart struct {
	cmd.GuessingCommand
	process string
	fs      *gnuflag.FlagSet
}

func (c *AppRestart) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/restart", appName))
	if err != nil {
		return err
	}
	body := strings.NewReader("process=" + c.process)
	request, err := http.NewRequest("POST", u, body)
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

func (c *AppRestart) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-restart",
		Usage:   "app-restart [-a/--app appname] [-p/--process processname]",
		Desc:    `Restarts an application, or one of the processes of the application.`,
		MinArgs: 0,
	}
}

func (c *AppRestart) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

type CnameAdd struct {
	cmd.GuessingCommand
}

func (c *CnameAdd) Run(context *cmd.Context, client *cmd.Client) error {
	err := addCName(context.Args, c.GuessingCommand, client)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully defined.")
	return nil
}

func (c *CnameAdd) Info() *cmd.Info {
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

type CnameRemove struct {
	cmd.GuessingCommand
}

func (c *CnameRemove) Run(context *cmd.Context, client *cmd.Client) error {
	err := unsetCName(context.Args, c.GuessingCommand, client)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully undefined.")
	return nil
}

func (c *CnameRemove) Info() *cmd.Info {
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

func unsetCName(cnames []string, g cmd.GuessingCommand, client *cmd.Client) error {
	appName, err := g.Guess()
	if err != nil {
		return err
	}
	v := url.Values{}
	for _, cname := range cnames {
		v.Add("cname", cname)
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/cname?%s", appName, v.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	return err
}

func addCName(cnames []string, g cmd.GuessingCommand, client *cmd.Client) error {
	appName, err := g.Guess()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/cname", appName))
	if err != nil {
		return err
	}
	v := url.Values{}
	for _, cname := range cnames {
		v.Add("cname", cname)
	}
	b := strings.NewReader(v.Encode())
	request, err := http.NewRequest("POST", u, b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	return err
}

type UnitAdd struct {
	cmd.GuessingCommand
	fs      *gnuflag.FlagSet
	process string
}

func (c *UnitAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-add",
		Usage: "unit-add <# of units> [-a/--app appname] [-p/--process processname]",
		Desc: `Adds new units to a process of an application. You need to have access to the
app to be able to add new units to it.`,
		MinArgs: 1,
	}
}

func (c *UnitAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

func (c *UnitAdd) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/units", appName))
	if err != nil {
		return err
	}
	val := url.Values{}
	val.Add("units", context.Args[0])
	val.Add("process", c.process)
	request, err := http.NewRequest("PUT", u, bytes.NewBufferString(val.Encode()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return cmd.StreamJSONResponse(context.Stdout, response)
}

type UnitRemove struct {
	cmd.GuessingCommand
	fs      *gnuflag.FlagSet
	process string
}

func (c *UnitRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-remove",
		Usage: "unit-remove <# of units> [-a/--app appname] [-p/-process processname]",
		Desc: `Removes units from a process of an application. You need to have access to the
app to be able to remove units from it.`,
		MinArgs: 1,
	}
}

func (c *UnitRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

func (c *UnitRemove) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	val := url.Values{}
	val.Add("units", context.Args[0])
	val.Add("process", c.process)
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/units?%s", appName, val.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(context.Stdout, response)
}
