// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
	apptypes "github.com/tsuru/tsuru/types/app"
	quotaTypes "github.com/tsuru/tsuru/types/quota"
	volumeTypes "github.com/tsuru/tsuru/types/volume"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/duration"
)

const (
	cutoffHexID = 12
)

var hexRegex = regexp.MustCompile(`(?i)^[a-f0-9]+$`)

type AppCreate struct {
	teamOwner   string
	plan        string
	router      string
	pool        string
	description string
	tags        cmd.StringSliceFlag
	routerOpts  cmd.MapFlag
	fs          *gnuflag.FlagSet
}

type unitSorter struct {
	Statuses []string
	Counts   []int
}

func (u *unitSorter) Len() int {
	return len(u.Statuses)
}

func (u *unitSorter) Swap(i, j int) {
	u.Statuses[i], u.Statuses[j] = u.Statuses[j], u.Statuses[i]
	u.Counts[i], u.Counts[j] = u.Counts[j], u.Counts[i]
}

func (u *unitSorter) Less(i, j int) bool {
	if u.Counts[i] > u.Counts[j] {
		return true
	}
	if u.Counts[i] == u.Counts[j] {
		return u.Statuses[i] < u.Statuses[j]
	}
	return false
}

func newUnitSorter(m map[string]int) *unitSorter {
	us := &unitSorter{
		Statuses: make([]string, 0, len(m)),
		Counts:   make([]int, 0, len(m)),
	}
	for k, v := range m {
		us.Statuses = append(us.Statuses, k)
		us.Counts = append(us.Counts, v)
	}
	return us
}

func (c *AppCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-create",
		Usage: "app create <appname> [platform] [--plan/-p plan name] [--router/-r router name] [--team/-t team owner] [--pool/-o pool name] [--description/-d description] [--tag/-g tag]... [--router-opts key=value]...",
		Desc: `Creates a new app using the given name and platform. For tsuru,
a platform is provisioner dependent. To check the available platforms, use the
command [[tsuru platform list]] and to add a platform use the command [[tsuru platform add]].

In order to create an app, you need to be member of at least one team. All
teams that you are member (see [[tsuru team-list]]) will be able to access the
app.

The [[--platform]] parameter is the name of the platform to be used when
creating the app. This will define how tsuru understands and executes your
app. The list of available platforms can be found running [[tsuru platform list]].

The [[--plan]] parameter defines the plan to be used. The plan specifies how
computational resources are allocated to your application. Typically this
means limits for memory and swap usage, and how much cpu share is allocated.
The list of available plans can be found running [[tsuru plan list]].

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

The [[--tag]] parameter sets a tag to your app. You can set multiple [[--tag]] parameters.

The [[--router-opts]] parameter allow passing custom parameters to the router
used by the application's plan. The key and values used depends on the router
implementation.`,
		MinArgs: 1,
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
		tagMessage := "App tag"
		c.fs.Var(&c.tags, "tag", tagMessage)
		c.fs.Var(&c.tags, "g", tagMessage)
		c.fs.Var(&c.routerOpts, "router-opts", "Router options")
	}
	return c.fs
}

func (c *AppCreate) Run(context *cmd.Context, client *cmd.Client) error {
	var platform string
	appName := context.Args[0]
	if len(context.Args) > 1 {
		platform = context.Args[1]
	}
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
	for _, tag := range c.tags {
		v.Add("tag", tag)
	}
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
	fmt.Fprintln(context.Stdout, "Use app info to check the status of the app and its units.")
	return nil
}

type AppUpdate struct {
	args tsuru.UpdateApp
	fs   *gnuflag.FlagSet
	cmd.AppNameMixIn
	cmd.ConfirmationCommand

	memory, cpu string
}

func (c *AppUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-update",
		Usage: "app update [-a/--app appname] [--description/-d description] [--plan/-p plan name] [--pool/-o pool] [--team-owner/-t team owner] [--platform/-l platform] [-i/--image-reset] [--cpu cpu] [--memory memory] [--tag/-g tag]...",
		Desc:  `Updates an app, changing its description, tags, plan or pool information.`,
	}
}

