// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/permission"
	"launchpad.net/gnuflag"
)

type permissionList struct {
	fs   *gnuflag.FlagSet
	tree bool
}

type permissionData struct {
	Name     string
	Contexts []string
	children []*permissionData
}

func (c *permissionList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "permission-list",
		Usage: "permission-list [-t/--tree]",
		Desc:  `Lists all permissions available to use when defining roles.`,
	}
}

func (c *permissionList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("plan-List", gnuflag.ExitOnError)
		tree := "Show permissions in tree format."
		c.fs.BoolVar(&c.tree, "tree", false, tree)
		c.fs.BoolVar(&c.tree, "t", false, tree)
	}
	return c.fs
}

func (c *permissionList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/permissions")
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
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var permissions []*permissionData
	err = json.Unmarshal(result, &permissions)
	if err != nil {
		return err
	}
	maxSize := 0
	maxCtx := 0
	for _, perm := range permissions[1:] {
		parts := strings.Split(perm.Name, ".")
		parentName := strings.Join(parts[:len(parts)-1], ".")
		size := (3 * len(parts)) + len(parts[len(parts)-1])
		if size > maxSize {
			maxSize = size
		}
		ctxSize := (len(perm.Contexts) - 1) * 2
		for _, c := range perm.Contexts {
			ctxSize += len(c)
		}
		if ctxSize > maxCtx {
			maxCtx = ctxSize
		}
		for _, parent := range permissions {
			if parent.Name == parentName {
				parent.children = append(parent.children, perm)
				break
			}
		}
	}
	permissions[0].Name = "*"
	if c.tree {
		lastMap := map[int]bool{}
		fmt.Fprintf(context.Stdout, "Permission%s | Context\n%s-+-%s\n", strings.Repeat(" ", maxSize-10), strings.Repeat("-", maxSize), strings.Repeat("-", maxCtx))
		renderTree(context.Stdout, permissions[0], 0, lastMap, maxSize)
	} else {
		renderList(context.Stdout, permissions)
	}
	return nil
}

func renderList(w io.Writer, permissions []*permissionData) {
	t := cmd.NewTable()
	t.Headers = cmd.Row{"Name", "Contexts"}
	for _, perm := range permissions {
		t.AddRow(cmd.Row{perm.Name, strings.Join(perm.Contexts, ", ")})
	}
	fmt.Fprint(w, t.String())
}

func renderTree(w io.Writer, item *permissionData, level int, lastMap map[int]bool, maxSize int) {
	parts := strings.Split(item.Name, ".")
	lastName := parts[len(parts)-1]
	padding := ""
	for i := 0; i < level; i++ {
		if i == level-1 {
			if lastMap[i+1] {
				padding += "└──"
			} else {
				padding += "├──"
			}
		} else {
			if lastMap[i+1] {
				padding += "   "
			} else {
				padding += "|  "
			}
		}
	}
	line := fmt.Sprintf("%s%s", padding, lastName)
	lineSize := len([]rune(line))
	if lineSize < maxSize {
		line += strings.Repeat(" ", maxSize-lineSize)
	}
	fmt.Fprintf(w, "%s | %s\n", line, strings.Join(item.Contexts, ", "))
	for i, child := range item.children {
		lastMap[level+1] = i == len(item.children)-1
		renderTree(w, child, level+1, lastMap, maxSize)
	}
}

type roleAdd struct{}

func (c *roleAdd) Info() *cmd.Info {
	info := &cmd.Info{
		Name:    "role-add",
		Usage:   "role-add <role-name> <context-type>",
		Desc:    `Create a new role for the specified context type. Valid context types are:`,
		MinArgs: 2,
	}
	allTypes := make([]string, len(permission.ContextTypes))
	for i := range permission.ContextTypes {
		allTypes[i] = "* " + string(permission.ContextTypes[i])
	}
	info.Desc = fmt.Sprintf("%s\n\n%s", info.Desc, strings.Join(allTypes, "\n"))
	return info
}

