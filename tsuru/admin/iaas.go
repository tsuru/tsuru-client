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

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas"
)

const filterMessage = "Filter metadata name and value"

type MachineList struct {
	fs                *gnuflag.FlagSet
	filter            cmd.MapFlag
	simplifiedID      bool
	simplifiedAddress bool
}

func (c *MachineList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "machine-list",
		Usage: "machine-list [--filter/-f <metadata>=<value>]...",
		Desc: `Lists all machines created using an IaaS provider.
These machines were created with the [[node-add]] command.`,
		MinArgs: 0,
	}
}

func (c *MachineList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.Var(&c.filter, "filter", filterMessage)
		c.fs.Var(&c.filter, "f", filterMessage)
		c.fs.BoolVar(&c.simplifiedID, "i", false, "Display only machine Id on IaaS")
		c.fs.BoolVar(&c.simplifiedAddress, "q", false, "Display only machine address on IaaS")
	}
	return c.fs
}

func (c *MachineList) Run(context *cmd.Context, client *cmd.Client) error {
	machines, err := c.List(client)
	if err != nil {
		return err
	}
	if len(machines) > 0 {
		machines = c.filterMachines(machines)
	}
	if c.simplifiedAddress && c.simplifiedID {
		return fmt.Errorf("-i and -q flags are mutually exclusive")
	}
	if c.simplifiedID {
		for _, machine := range machines {
			fmt.Fprintln(context.Stdout, machine.Id)
		}
		return nil
	}
	if c.simplifiedAddress {
		for _, machine := range machines {
			fmt.Fprintln(context.Stdout, machine.Address)
		}
		return nil
	}
	tmplList := TemplateList{}
	templates, err := tmplList.List(client)
	if err != nil {
		return err
	}
	tmplParams := make(map[string]map[string]string)
	for _, t := range templates {
		tmplMap := make(map[string]string)
		for _, d := range t.Data {
			tmplMap[d.Name] = d.Value
		}
		tmplParams[t.Name] = tmplMap
	}
	machineToTemplates := make(map[string][]string)
	for _, m := range machines {
	loop:
		for t, tmap := range tmplParams {
			if len(tmap) > len(m.CreationParams) {
				continue
			}

			for k, v := range tmap {
				if m.CreationParams[k] != v {
					continue loop
				}
			}
			machineToTemplates[m.Id] = append(machineToTemplates[m.Id], t)
		}
	}
	table := c.Tabulate(machines, machineToTemplates)
	context.Stdout.Write(table.Bytes())
	return nil
}

func (c *MachineList) List(client *cmd.Client) ([]iaas.Machine, error) {
	url, err := cmd.GetURL("/iaas/machines")
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
	var machines []iaas.Machine
	err = json.NewDecoder(response.Body).Decode(&machines)
	return machines, err
}

func (c *MachineList) Tabulate(machines []iaas.Machine, machineToTemplate map[string][]string) *tablecli.Table {
	table := tablecli.NewTable()
	table.Headers = tablecli.Row([]string{"Id", "IaaS", "Address", "Creation Params", "Matching Templates"})
	table.LineSeparator = true
	for _, machine := range machines {
		var params []string
		for k, v := range machine.CreationParams {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(params)
		sort.Strings(machineToTemplate[machine.Id])
		table.AddRow(tablecli.Row([]string{machine.Id, machine.Iaas, machine.Address, strings.Join(params, "\n"),
			strings.Join(machineToTemplate[machine.Id], "\n")}))
	}
	table.Sort()
	return table
}

func (c *MachineList) filterMachines(machines []iaas.Machine) []iaas.Machine {
	filteredMachines := make([]iaas.Machine, 0)
	for _, m := range machines {
		if c.machineMetadataMatchesFilters(m) {
			filteredMachines = append(filteredMachines, m)
		}
	}
	return filteredMachines
}

func (c *MachineList) machineMetadataMatchesFilters(machine iaas.Machine) bool {
	for key, value := range c.filter {
		metaVal := machine.CreationParams[key]
		if metaVal != value {
			return false
		}
	}
	return true
}

type MachineDestroy struct {
	cmd.ConfirmationCommand
}

func (c *MachineDestroy) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "machine-destroy",
		Usage:   "machine-destroy <machine id> [-y/--assume-yes]",
		Desc:    "Destroys an existing machine created using a IaaS.",
		MinArgs: 1,
	}
}