func (c *AppUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		flagSet := gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		descriptionMessage := "Changes description for the app"
		planMessage := "Changes plan for the app"
		poolMessage := "Changes pool for the app"
		teamOwnerMessage := "Changes owner team for the app"
		tagMessage := "Add tags for the app. You can add multiple tags repeating the --tag argument"
		platformMsg := "Changes platform for the app"
		imgReset := "Forces next deploy to build app image from scratch"
		noRestartMessage := "Prevent tsuru from restarting the application"
		flagSet.StringVar(&c.args.Description, "description", "", descriptionMessage)
		flagSet.StringVar(&c.args.Description, "d", "", descriptionMessage)
		flagSet.StringVar(&c.args.Plan, "plan", "", planMessage)
		flagSet.StringVar(&c.args.Plan, "p", "", planMessage)
		flagSet.StringVar(&c.args.Platform, "l", "", platformMsg)
		flagSet.StringVar(&c.args.Platform, "platform", "", platformMsg)
		flagSet.StringVar(&c.args.Pool, "o", "", poolMessage)
		flagSet.StringVar(&c.args.Pool, "pool", "", poolMessage)
		flagSet.BoolVar(&c.args.ImageReset, "i", false, imgReset)
		flagSet.BoolVar(&c.args.ImageReset, "image-reset", false, imgReset)
		flagSet.BoolVar(&c.args.NoRestart, "no-restart", false, noRestartMessage)
		flagSet.StringVar(&c.args.TeamOwner, "t", "", teamOwnerMessage)
		flagSet.StringVar(&c.args.TeamOwner, "team-owner", "", teamOwnerMessage)
		flagSet.Var((*cmd.StringSliceFlag)(&c.args.Tags), "g", tagMessage)
		flagSet.Var((*cmd.StringSliceFlag)(&c.args.Tags), "tag", tagMessage)
		flagSet.StringVar(&c.cpu, "cpu", "", "CPU limit for app, this will override the plan cpu value. One cpu is equivalent to 1 vCPU/Core, fractional requests are allowed and the expression 0.1 is equivalent to the expression 100m")
		flagSet.StringVar(&c.memory, "memory", "", "Memory limit for app, this will override the plan memory value. You can express memory as a bytes integer or using one of these suffixes: E, P, T, G, M, K, Ei, Pi, Ti, Gi, Mi, Ki")
		c.fs = cmd.MergeFlagSet(
			c.AppNameMixIn.Flags(),
			flagSet,
		)
	}
	return c.fs
}

func (c *AppUpdate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()

	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}

	if c.cpu != "" {
		var cpuQuantity resource.Quantity
		cpuQuantity, err = resource.ParseQuantity(c.cpu)
		if err != nil {
			return err
		}
		milliValue := int(cpuQuantity.MilliValue())
		c.args.Planoverride.Cpumilli = &milliValue
	}

	if c.memory != "" {
		var memoryQuantity resource.Quantity
		memoryQuantity, err = resource.ParseQuantity(c.memory)
		if err != nil {
			return err
		}
		val := memoryQuantity.Value()
		c.args.Planoverride.Memory = &val
	}

	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("Please use the -a/--app flag to specify which app you want to update.")
	}

	response, err := apiClient.AppApi.AppUpdate(context.TODO(), appName, c.args)
	if err != nil {
		return err
	}

	err = cmd.StreamJSONResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "App %q has been updated!\n", appName)
	return nil
}

type AppRemove struct {
	cmd.AppNameMixIn
	cmd.ConfirmationCommand
	fs *gnuflag.FlagSet
}

func (c *AppRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-remove",
		Usage: "app remove [-a/--app appname] [-y/--assume-yes]",
		Desc: `Removes an application. If the app is bound to any service instance, all binds
will be removed before the app gets deleted (see [[tsuru service-unbind]]).

You need to be a member of a team that has access to the app to be able to
remove it (you are able to remove any app that you see in [[tsuru app list]]).`,
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
			c.AppNameMixIn.Flags(),
			c.ConfirmationCommand.Flags(),
		)
	}
	return c.fs
}

type AppInfo struct {
	cmd.AppNameMixIn

	json         bool
	simplified   bool
	flagsApplied bool
}

func (c *AppInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-info",
		Usage: "app info [-a/--app appname]",
		Desc: `Shows information about a specific app. Its state, platform, git repository,
etc. You need to be a member of a team that has access to the app to be able to
see information about it.`,
		MinArgs: 0,
	}
}

func (cmd *AppInfo) Flags() *gnuflag.FlagSet {
	fs := cmd.AppNameMixIn.Flags()
	if !cmd.flagsApplied {
		fs.BoolVar(&cmd.simplified, "simplified", false, "Show simplified view of app")
		fs.BoolVar(&cmd.simplified, "s", false, "Show simplified view of app")
		fs.BoolVar(&cmd.json, "json", false, "Show JSON view of app")

		cmd.flagsApplied = true
	}
	return fs
}

func (c *AppInfo) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
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
	var a app
	err = json.NewDecoder(response.Body).Decode(&a)
	if err != nil {
		return err
	}
	return c.Show(&a, context, c.simplified)
}

type unit struct {
	ID           string
	IP           string
	Status       string
	StatusReason string
	ProcessName  string
	Address      *url.URL
	Addresses    []url.URL
	Version      int
	Routable     *bool
	Ready        *bool
	Restarts     *int
	CreatedAt    *time.Time
}

func (u *unit) Host() string {
	address := ""
	if len(u.Addresses) > 0 {
		address = u.Addresses[0].Host
	} else if u.Address != nil {
		address = u.Address.Host
	}
	if address == "" {
		return address
	}

	host, _, _ := net.SplitHostPort(address)
	return host

}

func (u *unit) ReadyAndStatus() string {
	if u.Ready != nil && *u.Ready {
		return "ready"
	}

	if u.StatusReason != "" {
		return u.Status + " (" + u.StatusReason + ")"
	}

	return u.Status
}