func (c *roleAdd) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	contextType := context.Args[1]
	params := url.Values{}
	params.Set("name", roleName)
	params.Set("context", contextType)
	addr, err := cmd.GetURL("/roles")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", addr, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Role successfully created!\n")
	return nil
}

type roleList struct{}

func (c *roleList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-list",
		Usage:   "role-list",
		Desc:    `List all existing roles.`,
		MinArgs: 0,
	}
}

func (c *roleList) Run(context *cmd.Context, client *cmd.Client) error {
	addr, err := cmd.GetURL("/roles")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", addr, nil)
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
	var roles []permission.Role
	err = json.Unmarshal(result, &roles)
	if err != nil {
		return err
	}
	table := cmd.NewTable()
	table.Headers = cmd.Row{"Role", "Context", "Permissions"}
	table.LineSeparator = true
	for _, r := range roles {
		table.AddRow(cmd.Row{r.Name, string(r.ContextType), strings.Join(r.SchemeNames, "\n")})
	}
	fmt.Fprint(context.Stdout, table.String())
	return nil
}

type rolePermissionAdd struct{}

func (c *rolePermissionAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-permission-add",
		Usage:   "role-permission-add <role-name> <permission-name>...",
		Desc:    `Add a new permission to an existing role.`,
		MinArgs: 2,
	}
}

func (c *rolePermissionAdd) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	params := url.Values{}
	for _, p := range context.Args[1:] {
		params.Add("permission", p)
	}
	addr, err := cmd.GetURL(fmt.Sprintf("/roles/%s/permissions", roleName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", addr, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Permission successfully added!\n")
	return nil
}

type rolePermissionRemove struct{}

func (c *rolePermissionRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-permission-remove",
		Usage:   "role-permission-remove <role-name> <permission-name>",
		Desc:    `Remove a permission from an existing role.`,
		MinArgs: 2,
	}
}

func (c *rolePermissionRemove) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	permissionName := context.Args[1]
	addr, err := cmd.GetURL(fmt.Sprintf("/roles/%s/permissions/%s", roleName, permissionName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", addr, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Permission successfully removed!\n")
	return nil
}

type roleAssign struct{}

func (c *roleAssign) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-assign",
		Usage:   "role-assign <role-name> <user-email> [<context-value>]",
		Desc:    `Assign an existing role to a user with some context value.`,
		MinArgs: 2,
	}
}

func (c *roleAssign) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	userEmail := context.Args[1]
	var contextValue string
	if len(context.Args) > 2 {
		contextValue = context.Args[2]
	}
	params := url.Values{}
	params.Set("email", userEmail)
	params.Set("context", contextValue)
	addr, err := cmd.GetURL(fmt.Sprintf("/roles/%s/user", roleName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", addr, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Role successfully assigned!\n")
	return nil
}

type roleDissociate struct{}

func (c *roleDissociate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-dissociate",
		Usage:   "role-dissociate <role-name> <user-email> [<context-value>]",
		Desc:    `Dissociate an existing role from a user for some context value.`,
		MinArgs: 2,
	}
}

func (c *roleDissociate) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	userEmail := context.Args[1]
	var contextValue string
	if len(context.Args) > 2 {
		contextValue = context.Args[2]
	}
	params := url.Values{}
	params.Set("context", contextValue)
	addr, err := cmd.GetURL(fmt.Sprintf("/roles/%s/user/%s?%s", roleName, userEmail, params.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", addr, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Role successfully dissociated!\n")
	return nil
}

type roleRemove struct{}

func (c *roleRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-remove",
		Usage:   "role-remove <role-name>",
		Desc:    `Remove an existing role.`,
		MinArgs: 1,
	}
}

func (c *roleRemove) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	addr, err := cmd.GetURL(fmt.Sprintf("/roles/%s", roleName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", addr, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Role successfully removed!\n")
	return nil
}
