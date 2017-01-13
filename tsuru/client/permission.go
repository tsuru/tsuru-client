// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/permission"
)

type PermissionList struct {
	fs   *gnuflag.FlagSet
	tree bool
}

type permissionData struct {
	Name     string
	Contexts []string
	children []*permissionData
}

func (c *PermissionList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "permission-list",
		Usage: "permission-list [-t/--tree]",
		Desc:  `Lists all permissions available to use when defining roles.`,
	}
}

func (c *PermissionList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("plan-List", gnuflag.ExitOnError)
		tree := "Show permissions in tree format."
		c.fs.BoolVar(&c.tree, "tree", false, tree)
		c.fs.BoolVar(&c.tree, "t", false, tree)
	}
	return c.fs
}

func (c *PermissionList) Run(context *cmd.Context, client *cmd.Client) error {
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
				padding += "│  "
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

type RoleInfo struct{}

func (c *RoleInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-info",
		Usage:   "role-info <role-name>",
		Desc:    "Get information about specific role.",
		MinArgs: 1,
	}
}

func (c *RoleInfo) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	addr, err := cmd.GetURL(fmt.Sprintf("/roles/%s", roleName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", addr, nil)
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
	var perm permission.Role
	err = json.Unmarshal(result, &perm)
	if err != nil {
		return err
	}
	tbl := cmd.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = cmd.Row{"Name", "Context", "Permissions", "Description"}
	tbl.AddRow(cmd.Row{perm.Name, string(perm.ContextType), strings.Join(perm.SchemeNames, "\n"), perm.Description})
	fmt.Fprintf(context.Stdout, tbl.String())
	return nil
}

type RoleAdd struct {
	description string
	fs          *gnuflag.FlagSet
}

func (c *RoleAdd) Info() *cmd.Info {
	info := &cmd.Info{
		Name:  "role-add",
		Usage: "role-add <role-name> <context-type> [--description/-d description]",
		Desc: `Create a new role for the specified context type.
Valid context types are:

%s

The [[--description]] parameter sets a description for your role.
It is an optional parameter, and if its not set the role will only not have a
description associated.
`,
		MinArgs: 2,
	}
	allTypes := make([]string, len(permission.ContextTypes))
	for i := range permission.ContextTypes {
		allTypes[i] = "* " + string(permission.ContextTypes[i])
	}
	info.Desc = fmt.Sprintf(info.Desc, strings.Join(allTypes, "\n"))
	return info
}

func (c *RoleAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		descriptionMessage := "Role description"
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.StringVar(&c.description, "description", "", descriptionMessage)
		c.fs.StringVar(&c.description, "d", "", descriptionMessage)
	}
	return c.fs
}

func (c *RoleAdd) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	contextType := context.Args[1]
	description := c.description
	params := url.Values{}
	params.Set("name", roleName)
	params.Set("context", contextType)
	params.Set("description", description)
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

type RoleList struct{}

func (c *RoleList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-list",
		Usage:   "role-list",
		Desc:    `List all existing roles.`,
		MinArgs: 0,
	}
}

func (c *RoleList) Run(context *cmd.Context, client *cmd.Client) error {
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

type RolePermissionAdd struct{}

func (c *RolePermissionAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-permission-add",
		Usage:   "role-permission-add <role-name> <permission-name>...",
		Desc:    `Add a new permission to an existing role.`,
		MinArgs: 2,
	}
}

func (c *RolePermissionAdd) Run(context *cmd.Context, client *cmd.Client) error {
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

type RolePermissionRemove struct{}

func (c *RolePermissionRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-permission-remove",
		Usage:   "role-permission-remove <role-name> <permission-name>",
		Desc:    `Remove a permission from an existing role.`,
		MinArgs: 2,
	}
}

func (c *RolePermissionRemove) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	permissionName := context.Args[1]
	addr, err := cmd.GetURL(fmt.Sprintf("/roles/%s/permissions/%s", roleName, permissionName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, addr, nil)
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

type RoleAssign struct{}

func (c *RoleAssign) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-assign",
		Usage:   "role-assign <role-name> <user-email> [<context-value>]",
		Desc:    `Assign an existing role to a user with some context value.`,
		MinArgs: 2,
	}
}

func (c *RoleAssign) Run(context *cmd.Context, client *cmd.Client) error {
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

type RoleDissociate struct{}

func (c *RoleDissociate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-dissociate",
		Usage:   "role-dissociate <role-name> <user-email> [<context-value>]",
		Desc:    `Dissociate an existing role from a user for some context value.`,
		MinArgs: 2,
	}
}

func (c *RoleDissociate) Run(context *cmd.Context, client *cmd.Client) error {
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
	request, err := http.NewRequest(http.MethodDelete, addr, nil)
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

type RoleRemove struct {
	cmd.ConfirmationCommand
}

func (c *RoleRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-remove",
		Usage:   "role-remove <role-name> [-y/--assume-yes]",
		Desc:    `Remove an existing role.`,
		MinArgs: 1,
	}
}