func (u *unit) Port() string {
	if len(u.Addresses) == 0 {
		if u.Address == nil {
			return ""
		}
		_, port, _ := net.SplitHostPort(u.Address.Host)
		return port
	}

	ports := []string{}
	for _, addr := range u.Addresses {
		_, port, _ := net.SplitHostPort(addr.Host)
		ports = append(ports, port)
	}
	return strings.Join(ports, ", ")
}

type unitMetrics struct {
	ID     string
	CPU    string
	Memory string
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
	Provisioner string
	Cluster     string
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
	Quota       quotaTypes.Quota
	Plan        apptypes.Plan
	Router      string
	RouterOpts  map[string]string
	Tags        []string
	Error       string
	Routers     []apptypes.AppRouter
	AutoScale   []tsuru.AutoScaleSpec

	InternalAddresses    []appInternalAddress
	UnitsMetrics         []unitMetrics
	VolumeBinds          []volumeTypes.VolumeBind
	ServiceInstanceBinds []serviceInstanceBind
}

type serviceInstanceBind struct {
	Service  string
	Instance string
	Plan     string
}

type appInternalAddress struct {
	Domain   string
	Protocol string
	Port     int
	Version  string
	Process  string
}

func (a *app) QuotaString() string {
	var limit strings.Builder
	if a.Quota.IsUnlimited() {
		limit.WriteString("unlimited")
	} else {
		fmt.Fprintf(&limit, "%d units", a.Quota.Limit)
	}
	return fmt.Sprintf("%d/%s", a.Quota.InUse, limit.String())
}

func (a *app) TeamList() string {
	teams := []string{}
	if a.TeamOwner != "" {
		teams = append(teams, a.TeamOwner+" (owner)")
	}

	for _, t := range a.Teams {
		if t != a.TeamOwner {
			teams = append(teams, t)
		}
	}

	return strings.Join(teams, ", ")

}

func (a *app) InternalAddr() string {

	addrs := []string{}
	for _, a := range a.InternalAddresses {
		if a.Protocol == "UDP" {
			addrs = append(addrs, fmt.Sprintf("%s:%d (UDP)", a.Domain, a.Port))
		} else {
			addrs = append(addrs, fmt.Sprintf("%s:%d", a.Domain, a.Port))
		}
	}

	return strings.Join(addrs, ", ")
}
func (a *app) Addr() string {
	var allAddrs []string
	for _, cname := range a.CName {
		if cname != "" {
			allAddrs = append(allAddrs, cname+" (cname)")
		}
	}
	if len(a.Routers) == 0 {
		if a.IP != "" {
			allAddrs = append(allAddrs, a.IP)
		}
	} else {
		for _, r := range a.Routers {
			if len(r.Addresses) > 0 {
				sort.Strings(r.Addresses)
				allAddrs = append(allAddrs, r.Addresses...)
			} else if r.Address != "" {
				allAddrs = append(allAddrs, r.Address)
			}
		}
	}
	return strings.Join(allAddrs, ", ")
}

func (a *app) TagList() string {
	return strings.Join(a.Tags, ", ")
}

