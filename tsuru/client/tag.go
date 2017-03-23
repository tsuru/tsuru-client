// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	if len(tagList) > 0 {
		table := cmd.NewTable()
		table.Headers = cmd.Row([]string{"Tag", "Apps", "Service Instances"})
		for _, t := range tagList {
			instanceNames := ""
			for serviceName, instances := range t.ServiceInstances {
				if len(instanceNames) > 0 {
					instanceNames += "\n"
				}
				instanceNames += fmt.Sprintf("%s: %s", serviceName, strings.Join(instances, ", "))
			}
			table.AddRow(cmd.Row([]string{t.Name, strings.Join(t.Apps, ", "), instanceNames}))
		}
		table.LineSeparator = true
		table.Sort()
		context.Stdout.Write(table.Bytes())
	}
	return nil
}

func loadApps(client *cmd.Client) ([]app, error) {
	result, err := getFromUrl("/apps", client)
	if err != nil {
		return nil, err
	}
	var apps []app
	err = json.Unmarshal(result, &apps)
	return apps, nil
}

func loadServices(client *cmd.Client) ([]service.ServiceModel, error) {
	result, err := getFromUrl("/services", client)
	if err != nil {
		return nil, err
	}
	var services []service.ServiceModel
	err = json.Unmarshal(result, &services)
	return services, nil
}

func getFromUrl(path string, client *cmd.Client) ([]byte, error) {
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
	if response.StatusCode != http.StatusOK {
		return nil, nil
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
				if tagList[t].ServiceInstances == nil {
					tagList[t].ServiceInstances = make(map[string][]string)
				}
				tagList[t].ServiceInstances[s.Service] = append(tagList[t].ServiceInstances[s.Service], instance.Name)
			}
		}
	}
	return tagList
}
