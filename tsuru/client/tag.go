// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/service"
)

type TagList struct{}

type tag struct {
	Name             string
	Apps             []string
	ServiceInstances map[string][]string
}

func (t *TagList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "tag-list",
		Usage: "tag-list",
		Desc:  `Retrieves and shows a list of tags with the respective apps and service instances.`,
	}
}

func (t *TagList) Run(context *cmd.Context) error {
	apps, appErr := loadApps()
	if appErr != nil {
		apps = []app{}
	}
	services, serviceErr := loadServices()
	if serviceErr != nil {
		services = []service.ServiceModel{}
	}

	if appErr != nil && serviceErr != nil {
		fmt.Fprintln(context.Stdout, "Unable to access apps and services tags")
		return nil
	}

	return t.Show(apps, services, context)
}

func (t *TagList) Show(apps []app, services []service.ServiceModel, context *cmd.Context) error {
	tagList := processTags(apps, services)
	if len(tagList) == 0 {
		return nil
	}

	// The output of this Table does not look good with tab writer enabled
	originalUseTabWriter := tablecli.TableConfig.UseTabWriter
	tablecli.TableConfig.UseTabWriter = false
	defer func() { tablecli.TableConfig.UseTabWriter = originalUseTabWriter }()

	table := tablecli.NewTable()
	table.Headers = tablecli.Row([]string{"Tag", "Apps", "Service Instances"})
	for _, tagName := range sortedTags(tagList) {
		t := tagList[tagName]
		var instanceNames []string
		for _, serviceName := range sortedServices(t.ServiceInstances) {
			instances := t.ServiceInstances[serviceName]
			for _, instanceName := range instances {
				instanceNames = append(instanceNames, fmt.Sprintf("%s: %s", serviceName, instanceName))
			}
		}
		table.AddRow(tablecli.Row([]string{t.Name, strings.Join(t.Apps, "\n"), strings.Join(instanceNames, "\n")}))
	}
	table.LineSeparator = true
	table.Sort()
	context.Stdout.Write(table.Bytes())
	return nil
}

func loadApps() ([]app, error) {
	result, err := getFromURL("/apps")
	if err != nil {
		return nil, err
	}
	var apps []app
	err = json.Unmarshal(result, &apps)
	return apps, err
}

func loadServices() ([]service.ServiceModel, error) {
	result, err := getFromURL("/services")
	if err != nil {
		return nil, err
	}
	var services []service.ServiceModel
	err = json.Unmarshal(result, &services)
	return services, err
}

func getFromURL(path string) ([]byte, error) {
	url, err := config.GetURL(path)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode, string(body))
	}
	return body, nil
}

func processTags(apps []app, services []service.ServiceModel) map[string]*tag {
	tagList := make(map[string]*tag)
	for _, app := range apps {
		for _, t := range app.Tags {
			if _, ok := tagList[t]; !ok {
				tagList[t] = &tag{Name: t, Apps: []string{app.Name}}
			} else {
				tagList[t].Apps = append(tagList[t].Apps, app.Name)
			}
		}
	}
	for _, s := range services {
		for _, instance := range s.ServiceInstances {
			for _, t := range instance.Tags {
				if _, ok := tagList[t]; !ok {
					tagList[t] = &tag{Name: t, ServiceInstances: make(map[string][]string)}
				}
				si := &tagList[t].ServiceInstances
				if *si == nil {
					*si = make(map[string][]string)
				}
				(*si)[s.Service] = append((*si)[s.Service], instance.Name)
			}
		}
	}
	return tagList
}

func sortedTags(tagList map[string]*tag) []string {
	tagNames := make([]string, len(tagList))
	i := 0
	for t := range tagList {
		tagNames[i] = t
		i++
	}
	sort.Strings(tagNames)
	return tagNames
}

func sortedServices(services map[string][]string) []string {
	serviceNames := make([]string, len(services))
	i := 0
	for s := range services {
		serviceNames[i] = s
		i++
	}
	sort.Strings(serviceNames)
	return serviceNames
}