func (a *app) GetRouterOpts() string {
	var kv []string
	for k, v := range a.RouterOpts {
		kv = append(kv, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(kv)
	return strings.Join(kv, ", ")
}

func ShortID(id string) string {
	if hexRegex.MatchString(id) && len(id) > cutoffHexID {
		return id[:cutoffHexID]
	}
	return id
}

const simplifiedFormat = `{{ if .Error -}}
Error: {{ .Error }}
{{ end -}}
Application: {{.Name}}
{{- if .Description }}
Description: {{.Description}}
{{- end }}
{{- if .TagList }}
Tags: {{.TagList}}
{{- end }}
Created by: {{.Owner}}
Platform: {{.Platform}}
Plan: {{ .Plan.Name }}
Pool: {{.Pool}} ({{ .Provisioner }}{{ if .Cluster}} | cluster: {{ .Cluster }}{{end}})
{{if not .Routers -}}
Router:{{if .Router}} {{.Router}}{{if .RouterOpts}} ({{.GetRouterOpts}}){{end}}{{end}}
{{end -}}
Teams: {{.TeamList}}
{{- if .InternalAddr }}
Cluster Internal Addresses: {{.InternalAddr}}
{{- end }}
{{- if .Addr }}
Cluster External Addresses: {{.Addr}}
{{- end }}
{{- if .SimpleServicesView }}
Bound Services: {{ .SimpleServicesView }}
{{- end }}
`

const fullFormat = `{{ if .Error -}}
Error: {{ .Error }}
{{ end -}}
Application: {{.Name}}
{{- if .Description }}
Description: {{.Description}}
{{- end }}
{{- if .TagList }}
Tags: {{.TagList}}
{{- end }}
Platform: {{.Platform}}
{{ if .Provisioner -}}
Provisioner: {{ .Provisioner }}
{{ end -}}
{{if not .Routers -}}
Router:{{if .Router}} {{.Router}}{{if .RouterOpts}} ({{.GetRouterOpts}}){{end}}{{end}}
{{end -}}
Teams: {{.TeamList}}
External Addresses: {{.Addr}}
Created by: {{.Owner}}
Deploys: {{.Deploys}}
{{if .Cluster -}}
Cluster: {{ .Cluster }}
{{ end -}}
Pool:{{if .Pool}} {{.Pool}}{{end}}{{if .Lock.Locked}}
{{.Lock.String}}{{end}}
Quota: {{ .QuotaString }}
`

func (a *app) String(simplified bool) string {
	var format string

	if simplified {
		format = simplifiedFormat
	} else {
		format = fullFormat
	}

	var buf bytes.Buffer
	tmpl := template.Must(template.New("app").Parse(format))

	if simplified {
		renderUnitsSummary(&buf, a.Units, a.UnitsMetrics, a.Provisioner)
	} else {
		renderUnits(&buf, a.Units, a.UnitsMetrics, a.Provisioner)
	}

	internalAddressesTable := tablecli.NewTable()
	internalAddressesTable.Headers = []string{"Domain", "Port", "Process", "Version"}
	for _, internalAddress := range a.InternalAddresses {
		internalAddressesTable.AddRow([]string{
			internalAddress.Domain,
			strconv.Itoa(internalAddress.Port) + "/" + internalAddress.Protocol,
			internalAddress.Process,
			internalAddress.Version,
		})
	}

	if !simplified {
		renderServiceInstanceBinds(&buf, a.ServiceInstanceBinds)
	}

	autoScaleTable := tablecli.NewTable()
	autoScaleTable.Headers = tablecli.Row([]string{"Process", "Min", "Max", "Target CPU"})
	for _, as := range a.AutoScale {
		cpu := cpuValue(as.AverageCPU)
		autoScaleTable.AddRow(tablecli.Row([]string{
			fmt.Sprintf("%s (v%d)", as.Process, as.Version),
			strconv.Itoa(int(as.MinUnits)),
			strconv.Itoa(int(as.MaxUnits)),
			cpu,
		}))
	}

	if autoScaleTable.Rows() > 0 {
		buf.WriteString("\n")
		buf.WriteString("Auto Scale:\n")
		buf.WriteString(autoScaleTable.String())
	}

	if !simplified && (a.Plan.Memory != 0 || a.Plan.Swap != 0 || a.Plan.CpuShare != 0) {
		buf.WriteString("\n")
		buf.WriteString("App Plan:\n")
		buf.WriteString(renderPlans([]apptypes.Plan{a.Plan}, false, false))
	}
	if !simplified && internalAddressesTable.Rows() > 0 {
		buf.WriteString("\n")
		buf.WriteString("Cluster internal addresses:\n")
		buf.WriteString(internalAddressesTable.String())
	}
	if !simplified && len(a.Routers) > 0 {
		buf.WriteString("\n")
		if a.Provisioner == "kubernetes" {
			buf.WriteString("Cluster external addresses:\n")
			renderRouters(a.Routers, &buf, "Router")
		} else {
			buf.WriteString("Routers:\n")
			renderRouters(a.Routers, &buf, "Name")
		}
	}

	renderVolumeBinds(&buf, a.VolumeBinds)

	var tplBuffer bytes.Buffer
	tmpl.Execute(&tplBuffer, a)
	return tplBuffer.String() + buf.String()
}

func (a *app) SimpleServicesView() string {
	sibs := make([]serviceInstanceBind, len(a.ServiceInstanceBinds))
	copy(sibs, a.ServiceInstanceBinds)

	sort.Slice(sibs, func(i, j int) bool {
		if sibs[i].Service < sibs[j].Service {
			return true
		}
		if sibs[i].Service > sibs[j].Service {
			return false
		}
		return sibs[i].Instance < sibs[j].Instance
	})
	pairs := []string{}
	for _, b := range sibs {
		pairs = append(pairs, b.Service+"/"+b.Instance)
	}

	return strings.Join(pairs, ", ")
}

func renderUnitsSummary(buf *bytes.Buffer, units []unit, metrics []unitMetrics, provisioner string) {
	type unitsKey struct {
		process  string
		version  int
		routable bool
	}
	groupedUnits := map[unitsKey][]unit{}
	for _, u := range units {
		routable := false
		if u.Routable != nil {
			routable = *u.Routable
		}
		key := unitsKey{process: u.ProcessName, version: u.Version, routable: routable}
		groupedUnits[key] = append(groupedUnits[key], u)
	}
	keys := make([]unitsKey, 0, len(groupedUnits))
	for key := range groupedUnits {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].version == keys[j].version {
			return keys[i].process < keys[j].process
		}
		return keys[i].version < keys[j].version
	})
	var titles []string
	if provisioner == "kubernetes" {
		titles = []string{"Process", "Ready", "Restarts", "Avg CPU (abs)", "Avg Memory"}
	} else {
		titles = []string{"Process", "Units"}
	}
	unitsTable := tablecli.NewTable()
	tablecli.TableConfig.ForceWrap = false
	unitsTable.Headers = tablecli.Row(titles)

	fmt.Fprintf(buf, "Units: %d\n", len(units))

	if len(units) == 0 {
		return
	}
	mapUnitMetrics := map[string]unitMetrics{}
	for _, unitMetric := range metrics {
		mapUnitMetrics[unitMetric.ID] = unitMetric
	}

	for _, key := range keys {
		summaryTitle := key.process
		if key.version > 0 {
			summaryTitle = fmt.Sprintf("%s (v%d)", key.process, key.version)
		}

		summaryUnits := groupedUnits[key]

		if !key.routable && provisioner == "kubernetes" {
			summaryTitle = summaryTitle + " (unroutable)"
		}

		readyUnits := 0
		restarts := 0
		cpuTotal := resource.NewQuantity(0, resource.DecimalSI)
		memoryTotal := resource.NewQuantity(0, resource.BinarySI)

		for _, unit := range summaryUnits {
			if unit.Ready != nil && *unit.Ready {
				readyUnits += 1
			}

			if unit.Restarts != nil {
				restarts += *unit.Restarts
			}

			unitMetric := mapUnitMetrics[unit.ID]
			qt, err := resource.ParseQuantity(unitMetric.CPU)
			if err == nil {
				cpuTotal.Add(qt)
			}
			qt, err = resource.ParseQuantity(unitMetric.Memory)
			if err == nil {
				memoryTotal.Add(qt)
			}
		}

		if provisioner == "kubernetes" {
			unitsTable.AddRow(tablecli.Row{
				summaryTitle,
				fmt.Sprintf("%d/%d", readyUnits, len(summaryUnits)),
				fmt.Sprintf("%d", restarts),
				fmt.Sprintf("%d%%", cpuTotal.MilliValue()/int64(10)/int64(len(summaryUnits))),
				fmt.Sprintf("%vMi", memoryTotal.Value()/int64(1024*1024)/int64(len(summaryUnits))),
			})
		} else {
			unitsTable.AddRow(tablecli.Row{
				summaryTitle,
				fmt.Sprintf("%d", len(summaryUnits)),
			})
		}
	}
	buf.WriteString(unitsTable.String())
}

