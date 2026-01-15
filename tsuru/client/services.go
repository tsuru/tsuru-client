// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/antihax/optional"
	"github.com/cezarsa/form"
	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/completions"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/service"
)

type serviceFilter struct {
	name      string
	pool      string
	plan      string
	service   string
	teamOwner string
	tags      cmd.StringSliceFlag
}

func (f *serviceFilter) queryString() (url.Values, error) {
	result := make(url.Values)
	if f.name != "" {
		result.Set("name", f.name)
	}
	if f.teamOwner != "" {
		result.Set("teamOwner", f.teamOwner)
	}
	if f.pool != "" {
		result.Set("pool", f.pool)
	}
	if f.plan != "" {
		result.Set("plan", f.plan)
	}
	if f.service != "" {
		result.Set("service", f.service)
	}
	for _, tag := range f.tags {
		result.Add("tag", tag)
	}
	return result, nil
}

type ServiceList struct {
	fs               *pflag.FlagSet
	filter           serviceFilter
	simplified       bool
	json             bool
	justServiceNames bool
}

func (s *ServiceList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-list",
		Usage: "service list",
		Desc:  `Retrieves and shows a list of instances of service the user has access.`,
	}
}

func (c *ServiceList) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("service-list", pflag.ExitOnError)
		c.fs.SortFlags = false

		c.fs.StringVarP(&c.filter.service, "service", "s", "", "Filter instances by service")
		c.fs.StringVarP(&c.filter.name, standards.FlagName, standards.ShortFlagName, "", "Filter service instances by name")
		c.fs.StringVarP(&c.filter.pool, standards.FlagPool, standards.ShortFlagPool, "", "Filter service instances by pool")
		c.fs.StringVarP(&c.filter.plan, standards.FlagPlan, standards.ShortFlagPlan, "", "Filter service instances by plan")
		c.fs.StringVarP(&c.filter.teamOwner, standards.FlagTeam, standards.ShortFlagTeam, "", "Filter service instances by team owner")

		c.fs.BoolVarP(&c.simplified, standards.FlagOnlyName, standards.ShortFlagOnlyName, false, "Display only service instances name")

		c.fs.BoolVar(&c.json, standards.FlagJSON, false, "Display in JSON format")
		c.fs.BoolVarP(&c.justServiceNames, "just-services", "j", false, "Display just service names")

		tagMessage := "Filter services by tag. Can be used multiple times"
		c.fs.VarP(&c.filter.tags, standards.FlagTag, standards.ShortFlagTag, tagMessage)
	}
	return c.fs
}

func (s ServiceList) Run(ctx *cmd.Context) error {
	qs, err := s.filter.queryString()
	if err != nil {
		return err
	}
	url, err := config.GetURL(fmt.Sprintf("/services/instances?%s", qs.Encode()))
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := tsuruHTTP.AuthenticatedClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	services := []service.ServiceModel{}
	err = json.NewDecoder(resp.Body).Decode(&services)
	if err != nil {
		return err
	}

	services = s.clientSideFilter(services)

	for _, s := range services {
		sort.Slice(s.ServiceInstances, func(i, j int) bool {
			return s.ServiceInstances[i].Name < s.ServiceInstances[j].Name
		})
	}

	if s.simplified {
		for _, s := range services {
			for _, instance := range s.ServiceInstances {
				fmt.Fprintln(ctx.Stdout, s.Service, instance.Name)
			}
		}
		return nil
	}

	if s.json {
		instances := []service.ServiceInstance{}
		for _, s := range services {
			instances = append(instances, s.ServiceInstances...)
		}

		return formatter.JSON(ctx.Stdout, instances)
	}

	if s.justServiceNames {
		t := tablecli.NewTable()
		t.Headers = tablecli.Row([]string{"Service"})
		for _, s := range services {
			t.AddRow(tablecli.Row([]string{s.Service}))
		}

		_, err = ctx.Stdout.Write(t.Bytes())
		return err
	}

	hasPool := false
	for _, service := range services {
		for _, instance := range service.ServiceInstances {
			if instance.Pool != "" {
				hasPool = true
			}
		}
	}
	table := tablecli.NewTable()
	header := []string{"Service", "Instance"}
	if hasPool {
		header = append(header, "Pool")
	}
	hasServiceWithInstances := false
	table.Headers = tablecli.Row(header)
	for _, s := range services {
		for _, instance := range s.ServiceInstances {
			hasServiceWithInstances = true
			row := []string{s.Service, instance.Name}
			if hasPool {
				row = append(row, instance.Pool)
			}
			r := tablecli.Row(row)
			table.AddRow(r)
		}
	}
	if !hasServiceWithInstances {
		return nil
	}

	_, err = ctx.Stdout.Write(table.Bytes())
	return err
}

