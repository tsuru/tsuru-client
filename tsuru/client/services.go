// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/ajg/form"
	"github.com/antihax/optional"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/tsuru/gnuflag"
	tsuruClient "github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/service"
)

type serviceFilter struct {
	name      string
	pool      string
	plan      string
	service   string
	teamOwner string
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
	return result, nil
}

type ServiceList struct {
	fs               *gnuflag.FlagSet
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

func (c *ServiceList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("service-list", gnuflag.ExitOnError)
		c.fs.StringVar(&c.filter.service, "service", "", "Filter instances by service")
		c.fs.StringVar(&c.filter.service, "s", "", "Filter instances by service")
		c.fs.StringVar(&c.filter.name, "name", "", "Filter service instances by name")
		c.fs.StringVar(&c.filter.name, "n", "", "Filter service instances by name")
		c.fs.StringVar(&c.filter.pool, "pool", "", "Filter service instances by pool")
		c.fs.StringVar(&c.filter.pool, "o", "", "Filter service instances by pool")
		c.fs.StringVar(&c.filter.plan, "plan", "", "Filter service instances by plan")
		c.fs.StringVar(&c.filter.plan, "p", "", "Filter service instances by plan")
		c.fs.StringVar(&c.filter.teamOwner, "team", "", "Filter service instances by team owner")
		c.fs.StringVar(&c.filter.teamOwner, "t", "", "Filter service instances by team owner")
		c.fs.BoolVar(&c.simplified, "q", false, "Display only service instances name")
		c.fs.BoolVar(&c.json, "json", false, "Display in JSON format")
		c.fs.BoolVar(&c.justServiceNames, "j", false, "Display just service names")

	}
	return c.fs
}