func renderUnits(buf *bytes.Buffer, units []unit, metrics []unitMetrics, provisioner string) {
	type unitsKey struct {
		process  string
		version  int
		routable bool
	}
	groupedUnits := map[unitsKey][]unit{}
	for _, u := range units {
		routable := false
		if u.Routable != nil {
			routable = *u.Routable
		}
		key := unitsKey{process: u.ProcessName, version: u.Version, routable: routable}
		groupedUnits[key] = append(groupedUnits[key], u)
	}
	keys := make([]unitsKey, 0, len(groupedUnits))
	for key := range groupedUnits {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].version == keys[j].version {
			return keys[i].process < keys[j].process
		}
		return keys[i].version < keys[j].version
	})

	var titles []string
	if provisioner == "kubernetes" {
		titles = []string{"Name", "Host", "Status", "Restarts", "Age", "CPU", "Memory"}
	} else {
		titles = []string{"Name", "Status", "Host", "Port"}
	}
	mapUnitMetrics := map[string]unitMetrics{}
	for _, unitMetric := range metrics {
		mapUnitMetrics[unitMetric.ID] = unitMetric
	}

	for _, key := range keys {
		units := groupedUnits[key]
		unitsTable := tablecli.NewTable()
		tablecli.TableConfig.ForceWrap = false
		unitsTable.Headers = tablecli.Row(titles)
		sort.Slice(units, func(i, j int) bool {
			return units[i].ID < units[j].ID
		})
		for _, unit := range units {
			if unit.ID == "" {
				continue
			}
			var row tablecli.Row
			if provisioner == "kubernetes" {
				row = tablecli.Row{
					unit.ID,
					unit.Host(),
					unit.ReadyAndStatus(),
					countValue(unit.Restarts),
					translateTimestampSince(unit.CreatedAt),
					cpuValue(mapUnitMetrics[unit.ID].CPU),
					memoryValue(mapUnitMetrics[unit.ID].Memory),
				}
			} else {
				row = tablecli.Row{
					ShortID(unit.ID),
					unit.Status,
					unit.Host(),
					unit.Port(),
				}
			}

			unitsTable.AddRow(row)
		}
		if unitsTable.Rows() > 0 {
			unitsTable.SortByColumn(2)
			buf.WriteString("\n")
			groupLabel := ""
			if key.process != "" {
				groupLabel = fmt.Sprintf(" [process %s]", key.process)
			}
			if key.version != 0 {
				groupLabel = fmt.Sprintf("%s [version %d]", groupLabel, key.version)
			}
			if key.routable {
				groupLabel = fmt.Sprintf("%s [routable]", groupLabel)
			}
			buf.WriteString(fmt.Sprintf("Units%s: %d\n", groupLabel, unitsTable.Rows()))
			buf.WriteString(unitsTable.String())
		}
	}
}

