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
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cezarsa/form"
	"github.com/fatih/color"
	"github.com/lnquy/cron"
	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	tsuruClientApp "github.com/tsuru/tsuru-client/tsuru/app"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/completions"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	appTypes "github.com/tsuru/tsuru/types/app"
	bindTypes "github.com/tsuru/tsuru/types/bind"
	provTypes "github.com/tsuru/tsuru/types/provision"
	volumeTypes "github.com/tsuru/tsuru/types/volume"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/duration"
)

const (
	cutoffHexID = 12
)

var hexRegex = regexp.MustCompile(`(?i)^[a-f0-9]+$`)

var _ cmd.AutoCompleteCommand = &AppCreate{}

type AppCreate struct {
	teamOwner   string
	plan        string
	router      string
	pool        string
	description string
	tags        cmd.StringSliceFlag
	routerOpts  cmd.MapFlag
	fs          *pflag.FlagSet
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
		Usage: "<appname> [platform] [--plan/-p plan name] [--router/-r router name] [--team/-t team owner] [--pool/-o pool name] [--description/-d description] [--tag/-g tag]... [--router-opts key=value]...",
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

func (c *AppCreate) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)

		infoMessage := "The plan used to create the app"
		c.fs.StringVarP(&c.plan, standards.FlagPlan, standards.ShortFlagPlan, "", infoMessage)

		routerMessage := "The router used by the app"
		c.fs.StringVarP(&c.router, standards.FlagRouter, "r", "", routerMessage)

		teamMessage := "Team owner app"
		c.fs.StringVarP(&c.teamOwner, standards.FlagTeam, standards.ShortFlagTeam, "", teamMessage)

		poolMessage := "Pool to deploy your app"
		c.fs.StringVarP(&c.pool, standards.FlagPool, standards.ShortFlagPool, "", poolMessage)

		descriptionMessage := "App description"
		c.fs.StringVarP(&c.description, standards.FlagDescription, standards.ShortFlagDescription, "", descriptionMessage)

		tagMessage := "App tag"
		c.fs.VarP(&c.tags, standards.FlagTag, standards.ShortFlagTag, tagMessage)

		c.fs.Var(&c.routerOpts, "router-opts", "Router options")
	}
	return c.fs
}

func (cmd *AppCreate) Complete(args []string, toComplete string) ([]string, error) {
	if len(args) == 1 {
		return completions.PlatformNameCompletionFunc(toComplete)
	}
	return nil, nil
}

func (c *AppCreate) Run(context *cmd.Context) error {
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
	u, err := config.GetURL("/apps")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	result, err := io.ReadAll(response.Body)
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
	fs   *pflag.FlagSet
	tsuruClientApp.AppNameMixIn
	cmd.ConfirmationCommand

	memory, cpu, cpuBurst string
}

func (c *AppUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-update",
		Usage: "[-a/--app appname] [--description/-d description] [--plan/-p plan name] [--pool/-o pool] [--team-owner/-t team owner] [--platform/-l platform] [-i/--image-reset] [--cpu cpu] [--memory memory] [--cpu-burst-factor cpu-burst-factor] [--tag/-g tag]...",
		Desc:  `Updates an app, changing its description, tags, plan or pool information.`,
	}
}

func (c *AppUpdate) Flags() *pflag.FlagSet {
	if c.fs == nil {
		flagSet := pflag.NewFlagSet("", pflag.ExitOnError)

		descriptionMessage := "Changes description for the app"
		flagSet.StringVarP(&c.args.Description, standards.FlagDescription, standards.ShortFlagDescription, "", descriptionMessage)

		planMessage := "Changes plan for the app"
		flagSet.StringVarP(&c.args.Plan, standards.FlagPlan, standards.ShortFlagPlan, "", planMessage)

		poolMessage := "Changes pool for the app"
		flagSet.StringVarP(&c.args.Pool, standards.FlagPool, standards.ShortFlagPool, "", poolMessage)

		teamOwnerMessage := "Changes owner team for the app"
		flagSet.StringVarP(&c.args.TeamOwner, standards.FlagTeam, standards.ShortFlagTeam, "", teamOwnerMessage)
		flagSet.StringVar(&c.args.TeamOwner, "team-owner", "", teamOwnerMessage)
		flagSet.MarkHidden("team-owner")

		tagMessage := "Add tags for the app. You can add multiple tags repeating the --tag argument"
		flagSet.VarP((*cmd.StringSliceFlag)(&c.args.Tags), standards.FlagTag, standards.ShortFlagTag, tagMessage)

		platformMsg := "Changes platform for the app"
		flagSet.StringVarP(&c.args.Platform, standards.FlagPlatform, "l", "", platformMsg)

		imgReset := "Forces next deploy to build app image from scratch"
		flagSet.BoolVarP(&c.args.ImageReset, "image-reset", "i", false, imgReset)

		noRestartMessage := "Prevent tsuru from restarting the application"
		flagSet.BoolVar(&c.args.NoRestart, "no-restart", false, noRestartMessage)

		flagSet.StringVar(&c.cpu, "cpu", "", "CPU limit for app, this will override the plan cpu value. One cpu is equivalent to 1 vCPU/Core, fractional requests are allowed and the expression 0.1 is equivalent to the expression 100m")
		flagSet.StringVar(&c.cpuBurst, "cpu-burst-factor", "", "The multiplier to determine the limits of the CPU burst. Setting 1 disables burst")
		flagSet.StringVar(&c.memory, "memory", "", "Memory limit for app, this will override the plan memory value. You can express memory as a bytes integer or using one of these suffixes: E, P, T, G, M, K, Ei, Pi, Ti, Gi, Mi, Ki")
		c.fs = mergeFlagSet(
			c.AppNameMixIn.Flags(),
			flagSet,
		)
	}
	return c.fs
}

