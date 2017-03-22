// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
)

type TagList struct{}

func (t *TagList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "tag-list",
		Usage: "tag-list",
		Desc:  `Retrieves and shows a list of tags with the respective apps and service instances.`,
	}
}

func (t *TagList) Run(context *cmd.Context, client *cmd.Client) error {
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
	if response.StatusCode != http.StatusOK {
		return nil
	}
	defer response.Body.Close()
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	return t.Show(b, context)
}

func (t *TagList) Show(result []byte, context *cmd.Context) error {
	var apps []app
	err := json.Unmarshal(result, &apps)
	if err != nil {
		return err
	}
	tagList := make(map[string][]string)
	for _, app := range apps {
		for _, tag := range app.Tags {
			tagList[tag] = append(tagList[tag], app.Name)
		}
	}
	if len(tagList) > 0 {
		table := cmd.NewTable()
		table.Headers = cmd.Row([]string{"Tag", "Apps"})
		for tag, appNames := range tagList {
			table.AddRow(cmd.Row([]string{tag, strings.Join(appNames, ", ")}))
		}
		table.LineSeparator = true
		table.Sort()
		context.Stdout.Write(table.Bytes())
	}
	return nil
}