func (s ServiceList) Run(ctx *cmd.Context, client *cmd.Client) error {
	qs, err := s.filter.queryString()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/services/instances?%s", qs.Encode()))
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
	table.Headers = tablecli.Row(header)
	for _, s := range services {
		for _, instance := range s.ServiceInstances {
			row := []string{s.Service, instance.Name}
			if hasPool {
				row = append(row, instance.Pool)
			}
			r := tablecli.Row(row)
			table.AddRow(r)
		}
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
	fs          *gnuflag.FlagSet
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

func (c *ServiceInstanceAdd) Run(ctx *cmd.Context, client *cmd.Client) error {
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
	u, err := cmd.GetURL(fmt.Sprintf("/services/%s/instances", serviceName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Service instance successfully added.\n")
	fmt.Fprintf(ctx.Stdout, "For additional information use: tsuru service instance info %s %s\n", serviceName, instanceName)
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
		c.fs.Var(&c.params, "plan-param", "Plan specific parameters")
		c.fs.StringVar(&c.pool, "pool", "", "pool name where this service instance is going to run into (valid only for multi-cluster service)")
	}
	return c.fs
}

type ServiceInstanceUpdate struct {
	fs           *gnuflag.FlagSet
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

func (c *ServiceInstanceUpdate) Run(ctx *cmd.Context, client *cmd.Client) error {
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
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

func (c *ServiceInstanceUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("service-instance-update", gnuflag.ExitOnError)

		teamOwnerMessage := "service instance team owner"
		c.fs.StringVar(&c.teamOwner, "team-owner", "", teamOwnerMessage)
		c.fs.StringVar(&c.teamOwner, "t", "", teamOwnerMessage)
		descriptionMessage := "service instance description"
		c.fs.StringVar(&c.description, "description", "", descriptionMessage)
		c.fs.StringVar(&c.description, "d", "", descriptionMessage)
		planMessage := "service instance plan"
		c.fs.StringVar(&c.plan, "plan", "", planMessage)
		c.fs.StringVar(&c.plan, "p", "", planMessage)
		tagMessage := "service instance tag"
		c.fs.Var(&c.tags, "tag", tagMessage)
		c.fs.Var(&c.tags, "g", tagMessage)
		c.fs.Var(&c.removeTags, "remove-tag", "tag to be removed from instance tags")
		planParamMessage := "parameter to be added/updated in instance parameters"
		c.fs.Var(&c.params, "plan-param", planParamMessage)
		c.fs.Var(&c.params, "add-param", planParamMessage)
		c.fs.Var(&c.removeParams, "remove-param", "parameter key to be removed from instance parameters")
	}
	return c.fs
}

type ServiceInstanceBind struct {
	appName   string
	jobName   string
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (sb *ServiceInstanceBind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()

	if sb.appName == "" && sb.jobName == "" {
		return errors.New("You must pass an application or job")
	}
	if sb.appName != "" && sb.jobName != "" {
		return errors.New("You must pass an application or job, never both")
	}

	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]

	var path string
	apiVersion := "1.0"
	if sb.appName != "" {
		path = "/services/" + serviceName + "/instances/" + instanceName + "/" + sb.appName
	} else {
		path = "/services/" + serviceName + "/instances/" + instanceName + "/jobs/" + sb.jobName
		apiVersion = "1.13"
	}

	u, err := cmd.GetURLVersion(apiVersion, path)
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
	return cmd.StreamJSONResponse(ctx.Stdout, resp)
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

func (sb *ServiceInstanceBind) Flags() *gnuflag.FlagSet {
	if sb.fs == nil {
		sb.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)

		sb.fs.StringVar(&sb.appName, "app", "", "The name of the app.")
		sb.fs.StringVar(&sb.appName, "a", "", "The name of the app.")
		sb.fs.StringVar(&sb.jobName, "job", "", "The name of the job.")
		sb.fs.StringVar(&sb.jobName, "j", "", "The name of the job.")
		sb.fs.BoolVar(&sb.noRestart, "no-restart", false, "Binds an application to a service instance without restart the application")
	}
	return sb.fs
}

type ServiceInstanceUnbind struct {
	cmd.AppNameMixIn
	fs        *gnuflag.FlagSet
	noRestart bool
	force     bool
}

func (su *ServiceInstanceUnbind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	appName, err := su.AppName()
	if err != nil {
		return err
	}
	serviceName := ctx.Args[0]
	instanceName := ctx.Args[1]
	u, err := cmd.GetURL(fmt.Sprintf("/services/%s/instances/%s/%s", serviceName, instanceName, appName))
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
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(ctx.Stdout, resp)
}

func (su *ServiceInstanceUnbind) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-instance-unbind",
		Usage: "service instance unbind <service-name> <service-instance-name> [-a/--app appname] [--no-restart] [--force]",
		Desc: `Unbinds an application from a service instance. After unbinding, the instance
will not be available anymore. For example, when unbinding an application from
a MySQL service, the application would lose access to the database.`,
		MinArgs: 2,
	}
}

func (su *ServiceInstanceUnbind) Flags() *gnuflag.FlagSet {
	if su.fs == nil {
		su.fs = su.AppNameMixIn.Flags()
		su.fs.BoolVar(&su.noRestart, "no-restart", false, "Unbinds an application from a service instance without restart the application")
		su.fs.BoolVar(&su.force, "force", false, "Forces the unbind even if the unbind API call to the service fails")
	}
	return su.fs
}

type ServiceInstanceInfo struct {
	fs   *gnuflag.FlagSet
	json bool
}

func (c *ServiceInstanceInfo) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("service-instance-info", gnuflag.ContinueOnError)
		c.fs.BoolVar(&c.json, "json", false, "Show JSON")
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
	si := &ServiceInstanceInfoModel{
		ServiceName:  serviceName,
		InstanceName: instanceName,
	}
	err = json.NewDecoder(resp.Body).Decode(si)
	if err != nil {
		return err
	}

	url, err = cmd.GetURL("/services/" + serviceName + "/instances/" + instanceName + "/status")
	if err != nil {
		return err
	}
	request, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err = client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bMsg, err := ioutil.ReadAll(resp.Body)
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
	fs   *gnuflag.FlagSet
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

type ServicePlanList struct {
	fs   *gnuflag.FlagSet
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

func (c *ServicePlanList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		flagDesc := "the pool used to fetch details (could be required if the service is a multi-cluster offering)"
		c.fs = gnuflag.NewFlagSet("service-plan-list", gnuflag.ExitOnError)
		c.fs.StringVar(&c.pool, "pool", "", flagDesc)
		c.fs.StringVar(&c.pool, "p", "", flagDesc)
	}
	return c.fs
}

func (c *ServicePlanList) Run(ctx *cmd.Context, client *cmd.Client) error {
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
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
			row := []string{instance.Name}
			if hasPlan {
				row = append(row, instance.PlanName)
			}
			if hasPool {
				row = append(row, instance.Pool)
			}
			row = append(row, apps)

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
			instanceParam, bindParam := parsePlanParams(plan.Schemas)
			data := []string{plan.Name, plan.Description, instanceParam, bindParam}
			table.AddRow(tablecli.Row(data))
		}
		table.Headers = tablecli.Row([]string{"Name", "Description", "Instance Params", "Binding Params"})
		ctx.Stdout.Write(table.Bytes())
	}
	return nil
}

