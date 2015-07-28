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
	"net/url"
	"sort"
	"strings"
	"text/template"
	"time"

	tsuruapp "github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
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
	Name        string
	Ip          string
	Status      string
	ProcessName string
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
	Pool       string
	Lock       lock
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
Pool: {{.Pool}}{{if .Lock.Locked}}
{{.Lock.String}}{{end}}
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
	for _, process := range processes {
		units := unitsByProcess[process]
		unitsTable := cmd.NewTable()
		unitsTable.Headers = cmd.Row(titles)
		for _, unit := range units {
			if unit.Name == "" {
				continue
			}
			id := unit.Name
			if len(unit.Name) > 10 {
				id = id[:10]
			}
			row := []string{id, unit.Status}
			cont, ok := contMap[id]
			if ok {
				row = append(row, []string{cont.HostAddr, cont.HostPort, cont.IP}...)
			}
			unitsTable.AddRow(cmd.Row(row))
		}
		if unitsTable.Rows() > 0 {
			if len(a.containers) > 0 {
				unitsTable.SortByColumn(2)
			}
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
	servicesTable.Headers = []string{"Service", "Instance"}
	for _, service := range a.services {
		if len(service.Instances) == 0 {
			continue
		}
		servicesTable.AddRow([]string{service.Service, strings.Join(service.Instances, ", ")})
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

type appFilter struct {
	name      string
	platform  string
	teamOwner string
	owner     string
	locked    bool
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
		result.Set("teamowner", f.teamOwner)
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
	return result, nil
}

type appList struct {
	fs     *gnuflag.FlagSet
	filter appFilter
}

func (c *appList) Run(context *cmd.Context, client *cmd.Client) error {
	qs, err := c.filter.queryString(client)
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps?%s", qs.Encode()))
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

func (c *appList) Show(result []byte, context *cmd.Context) error {
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

func (c *appList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("app-list", gnuflag.ExitOnError)
		c.fs.StringVar(&c.filter.name, "name", "", "Filter applications by name")
		c.fs.StringVar(&c.filter.name, "n", "", "Filter applications by name")
		c.fs.StringVar(&c.filter.platform, "platform", "", "Display only applications that use the given platform")
		c.fs.StringVar(&c.filter.platform, "p", "", "Display only applications that use the given platform")
		c.fs.StringVar(&c.filter.teamOwner, "team", "", "Display only applications owned by the given team")
		c.fs.StringVar(&c.filter.teamOwner, "t", "", "Display only applications owned by the given team")
		c.fs.StringVar(&c.filter.owner, "user", "", "Display only applications owner by the given user")
		c.fs.StringVar(&c.filter.owner, "u", "", "Display only applications owner by the given user")
		c.fs.BoolVar(&c.filter.locked, "locked", false, "Display only applications that are locked")
		c.fs.BoolVar(&c.filter.locked, "l", false, "Display only applications that are locked")
	}
	return c.fs
}

func (c *appList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-list",
		Usage: "app-list",
		Desc: `Lists all apps that you have access to. App access is controlled by teams. If
your team has access to an app, then you have access to it.

Flags can be used to filter the list of applications.`,
	}
}

type appStop struct {
	cmd.GuessingCommand
	process string
	fs      *gnuflag.FlagSet
}

func (c *appStop) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-stop",
		Usage:   "app-stop [-a/--app appname] [-p/--process processname]",
		Desc:    "Stops an application, or one of the processes of the application.",
		MinArgs: 0,
	}
}

func (c *appStop) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/stop?process=%s", appName, c.process))
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

func (c *appStop) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

type appStart struct {
	cmd.GuessingCommand
	process string
	fs      *gnuflag.FlagSet
}

func (c *appStart) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-start",
		Usage:   "app-start [-a/--app appname] [-p/--process processname]",
		Desc:    "Starts an application, or one of the processes of the application.",
		MinArgs: 0,
	}
}

func (c *appStart) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/start?process=%s", appName, c.process))
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

func (c *appStart) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

type appRestart struct {
	cmd.GuessingCommand
	process string
	fs      *gnuflag.FlagSet
}

func (c *appRestart) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/restart?process=%s", appName, c.process))
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
	return cmd.StreamJSONResponse(context.Stdout, response)
}

func (c *appRestart) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-restart",
		Usage:   "app-restart [-a/--app appname] [-p/--process processname]",
		Desc:    `Restarts an application, or one of the processes of the application.`,
		MinArgs: 0,
	}
}

func (c *appRestart) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
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
	fs      *gnuflag.FlagSet
	process string
}

func (c *unitAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-add",
		Usage: "unit-add <# of units> [-a/--app appname] [-p/--process processname]",
		Desc: `Adds new units to a process of an application. You need to have access to the
app to be able to add new units to it.`,
		MinArgs: 1,
	}
}

func (c *unitAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

func (c *unitAdd) Run(context *cmd.Context, client *cmd.Client) error {
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

type unitRemove struct {
	cmd.GuessingCommand
	fs      *gnuflag.FlagSet
	process string
}

func (c *unitRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-remove",
		Usage: "unit-remove <# of units> [-a/--app appname] [-p/-process processname]",
		Desc: `Removes units from a process of an application. You need to have access to the
app to be able to remove units from it.`,
		MinArgs: 1,
	}
}

func (c *unitRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
	}
	return c.fs
}

func (c *unitRemove) Run(context *cmd.Context, client *cmd.Client) error {
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
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(context.Stdout, response)
}

type appPoolChange struct {
	cmd.GuessingCommand
}

func (a *appPoolChange) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-pool-change",
		Usage:   "app-pool-change <pool_name> [-a/--app appname]",
		Desc:    `Change app pool. You need to have access to the pool to be able to do it.`,
		MinArgs: 1,
	}
}

func (a *appPoolChange) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := a.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/pool", appName))
	if err != nil {
		return err
	}
	body := bytes.NewBufferString(context.Args[0])
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Pool successfully changed!")
	return nil
}