func (s *ServiceList) clientSideFilter(services []service.ServiceModel) []service.ServiceModel {
	result := make([]service.ServiceModel, 0, len(services))

	for _, service := range services {
		if (s.filter.service != "" && service.Service == s.filter.service) || s.filter.service == "" {
			service.ServiceInstances = s.clientSideFilterInstances(service.ServiceInstances)
			service.Instances = nil
			result = append(result, service)
		}
	}

	return result
}

func (c *ServiceList) clientSideFilterInstances(serviceInstances []service.ServiceInstance) []service.ServiceInstance {
	result := make([]service.ServiceInstance, 0, len(serviceInstances))

	for _, s := range serviceInstances {
		insert := true
		if c.filter.name != "" && !strings.Contains(s.Name, c.filter.name) {
			insert = false
		}

		if c.filter.pool != "" && s.Pool != c.filter.pool {
			insert = false
		}

		if c.filter.plan != "" && s.PlanName != c.filter.plan {
			insert = false
		}

		if c.filter.teamOwner != "" && s.TeamOwner != c.filter.teamOwner {
			insert = false
		}

		if insert {
			result = append(result, s)
		}
	}

	return result
}

type ServiceInstanceAdd struct {
	fs          *pflag.FlagSet
	teamOwner   string
	description string
	tags        cmd.StringSliceFlag
	params      cmd.MapFlag
	pool        string
}

func (c *ServiceInstanceAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-add",
		Usage: "service instance add <service-name> <service-instance-name> [plan] [-t/--team-owner team] [-d/--description description] [-g/--tag tag]... [--plan-param key=value]... [--pool name]",
		Desc: `Creates a service instance of a service. There can later be binded to
applications with [[tsuru service-bind]].

This example shows how to add a new instance of **mongodb** service, named
**tsuru_mongodb** with the plan **small**:

::

	$ tsuru service instance add mongodb tsuru_mongodb small -t myteam
`,
		MinArgs: 2,
		MaxArgs: 3,
	}
}

var _ cmd.AutoCompleteCommand = &ServiceInstanceAdd{}

func (c *ServiceInstanceAdd) Complete(args []string, toComplete string) ([]string, error) {
	if len(args) == 0 {
		return completions.ServiceNameCompletionFunc(toComplete)
	}
	return nil, nil
}

func (c *ServiceInstanceAdd) Run(ctx *cmd.Context) error {
	serviceName, instanceName := ctx.Args[0], ctx.Args[1]
	var plan string
	if len(ctx.Args) > 2 {
		plan = ctx.Args[2]
	}
	parameters := make(map[string]interface{})
	for k, v := range c.params {
		parameters[k] = v
	}
	v, err := form.EncodeToValues(map[string]interface{}{"parameters": parameters})
	if err != nil {
		return err
	}
	// This is kept as this to keep backwards compatibility with older API versions
	v.Set("name", instanceName)
	v.Set("plan", plan)
	v.Set("owner", c.teamOwner)
	v.Set("description", c.description)
	v.Set("pool", c.pool)
	for _, tag := range c.tags {
		v.Add("tag", tag)
	}
	u, err := config.GetURL(fmt.Sprintf("/services/%s/instances", serviceName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Service instance successfully added.\n")
	fmt.Fprintf(ctx.Stdout, "For additional information use: tsuru service instance info %s %s\n", serviceName, instanceName)
	return nil
}

func (c *ServiceInstanceAdd) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("service-instance-add", pflag.ExitOnError)

		flagDesc := "the team that owns the service (mandatory if the user is member of more than one team)"
		c.fs.StringVar(&c.teamOwner, "team-owner", "", flagDesc)
		c.fs.MarkHidden("team-owner")
		c.fs.StringVarP(&c.teamOwner, standards.FlagTeam, standards.ShortFlagTeam, "", flagDesc)

		descriptionMessage := "service instance description"
		c.fs.StringVarP(&c.description, standards.FlagDescription, standards.ShortFlagDescription, "", descriptionMessage)

		tagMessage := "service instance tag"
		c.fs.VarP(&c.tags, standards.FlagTag, standards.ShortFlagTag, tagMessage)

		c.fs.Var(&c.params, "plan-param", "Plan specific parameters")
		c.fs.StringVar(&c.pool, "pool", "", "pool name where this service instance is going to run into (valid only for multi-cluster service)")
	}
	return c.fs
}