func parsePlanParams(schemas *osb.Schemas) (instanceParams string, bindingParams string) {
	if schemas == nil {
		return instanceParams, bindingParams
	}
	var err error
	if schemas.ServiceInstance != nil {
		instanceParams, err = parseParams(schemas.ServiceInstance.Create.Parameters)
		if err != nil {
			instanceParams = fmt.Sprintf("error parsing %+v: %v", schemas.ServiceInstance.Create, err)
		}
	}
	if schemas.ServiceBinding != nil {
		bindingParams, err = parseParams(schemas.ServiceBinding.Create.Parameters)
		if err != nil {
			bindingParams = fmt.Sprintf("error parsing %+v: %v", schemas.ServiceBinding.Create, err)
		}
	}
	return instanceParams, bindingParams
}

type jsonSchema struct {
	Properties  map[string]*jsonSchema
	Default     interface{}
	Type        string
	Description string
	Required    []string
}

func parseParams(params interface{}) (string, error) {
	if params == nil {
		return "", nil
	}
	d, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	var schema jsonSchema
	err = json.Unmarshal(d, &schema)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	var props []string
	for k := range schema.Properties {
		props = append(props, k)
	}
	sort.Strings(props)
	requireMap := make(map[string]struct{})
	for _, r := range schema.Required {
		requireMap[r] = struct{}{}
	}
	for _, k := range props {
		v := schema.Properties[k]
		if v == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("%v: \n", k))
		sb.WriteString(fmt.Sprintf("  description: %v\n", v.Description))
		sb.WriteString(fmt.Sprintf("  type: %v\n", v.Type))
		if v.Default != nil {
			sb.WriteString(fmt.Sprintf("  default: %v\n", v.Default))
		}
		if _, ok := requireMap[k]; ok {
			sb.WriteString("  required: true\n")
		}
	}
	return sb.String(), nil
}

func (c *ServiceInfo) WriteDoc(ctx *cmd.Context, client *cmd.Client) error {
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

func (c *ServiceInfo) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		flagDesc := "the pool used to fetch details (could be required if the service is a multi-cluster offering)"
		c.fs = gnuflag.NewFlagSet("service-instance-add", gnuflag.ExitOnError)
		c.fs.StringVar(&c.pool, "pool", "", flagDesc)
		c.fs.StringVar(&c.pool, "p", "", flagDesc)
	}
	return c.fs
}

func (c *ServiceInfo) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]

	instances, err := c.fetchInstances(serviceName, client)
	if err != nil {
		return err
	}

	plans, err := c.fetchPlans(serviceName, client)
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
	return c.WriteDoc(ctx, client)
}

func (c *ServiceInfo) fetchInstances(serviceName string, client *cmd.Client) ([]ServiceInstanceModel, error) {
	url, err := cmd.GetURL("/services/" + serviceName)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
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
	Schemas     *osb.Schemas
}

func (c *ServiceInfo) fetchPlans(serviceName string, client *cmd.Client) ([]plan, error) {
	v := url.Values{}
	if c.pool != "" {
		v.Set("pool", c.pool)
	}
	url, err := cmd.GetURL(fmt.Sprintf("/services/%s/plans?", serviceName) + v.Encode())
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
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

type ServiceInstanceRemove struct {
	cmd.ConfirmationCommand
	fs           *gnuflag.FlagSet
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
	qs := url.Values{}
	qs.Set("unbindall", strconv.FormatBool(c.force))
	qs.Set("ignoreerrors", strconv.FormatBool(c.ignoreErrors))
	url := fmt.Sprintf("/services/%s/instances/%s?%s", serviceName, instanceName, qs.Encode())
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
	return cmd.StreamJSONResponse(ctx.Stdout, resp)
}

func (c *ServiceInstanceRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.ConfirmationCommand.Flags()
		c.fs.BoolVar(&c.force, "f", false, "Forces the removal of a service instance binded to apps.")
		c.fs.BoolVar(&c.force, "force", false, "Forces the removal of a service instance binded to apps.")
		c.fs.BoolVar(&c.ignoreErrors, "ignore-errors", false, "Ignore errors returned by service backend.")
	}
	return c.fs
}

type ServiceInstanceGrant struct{}

func (c *ServiceInstanceGrant) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-instance-grant",
		Usage:   "service instance grant <service-name> <service-instance-name> <team-name>",
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
		Usage:   "service instance revoke <service-name> <service-instance-name> <team-name>",
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

func findString(strs []string, s string) (int, bool) {
	for i, ss := range strs {
		if ss == s {
			return i, true
		}
	}

	return -1, false
}
