// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/cezarsa/form"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas"
)

type MachineList struct{}

func (c *MachineList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "machine-list",
		Usage: "machine-list",
		Desc: `Lists all machines created using an IaaS provider.
These machines were created with the [[docker-node-add]] command.`,
		MinArgs: 0,
	}
}

func (c *MachineList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/iaas/machines")
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
	var machines []iaas.Machine
	err = json.NewDecoder(response.Body).Decode(&machines)
	if err != nil {
		return err
	}
	table := cmd.NewTable()
	table.Headers = cmd.Row([]string{"Id", "IaaS", "Address", "Creation Params"})
	table.LineSeparator = true
	for _, machine := range machines {
		var params []string
		for k, v := range machine.CreationParams {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(params)
		table.AddRow(cmd.Row([]string{machine.Id, machine.Iaas, machine.Address, strings.Join(params, "\n")}))
	}
	table.Sort()
	context.Stdout.Write(table.Bytes())
	return nil
}

type MachineDestroy struct{}

func (c *MachineDestroy) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "machine-destroy",
		Usage:   "machine-destroy <machine id>",
		Desc:    "Destroys an existing machine created using a IaaS.",
		MinArgs: 1,
	}
}
func (c *MachineDestroy) Run(context *cmd.Context, client *cmd.Client) error {
	machineId := context.Args[0]
	url, err := cmd.GetURL("/iaas/machines/" + machineId)
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
	fmt.Fprintln(context.Stdout, "Machine successfully destroyed.")
	return nil
}

type TemplateList struct{}

func (c *TemplateList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "machine-template-list",
		Usage:   "machine-template-list",
		Desc:    "Lists all machine templates.",
		MinArgs: 0,
	}
}

func (c *TemplateList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/iaas/templates")
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
	var templates []iaas.Template
	err = json.NewDecoder(response.Body).Decode(&templates)
	if err != nil {
		return err
	}
	table := cmd.NewTable()
	table.Headers = cmd.Row([]string{"Name", "IaaS", "Params"})
	table.LineSeparator = true
	for _, template := range templates {
		var params []string
		for _, data := range template.Data {
			params = append(params, fmt.Sprintf("%s=%s", data.Name, data.Value))
		}
		sort.Strings(params)
		table.AddRow(cmd.Row([]string{template.Name, template.IaaSName, strings.Join(params, "\n")}))
	}
	table.Sort()
	context.Stdout.Write(table.Bytes())
	return nil
}

type TemplateAdd struct{}

func (c *TemplateAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "machine-template-add",
		Usage: "machine-template-add <name> <iaas> <param>=<value>...",
		Desc: `Creates a new machine template.

Templates can be used with the [[docker-node-add]] command running it with
the [[template=<template name>]] parameter. Templates can contain a list of
parameters that will be sent to the IaaS provider.`,
		MinArgs: 3,
	}
}

func (c *TemplateAdd) Run(context *cmd.Context, client *cmd.Client) error {
	var template iaas.Template
	template.Name = context.Args[0]
	template.IaaSName = context.Args[1]
	for _, param := range context.Args[2:] {
		if strings.Contains(param, "=") {
			keyValue := strings.SplitN(param, "=", 2)
			template.Data = append(template.Data, iaas.TemplateData{
				Name:  keyValue[0],
				Value: keyValue[1],
			})
		}
	}
	v, err := form.EncodeToValues(&template)
	if err != nil {
		return err
	}
	u, err := cmd.GetURL("/iaas/templates")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, bytes.NewBufferString(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		context.Stderr.Write([]byte("Failed to add template.\n"))
		return err
	}
	context.Stdout.Write([]byte("Template successfully added.\n"))
	return nil
}

type TemplateRemove struct{}

func (c *TemplateRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "machine-template-remove",
		Usage:   "machine-template-remove <name>",
		Desc:    "Removes an existing machine template.",
		MinArgs: 1,
	}
}

func (c *TemplateRemove) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/iaas/templates/" + context.Args[0])
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		context.Stderr.Write([]byte("Failed to remove template.\n"))
		return err
	}
	context.Stdout.Write([]byte("Template successfully removed.\n"))
	return nil
}

type TemplateUpdate struct{}

func (c *TemplateUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "machine-template-update",
		Usage:   "machine-template-update <name> <param>=<value>...",
		Desc:    "Update an existing machine template.",
		MinArgs: 2,
	}
}

func (c *TemplateUpdate) Run(context *cmd.Context, client *cmd.Client) error {
	template := iaas.Template{Name: context.Args[0]}
	for _, param := range context.Args[1:] {
		if strings.Contains(param, "=") {
			keyValue := strings.SplitN(param, "=", 2)
			template.Data = append(template.Data, iaas.TemplateData{
				Name:  keyValue[0],
				Value: keyValue[1],
			})
		}
	}
	v, err := form.EncodeToValues(&template)
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/iaas/templates/%s", template.Name))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		context.Stderr.Write([]byte("Failed to update template.\n"))
		return err
	}
	context.Stdout.Write([]byte("Template successfully updated.\n"))
	return nil
}