type ServiceInstanceCompletionMixIn struct{}

func (c *ServiceInstanceCompletionMixIn) Complete(args []string, toComplete string) ([]string, error) {
	if len(args) == 0 {
		return completions.ServiceNameCompletionFunc(toComplete)
	}
	if len(args) == 1 {
		return completions.ServiceInstanceCompletionFunc(args[0], toComplete)
	}
	return nil, nil
}

var _ cmd.AutoCompleteCommand = &ServiceInstanceUpdate{}

type ServiceInstanceUpdate struct {
	ServiceInstanceCompletionMixIn

	fs           *pflag.FlagSet
	teamOwner    string
	description  string
	plan         string
	tags         cmd.StringSliceFlag
	removeTags   cmd.StringSliceFlag
	params       cmd.MapFlag
	removeParams cmd.StringSliceFlag
}

func (c *ServiceInstanceUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-update",
		Usage: "service instance update <service-name> <service-instance-name> [-t/--team-owner team] [-d/--description description] [-p/--plan plan] [-g/--tag tag]... [--remove-tag tag]... [--add-param key=value]... [--remove-param key]...",
		Desc: `Updates a service instance.

The --team-owner (or -t) parameter updates the team owner of a service instance.

The --description (or -d) parameter sets a description for your service instance.

The --plan (or -p) parameter updates the service instance plan.

The --tag (or -g) parameter adds a tag to your service instance. This parameter may be used multiple times.

The --remove-tag removes a tag. This parameter may be used multiple times.

The --add-param (or --plan-param) adds a parameter in the service instance. This parameter may be used multiple times.

The --remove-param removes a parameter. This parameter may be used multiple times.
`,
		MinArgs: 2,
	}
}

func (c *ServiceInstanceUpdate) Run(ctx *cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	serviceName, instanceName := ctx.Args[0], ctx.Args[1]
	si, _, err := apiClient.ServiceApi.InstanceGet(context.Background(), serviceName, instanceName)
	if err != nil {
		return err
	}
	data := tsuru.ServiceInstanceUpdateData{
		Description: si.Description,
		Teamowner:   si.Teamowner,
		Plan:        si.Planname,
		Tags:        si.Tags,
		Parameters:  si.Parameters,
	}
	if c.description != "" {
		data.Description = c.description
	}
	if c.teamOwner != "" {
		data.Teamowner = c.teamOwner
	}
	if c.plan != "" {
		data.Plan = c.plan
	}
	for _, t := range c.tags {
		data.Tags = append(data.Tags, t)
	}
	for _, t := range c.removeTags {
		if i, found := findString(data.Tags, t); found {
			data.Tags = append(data.Tags[:i], data.Tags[i+1:]...)
		}
	}
	for k, v := range c.params {
		if data.Parameters == nil {
			data.Parameters = make(map[string]string)
		}
		data.Parameters[k] = v
	}
	for _, k := range c.removeParams {
		delete(data.Parameters, k)
	}
	_, err = apiClient.ServiceApi.InstanceUpdate(context.Background(), serviceName, instanceName, data)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Service successfully updated.\n")
	return nil
}