func (c *AppUpdate) Run(ctx *cmd.Context) error {
	ctx.RawOutput()

	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
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

	if c.cpuBurst != "" {
		var cpuBurst float64
		cpuBurst, err = strconv.ParseFloat(c.cpuBurst, 64)
		if err != nil {
			return err
		}

		if cpuBurst < 1 {
			return errors.New("invalid factor, please use a value greater equal 1")
		}

		c.args.Planoverride.CpuBurst = &cpuBurst
	}

	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("please use the -a/--app flag to specify which app you want to update")
	}

	response, err := apiClient.AppApi.AppUpdate(context.TODO(), appName, c.args)
	if err != nil {
		return err
	}

	err = formatter.StreamJSONResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "App %q has been updated!\n", appName)
	return nil
}

type AppRemove struct {
	tsuruClientApp.AppNameMixIn
	cmd.ConfirmationCommand
	fs *pflag.FlagSet
}

func (c *AppRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-remove",
		Usage: "[-a/--app appname] [-y/--assume-yes]",
		Desc: `Removes an application. If the app is bound to any service instance, all binds
will be removed before the app gets deleted (see [[tsuru service-unbind]]).

You need to be a member of a team that has access to the app to be able to
remove it (you are able to remove any app that you see in [[tsuru app list]]).`,
	}
}

func (c *AppRemove) Run(context *cmd.Context) error {
	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("please use the -a/--app flag to specify which app you want to remove")
	}
	if len(c.fs.Args()) > 0 {
		return errors.New("wrong number of parameters, are you using the correct command?")
	}
	if !c.Confirm(context, fmt.Sprintf(`Are you sure you want to remove app "%s"?`, appName)) {
		return nil
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(context.Stdout, response)
}

func (c *AppRemove) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = mergeFlagSet(
			c.AppNameMixIn.Flags(),
			c.ConfirmationCommand.Flags(),
		)
	}
	return c.fs
}

var _ cmd.AutoCompleteCommand = &AppInfo{}

type AppInfo struct {
	tsuruClientApp.AppNameMixIn

	json         bool
	simplified   bool
	flagsApplied bool
}

func (c *AppInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-info",
		Usage: "[appname]",
		Desc: `Shows information about a specific app. Its state, platform, etc.
You need to be a member of a team that has access to the app to be able to see information about it.`,
	}
}

func (cmd *AppInfo) Flags() *pflag.FlagSet {
	fs := cmd.AppNameMixIn.Flags()
	if !cmd.flagsApplied {
		fs.BoolVarP(&cmd.simplified, "simplified", "s", false, "Show simplified view of app")
		fs.BoolVar(&cmd.json, standards.FlagJSON, false, "Show JSON view of app")

		cmd.flagsApplied = true
	}
	return fs
}

func (cmd *AppInfo) Complete(args []string, toComplete string) ([]string, error) {
	return completions.AppNameCompletionFunc(toComplete)
}

