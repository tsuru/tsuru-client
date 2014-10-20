// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
	"launchpad.net/gnuflag"
)

type AppCreate struct {
	teamOwner string
	plan      string
	fs        *gnuflag.FlagSet
}

func (c *AppCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-create",
		Usage:   "app-create <appname> <platform> [--plan/-p plan_name] [--team/-t (team owner)]",
		Desc:    "create a new app.",
		MinArgs: 2,
	}
}

func (c *AppCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		infoMessage := "The plan used to create the app"
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.StringVar(&c.plan, "plan", "", infoMessage)
		c.fs.StringVar(&c.plan, "p", "", infoMessage)
		teamMessage := "Team owner app"
		c.fs.StringVar(&c.teamOwner, "team", "", teamMessage)
		c.fs.StringVar(&c.teamOwner, "t", "", teamMessage)
	}
	return c.fs
}

func (c *AppCreate) Run(context *cmd.Context, client *cmd.Client) error {
	appName := context.Args[0]
	platform := context.Args[1]
	params := map[string]interface{}{
		"name":      appName,
		"platform":  platform,
		"plan":      map[string]interface{}{"name": c.plan},
		"teamOwner": c.teamOwner,
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
	fmt.Fprintf(context.Stdout, "App %q is being created!\n", appName)
	fmt.Fprintln(context.Stdout, "Use app-info to check the status of the app and its units.")
	fmt.Fprintf(context.Stdout, "Your repository for %q project is %q\n", appName, out["repository_url"])
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
		Usage: "app-remove [--app appname] [--assume-yes]",
		Desc: `removes an app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 0,
	}
}

func (c *AppRemove) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
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
		Usage: "app-info [--app appname]",
		Desc: `show information about your app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 0,
	}
}

func (c *AppInfo) Run(context *cmd.Context, client *cmd.Client) error {
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
	Ready      bool
	Owner      string
	TeamOwner  string
	Deploys    uint
	containers []container
	services   []serviceData
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

func (a *app) IsReady() string {
	if a.Ready {
		return "Yes"
	}
	return "No"
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
	var buf bytes.Buffer
	tmpl.Execute(&buf, a)
	var suffix string
	if units.Rows() > 0 {
		suffix = fmt.Sprintf("Units: %d\n%s", units.Rows(), units)
	}
	if servicesTable.Rows() > 0 {
		suffix = fmt.Sprintf("%s\nService instances: %d\n%s", suffix, servicesTable.Rows(), servicesTable)
	}
	return buf.String() + suffix
}

func (c *AppInfo) Show(result []byte, adminResult []byte, servicesResult []byte, context *cmd.Context) error {
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

type AppGrant struct {
	cmd.GuessingCommand
}

func (c *AppGrant) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-grant",
		Usage: "app-grant <teamname> [--app appname]",
		Desc: `grants access to an app to a team.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
}

func (c *AppGrant) Run(context *cmd.Context, client *cmd.Client) error {
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

type AppRevoke struct {
	cmd.GuessingCommand
}

func (c *AppRevoke) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-revoke",
		Usage: "app-revoke <teamname> [--app appname]",
		Desc: `revokes access to an app from a team.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
}

func (c *AppRevoke) Run(context *cmd.Context, client *cmd.Client) error {
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

type AppList struct{}

func (c AppList) Run(context *cmd.Context, client *cmd.Client) error {
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

func (c AppList) Show(result []byte, context *cmd.Context) error {
	var apps []app
	err := json.Unmarshal(result, &apps)
	if err != nil {
		return err
	}
	table := cmd.NewTable()
	table.Headers = cmd.Row([]string{"Application", "Units State Summary", "Address", "Ready?"})
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
		table.AddRow(cmd.Row([]string{app.Name, summary, addrs, app.IsReady()}))
	}
	table.LineSeparator = true
	table.Sort()
	context.Stdout.Write(table.Bytes())
	return nil
}

func (c AppList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-list",
		Usage: "app-list",
		Desc:  "list all your apps.",
	}
}

type AppStop struct {
	cmd.GuessingCommand
}

func (c *AppStop) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-stop",
		Usage: "app-stop [--app appname]",
		Desc: `stops an app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 0,
	}
}

func (c *AppStop) Run(context *cmd.Context, client *cmd.Client) error {
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

type AppStart struct {
	cmd.GuessingCommand
}

func (c *AppStart) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-start",
		Usage: "app-start [--app appname]",
		Desc: `starts an app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 0,
	}
}

func (c *AppStart) Run(context *cmd.Context, client *cmd.Client) error {
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

type AppRestart struct {
	cmd.GuessingCommand
}

func (c *AppRestart) Run(context *cmd.Context, client *cmd.Client) error {
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

func (c *AppRestart) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-restart",
		Usage: "app-restart [--app appname]",
		Desc: `restarts an app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 0,
	}
}

type AddCName struct {
	cmd.GuessingCommand
}

func (c *AddCName) Run(context *cmd.Context, client *cmd.Client) error {
	err := addCName(context.Args, c.GuessingCommand, client)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully defined.")
	return nil
}

func (c *AddCName) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "cname-add",
		Usage:   "cname-add <cname> [<cname> ...] [--app appname]",
		Desc:    `adds a cname for your app.`,
		MinArgs: 1,
	}
}

type RemoveCName struct {
	cmd.GuessingCommand
}

func (c *RemoveCName) Run(context *cmd.Context, client *cmd.Client) error {
	err := unsetCName(context.Args, c.GuessingCommand, client)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully undefined.")
	return nil
}

func (c *RemoveCName) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "cname-remove",
		Usage:   "cname-remove <cname> [<cname> ...] [--app appname]",
		Desc:    `removes cnames of your app.`,
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
		Usage:   "app-set-team-owner <new-team-owner> [--app appname]",
		Desc:    "set app's owner team",
		MinArgs: 1,
	}
}

type UnitAdd struct {
	cmd.GuessingCommand
}

func (c *UnitAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "unit-add",
		Usage:   "unit-add <# of units> [--app appname]",
		Desc:    "add new units to an app.",
		MinArgs: 1,
	}
}

func (c *UnitAdd) Run(context *cmd.Context, client *cmd.Client) error {
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

type UnitRemove struct {
	cmd.GuessingCommand
}

func (c *UnitRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "unit-remove",
		Usage:   "unit-remove <# of units> [--app appname]",
		Desc:    "remove units from an app.",
		MinArgs: 1,
	}
}

func (c *UnitRemove) Run(context *cmd.Context, client *cmd.Client) error {
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