func (c *ServiceInstanceUpdate) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("service-instance-update", pflag.ExitOnError)
		c.fs.SortFlags = false

		teamOwnerMessage := "service instance team owner"
		c.fs.StringVar(&c.teamOwner, "team-owner", "", teamOwnerMessage)
		c.fs.MarkHidden("team-owner")
		c.fs.StringVarP(&c.teamOwner, standards.FlagTeam, standards.ShortFlagTeam, "", teamOwnerMessage)

		descriptionMessage := "service instance description"
		c.fs.StringVarP(&c.description, standards.FlagDescription, standards.ShortFlagDescription, "", descriptionMessage)

		planMessage := "service instance plan"
		c.fs.StringVarP(&c.plan, standards.FlagPlan, standards.ShortFlagPlan, "", planMessage)

		tagMessage := "service instance tag"
		c.fs.VarP(&c.tags, standards.FlagTag, standards.ShortFlagTag, tagMessage)

		c.fs.Var(&c.removeTags, "remove-tag", "tag to be removed from instance tags")

		planParamMessage := "parameter to be added/updated in instance parameters"
		c.fs.Var(&c.params, "plan-param", planParamMessage)
		c.fs.Var(&c.params, "add-param", planParamMessage)

		c.fs.Var(&c.removeParams, "remove-param", "parameter key to be removed from instance parameters")
	}
	return c.fs
}

var _ cmd.AutoCompleteCommand = &ServiceInstanceBind{}

type ServiceInstanceBind struct {
	ServiceInstanceCompletionMixIn

	appName   string
	jobName   string
	fs        *pflag.FlagSet
	noRestart bool
}

func (sb *ServiceInstanceBind) Run(ctx *cmd.Context) error {
	ctx.RawOutput()

	err := checkAppAndJobInputs(sb.appName, sb.jobName)
	if err != nil {
		return err
	}

	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]

	var path string
	apiVersion := "1.13"
	if sb.appName != "" {
		path = "/services/" + serviceName + "/instances/" + instanceName + "/apps/" + sb.appName
	} else {
		path = "/services/" + serviceName + "/instances/" + instanceName + "/jobs/" + sb.jobName
	}

	u, err := config.GetURLVersion(apiVersion, path)
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
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(ctx.Stdout, resp)
}

func (sb *ServiceInstanceBind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-bind",
		Usage: "service instance bind <service-name> <service-instance-name> [-a/--app appname] [-j/--job jobname] [--no-restart]",
		Desc: `Binds an application or job to a previously created service instance. See [[tsuru
service instance add]] for more details on how to create a service instance.

When binding an application or job to a service instance, tsuru will add new
environment variables to the application. All environment variables exported
by bind will be private (not accessible via [[tsuru env-get]]).`,
		MinArgs: 2,
	}
}

func (sb *ServiceInstanceBind) Flags() *pflag.FlagSet {
	if sb.fs == nil {
		sb.fs = pflag.NewFlagSet("", pflag.ExitOnError)

		sb.fs.StringVarP(&sb.appName, standards.FlagApp, standards.ShortFlagApp, "", "The name of the app.")
		sb.fs.StringVarP(&sb.jobName, standards.FlagJob, standards.ShortFlagJob, "", "The name of the job.")

		sb.fs.BoolVar(&sb.noRestart, standards.FlagNoRestart, false, "Binds an application to a service instance without restarting the application. Does not apply to jobs")
	}
	return sb.fs
}

var _ cmd.AutoCompleteCommand = &ServiceInstanceUnbind{}

type ServiceInstanceUnbind struct {
	ServiceInstanceCompletionMixIn

	appName   string
	jobName   string
	fs        *pflag.FlagSet
	noRestart bool
	force     bool
}

func (su *ServiceInstanceUnbind) Run(ctx *cmd.Context) error {
	ctx.RawOutput()

	err := checkAppAndJobInputs(su.appName, su.jobName)
	if err != nil {
		return err
	}

	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]

	var path string
	apiVersion := "1.13"
	if su.appName != "" {
		path = "/services/" + serviceName + "/instances/" + instanceName + "/apps/" + su.appName
	} else {
		path = "/services/" + serviceName + "/instances/" + instanceName + "/jobs/" + su.jobName
	}

	u, err := config.GetURLVersion(apiVersion, path)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Set("noRestart", strconv.FormatBool(su.noRestart))
	query.Set("force", strconv.FormatBool(su.force))
	request.URL.RawQuery = query.Encode()
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(ctx.Stdout, resp)
}

func (su *ServiceInstanceUnbind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-unbind",
		Usage: "service instance unbind <service-name> <service-instance-name> [-a/--app appname] [-j/--job jobname] [--no-restart] [--force]",
		Desc: `Unbinds an application or job from a service instance. After unbinding, the instance
will not be available anymore. For example, when unbinding an application from
a MySQL service, the application would lose access to the database.`,
		MinArgs: 2,
	}
}

