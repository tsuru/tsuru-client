// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/tsuru/gnuflag"
	tsuruClient "github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/permission"
	permTypes "github.com/tsuru/tsuru/types/permission"
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
		Usage: "permission list [-t/--tree]",
		Desc:  `Lists all permissions available to use when defining roles.`,
	}
}

func (c *PermissionList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("permission-list", gnuflag.ExitOnError)
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
	t := tablecli.NewTable()
	t.Headers = tablecli.Row{"Name", "Contexts"}
	for _, perm := range permissions {
		t.AddRow(tablecli.Row{perm.Name, strings.Join(perm.Contexts, ", ")})
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
		Usage:   "role info <role-name>",
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
	tbl := tablecli.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = tablecli.Row{"Name", "Context", "Permissions", "Description"}
	tbl.AddRow(tablecli.Row{perm.Name, string(perm.ContextType), strings.Join(perm.SchemeNames, "\n"), perm.Description})
	fmt.Fprint(context.Stdout, tbl.String())
	return nil
}

type RoleAdd struct {
	description string
	fs          *gnuflag.FlagSet
}

func (c *RoleAdd) Info() *cmd.Info {
	info := &cmd.Info{
		Name:  "role-add",
		Usage: "role add <role-name> <context-type> [--description/-d description]",
		Desc: `Create a new role for the specified context type.
Valid context types are:

%s

The [[--description]] parameter sets a description for your role.
It is an optional parameter, and if its not set the role will only not have a
description associated.
`,
		MinArgs: 2,
	}
	allTypes := make([]string, len(permTypes.ContextTypes))
	for i := range permTypes.ContextTypes {
		allTypes[i] = "* " + string(permTypes.ContextTypes[i])
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

func (c *RoleAdd) RoleSet(roleName, contextType, description string) tsuru.RoleAddData {
	RoleAdd := tsuru.RoleAddData{
		Name:        roleName,
		Contexttype: contextType,
		Description: description,
	}
	return RoleAdd
}
func (c *RoleAdd) Run(ctx *cmd.Context, client *cmd.Client) error {
	roleName := ctx.Args[0]
	contextType := ctx.Args[1]
	description := c.description
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
	if err != nil {
		return err
	}
	roleAdd := c.RoleSet(roleName, contextType, description)
	_, err = apiClient.AuthApi.CreateRole(context.TODO(), roleAdd)

	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Role successfully created!\n")
	return nil
}

type RoleList struct{}

func (c *RoleList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-list",
		Usage:   "role list",
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
	table := tablecli.NewTable()
	table.Headers = tablecli.Row{"Role", "Context", "Permissions"}
	table.LineSeparator = true
	for _, r := range roles {
		table.AddRow(tablecli.Row{r.Name, string(r.ContextType), strings.Join(r.SchemeNames, "\n")})
	}
	fmt.Fprint(context.Stdout, table.String())
	return nil
}

type RolePermissionAdd struct{}

func (c *RolePermissionAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-permission-add",
		Usage:   "role permission add <role-name> <permission-name>...",
		Desc:    `Add a new permission to an existing role.`,
		MinArgs: 2,
	}
}
func (c *RolePermissionAdd) RolePermAdd(rolename string, permission []string) tsuru.PermissionData {
	rolePermAdd := tsuru.PermissionData{
		Name:       rolename,
		Permission: permission,
	}
	return rolePermAdd
}
func (c *RolePermissionAdd) Run(ctx *cmd.Context, client *cmd.Client) error {
	roleName := ctx.Args[0]
	params := url.Values{}
	for _, p := range ctx.Args[1:] {
		params.Add("permission", p)
	}
	rolePermAdd := c.RolePermAdd(roleName, params["permission"])
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
	if err != nil {
		return err
	}
	_, err = apiClient.AuthApi.PermissionAdd(context.TODO(), roleName, rolePermAdd)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Permission successfully added!\n")
	return nil
}

type RolePermissionRemove struct{}

func (c *RolePermissionRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-permission-remove",
		Usage:   "role permission remove <role-name> <permission-name>",
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
		Usage:   "role assign <role-name> <user-email>|<token-id>|group:<group-id> [<context-value>]",
		Desc:    `Assign an existing role to a user, token or group with some context value.`,
		MinArgs: 2,
	}
}
func (c *RoleAssign) RoleAssignData(roleName, contextValue, roleTarget, suffix, version string) tsuru.RoleAssignData {
	roleAdd := tsuru.RoleAssignData{
		Name:         roleName,
		Contextvalue: contextValue,
		Roletarget:   roleTarget,
		Sufix:        suffix,
		Version:      version,
	}
	return roleAdd
}
func (c *RoleAssign) Run(ctx *cmd.Context, client *cmd.Client) error {
	roleName := ctx.Args[0]
	roleTarget := ctx.Args[1]
	var contextValue string
	if len(ctx.Args) > 2 {
		contextValue = ctx.Args[2]
	}
	params := url.Values{}
	var suffix, version string
	if strings.HasPrefix(roleTarget, "group:") {
		suffix = "group"
		version = "1.9"
		params.Set("group_name", strings.TrimPrefix(roleTarget, "group:"))
	} else if strings.Contains(roleTarget, "@") {
		suffix = "user"
		version = "1.0"
		params.Set("email", roleTarget)
	} else {
		suffix = "token"
		version = "1.6"
		params.Set("token_id", roleTarget)
	}
	params.Set("context", contextValue)
	roleAssginAdd := c.RoleAssignData(roleName, contextValue, roleTarget, suffix, version)
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
	if err != nil {
		return err
	}
	_, err = apiClient.AuthApi.RoleAssign(context.TODO(), roleName, roleAssginAdd)

	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Role successfully assigned!\n")
	return nil
}

type RoleDissociate struct{}

func (c *RoleDissociate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-dissociate",
		Usage:   "role dissociate <role-name> <user-email>|<token-id> [<context-value>]",
		Desc:    `Dissociate an existing role from a user or token for some context value.`,
		MinArgs: 2,
	}
}

