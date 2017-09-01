// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/tsuru/tsuru/cmd"
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

func (t *TagList) Run(context *cmd.Context, client *cmd.Client) error {
	apps, err := loadApps(client)
	if err != nil {
		return err
	}
	services, err := loadServices(client)
	if err != nil {
		return err
	}
	return t.Show(apps, services, context)
}

func (t *TagList) Show(apps []app, services []service.ServiceModel, context *cmd.Context) error {
	tagList := processTags(apps, services)
	if len(tagList) == 0 {
		return nil
	}
	table := cmd.NewTable()
	table.Headers = cmd.Row([]string{"Tag", "Apps", "Service Instances"})
	for _, tagName := range sortedTags(tagList) {
		t := tagList[tagName]
		instanceNames := make([]string, len(t.ServiceInstances))
		for i, serviceName := range sortedServices(t.ServiceInstances) {
			instances := t.ServiceInstances[serviceName]
			instanceNames[i] = fmt.Sprintf("%s: %s", serviceName, strings.Join(instances, ", "))
		}
		table.AddRow(cmd.Row([]string{t.Name, strings.Join(t.Apps, ", "), strings.Join(instanceNames, "\n")}))
	}
	table.LineSeparator = true
	table.Sort()
	context.Stdout.Write(table.Bytes())
	return nil
}

func loadApps(client *cmd.Client) ([]app, error) {
	result, err := getFromURL("/apps", client)
	if err != nil {
		return nil, err
	}
	var apps []app
	err = json.Unmarshal(result, &apps)
	return apps, err
}

func loadServices(client *cmd.Client) ([]service.ServiceModel, error) {
	result, err := getFromURL("/services", client)
	if err != nil {
		return nil, err
	}
	var services []service.ServiceModel
	err = json.Unmarshal(result, &services)
	return services, err
}

func getFromURL(path string, client *cmd.Client) ([]byte, error) {
	url, err := cmd.GetURL(path)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
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