func (su *ServiceInstanceUnbind) Flags() *pflag.FlagSet {
	if su.fs == nil {
		su.fs = pflag.NewFlagSet("", pflag.ExitOnError)
		su.fs.SortFlags = false
		su.fs.StringVarP(&su.appName, standards.FlagApp, standards.ShortFlagApp, "", "The name of the app.")
		su.fs.StringVarP(&su.jobName, standards.FlagJob, standards.ShortFlagJob, "", "The name of the job.")
		su.fs.BoolVar(&su.noRestart, standards.FlagNoRestart, false, "Unbinds an application from a service instance without restarting the application. Does not apply to jobs")
		su.fs.BoolVar(&su.force, "force", false, "Forces the unbind even if the unbind API call to the service fails")
	}
	return su.fs
}

var _ cmd.AutoCompleteCommand = &ServiceInstanceInfo{}

type ServiceInstanceInfo struct {
	ServiceInstanceCompletionMixIn
	fs   *pflag.FlagSet
	json bool
}

func (c *ServiceInstanceInfo) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("service-instance-info", pflag.ContinueOnError)
		c.fs.BoolVar(&c.json, standards.FlagJSON, false, "Show JSON")
	}
	return c.fs
}

func (c ServiceInstanceInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-instance-info",
		Usage:   "service instance info <service-name> <instance-name>",
		Desc:    `Displays the information of the given service instance.`,
		MinArgs: 2,
	}
}

type ServiceInstanceInfoModel struct {
	ServiceName     string
	InstanceName    string
	Pool            string
	Apps            []string
	Jobs            []string
	Teams           []string
	TeamOwner       string
	Description     string
	PlanName        string
	PlanDescription string
	CustomInfo      map[string]string
	Tags            []string
	Parameters      map[string]interface{}
	Status          string
}

func (c ServiceInstanceInfo) Run(ctx *cmd.Context) error {
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	url, err := config.GetURL("/services/" + serviceName + "/instances/" + instanceName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	si := &ServiceInstanceInfoModel{
		ServiceName:  serviceName,
		InstanceName: instanceName,
	}
	err = json.NewDecoder(resp.Body).Decode(si)
	if err != nil {
		return err
	}

	url, err = config.GetURL("/services/" + serviceName + "/instances/" + instanceName + "/status")
	if err != nil {
		return err
	}
	request, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err = tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bMsg, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	si.Status = string(bMsg)

	if c.json {
		return formatter.JSON(ctx.Stdout, si)
	}

	fmt.Fprintf(ctx.Stdout, "Service: %s\n", serviceName)
	fmt.Fprintf(ctx.Stdout, "Instance: %s\n", instanceName)
	if si.Pool != "" {
		fmt.Fprintf(ctx.Stdout, "Pool: %s\n", si.Pool)
	}
	if len(si.Apps) > 0 {
		fmt.Fprintf(ctx.Stdout, "Apps: %s\n", strings.Join(si.Apps, ", "))
	}

	if len(si.Jobs) > 0 {
		fmt.Fprintf(ctx.Stdout, "Jobs: %s\n", strings.Join(si.Jobs, ", "))
	}

	fmt.Fprintf(ctx.Stdout, "Teams: %s\n", formatTeams(si.TeamOwner, si.Teams))

	if si.Description != "" {
		fmt.Fprintf(ctx.Stdout, "Description: %s\n", si.Description)
	}

	if len(si.Tags) > 0 {
		fmt.Fprintf(ctx.Stdout, "Tags: %s\n", strings.Join(si.Tags, ", "))
	}

	if si.PlanName != "" {
		fmt.Fprintf(ctx.Stdout, "Plan: %s\n", si.PlanName)
	}

	if si.PlanDescription != "" {
		fmt.Fprintf(ctx.Stdout, "Plan description: %s\n", si.PlanDescription)
	}

	if len(si.Parameters) != 0 {
		var keys []string
		for k := range si.Parameters {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(ctx.Stdout, "Plan parameters:\n")
		for _, k := range keys {
			fmt.Fprintf(ctx.Stdout, "\t%s = %v\n", k, si.Parameters[k])
		}
	}
	if len(si.CustomInfo) != 0 {
		keyList := make([]string, 0)
		for key := range si.CustomInfo {
			keyList = append(keyList, key)
		}
		sort.Strings(keyList)
		for ind, key := range keyList {
			if !strings.Contains(si.CustomInfo[key], "\n") {
				fmt.Fprintf(ctx.Stdout, "%s: %s\n", key, si.CustomInfo[key])
				continue
			}

			ctx.Stdout.Write([]byte(key + ":" + "\n"))
			ctx.Stdout.Write([]byte("\t" + si.CustomInfo[key] + "\n"))
			if ind != len(keyList)-1 {
				ctx.Stdout.Write([]byte("\n"))
			}
		}
	}

	fmt.Fprintf(ctx.Stdout, "Status: %s\n", si.Status)
	return nil
}

func formatTeams(teamOwner string, teams []string) string {
	result := []string{}
	if teamOwner != "" {
		result = append(result, teamOwner+" (owner)")
	}

	for _, team := range teams {
		if team != teamOwner {
			result = append(result, team)
		}
	}

	return strings.Join(result, ", ")
}

type ServiceInfo struct {
	fs   *pflag.FlagSet
	pool string
}

func (c *ServiceInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-info",
		Usage: "service info <service-name> [-p/--pool pool]",
		Desc: `Displays a list of all instances of a given service (that the user has access
to), and apps bound to these instances.`,
		MinArgs: 1,
	}
}

var _ cmd.AutoCompleteCommand = &ServiceInfo{}

func (c *ServiceInfo) Complete(args []string, toComplete string) ([]string, error) {
	if len(args) == 0 {
		return completions.ServiceNameCompletionFunc(toComplete)
	}
	return nil, nil
}

type ServicePlanList struct {
	fs   *pflag.FlagSet
	pool string
}

func (c *ServicePlanList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-plan-list",
		Usage:   "service plan list <service-name> [-p/--pool pool]",
		Desc:    `Displays a list of all plans of a given service.`,
		MinArgs: 1,
	}
}