func (c *RoleRemove) Run(context *cmd.Context, client *cmd.Client) error {
	roleName := context.Args[0]
	if !c.Confirm(context, fmt.Sprintf(`Are you sure you want to remove role "%s"?`, roleName)) {
		return nil
	}
	addr, err := cmd.GetURL(fmt.Sprintf("/roles/%s", roleName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, addr, nil)
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

type RoleDefaultAdd struct {
	fs    *gnuflag.FlagSet
	roles map[string]*cmd.StringSliceFlag
}

func (c *RoleDefaultAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.roles = map[string]*cmd.StringSliceFlag{}
		for eventName, event := range permission.RoleEventMap {
			flag := &cmd.StringSliceFlag{}
			c.roles[eventName] = flag
			c.fs.Var(flag, eventName, event.Description)
		}
	}
	return c.fs
}

func (c *RoleDefaultAdd) Info() *cmd.Info {
	info := &cmd.Info{
		Name:  "role-default-add",
		Usage: "role-default-add",
		Desc:  `Add a new default role on a specific event.`,
	}
	var usage []string
	for eventName := range permission.RoleEventMap {
		usage = append(usage, fmt.Sprintf("[--%s <role name>]...", eventName))
	}
	info.Usage = fmt.Sprintf("%s %s", info.Usage, strings.Join(usage, " "))
	return info
}

func (c *RoleDefaultAdd) Run(context *cmd.Context, client *cmd.Client) error {
	params := url.Values{}
	for name, values := range c.roles {
		for _, val := range []string(*values) {
			params.Add(name, val)
		}
	}
	encodedParams := params.Encode()
	if encodedParams == "" {
		return fmt.Errorf("You must choose which event to add default roles.")
	}
	addr, err := cmd.GetURL("/role/default")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", addr, strings.NewReader(encodedParams))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Roles successfully added as default!\n")
	return nil
}

type RoleDefaultRemove struct {
	fs    *gnuflag.FlagSet
	roles map[string]*cmd.StringSliceFlag
}

func (c *RoleDefaultRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.roles = map[string]*cmd.StringSliceFlag{}
		for eventName, event := range permission.RoleEventMap {
			flag := &cmd.StringSliceFlag{}
			c.roles[eventName] = flag
			c.fs.Var(flag, eventName, event.Description)
		}
	}
	return c.fs
}

func (c *RoleDefaultRemove) Info() *cmd.Info {
	info := &cmd.Info{
		Name:  "role-default-remove",
		Usage: "role-default-remove",
		Desc:  `Remove a default role from a specific event.`,
	}
	var usage []string
	for eventName := range permission.RoleEventMap {
		usage = append(usage, fmt.Sprintf("[--%s <role name>]...", eventName))
	}
	info.Usage = fmt.Sprintf("%s %s", info.Usage, strings.Join(usage, " "))
	return info
}

func (c *RoleDefaultRemove) Run(context *cmd.Context, client *cmd.Client) error {
	params := url.Values{}
	for name, values := range c.roles {
		for _, val := range []string(*values) {
			params.Add(name, val)
		}
	}
	encodedParams := params.Encode()
	if encodedParams == "" {
		return fmt.Errorf("You must choose which event to remove default roles.")
	}
	addr, err := cmd.GetURL(fmt.Sprintf("/role/default?%s", encodedParams))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, addr, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Roles successfully removed as default!\n")
	return nil
}

type RoleDefaultList struct{}

func (c *RoleDefaultList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "role-default-list",
		Usage: "role-default-list",
		Desc:  `List all roles set as default on any event.`,
	}
}

func (c *RoleDefaultList) Run(context *cmd.Context, client *cmd.Client) error {
	addr, err := cmd.GetURL("/role/default")
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
	defer response.Body.Close()
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var roles []permission.Role
	err = json.Unmarshal(result, &roles)
	if err != nil {
		return err
	}
	rolesByEvent := map[string][]permission.Role{}
	for _, r := range roles {
		for _, evt := range r.Events {
			rolesByEvent[evt] = append(rolesByEvent[evt], r)
		}
	}
	tbl := cmd.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = cmd.Row{"Event", "Description", "Roles"}
	for _, event := range permission.RoleEventMap {
		roles := rolesByEvent[event.String()]
		roleNames := make([]string, len(roles))
		for i := range roles {
			roleNames[i] = roles[i].Name
		}
		tbl.AddRow(cmd.Row{event.String(), event.Description, strings.Join(roleNames, "\n")})
	}
	tbl.Sort()
	fmt.Fprint(context.Stdout, tbl.String())
	return nil
}