func (c *MachineDestroy) Run(context *cmd.Context, client *cmd.Client) error {
	machineID := context.Args[0]
	if !c.Confirm(context, fmt.Sprintf("Are you sure you want to remove machine %q?", machineID)) {
		return nil
	}
	url, err := cmd.GetURL("/iaas/machines/" + machineID)
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
	fmt.Fprintln(context.Stdout, "Machine successfully destroyed.")
	return nil
}

type TemplateList struct {
	countMachines bool
	fs            *gnuflag.FlagSet
	filter        cmd.MapFlag
	simplified    bool
}

func (c *TemplateList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "machine-template-list",
		Usage:   "machine-template-list [--count] [--filter/-f <metadata>=<value>]",
		Desc:    "Lists all machine templates.",
		MinArgs: 0,
	}
}

func (c *TemplateList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.countMachines, "count", false, "Count machines using each template.")
		c.fs.BoolVar(&c.countMachines, "c", false, "Count machines using each template.")
		c.fs.Var(&c.filter, "filter", filterMessage)
		c.fs.Var(&c.filter, "f", filterMessage)
		c.fs.BoolVar(&c.simplified, "q", false, "Display only machine template name")
	}
	return c.fs
}

func (c *TemplateList) List(client *cmd.Client) ([]iaas.Template, error) {
	url, err := cmd.GetURL("/iaas/templates")
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
	var templates []iaas.Template
	err = json.NewDecoder(response.Body).Decode(&templates)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func (c *TemplateList) Run(context *cmd.Context, client *cmd.Client) error {
	templates, err := c.List(client)
	if err != nil {
		return err
	}
	if len(templates) > 0 {
		templates = c.filterTemplates(templates)
	}
	if c.simplified {
		for _, template := range templates {
			fmt.Fprintln(context.Stdout, template.Name)
		}
		return nil
	}
	templateIndex := make(map[string]int)
	if c.countMachines {
		mList := MachineList{}
		machines, err := mList.List(client)
		if err != nil {
			return err
		}
		for _, t := range templates {
			tParams := make(map[string]string)
			for _, data := range t.Data {
				tParams[data.Name] = data.Value
			}

		loop:
			for _, m := range machines {
				if len(tParams) > len(m.CreationParams) {
					continue
				}

				for k, v := range tParams {
					if m.CreationParams[k] != v {
						continue loop
					}
				}
				templateIndex[t.Name]++
			}
		}
	}
	table := tablecli.NewTable()
	headers := []string{"Name", "IaaS", "Params"}
	if c.countMachines {
		headers = append(headers, "# Machines")
	}
	table.Headers = tablecli.Row(headers)
	table.LineSeparator = true
	for _, template := range templates {
		var params []string
		for _, data := range template.Data {
			params = append(params, fmt.Sprintf("%s=%s", data.Name, data.Value))
		}
		sort.Strings(params)
		row := []string{template.Name, template.IaaSName, strings.Join(params, "\n")}
		if c.countMachines {
			row = append(row, fmt.Sprintf("%d", templateIndex[template.Name]))
		}
		table.AddRow(tablecli.Row(row))
	}
	table.Sort()
	context.Stdout.Write(table.Bytes())
	return nil
}

func (c *TemplateList) filterTemplates(templates []iaas.Template) []iaas.Template {
	filteredTemplates := make([]iaas.Template, 0)
	for _, t := range templates {
		if c.templateMetadataMatchesFilters(t) {
			filteredTemplates = append(filteredTemplates, t)
		}
	}
	return filteredTemplates
}

func (c *TemplateList) templateMetadataMatchesFilters(template iaas.Template) bool {
	for key, value := range c.filter {
		hasKey := false
		for _, templateData := range template.Data {
			if key == templateData.Name {
				hasKey = true
				if value != templateData.Value {
					return false
				}
			}
		}
		if !hasKey {
			return false
		}
	}
	return true
}

type TemplateAdd struct{}

func (c *TemplateAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "machine-template-add",
		Usage: "machine-template-add <name> <iaas> <param>=<value>...",
		Desc: `Creates a new machine template.

Templates can be used with the [[node-add]] command running it with the
[[template=<template name>]] parameter. Templates can contain a list of
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

type TemplateRemove struct {
	cmd.ConfirmationCommand
}

func (c *TemplateRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "machine-template-remove",
		Usage:   "machine-template-remove <name> [-y/--assume-yes]",
		Desc:    "Removes an existing machine template.",
		MinArgs: 1,
	}
}

func (c *TemplateRemove) Run(context *cmd.Context, client *cmd.Client) error {
	templateName := context.Args[0]
	if !c.Confirm(context, fmt.Sprintf("Are you sure you want to remove template %q?", templateName)) {
		return nil
	}
	url, err := cmd.GetURL("/iaas/templates/" + templateName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
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

type TemplateUpdate struct {
	iaasName string
	fs       *gnuflag.FlagSet
}

func (c *TemplateUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "machine-template-update",
		Usage:   "machine-template-update <name> <param>=<value>... [-i/--iaas <iaas_name>]",
		Desc:    "Update an existing machine template.",
		MinArgs: 2,
	}
}

func (c *TemplateUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		iaasName := "The iaas name to be used"
		c.fs.StringVar(&c.iaasName, "iaas", "", iaasName)
		c.fs.StringVar(&c.iaasName, "i", "", iaasName)
	}
	return c.fs
}

func (c *TemplateUpdate) Run(context *cmd.Context, client *cmd.Client) error {
	template := iaas.Template{Name: context.Args[0], IaaSName: c.iaasName}
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

type TemplateCopy struct{}

func (c *TemplateCopy) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "machine-template-copy",
		Usage: "machine-template-copy <new-name> <src-name> <param>=<value>...",
		Desc: `Copies an existing template.

Templates can be used with the [[node-add]] command running it with the
[[template=<template name>]] parameter. Templates can contain a list of
parameters that will be sent to the IaaS provider.`,
		MinArgs: 2,
	}
}

func (c *TemplateCopy) Run(context *cmd.Context, client *cmd.Client) error {
	newName, srcName := context.Args[0], context.Args[1]
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
	defer response.Body.Close()
	var templates []iaas.Template
	err = json.NewDecoder(response.Body).Decode(&templates)
	if err != nil {
		return err
	}
	var template iaas.Template
	for _, tpl := range templates {
		if tpl.Name == srcName {
			template = tpl
			break
		}
	}
	if template.Name == "" {
		return fmt.Errorf("Template not found with name %s", srcName)
	}
	varMap := map[string]string{}
	for _, entry := range template.Data {
		varMap[entry.Name] = entry.Value
	}
	template.Name = newName
	for _, param := range context.Args[2:] {
		if strings.Contains(param, "=") {
			keyValue := strings.SplitN(param, "=", 2)
			varMap[keyValue[0]] = keyValue[1]
		}
	}
	template.Data = make([]iaas.TemplateData, 0, len(varMap))
	for k, v := range varMap {
		template.Data = append(template.Data, iaas.TemplateData{
			Name:  k,
			Value: v,
		})
	}
	v, err := form.EncodeToValues(&template)
	if err != nil {
		return err
	}
	u, err := cmd.GetURL("/iaas/templates")
	if err != nil {
		return err
	}
	request, err = http.NewRequest("POST", u, bytes.NewBufferString(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		context.Stderr.Write([]byte("Failed to copy template.\n"))
		return err
	}
	context.Stdout.Write([]byte("Template successfully copied.\n"))
	return nil
}