func (c *ServicePlanList) Flags() *pflag.FlagSet {
	if c.fs == nil {
		flagDesc := "the pool used to fetch details (could be required if the service is a multi-cluster offering)"
		c.fs = pflag.NewFlagSet("service-plan-list", pflag.ExitOnError)
		c.fs.StringVarP(&c.pool, standards.FlagPool, "p", "", flagDesc)
	}
	return c.fs
}

var _ cmd.AutoCompleteCommand = &ServicePlanList{}

func (c *ServicePlanList) Complete(args []string, toComplete string) ([]string, error) {
	if len(args) == 0 {
		return completions.ServiceNameCompletionFunc(toComplete)
	}
	return nil, nil
}

func (c *ServicePlanList) Run(ctx *cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	serviceName := ctx.Args[0]
	plans, _, err := apiClient.ServiceApi.ServicePlans(context.Background(), serviceName, &tsuru.ServicePlansOpts{
		Pool: optional.NewString(c.pool),
	})
	if err != nil {
		return err
	}

	if c.pool == "" {
		fmt.Fprintf(ctx.Stdout, "Plans for \"%s\"\n", serviceName)
	} else {
		fmt.Fprintf(ctx.Stdout, "Plans for \"%s\" in pool \"%s\"\n", serviceName, c.pool)
	}

	table := tablecli.NewTable()
	table.LineSeparator = true
	for _, plan := range plans {
		data := []string{plan.Name, plan.Description}
		table.AddRow(tablecli.Row(data))
	}
	table.Headers = tablecli.Row([]string{"Name", "Description"})
	ctx.Stdout.Write(table.Bytes())
	return nil
}