func renderServiceInstanceBinds(w io.Writer, binds []serviceInstanceBind) {
	sibs := make([]serviceInstanceBind, len(binds))
	copy(sibs, binds)

	sort.Slice(sibs, func(i, j int) bool {
		if sibs[i].Service < sibs[j].Service {
			return true
		}
		if sibs[i].Service > sibs[j].Service {
			return false
		}
		return sibs[i].Instance < sibs[j].Instance
	})

	type instanceAndPlan struct {
		Instance string
		Plan     string
	}

	instancesByService := map[string][]instanceAndPlan{}
	for _, sib := range sibs {
		instancesByService[sib.Service] = append(instancesByService[sib.Service], instanceAndPlan{
			Instance: sib.Instance,
			Plan:     sib.Plan,
		})
	}

	var services []string
	for _, sib := range sibs {
		if len(services) > 0 && services[len(services)-1] == sib.Service {
			continue
		}
		services = append(services, sib.Service)
	}

	table := tablecli.NewTable()
	table.Headers = []string{"Service", "Instance (Plan)"}

	for _, s := range services {
		var sb strings.Builder
		for i, inst := range instancesByService[s] {
			sb.WriteString(inst.Instance)
			if inst.Plan != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", inst.Plan))
			}

			if i < len(instancesByService[s])-1 {
				sb.WriteString("\n")
			}
		}
		table.AddRow([]string{s, sb.String()})
	}

	if table.Rows() > 0 {
		fmt.Fprintf(w, "\nService instances: %d\n", table.Rows())
		fmt.Fprint(w, table.String())
	}
}

func renderVolumeBinds(w io.Writer, binds []volumeTypes.VolumeBind) {
	table := tablecli.NewTable()
	table.Headers = tablecli.Row([]string{"Name", "MountPoint", "Mode"})
	table.LineSeparator = true

	for _, b := range binds {
		mode := "rw"
		if b.ReadOnly {
			mode = "ro"
		}
		table.AddRow(tablecli.Row([]string{b.ID.Volume, b.ID.MountPoint, mode}))
	}

	if table.Rows() > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Volumes:", table.Rows())
		fmt.Fprint(w, table.String())
	}
}

func countValue(i *int) string {
	if i == nil {
		return ""
	}

	return fmt.Sprintf("%d", *i)
}

func cpuValue(q string) string {
	var cpu string
	qt, err := resource.ParseQuantity(q)
	if err == nil {
		cpu = fmt.Sprintf("%d%%", qt.MilliValue()/10)
	}

	return cpu
}

func memoryValue(q string) string {
	var memory string
	qt, err := resource.ParseQuantity(q)
	if err == nil {
		memory = fmt.Sprintf("%vMi", qt.Value()/(1024*1024))

	}
	return memory
}

func translateTimestampSince(timestamp *time.Time) string {
	if timestamp == nil || timestamp.IsZero() {
		return ""
	}

	return duration.HumanDuration(time.Since(*timestamp))
}

func (c *AppInfo) Show(a *app, context *cmd.Context, simplified bool) error {
	if c.json {
		return formatter.JSON(context.Stdout, a)
	}
	fmt.Fprintln(context.Stdout, a.String(simplified))
	return nil
}

type AppGrant struct {
	cmd.AppNameMixIn
}

func (c *AppGrant) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-grant",
		Usage: "app grant <teamname> [-a/--app appname]",
		Desc: `Allows a team to access an application. You need to be a member of a team that
has access to the app to allow another team to access it. grants access to an
app to a team.`,
		MinArgs: 1,
	}
}

func (c *AppGrant) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
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
	cmd.AppNameMixIn
}

func (c *AppRevoke) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-revoke",
		Usage: "app revoke <teamname> [-a/--app appname]",
		Desc: `Revokes the permission to access an application from a team. You need to have
access to the application to revoke access from a team.

An application cannot be orphaned, so it will always have at least one
authorized team.`,
		MinArgs: 1,
	}
}

func (c *AppRevoke) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
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
	tags      cmd.StringSliceFlag
}

func (f *appFilter) queryString(cli *cmd.Client) (url.Values, error) {
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
			var err error
			owner, err = currentUserEmail(cli)
			if err != nil {
				return nil, err
			}
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
	for _, tag := range f.tags {
		result.Add("tag", tag)
	}
	return result, nil
}

func currentUserEmail(cli *cmd.Client) (string, error) {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return "", err
	}
	user, _, err := apiClient.UserApi.UserGet(context.TODO())
	if err != nil {
		return "", err
	}
	return user.Email, nil
}

type AppList struct {
	fs         *gnuflag.FlagSet
	filter     appFilter
	simplified bool
	json       bool
}