func (c *AppInfo) Run(context *cmd.Context) error {
	appName, err := c.AppNameByArgsAndFlag(context.Args)
	if err != nil {
		return err
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
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

func unitHost(u provTypes.Unit) string {
	address := ""
	if len(u.Addresses) > 0 {
		address = u.Addresses[0].Host
	} else if u.Address != nil {
		address = u.Address.Host
	} else if u.IP != "" {
		return u.IP
	}
	if address == "" {
		return address
	}

	host, _, _ := net.SplitHostPort(address)
	return host
}

func unitReadyAndStatus(u provTypes.Unit) string {
	if u.Ready != nil && *u.Ready {
		return "ready"
	}

	if u.StatusReason != "" {
		return u.Status.String() + " (" + u.StatusReason + ")"
	}

	return u.Status.String()
}

func unitPort(u provTypes.Unit) string {
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

type app appTypes.AppInfo

func (a *app) QuotaString() string {
	if a.Quota == nil {
		return "0/0 units"
	}
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

	return strings.Join(addrs, "\n")
}

func (a *app) Addr() string {
	return appAddrs(a.CName, a.IP, a.Routers)
}

func AppResumeAddr(a *appTypes.AppResume) string {
	return appAddrs(a.CName, a.IP, a.Routers)
}

func appAddrs(cnames []string, ip string, routers []appTypes.AppRouter) string {
	var cnameAddrs []string
	var routerAddrs []string

	for _, cname := range cnames {
		if cname != "" {
			cnameAddrs = append(cnameAddrs, ensureHTTP(cname)+" (cname)")
		}
	}

	if len(routers) == 0 {
		if ip != "" {
			routerAddrs = append(routerAddrs, ensureHTTP(ip))
		}
	} else {
		for _, r := range routers {
			if len(r.Addresses) > 0 {
				for _, addr := range r.Addresses {
					routerAddrs = append(routerAddrs, ensureHTTP(addr))
				}
			} else if r.Address != "" {
				routerAddrs = append(routerAddrs, ensureHTTP(r.Address))
			}
		}
	}

	sort.Strings(cnameAddrs)
	sort.Strings(routerAddrs)

	allAddrs := append(cnameAddrs, routerAddrs...)

	return strings.Join(allAddrs, "\n")
}

func ensureHTTP(addr string) string {
	if addr == "" {
		return addr
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	return "http://" + addr
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
Error:        {{ .Error }}
{{ end -}}
Application:  {{.Name}}
{{- if .DashboardURL }}
Dashboard:    {{ .DashboardURL }}
{{- end }}
{{- if .Description }}
Description:  {{.Description}}
{{- end }}
{{- if .TagList }}
Tags:         {{.TagList}}
{{- end }}
Created by:   {{.Owner}}
Platform:     {{.Platform}}
Plan:         {{ .Plan.Name }}
Pool:         {{.Pool}} ({{ .Provisioner }}{{ if .Cluster}} | cluster: {{ .Cluster }}{{end}})
{{if and (not .Routers) (.Router) -}}
Router:       {{.Router}}{{if .RouterOpts}} ({{.GetRouterOpts}}){{end}}
{{end -}}
Teams:        {{.TeamList}}

{{ with .InternalAddr -}}
Cluster Internal Addresses:  {{. | replace "\n" "\n                             "}}
{{ end -}}
{{ with .Addr -}}
Cluster External Addresses:  {{. | replace "\n" "\n                             "}}
{{ end -}}
{{ with .SimpleServicesView -}}
Bound Services:              {{. | replace "\n" "\n                             "}}
{{ end -}}
`

const fullFormat = `{{ if .Error -}}
Error:        {{ .Error }}
{{ end -}}
Application:  {{.Name}}
{{- if .DashboardURL }}
Dashboard:    {{ .DashboardURL }}
{{- end }}
{{- if .Description }}
Description:  {{.Description}}
{{- end }}
{{- if .TagList }}
Tags:         {{.TagList}}
{{- end }}
Platform:     {{.Platform}}
{{ if .Provisioner -}}
Provisioner:  {{ .Provisioner }}
{{ end -}}
{{if and (not .Routers) (.Router) -}}
Router:       {{.Router}}{{if .RouterOpts}} ({{.GetRouterOpts}}){{end}}
{{end -}}
Teams:        {{.TeamList}}
Created by:   {{.Owner}}
Deploys:      {{.Deploys}}
{{if .Cluster -}}
Cluster:      {{ .Cluster }}
{{ end -}}
{{if .Pool -}}
Pool:         {{.Pool}}
{{ end -}}
Quota:        {{ .QuotaString }}
{{ if .Addr -}}
Addresses:    {{.Addr | replace "\n" "\n              " }}
{{ end -}}
`

func (a *app) String(simplified bool) string {
	var format string

	if simplified {
		format = simplifiedFormat
	} else {
		format = fullFormat
	}

	var buf bytes.Buffer
	tmpl := template.Must(
		template.New("app").
			Funcs(template.FuncMap{
				"replace": func(old, new, s string) string {
					return strings.ReplaceAll(s, old, new)
				},
			}).
			Parse(format),
	)

	if simplified {
		renderUnitsSummary(&buf, a.Units, a.UnitsMetrics)
	} else {
		renderUnits(&buf, a.Units, a.UnitsMetrics)
	}

	internalAddressesTable := tablecli.NewTable()
	internalAddressesTable.Headers = []string{"Domain", "Port", "Process"}

	containsVersion := false
	for _, internalAddress := range a.InternalAddresses {
		if internalAddress.Version != "" {
			containsVersion = true
			break
		}
	}
	if containsVersion {
		internalAddressesTable.Headers = append(internalAddressesTable.Headers, "Version")
	}

	internalAddressesTable.TableWriterPadding = standards.SubTableWriterPadding

	for _, internalAddress := range a.InternalAddresses {
		port := strconv.Itoa(int(internalAddress.Port))
		if internalAddress.TargetPort > 0 && int(internalAddress.Port) != internalAddress.TargetPort {
			port = fmt.Sprintf("%d->%d", internalAddress.Port, internalAddress.TargetPort)
		}

		row := []string{
			internalAddress.Domain,
			port + "/" + internalAddress.Protocol,
			internalAddress.Process,
		}

		if containsVersion {
			row = append(row, internalAddress.Version)
		}

		internalAddressesTable.AddRow(row)
	}

	if !simplified {
		renderServiceInstanceBindsForApps(&buf, a.ServiceInstanceBinds)
	}

	var autoScaleTables []*tablecli.Table
	processes := []string{}

	for _, as := range a.Autoscale {
		autoScaleTable := tablecli.NewTable()
		autoScaleTable.LineSeparator = true
		autoScaleTable.TableWriterPadding = standards.SubTableWriterPadding

		processString := fmt.Sprintf(
			"Autoscale [process %s] [version %d] [min units %d] [max units %d]:",
			as.Process, as.Version, int(as.MinUnits), int(as.MaxUnits),
		)
		processes = append(processes, processString)

		autoScaleTable.Headers = tablecli.Row([]string{
			"Triggers",
			"Trigger details",
		})

		if as.AverageCPU != "" {
			cpu := cpuValue(as.AverageCPU)
			autoScaleTable.AddRow(tablecli.Row([]string{
				"CPU",
				fmt.Sprintf("Target: %s", cpu),
			}))
		}

		autoScaleTable.TableWriterExpandRows = true

		addSection := func(title string, lines []string) {
			autoScaleTable.AddRow([]string{
				title,
				strings.Join(lines, "\n"),
			})
		}

		for _, schedule := range as.Schedules {
			scheduleInfo := buildScheduleInfo(schedule)
			addSection("Schedule", scheduleInfo)
		}

		for _, prometheus := range as.Prometheus {
			prometheusInfo := buildPrometheusInfo(prometheus)
			addSection("Prometheus", prometheusInfo)
		}
		scaleDownLines := getParamsScaleDownLines(as.Behavior)
		if len(scaleDownLines) > 0 {
			addSection("Scale down behavior", scaleDownLines)
		}
		autoScaleTables = append(autoScaleTables, autoScaleTable)
	}

	if len(processes) > 0 {
		for i, asTable := range autoScaleTables {
			buf.WriteString("\n")
			buf.WriteString(processes[i])
			buf.WriteString("\n")
			buf.WriteString(asTable.String())
		}
	}

	if !simplified && a.Plan != nil && (a.Plan.Memory != 0 || a.Plan.CPUMilli != 0) {
		planByProcess := map[string]string{}
		for _, p := range a.Processes {
			if p.Plan != "" {
				planByProcess[p.Name] = p.Plan
			}
		}

		if len(planByProcess) == 0 {
			buf.WriteString("\n")
			buf.WriteString("Plan:\n")
			buf.WriteString(renderPlans([]appTypes.Plan{*a.Plan}, renderPlansOpts{
				tableWriterPadding: standards.SubTableWriterPadding,
			}))
		} else {
			buf.WriteString("\n")
			buf.WriteString("Process plans:\n")
			buf.WriteString(renderProcessPlan(*a.Plan, planByProcess))
		}
	}

	if !simplified && internalAddressesTable.Rows() > 0 {
		buf.WriteString("\n")
		buf.WriteString("Cluster internal addresses:\n")
		buf.WriteString(internalAddressesTable.String())
	}
	if !simplified && len(a.Routers) > 0 {
		buf.WriteString("\n")
		buf.WriteString("Cluster external addresses:\n")
		renderRouters(a.Routers, &buf, "Router", standards.SubTableWriterPadding)
	}

	renderVolumeBinds(&buf, a.VolumeBinds)

	var tplBuffer bytes.Buffer
	tmpl.Execute(&tplBuffer, a)
	return tplBuffer.String() + buf.String()
}

func buildScheduleInfo(schedule provTypes.AutoScaleSchedule) []string {
	// Init with default EN locale
	exprDesc, _ := cron.NewDescriptor()

	startTimeHuman, _ := exprDesc.ToDescription(schedule.Start, cron.Locale_en)
	endTimeHuman, _ := exprDesc.ToDescription(schedule.End, cron.Locale_en)

	return []string{
		"Start: " + startTimeHuman + " (" + schedule.Start + ")",
		"End: " + endTimeHuman + " (" + schedule.End + ")",
		"Units: " + strconv.Itoa(schedule.MinReplicas),
		"Timezone: " + schedule.Timezone,
	}
}

func buildPrometheusInfo(prometheus provTypes.AutoScalePrometheus) []string {
	thresholdValue := strconv.FormatFloat(prometheus.Threshold, 'f', -1, 64)

	return []string{
		"Name: " + prometheus.Name,
		"Query: " + prometheus.Query,
		"Threshold: " + thresholdValue,
		"PrometheusAddress: " + prometheus.PrometheusAddress,
	}
}

func (a *app) SimpleServicesView() string {
	sibs := make([]bindTypes.ServiceInstanceBind, len(a.ServiceInstanceBinds))
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

	return strings.Join(pairs, "\n")
}

func renderUnitsSummary(buf *bytes.Buffer, units []provTypes.Unit, metrics []provTypes.UnitMetric) {
	type unitsKey struct {
		process  string
		version  int
		routable bool
	}
	groupedUnits := map[unitsKey][]provTypes.Unit{}
	for _, u := range units {
		routable := false
		if u.Routable {
			routable = u.Routable
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
	titles = []string{"Process", "Ready", "Restarts", "Avg CPU (abs)", "Avg Memory"}

	unitsTable := tablecli.NewTable()
	tablecli.TableConfig.ForceWrap = false
	unitsTable.Headers = tablecli.Row(titles)
	unitsTable.TableWriterPadding = standards.SubTableWriterPadding
	fmt.Fprintf(buf, "\nUnits: %d\n", len(units))

	if len(units) == 0 {
		return
	}
	mapUnitMetrics := map[string]provTypes.UnitMetric{}
	for _, unitMetric := range metrics {
		mapUnitMetrics[unitMetric.ID] = unitMetric
	}

	for _, key := range keys {
		summaryTitle := key.process
		if key.version > 0 {
			summaryTitle = fmt.Sprintf("%s (v%d)", key.process, key.version)
		}

		summaryUnits := groupedUnits[key]

		if !key.routable {
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
				restarts += int(*unit.Restarts)
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

		unitsTable.AddRow(tablecli.Row{
			summaryTitle,
			fmt.Sprintf("%d/%d", readyUnits, len(summaryUnits)),
			fmt.Sprintf("%d", restarts),
			fmt.Sprintf("%d%%", cpuTotal.MilliValue()/int64(10)/int64(len(summaryUnits))),
			fmt.Sprintf("%vMi", memoryTotal.Value()/int64(1024*1024)/int64(len(summaryUnits))),
		})

	}
	buf.WriteString(unitsTable.String())
}

func renderUnits(buf *bytes.Buffer, units []provTypes.Unit, metrics []provTypes.UnitMetric) {
	type unitsKey struct {
		process  string
		version  int
		routable bool
	}
	groupedUnits := map[unitsKey][]provTypes.Unit{}
	for _, u := range units {
		routable := false
		if u.Routable {
			routable = u.Routable
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

	mapUnitMetrics := map[string]provTypes.UnitMetric{}
	for _, unitMetric := range metrics {
		mapUnitMetrics[unitMetric.ID] = unitMetric
	}

	titles := []string{"Name", "Host", "Status", "Restarts", "Age"}

	containsMetrics := len(metrics) > 0
	if containsMetrics {
		titles = append(titles, "CPU", "Memory")
	}

	for _, key := range keys {
		units := groupedUnits[key]
		unitsTable := tablecli.NewTable()
		tablecli.TableConfig.ForceWrap = false
		unitsTable.TableWriterPadding = standards.SubTableWriterPadding
		unitsTable.Headers = tablecli.Row(titles)
		sort.Slice(units, func(i, j int) bool {
			return units[i].ID < units[j].ID
		})
		for _, unit := range units {
			if unit.ID == "" {
				continue
			}
			row := tablecli.Row{
				unit.ID,
				unitHost(unit),
				unitReadyAndStatus(unit),
				countValue(unit.Restarts),
				translateTimestampSince(unit.CreatedAt),
			}

			if containsMetrics {
				row = append(row,
					cpuValue(mapUnitMetrics[unit.ID].CPU),
					memoryValue(mapUnitMetrics[unit.ID].Memory),
				)
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
			fmt.Fprintf(buf, "Units%s: %d\n", groupLabel, unitsTable.Rows())
			fmt.Fprint(buf, unitsTable.String())
		}
	}
}

func renderServiceInstanceBindsForApps(w io.Writer, binds []bindTypes.ServiceInstanceBind) {
	sibs := make([]bindTypes.ServiceInstanceBind, len(binds))
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
	table.TableWriterPadding = standards.SubTableWriterPadding

	count := 0
	for _, s := range services {
		for i, inst := range instancesByService[s] {
			count++
			desc := ""
			if i == 0 {
				desc = s
			}

			var sb strings.Builder
			sb.WriteString(inst.Instance)
			if inst.Plan != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", inst.Plan))
			}

			table.AddRow([]string{desc, sb.String()})
		}
	}

	if count > 0 {
		fmt.Fprintf(w, "\nService instances: %d\n", count)
		fmt.Fprint(w, table.String())
	}
}

func renderServiceInstanceBinds(w io.Writer, binds []tsuru.AppServiceInstanceBinds) {
	sibs := make([]tsuru.AppServiceInstanceBinds, len(binds))
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
	table.TableWriterPadding = standards.SubTableWriterPadding

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
	table.TableWriterPadding = standards.SubTableWriterPadding

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

func countValue[T any](v *T) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", *v)
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

var _ cmd.AutoCompleteCommand = &AppGrant{}

type AppGrant struct {
	tsuruClientApp.AppNameMixIn
}

func (c *AppGrant) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-grant",
		Usage: "<teamname> [-a/--app appname]",
		Desc: `Allows a team to access an application. You need to be a member of a team that
has access to the app to allow another team to access it. grants access to an
app to a team.`,
		MinArgs: 1,
	}
}

func (c *AppGrant) Run(context *cmd.Context) error {
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	teamName := context.Args[0]
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/teams/%s", appName, teamName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", u, nil)
	if err != nil {
		return err
	}
	_, err = tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `Team "%s" was added to the "%s" app`+"\n", teamName, appName)
	return nil
}

func (c *AppGrant) Complete(args []string, toComplete string) ([]string, error) {
	return completions.TeamNameCompletionFunc(toComplete)
}

type AppRevoke struct {
	tsuruClientApp.AppNameMixIn
}

func (c *AppRevoke) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-revoke",
		Usage: "<teamname> [-a/--app appname]",
		Desc: `Revokes the permission to access an application from a team. You need to have
access to the application to revoke access from a team.

An application cannot be orphaned, so it will always have at least one
authorized team.`,
		MinArgs: 1,
	}
}

var _ cmd.AutoCompleteCommand = &AppRevoke{}

func (c *AppRevoke) Complete(args []string, toComplete string) ([]string, error) {
	return completions.TeamNameCompletionFunc(toComplete)
}

func (c *AppRevoke) Run(context *cmd.Context) error {
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	teamName := context.Args[0]
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/teams/%s", appName, teamName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	_, err = tsuruHTTP.AuthenticatedClient.Do(request)
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
	status    string
	tags      cmd.StringSliceFlag
}

func (f *appFilter) queryString() (url.Values, error) {
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
			owner, err = currentUserEmail()
			if err != nil {
				return nil, err
			}
		}
		result.Set("owner", owner)
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

func currentUserEmail() (string, error) {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
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
	fs         *pflag.FlagSet
	filter     appFilter
	simplified bool
	json       bool
}

func (c *AppList) Run(context *cmd.Context) error {
	qs, err := c.filter.queryString()
	if err != nil {
		return err
	}
	if c.simplified {
		qs.Set("simplified", "true")
	}
	u, err := config.GetURL(fmt.Sprintf("/apps?%s", qs.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	defer response.Body.Close()
	result, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return c.Show(result, context)
}

func (c *AppList) Show(result []byte, context *cmd.Context) error {
	var apps []appTypes.AppResume
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

	if tablecli.TableConfig.UseTabWriter {
		table.Headers = tablecli.Row([]string{"Application", "Ready", "Reason"})
		for _, app := range apps {
			stats := collectUnitStats(&app, false)
			ready := appListReadyUnitsSummary(stats)
			summary := appListCompactSummary(&app, stats)
			table.AddRow([]string{app.Name, ready, summary})
		}
	} else {
		table.Headers = tablecli.Row([]string{"Application", "Units", "Address"})
		for _, app := range apps {
			summary := appListSummary(&app)
			addrs := strings.ReplaceAll(AppResumeAddr(&app), ", ", "\n")
			row := []string{app.Name, summary, addrs}

			table.AddRow(row)
		}
	}
	table.LineSeparator = true
	table.Sort()
	context.Stdout.Write(table.Bytes())
	return nil
}

type unitStats struct {
	totalUnits      int
	unitsWithErrors int
	readyUnits      int
	unitsStatus     map[string]int
}

func collectUnitStats(app *appTypes.AppResume, includeReady bool) unitStats {
	stats := unitStats{unitsStatus: make(map[string]int)}
	for _, unit := range app.Units {
		if unit.ID == "" {
			continue
		}
		stats.totalUnits++
		if unit.Status == provTypes.UnitStatusError {
			stats.unitsWithErrors++
		}
		if unit.Ready != nil && *unit.Ready {
			stats.readyUnits++
			if includeReady {
				stats.unitsStatus["ready"]++
			}
		} else if unit.StatusReason != "" {
			stats.unitsStatus[unit.Status.String()+" ("+unit.StatusReason+")"]++
		} else {
			stats.unitsStatus[unit.Status.String()]++
		}
	}
	return stats
}

func (s unitStats) errorColor() color.Attribute {
	if s.unitsWithErrors == s.totalUnits && s.totalUnits > 0 {
		return color.FgRed
	}
	if s.unitsWithErrors > 0 {
		return color.FgYellow
	}
	return 0
}

func colorSprint(attr color.Attribute, text string) string {
	if attr == 0 {
		return text
	}
	return color.New(attr).Sprint(text)
}

func formatUnitStatuses(unitsStatus map[string]int, c color.Attribute, separator string) string {
	statusText := make([]string, 0, len(unitsStatus))
	us := newUnitSorter(unitsStatus)
	sort.Sort(us)
	for _, status := range us.Statuses {
		text := fmt.Sprintf("%d %s", unitsStatus[status], status)
		if status == "error" || strings.HasPrefix(status, "error (") {
			text = colorSprint(c, text)
		}
		statusText = append(statusText, text)
	}
	return strings.Join(statusText, separator)
}

func appListSummary(app *appTypes.AppResume) string {
	if app.Error != "" {
		return color.RedString("error fetching units: " + app.Error)
	}
	stats := collectUnitStats(app, true)
	return formatUnitStatuses(stats.unitsStatus, stats.errorColor(), "\n")
}

func appListReadyUnitsSummary(stats unitStats) string {
	return colorSprint(stats.errorColor(), fmt.Sprintf("%d/%d", stats.readyUnits, stats.totalUnits))
}

func appListCompactSummary(app *appTypes.AppResume, stats unitStats) string {
	if app.Error != "" {
		return color.RedString("error fetching units: " + app.Error)
	}
	return formatUnitStatuses(stats.unitsStatus, stats.errorColor(), ", ")
}

func (c *AppList) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("app-list", pflag.ExitOnError)
		c.fs.StringVarP(&c.filter.name, standards.FlagName, standards.ShortFlagName, "", "Filter applications by name")

		c.fs.StringVarP(&c.filter.pool, standards.FlagPool, standards.ShortFlagPool, "", "Filter applications by pool")
		c.fs.StringVarP(&c.filter.status, "status", "s", "", "Filter applications by unit status. Accepts multiple values separated by commas. Possible values can be: building, created, starting, error, started, stopped, asleep")
		c.fs.StringVarP(&c.filter.platform, "platform", "p", "", "Filter applications by platform")
		c.fs.StringVarP(&c.filter.teamOwner, standards.FlagTeam, standards.ShortFlagTeam, "", "Filter applications by team owner")
		c.fs.StringVarP(&c.filter.owner, standards.FlagUser, standards.ShortFlagUser, "", "Filter applications by owner")
		c.fs.BoolVarP(&c.simplified, standards.FlagOnlyName, standards.ShortFlagOnlyName, false, "Display only applications name")
		c.fs.BoolVar(&c.json, standards.FlagJSON, false, "Display applications in JSON format")
		tagMessage := "Filter applications by tag. Can be used multiple times"
		c.fs.VarP(&c.filter.tags, standards.FlagTag, standards.ShortFlagTag, tagMessage)
	}
	return c.fs
}

func (c *AppList) Info() *cmd.Info {
	return &cmd.Info{
		Name: "app-list",
		Desc: `Lists all apps that you have access to. App access is controlled by teams. If
your team has access to an app, then you have access to it.

Flags can be used to filter the list of applications.`,
	}
}

type AppStop struct {
	tsuruClientApp.AppNameMixIn
	process string
	version string
	fs      *pflag.FlagSet
}

func (c *AppStop) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-stop",
		Usage: "[appname] [-p/--process processname] [--version version]",
		Desc:  "Stops an application, or one of the processes of the application.",
	}
}

func (c *AppStop) Run(context *cmd.Context) error {
	context.RawOutput()
	appName, err := c.AppNameByArgsAndFlag(context.Args)
	if err != nil {
		return err
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/stop", appName))
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
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(context.Stdout, response)
}

func (c *AppStop) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVarP(&c.process, "process", "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

var _ cmd.AutoCompleteCommand = &AppStop{}

func (c *AppStop) Complete(args []string, toComplete string) ([]string, error) {
	return completions.AppNameCompletionFunc(toComplete)
}

type AppStart struct {
	tsuruClientApp.AppNameMixIn
	process string
	version string
	fs      *pflag.FlagSet
}

func (c *AppStart) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-start",
		Usage: "[appname] [-p/--process processname] [--version version]",
		Desc:  "Starts an application, or one of the processes of the application.",
	}
}

func (c *AppStart) Run(context *cmd.Context) error {
	context.RawOutput()
	appName, err := c.AppNameByArgsAndFlag(context.Args)
	if err != nil {
		return err
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/start", appName))
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
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(context.Stdout, response)
}

func (c *AppStart) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVarP(&c.process, "process", "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

var _ cmd.AutoCompleteCommand = &AppStart{}

func (c *AppStart) Complete(args []string, toComplete string) ([]string, error) {
	return completions.AppNameCompletionFunc(toComplete)
}

type AppRestart struct {
	tsuruClientApp.AppNameMixIn
	process string
	version string
	fs      *pflag.FlagSet
}

func (c *AppRestart) Run(context *cmd.Context) error {
	context.RawOutput()
	appName, err := c.AppNameByArgsAndFlag(context.Args)
	if err != nil {
		return err
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/restart", appName))
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
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(context.Stdout, response)
}

var _ cmd.AutoCompleteCommand = &AppRestart{}

func (c *AppRestart) Complete(args []string, toComplete string) ([]string, error) {
	return completions.AppNameCompletionFunc(toComplete)
}

func (c *AppRestart) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-restart",
		Usage: "[appname] [-p/--process processname] [--version version]",
		Desc:  `Restarts an application, or one of the processes of the application.`,
	}
}

func (c *AppRestart) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVarP(&c.process, "process", "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

type CnameAdd struct {
	tsuruClientApp.AppNameMixIn
}

func (c *CnameAdd) Run(context *cmd.Context) error {
	err := addCName(context.Args, c.AppNameMixIn)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully defined.")
	return nil
}

func (c *CnameAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "cname-add",
		Usage: "<cname> [<cname> ...] [-a/--app appname]",
		Desc: `Adds a new CNAME to the application.

It will not manage any DNS register, it's up to the user to create the DNS
register. Once the app contains a custom CNAME, it will be displayed by "app list" and "app info".`,
		MinArgs: 1,
	}
}

type CnameRemove struct {
	tsuruClientApp.AppNameMixIn
}

func (c *CnameRemove) Run(context *cmd.Context) error {
	err := unsetCName(context.Args, c.AppNameMixIn)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "cname successfully undefined.")
	return nil
}

func (c *CnameRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "cname-remove",
		Usage: "<cname> [<cname> ...] [-a/--app appname]",
		Desc: `Removes a CNAME from the application. This undoes the change that cname-add
does.

After unsetting the CNAME from the app, [[tsuru app list]] and [[tsuru app info]] will display the internal, unfriendly address that tsuru uses.`,
		MinArgs: 1,
	}
}

func unsetCName(cnames []string, g tsuruClientApp.AppNameMixIn) error {
	appName, err := g.AppNameByFlag()
	if err != nil {
		return err
	}
	v := url.Values{}
	for _, cname := range cnames {
		v.Add("cname", cname)
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/cname?%s", appName, v.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	_, err = tsuruHTTP.AuthenticatedClient.Do(request)
	return err
}

func addCName(cnames []string, g tsuruClientApp.AppNameMixIn) error {
	appName, err := g.AppNameByFlag()
	if err != nil {
		return err
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/cname", appName))
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
	_, err = tsuruHTTP.AuthenticatedClient.Do(request)
	return err
}

type AppProcessUpdate struct {
	plan             string
	resetDefaultPlan bool
	noRestart        bool
	fs               *pflag.FlagSet
}

func (c *AppProcessUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-process-update",
		Usage:   "[app] [process] [--plan/-p plan name] [--default-plan]",
		Desc:    `Updates a plan of app process`,
		MinArgs: 2,
	}
}

func (c *AppProcessUpdate) Flags() *pflag.FlagSet {
	if c.fs == nil {
		flagSet := pflag.NewFlagSet("", pflag.ExitOnError)
		flagSet.SortFlags = false

		planMessage := "Changes plan for the app"
		flagSet.StringVarP(&c.plan, standards.FlagPlan, standards.ShortFlagPlan, "", planMessage)

		planReset := "Reset process to default plan of app"
		flagSet.BoolVar(&c.resetDefaultPlan, "default-plan", false, planReset)

		noRestartMessage := "Prevent tsuru from restarting the application"
		flagSet.BoolVar(&c.noRestart, standards.FlagNoRestart, false, noRestartMessage)

		c.fs = flagSet
	}
	return c.fs
}

func (c *AppProcessUpdate) Run(ctx *cmd.Context) error {
	ctx.RawOutput()

	if c.resetDefaultPlan {
		c.plan = "$default"
	}

	a := tsuru.UpdateApp{
		NoRestart: c.noRestart,
		Processes: []tsuru.AppProcess{
			{
				Name: ctx.Args[1],
				Plan: c.plan,
			},
		},
	}

	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}

	resp, err := apiClient.AppApi.AppUpdate(context.Background(), ctx.Args[0], a)
	if err != nil {
		return err
	}
	err = formatter.StreamJSONResponse(ctx.Stdout, resp)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Process %q of app %q has been updated!\n", ctx.Args[1], ctx.Args[0])

	return nil
}