type ServiceInstanceModel struct {
	Name     string
	PlanName string
	Pool     string
	Apps     []string
	Jobs     []string
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

func (*ServiceInfo) ExtraHeaders(instances []ServiceInstanceModel) []string {
	var headers []string
	for _, instance := range instances {
		for key := range instance.Info {
			if !in(key, headers) {
				headers = append(headers, key)
			}
		}
	}
	sort.Strings(headers)
	return headers
}

func (c *ServiceInfo) BuildInstancesTable(ctx *cmd.Context, serviceName string, instances []ServiceInstanceModel) error {
	if c.pool == "" {
		fmt.Fprintf(ctx.Stdout, "Info for \"%s\"\n", serviceName)
	} else {
		fmt.Fprintf(ctx.Stdout, "Info for \"%s\" in pool \"%s\"\n", serviceName, c.pool)
		instances = filterInstancesByPool(instances, c.pool)
	}

	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Name < instances[j].Name
	})

	if len(instances) > 0 {
		fmt.Fprintln(ctx.Stdout, "\nInstances")
		table := tablecli.NewTable()
		extraHeaders := c.ExtraHeaders(instances)
		hasPlan := false
		hasPool := false
		var headers []string
		for _, instance := range instances {
			if instance.PlanName != "" {
				hasPlan = true
			}
			if instance.Pool != "" && c.pool == "" {
				hasPool = true
			}
		}
		for _, instance := range instances {
			apps := strings.Join(instance.Apps, ", ")
			jobs := strings.Join(instance.Jobs, ", ")
			row := []string{instance.Name}
			if hasPlan {
				row = append(row, instance.PlanName)
			}
			if hasPool {
				row = append(row, instance.Pool)
			}
			row = append(row, apps)
			row = append(row, jobs)

			for _, h := range extraHeaders {
				row = append(row, instance.Info[h])
			}
			table.AddRow(tablecli.Row(row))
		}
		headers = []string{"Instances"}
		if hasPlan {
			headers = append(headers, "Plan")
		}
		if hasPool {
			headers = append(headers, "Pool")
		}

		headers = append(headers, "Apps")
		headers = append(headers, "Jobs")
		headers = append(headers, extraHeaders...)

		table.Headers = tablecli.Row(headers)
		ctx.Stdout.Write(table.Bytes())
	}
	return nil
}

func filterInstancesByPool(instances []ServiceInstanceModel, pool string) []ServiceInstanceModel {
	n := 0
	for _, instance := range instances {
		if instance.Pool == pool {
			instances[n] = instance
			n++
		}
	}
	return instances[:n]
}

func (c *ServiceInfo) BuildPlansTable(ctx *cmd.Context, plans []plan) error {
	if len(plans) > 0 {
		fmt.Fprint(ctx.Stdout, "\nPlans\n")
		table := tablecli.NewTable()
		table.LineSeparator = true
		for _, plan := range plans {
			data := []string{plan.Name, plan.Description}
			table.AddRow(tablecli.Row(data))
		}
		table.Headers = tablecli.Row([]string{"Name", "Description"})
		ctx.Stdout.Write(table.Bytes())
	}
	return nil
}

func (c *ServiceInfo) WriteDoc(ctx *cmd.Context) error {
	sName := ctx.Args[0]
	url := fmt.Sprintf("/services/%s/doc", sName)
	url, err := config.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(result) != 0 {
		fmt.Fprint(ctx.Stdout, "\nDocumentation:\n")
		ctx.Stdout.Write(result)
	}
	return nil
}

func (c *ServiceInfo) Flags() *pflag.FlagSet {
	if c.fs == nil {
		flagDesc := "the pool used to fetch details (could be required if the service is a multi-cluster offering)"
		c.fs = pflag.NewFlagSet("service-info", pflag.ExitOnError)
		c.fs.StringVarP(&c.pool, standards.FlagPool, "p", "", flagDesc)
	}
	return c.fs
}

func (c *ServiceInfo) Run(ctx *cmd.Context) error {
	serviceName := ctx.Args[0]

	instances, err := c.fetchInstances(serviceName)
	if err != nil {
		return err
	}

	plans, err := c.fetchPlans(serviceName)
	if err != nil {
		return err
	}

	err = c.BuildInstancesTable(ctx, serviceName, instances)
	if err != nil {
		return err
	}
	err = c.BuildPlansTable(ctx, plans)
	if err != nil {
		return err
	}
	return c.WriteDoc(ctx)
}

func (c *ServiceInfo) fetchInstances(serviceName string) ([]ServiceInstanceModel, error) {
	url, err := config.GetURL("/services/" + serviceName)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var instances []ServiceInstanceModel
	err = json.NewDecoder(resp.Body).Decode(&instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// TODO: swap with service.Plan
type plan struct {
	Name        string
	Description string
}

func (c *ServiceInfo) fetchPlans(serviceName string) ([]plan, error) {
	v := url.Values{}
	if c.pool != "" {
		v.Set("pool", c.pool)
	}
	url, err := config.GetURL(fmt.Sprintf("/services/%s/plans?", serviceName) + v.Encode())
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	plans := []plan{}
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, err
	}
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Name < plans[j].Name
	})

	return plans, nil
}