func (c *AppList) Run(context *cmd.Context, client *cmd.Client) error {
	qs, err := c.filter.queryString(client)
	if err != nil {
		return err
	}
	if c.simplified {
		qs.Set("simplified", "true")
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
	return c.Show(result, context, client)
}

func (c *AppList) Show(result []byte, context *cmd.Context, client *cmd.Client) error {
	var apps []app
	err := json.Unmarshal(result, &apps)
	if err != nil {
		return err
	}
	table := tablecli.NewTable()
	if c.simplified {
		for _, app := range apps {
			fmt.Fprintln(context.Stdout, app.Name)
		}
		return nil
	}
	if c.json {
		return formatter.JSON(context.Stdout, apps)
	}
	table.Headers = tablecli.Row([]string{"Application", "Units", "Address"})
	for _, app := range apps {
		var summary string
		if app.Error == "" {
			unitsStatus := make(map[string]int)
			for _, unit := range app.Units {
				if unit.ID != "" {
					if unit.Ready != nil && *unit.Ready {
						unitsStatus["ready"]++
					} else if unit.StatusReason != "" {
						unitsStatus[unit.Status+" ("+unit.StatusReason+")"]++
					} else {
						unitsStatus[unit.Status]++
					}
				}
			}
			statusText := make([]string, len(unitsStatus))
			i := 0
			us := newUnitSorter(unitsStatus)
			sort.Sort(us)
			for _, status := range us.Statuses {
				statusText[i] = fmt.Sprintf("%d %s", unitsStatus[status], status)
				i++
			}
			summary = strings.Join(statusText, "\n")
		} else {
			summary = "error fetching units"
			if client.Verbosity > 0 {
				summary += fmt.Sprintf(": %s", app.Error)
			}
		}
		addrs := strings.Replace(app.Addr(), ", ", "\n", -1)
		table.AddRow(tablecli.Row([]string{app.Name, summary, addrs}))
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
		c.fs.BoolVar(&c.json, "json", false, "Display applications in JSON format")
		tagMessage := "Filter applications by tag. Can be used multiple times"
		c.fs.Var(&c.filter.tags, "tag", tagMessage)
		c.fs.Var(&c.filter.tags, "g", tagMessage)
	}
	return c.fs
}

func (c *AppList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-list",
		Usage: "app list",
		Desc: `Lists all apps that you have access to. App access is controlled by teams. If
your team has access to an app, then you have access to it.

Flags can be used to filter the list of applications.`,
	}
}

type AppStop struct {
	cmd.AppNameMixIn
	process string
	version string
	fs      *gnuflag.FlagSet
}

func (c *AppStop) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-stop",
		Usage:   "app stop [-a/--app appname] [-p/--process processname] [--version version]",
		Desc:    "Stops an application, or one of the processes of the application.",
		MinArgs: 0,
	}
}

func (c *AppStop) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/stop", appName))
	if err != nil {
		return err
	}
	qs := url.Values{}
	qs.Set("process", c.process)
	qs.Set("version", c.version)
	body := strings.NewReader(qs.Encode())
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
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

type AppStart struct {
	cmd.AppNameMixIn
	process string
	version string
	fs      *gnuflag.FlagSet
}

func (c *AppStart) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-start",
		Usage:   "app start [-a/--app appname] [-p/--process processname] [--version version]",
		Desc:    "Starts an application, or one of the processes of the application.",
		MinArgs: 0,
	}
}

func (c *AppStart) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/start", appName))
	if err != nil {
		return err
	}
	qs := url.Values{}
	qs.Set("process", c.process)
	qs.Set("version", c.version)
	body := strings.NewReader(qs.Encode())
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
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

type AppRestart struct {
	cmd.AppNameMixIn
	process string
	version string
	fs      *gnuflag.FlagSet
}

func (c *AppRestart) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/restart", appName))
	if err != nil {
		return err
	}
	qs := url.Values{}
	qs.Set("process", c.process)
	qs.Set("version", c.version)
	body := strings.NewReader(qs.Encode())
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
		Usage:   "app restart [-a/--app appname] [-p/--process processname] [--version version]",
		Desc:    `Restarts an application, or one of the processes of the application.`,
		MinArgs: 0,
	}
}

func (c *AppRestart) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

type CnameAdd struct {
	cmd.AppNameMixIn
}

func (c *CnameAdd) Run(context *cmd.Context, client *cmd.Client) error {
	err := addCName(context.Args, c.AppNameMixIn, client)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully defined.")
	return nil
}

func (c *CnameAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "cname-add",
		Usage: "cname add <cname> [<cname> ...] [-a/--app appname]",
		Desc: `Adds a new CNAME to the application.

It will not manage any DNS register, it's up to the user to create the DNS
register. Once the app contains a custom CNAME, it will be displayed by "app list" and "app info".`,
		MinArgs: 1,
	}
}

type CnameRemove struct {
	cmd.AppNameMixIn
}

func (c *CnameRemove) Run(context *cmd.Context, client *cmd.Client) error {
	err := unsetCName(context.Args, c.AppNameMixIn, client)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully undefined.")
	return nil
}

func (c *CnameRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "cname-remove",
		Usage: "cname remove <cname> [<cname> ...] [-a/--app appname]",
		Desc: `Removes a CNAME from the application. This undoes the change that cname-add
does.

After unsetting the CNAME from the app, [[tsuru app list]] and [[tsuru app info]] will display the internal, unfriendly address that tsuru uses.`,
		MinArgs: 1,
	}
}