func (c *RoleDissociate) Run(ctx *cmd.Context, client *cmd.Client) error {
	roleName := ctx.Args[0]
	emailOrToken := ctx.Args[1]
	var contextValue string
	if len(ctx.Args) > 2 {
		contextValue = ctx.Args[2]
	}
	params := url.Values{}
	var suffix, version string
	if strings.Contains(emailOrToken, "@") {
		suffix = "user/" + emailOrToken
		version = "1.0"
	} else {
		suffix = "token/" + emailOrToken
		version = "1.6"
	}
	params.Set("context", contextValue)
	resp, err := cmd.GetURLVersion(version, fmt.Sprintf("/roles/%s/%s?%s", roleName, suffix, params.Encode()))
	if err != nil {
		return err
	}
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
	if err != nil {
		return err
	}
	if strings.Contains(emailOrToken, "@") {
		_, err = apiClient.AuthApi.DissociateRole(context.TODO(), roleName, resp)
	} else {
		_, err = apiClient.AuthApi.DissociateRoleFromToken(context.TODO(), roleName, emailOrToken, contextValue)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Role successfully dissociated!\n")
	return nil
}

type RoleRemove struct {
	cmd.ConfirmationCommand
}

func (c *RoleRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-remove",
		Usage:   "role remove <role-name> [-y/--assume-yes]",
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
		for eventName, event := range permTypes.RoleEventMap {
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
		Usage: "role default add",
		Desc:  `Add a new default role on a specific event.`,
	}
	var usage []string
	for eventName := range permTypes.RoleEventMap {
		usage = append(usage, fmt.Sprintf("[--%s <role name>]...", eventName))
	}
	info.Usage = fmt.Sprintf("%s %s", info.Usage, strings.Join(usage, " "))
	return info
}
func (c *RoleDefaultAdd) RoleDefAdd(roleMap map[string][]string) tsuru.RoleDefaultData {
	userData := tsuru.RoleDefaultData{
		Rolesmap: roleMap,
	}
	return userData
}
func (c *RoleDefaultAdd) Run(ctx *cmd.Context, client *cmd.Client) error {
	params := url.Values{}
	for name, values := range c.roles {
		for _, val := range []string(*values) {
			params.Add(name, val)
		}
	}
	rolname := []string{}
	roleMap := make(map[string][]string)
	for k := range params {
		rolname = append(rolname, k)
		for _, n := range params[k] {
			roleMap[k] = append(roleMap[k], n)
		}
	}
	rolDef := c.RoleDefAdd(roleMap)
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
	if err != nil {
		return err
	}
	_, err = apiClient.AuthApi.DefaultRoleAdd(context.TODO(), rolDef)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Roles successfully added as default!\n")
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
		for eventName, event := range permTypes.RoleEventMap {
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
		Usage: "role default remove",
		Desc:  `Remove a default role from a specific event.`,
	}
	var usage []string
	for eventName := range permTypes.RoleEventMap {
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
		Usage: "role default list",
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
	tbl := tablecli.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = tablecli.Row{"Event", "Description", "Roles"}
	for _, event := range permTypes.RoleEventMap {
		roles := rolesByEvent[event.String()]
		roleNames := make([]string, len(roles))
		for i := range roles {
			roleNames[i] = roles[i].Name
		}
		tbl.AddRow(tablecli.Row{event.String(), event.Description, strings.Join(roleNames, "\n")})
	}
	tbl.Sort()
	fmt.Fprint(context.Stdout, tbl.String())
	return nil
}

type RoleUpdate struct {
	newName     string
	description string
	contextType string
	fs          *gnuflag.FlagSet
}

func (c *RoleUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "role-update",
		Usage:   "role update <role> [-d/--description <description>] [-c/--context <context type>] [-n/--name <role new name>]",
		Desc:    "Updates a role description",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *RoleUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		roleDescription := "Updates a role description"
		c.fs.StringVar(&c.description, "d", "", roleDescription)
		c.fs.StringVar(&c.description, "description", "", roleDescription)
		contextType := "Updates the context type of a role"
		c.fs.StringVar(&c.contextType, "c", "", contextType)
		c.fs.StringVar(&c.contextType, "context", "", contextType)
		newName := "Updates the name of a role"
		c.fs.StringVar(&c.newName, "n", "", newName)
		c.fs.StringVar(&c.newName, "name", "", newName)
	}
	return c.fs
}
func (c *RoleUpdate) RoleUpdate(name string) tsuru.RoleUpdateData {
	userData := tsuru.RoleUpdateData{
		Name:        name,
		ContextType: c.contextType,
		Description: c.description,
		NewName:     c.newName,
	}
	return userData
}
func (c *RoleUpdate) Run(ctx *cmd.Context, client *cmd.Client) error {
	if (c.newName == "") && (c.description == "") && (c.contextType == "") {
		return errors.New("Neither the description, context or new name were set. You must define at least one.")
	}
	params := url.Values{}
	params.Set("name", ctx.Args[0])
	params.Set("newName", c.newName)
	params.Set("description", c.description)
	params.Set("contextType", c.contextType)
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
	if err != nil {
		return err
	}
	roleUpdate := c.RoleUpdate(ctx.Args[0])
	_, err = apiClient.AuthApi.UpdateRole(context.TODO(), roleUpdate)
	if err != nil {
		ctx.Stderr.Write([]byte("Failed to update role\n"))
		return err
	}
	ctx.Stdout.Write([]byte("Role successfully updated\n"))
	return nil
}