var _ cmd.AutoCompleteCommand = &ServiceInstanceRemove{}

type ServiceInstanceRemove struct {
	cmd.ConfirmationCommand
	ServiceInstanceCompletionMixIn
	fs           *pflag.FlagSet
	force        bool
	ignoreErrors bool
}

func (c *ServiceInstanceRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-remove",
		Usage: "service instance remove <service-name> <service-instance-name> [-f/--force] [--ignore-errors] [-y/--assume-yes]",
		Desc: `Destroys a service instance. It can't remove a service instance that is bound
to an app, so before remove a service instance, make sure there is no apps
bound to it (see [[tsuru service-instance-info]] command).`,
		MinArgs: 2,
	}
}

func (c *ServiceInstanceRemove) Run(ctx *cmd.Context) error {
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	msg := fmt.Sprintf("Are you sure you want to remove the instance %q", instanceName)
	if c.force {
		msg += " and all binds"
	}
	if !c.Confirm(ctx, msg+"?") {
		return nil
	}
	qs := url.Values{}
	qs.Set("unbindall", strconv.FormatBool(c.force))
	qs.Set("ignoreerrors", strconv.FormatBool(c.ignoreErrors))
	url := fmt.Sprintf("/services/%s/instances/%s?%s", serviceName, instanceName, qs.Encode())
	url, err := config.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(ctx.Stdout, resp)
}

func (c *ServiceInstanceRemove) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.ConfirmationCommand.Flags()
		c.fs.BoolVarP(&c.force, "force", "f", false, "Forces the removal of a service instance binded to apps.")
		c.fs.BoolVar(&c.ignoreErrors, "ignore-errors", false, "Ignore errors returned by service backend.")
	}
	return c.fs
}

var _ cmd.AutoCompleteCommand = &ServiceInstanceGrant{}

type ServiceInstanceGrant struct {
	ServiceInstanceCompletionMixIn
}

func (c *ServiceInstanceGrant) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-instance-grant",
		Usage:   "service instance grant <service-name> <service-instance-name> <team-name>",
		Desc:    `Grant access to team in a service instance.`,
		MinArgs: 3,
	}
}

func (c *ServiceInstanceGrant) Complete(args []string, toComplete string) ([]string, error) {
	if len(args) == 2 {
		return completions.TeamNameCompletionFunc(toComplete)
	}

	return c.ServiceInstanceCompletionMixIn.Complete(args, toComplete)
}

func (c *ServiceInstanceGrant) Run(ctx *cmd.Context) error {
	sName := ctx.Args[0]
	siName := ctx.Args[1]
	teamName := ctx.Args[2]
	url := fmt.Sprintf("/services/%s/instances/permission/%s/%s", sName, siName, teamName)
	url, err := config.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	_, err = tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, `Granted access to team %s in %s service instance.`+"\n", teamName, siName)
	return nil
}

var _ cmd.AutoCompleteCommand = &ServiceInstanceRevoke{}

type ServiceInstanceRevoke struct {
	ServiceInstanceCompletionMixIn
}

func (c *ServiceInstanceRevoke) Complete(args []string, toComplete string) ([]string, error) {
	if len(args) == 2 {
		return completions.TeamNameCompletionFunc(toComplete)
	}

	return c.ServiceInstanceCompletionMixIn.Complete(args, toComplete)
}

func (c *ServiceInstanceRevoke) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-instance-revoke",
		Usage:   "service instance revoke <service-name> <service-instance-name> <team-name>",
		Desc:    `Revoke access to team in a service instance.`,
		MinArgs: 3,
	}
}

func (c *ServiceInstanceRevoke) Run(ctx *cmd.Context) error {
	sName := ctx.Args[0]
	siName := ctx.Args[1]
	teamName := ctx.Args[2]
	url := fmt.Sprintf("/services/%s/instances/permission/%s/%s", sName, siName, teamName)
	url, err := config.GetURL(url)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, `Revoked access to team %s in %s service instance.`+"\n", teamName, siName)
	return nil
}

func findString(strs []string, s string) (int, bool) {
	for i, ss := range strs {
		if ss == s {
			return i, true
		}
	}

	return -1, false
}