func unsetCName(cnames []string, g cmd.AppNameMixIn, client *cmd.Client) error {
	appName, err := g.AppName()
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

func addCName(cnames []string, g cmd.AppNameMixIn, client *cmd.Client) error {
	appName, err := g.AppName()
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
	cmd.AppNameMixIn
	fs      *gnuflag.FlagSet
	process string
	version string
}

func (c *UnitAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-add",
		Usage: "unit add <# of units> [-a/--app appname] [-p/--process processname] [--version version]",
		Desc: `Adds new units to a process of an application. You need to have access to the
app to be able to add new units to it.`,
		MinArgs: 1,
	}
}

func (c *UnitAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

func (c *UnitAdd) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
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
	val.Set("version", c.version)
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
	cmd.AppNameMixIn
	fs      *gnuflag.FlagSet
	process string
	version string
}

func (c *UnitRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-remove",
		Usage: "unit remove <# of units> [-a/--app appname] [-p/-process processname] [--version version]",
		Desc: `Removes units from a process of an application. You need to have access to the
app to be able to remove units from it.`,
		MinArgs: 1,
	}
}

func (c *UnitRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

func (c *UnitRemove) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	val := url.Values{}
	val.Add("units", context.Args[0])
	val.Add("process", c.process)
	val.Set("version", c.version)
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

type UnitKill struct {
	cmd.AppNameMixIn
	fs    *gnuflag.FlagSet
	force bool
}

func (c *UnitKill) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-kill",
		Usage: "unit kill [-a/--app appname] [-f/--force] <unit>",
		Desc: `Kills units from a process of an application. You need to have access to the
app to be able to remove unit from it.`,
		MinArgs: 1,
	}
}

func (c *UnitKill) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVar(&c.force, "f", false, "Forces the termination of unit.")
	}
	return c.fs
}

func (c *UnitKill) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	unit := context.Args[0]

	v := url.Values{}
	if c.force {
		v.Set("force", "true")
	}

	url, err := cmd.GetURLVersion("1.12", fmt.Sprintf("/apps/%s/units/%s?%s", appName, unit, v.Encode()))
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

type UnitSet struct {
	cmd.AppNameMixIn
	fs      *gnuflag.FlagSet
	process string
	version int
}

func (c *UnitSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-set",
		Usage: "unit set <# of units> [-a/--app appname] [-p/--process processname] [--version version]",
		Desc: `Set the number of units for a process of an application, adding or removing units as needed. You need to have access to the
app to be able to set the number of units for it. The process flag is optional if the app has only 1 process.`,
		MinArgs: 1,
	}
}

func (c *UnitSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		processMessage := "Process name"
		c.fs.StringVar(&c.process, "process", "", processMessage)
		c.fs.StringVar(&c.process, "p", "", processMessage)
		c.fs.IntVar(&c.version, "version", 0, "Version number")
	}
	return c.fs
}

func (c *UnitSet) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var a app
	err = json.Unmarshal(result, &a)
	if err != nil {
		return err
	}

	unitsByProcess := map[string][]unit{}
	unitsByVersion := map[int][]unit{}
	for _, u := range a.Units {
		unitsByProcess[u.ProcessName] = append(unitsByProcess[u.ProcessName], u)
		unitsByVersion[u.Version] = append(unitsByVersion[u.Version], u)
	}

	if len(unitsByProcess) != 1 && c.process == "" {
		return errors.New("Please use the -p/--process flag to specify which process you want to set units for.")
	}

	if len(unitsByVersion) != 1 && c.version == 0 {
		return errors.New("Please use the --version flag to specify which version you want to set units for.")
	}

	if c.process == "" {
		for p := range unitsByProcess {
			c.process = p
			break
		}
	}

	if c.version == 0 {
		for v := range unitsByVersion {
			c.version = v
			break
		}
	}

	existingUnits := 0
	for _, unit := range a.Units {
		if unit.ProcessName == c.process && unit.Version == c.version {
			existingUnits++
		}
	}

	desiredUnits, err := strconv.Atoi(context.Args[0])
	if err != nil {
		return err
	}

	if existingUnits < desiredUnits {
		u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/units", appName))
		if err != nil {
			return err
		}

		unitsToAdd := desiredUnits - existingUnits
		val := url.Values{}
		val.Add("units", strconv.Itoa(unitsToAdd))
		val.Add("process", c.process)
		val.Add("version", strconv.Itoa(c.version))
		request, err := http.NewRequest(http.MethodPut, u, bytes.NewBufferString(val.Encode()))
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

	if existingUnits > desiredUnits {
		unitsToRemove := existingUnits - desiredUnits
		val := url.Values{}
		val.Add("units", strconv.Itoa(unitsToRemove))
		val.Add("process", c.process)
		val.Add("version", strconv.Itoa(c.version))
		u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/units?%s", appName, val.Encode()))
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

		defer response.Body.Close()
		return cmd.StreamJSONResponse(context.Stdout, response)
	}

	fmt.Fprintf(context.Stdout, "The process %s, version %d already has %d units.\n", c.process, c.version, existingUnits)
	return nil
}
