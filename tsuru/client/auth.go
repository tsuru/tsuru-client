// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"text/template"

	"github.com/antihax/optional"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
)

type UserCreate struct{}

func (c *UserCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "user-create",
		Usage:   "user create <email>",
		Desc:    "Creates a user within tsuru remote server. It will ask for the password before issue the request.",
		MinArgs: 1,
	}
}

func (c *UserCreate) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	u, err := cmd.GetURL("/users")
	if err != nil {
		return err
	}
	email := context.Args[0]
	fmt.Fprint(context.Stdout, "Password: ")
	password, err := cmd.PasswordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "\nConfirm: ")
	confirm, err := cmd.PasswordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout)
	if password != confirm {
		return errors.New("Passwords didn't match.")
	}
	v := url.Values{}
	v.Set("email", email)
	v.Set("password", password)
	b := strings.NewReader(v.Encode())
	request, err := http.NewRequest("POST", u, b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(request)
	if resp != nil {
		if resp.StatusCode == http.StatusNotFound ||
			resp.StatusCode == http.StatusMethodNotAllowed {
			return errors.New("User creation is disabled.")
		}
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `User "%s" successfully created!`+"\n", email)
	return nil
}

type UserRemove struct{}

func (c *UserRemove) currentUserEmail(client *cmd.Client) (string, error) {
	u, err := cmd.GetURL("/users/info")
	if err != nil {
		return "", err
	}
	request, _ := http.NewRequest("GET", u, nil)
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var r struct{ Email string }
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return "", err
	}
	return r.Email, nil
}

func (c *UserRemove) Run(context *cmd.Context, client *cmd.Client) error {
	var (
		answer string
		email  string
		err    error
	)
	if len(context.Args) > 0 {
		email = context.Args[0]
	} else {
		email, err = c.currentUserEmail(client)
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(context.Stdout, `Are you sure you want to remove the user %q from tsuru? (y/n) `, email)
	fmt.Fscanf(context.Stdin, "%s", &answer)
	if answer != "y" {
		fmt.Fprintln(context.Stdout, "Abort.")
		return nil
	}
	u, err := cmd.GetURL("/users")
	if err != nil {
		return err
	}
	var qs string
	if email != "" {
		qs = "?user=" + url.QueryEscape(email)
	}
	request, err := http.NewRequest(http.MethodDelete, u+qs, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "User %q successfully removed.\n", email)
	return nil
}

func (c *UserRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "user-remove",
		Usage: "user remove [email]",
		Desc: `Remove currently authenticated user from remote tsuru
server. Since there cannot exist any orphan teams, tsuru will refuse to remove
a user that is the last member of some team. If this is your case, make sure
you remove the team using ` + "`team-remove`" + ` before removing the user.`,
		MinArgs: 0,
		MaxArgs: 1,
	}
}

type TeamCreate struct {
	tags cmd.StringSliceFlag
	fs   *gnuflag.FlagSet
}

func (c *TeamCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "team-create",
		Usage: "team create <teamname> [--tag/-t tag]...",
		Desc: `Create a team for the user. tsuru requires a user to be a member of at least
one team in order to create an app or a service instance.

When you create a team, you're automatically member of this team.
`,
		MinArgs: 1,
	}
}

func (c *TeamCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.Var(&c.tags, "tag", "Sets tags to the team.")
		c.fs.Var(&c.tags, "t", "Sets tags to the team.")
	}
	return c.fs
}

func (c *TeamCreate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	team := ctx.Args[0]
	_, err = apiClient.TeamApi.TeamCreate(context.TODO(), tsuru.TeamCreateArgs{
		Name: team,
		Tags: c.tags,
	})
	if err != nil {
		return parseErrBody(err)
	}
	fmt.Fprintf(ctx.Stdout, `Team "%s" successfully created!`+"\n", team)
	return nil
}

type TeamUpdate struct {
	newName string
	tags    cmd.StringSliceFlag
	fs      *gnuflag.FlagSet
}

func (t *TeamUpdate) Flags() *gnuflag.FlagSet {
	if t.fs == nil {
		t.fs = gnuflag.NewFlagSet("team-update", gnuflag.ExitOnError)
		desc := "New team name."
		t.fs.StringVar(&t.newName, "name", "", desc)
		t.fs.StringVar(&t.newName, "n", "", desc)
		t.fs.Var(&t.tags, "tag", "New team tags.")
		t.fs.Var(&t.tags, "t", "New team tags.")
	}
	return t.fs
}

func (t *TeamUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "team-update",
		Usage: "team update <team-name> -n <new-team-name> [--tag/-t tag]...",
		Desc: `Updates a team.
`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (t *TeamUpdate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	team := ctx.Args[0]
	_, err = apiClient.TeamApi.TeamUpdate(context.TODO(), team, tsuru.TeamUpdateArgs{
		Newname: t.newName,
		Tags:    t.tags,
	})
	if err != nil {
		return parseErrBody(err)
	}
	fmt.Fprintln(ctx.Stdout, "Team successfully updated!")
	return nil
}

type TeamRemove struct {
	cmd.ConfirmationCommand
}

func (c *TeamRemove) Run(ctx *cmd.Context, cli *cmd.Client) error {
	team := ctx.Args[0]
	question := fmt.Sprintf("Are you sure you want to remove team %q?", team)
	if !c.Confirm(ctx, question) {
		return nil
	}
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	_, err = apiClient.TeamApi.TeamDelete(context.TODO(), team)
	if err != nil {
		return parseErrBody(err)
	}
	fmt.Fprintf(ctx.Stdout, `Team "%s" successfully removed!`+"\n", team)
	return nil
}

func (c *TeamRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "team-remove",
		Usage: "team remove <team-name>",
		Desc: `Removes a team from tsuru server. You're able to remove teams that you're
member of. A team that has access to any app cannot be removed. Before
removing a team, make sure it does not have access to any app (see "app grant"
and "app revoke" commands for details).`,
		MinArgs: 1,
	}
}

type TeamList struct {
	fs         *gnuflag.FlagSet
	simplified bool
}

func (c *TeamList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-list",
		Usage:   "team list",
		Desc:    "List all teams that you are member.",
		MinArgs: 0,
	}
}

func (c *TeamList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("team-list", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.simplified, "q", false, "Display only team's name")
	}
	return c.fs
}

func (c *TeamList) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	teams, resp, err := apiClient.TeamApi.TeamsList(context.TODO())
	if resp != nil && resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err != nil {
		return parseErrBody(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	if c.simplified {
		for _, team := range teams {
			fmt.Fprintln(ctx.Stdout, team.Name)
		}
		return nil
	}

	table := tablecli.NewTable()
	table.Headers = tablecli.Row{"Team", "Permissions", "Tags"}
	table.LineSeparator = true
	for _, team := range teams {
		table.AddRow(tablecli.Row{team.Name, strings.Join(team.Permissions, "\n"), strings.Join(team.Tags, "\n")})
	}
	fmt.Fprint(ctx.Stdout, table.String())
	return nil
}

func formatRoleInstances(userRoles []tsuru.RoleUser) []string {
	roles := make([]string, len(userRoles))
	for i, r := range userRoles {
		if r.Contextvalue != "" {
			r.Contextvalue = " " + r.Contextvalue
		}
		if r.Group != "" {
			r.Group = fmt.Sprintf(" (group %s)", r.Group)
		}
		roles[i] = fmt.Sprintf("%s(%s%s)%s", r.Name, r.Contexttype, r.Contextvalue, r.Group)
	}
	sort.Strings(roles)
	return roles
}

func formatPermissionInstances(userPerms []tsuru.PermissionUser) []string {
	permissions := make([]string, len(userPerms))
	for i, r := range userPerms {
		if r.Name == "" {
			r.Name = "*"
		}
		if r.Contextvalue != "" {
			r.Contextvalue = " " + r.Contextvalue
		}
		if r.Group != "" {
			r.Group = fmt.Sprintf(" (group %s)", r.Group)
		}
		permissions[i] = fmt.Sprintf("%s(%s%s)%s", r.Name, r.Contexttype, r.Contextvalue, r.Group)
	}
	sort.Strings(permissions)
	return permissions
}

type apiUser struct {
	Email       string
	Roles       []tsuru.RoleUser
	Permissions []tsuru.PermissionUser
}

type ContentTeam struct {
	Name  string    `json:"name"`
	Users []apiUser `json:"users"`
	Pools []Pool    `json:"pools"`
	Apps  []app     `json:"apps"`
	Tags  []string  `json:"tags"`
}

type TeamInfo struct{}

func (c *TeamInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-info",
		Usage:   "team info <team>",
		Desc:    `Shows information about a specific team.`,
		MinArgs: 1,
	}
}

func (c *TeamInfo) Run(ctx *cmd.Context, cli *cmd.Client) error {
	team := ctx.Args[0]
	u, err := cmd.GetURLVersion("1.4", fmt.Sprintf("/teams/%v", team))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	resp, err := cli.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var contentTeam ContentTeam
	err = json.Unmarshal(b, &contentTeam)
	if err != nil {
		return err
	}
	format := `Team: {{.Name}}
Tags: {{.Tags}}
`
	tmpl := template.Must(template.New("app").Parse(format))
	var tplBuffer bytes.Buffer
	var buf bytes.Buffer
	tmpl.Execute(&tplBuffer, contentTeam)
	usersTable := tablecli.NewTable()
	usersTable.Headers = tablecli.Row{"User", "Roles"}
	usersTable.LineSeparator = true
	for _, user := range contentTeam.Users {
		usersTable.AddRow(tablecli.Row{user.Email, strings.Join(formatRoleInstances(user.Roles), "\n")})
	}
	if usersTable.Rows() > 0 {
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("Users: %d\n", usersTable.Rows()))
		buf.WriteString(usersTable.String())
	}
	poolsTable := tablecli.NewTable()
	poolsTable.Headers = tablecli.Row{"Pool", "Kind", "Provisioner", "Routers"}
	poolsTable.LineSeparator = true
	for _, pool := range contentTeam.Pools {
		poolsTable.AddRow(tablecli.Row{pool.Name, pool.Kind(), pool.GetProvisioner(), strings.Join(pool.Allowed["router"], "\n")})
	}
	if poolsTable.Rows() > 0 {
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("Pools: %d\n", poolsTable.Rows()))
		buf.WriteString(poolsTable.String())
	}
	appsTable := tablecli.NewTable()
	appsTable.Headers = tablecli.Row{"Application", "Units", "Address"}
	appsTable.LineSeparator = true
	for _, app := range contentTeam.Apps {
		var summary string
		if app.Error == "" {
			unitsStatus := make(map[string]int)
			for _, unit := range app.Units {
				if unit.ID != "" {
					unitsStatus[unit.Status]++
				}
			}
			statusText := make([]string, len(unitsStatus))
			i := 0
			us := newUnitSorter(unitsStatus)
			sort.Sort(us)
			for _, status := range us.Statuses {
				statusText[i] = fmt.Sprintf("%d %s", unitsStatus[status], status)
				i++
			}
			summary = strings.Join(statusText, "\n")
		} else {
			summary = "error fetching units"
			if cli.Verbosity > 0 {
				summary += fmt.Sprintf(": %s", app.Error)
			}
		}
		addrs := strings.Replace(app.Addr(), ", ", "\n", -1)
		appsTable.AddRow(tablecli.Row([]string{app.Name, summary, addrs}))
	}
	if appsTable.Rows() > 0 {
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("Applications: %d\n", appsTable.Rows()))
		buf.WriteString(appsTable.String())
	}

	fmt.Fprint(ctx.Stdout, tplBuffer.String()+buf.String())
	return nil
}

type ChangePassword struct{}

func (c *ChangePassword) Run(context *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURL("/users/password")
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "Current password: ")
	old, err := cmd.PasswordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "\nNew password: ")
	new, err := cmd.PasswordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "\nConfirm: ")
	confirm, err := cmd.PasswordFromReader(context.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout)
	v := url.Values{}
	v.Set("old", old)
	v.Set("new", new)
	v.Set("confirm", confirm)
	request, err := http.NewRequest("PUT", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Password successfully updated!")
	return nil
}

func (c *ChangePassword) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "change-password",
		Usage: "change password",
		Desc: `Changes the password of the logged in user. It will ask for the current
password, the new and the confirmation.`,
	}
}

type ResetPassword struct {
	token string
}

func (c *ResetPassword) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "reset-password",
		Usage: "reset password <email> [--token|-t <token>]",
		Desc: `Resets the user password.

This process is composed of two steps:

1. Generate a new token
2. Reset the password using the token

In order to generate the token, users should run this command without the
--token flag. The token will be mailed to the user.

With the token in hand, the user can finally reset the password using the
--token flag. The new password will also be mailed to the user.`,
		MinArgs: 1,
	}
}

func (c *ResetPassword) msg() string {
	if c.token == "" {
		return `You've successfully started the password reset process.

Please check your email.`
	}
	return `Your password has been reset and mailed to you.

Please check your email.`
}

func (c *ResetPassword) Run(context *cmd.Context, client *cmd.Client) error {
	url := fmt.Sprintf("/users/%s/password", context.Args[0])
	if c.token != "" {
		url += "?token=" + c.token
	}
	url, err := cmd.GetURL(url)
	if err != nil {
		return err
	}
	request, _ := http.NewRequest("POST", url, nil)
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, c.msg())
	return nil
}

func (c *ResetPassword) Flags() *gnuflag.FlagSet {
	fs := gnuflag.NewFlagSet("reset-password", gnuflag.ExitOnError)
	fs.StringVar(&c.token, "token", "", "Token to reset the password")
	fs.StringVar(&c.token, "t", "", "Token to reset the password")
	return fs
}

type ShowAPIToken struct {
	user string
	fs   *gnuflag.FlagSet
}

func (c *ShowAPIToken) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "token-show",
		Usage: "token show [--user/-u useremail]",
		Desc: `Shows API token for the user. This command is deprecated, [[tsuru
token-create]] should be used instead. This token allow authenticated API
calls to tsuru impersonating the user who created it and will never expire.

The token key will be generated the first time this command is called. See
[[tsuru token regenerate]] if you need to invalidate an existing token.`,
		MinArgs: 0,
	}
}

func (c *ShowAPIToken) Run(context *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURL("/users/api-key")
	if err != nil {
		return err
	}
	if c.user != "" {
		u += fmt.Sprintf("?user=%s", url.QueryEscape(c.user))
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var APIKey string
		err = json.Unmarshal(b, &APIKey)
		if err != nil {
			return err
		}
		fmt.Fprintf(context.Stdout, "API key: %s\n", APIKey)
	}
	return nil
}

func (c *ShowAPIToken) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.StringVar(&c.user, "user", "", "Shows API token for the given user email")
		c.fs.StringVar(&c.user, "u", "", "Shows API token for the given user email")
	}
	return c.fs
}

type RegenerateAPIToken struct {
	user string
	fs   *gnuflag.FlagSet
}

func (c *RegenerateAPIToken) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "token-regenerate",
		Usage: "token regenerate [--user/-u useremail]",
		Desc: `Generates a new API token associated to the user. This invalidates and
replaces the token previously shown in [[tsuru token-show]]. This command is
deprecated, [[tsuru token-create]] and [[tsuru token-update]] should be used
instead.`,
		MinArgs: 0,
	}
}

func (c *RegenerateAPIToken) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users/api-key")
	if err != nil {
		return err
	}
	if c.user != "" {
		url += fmt.Sprintf("?user=%s", c.user)
	}
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var APIKey string
		err = json.Unmarshal(b, &APIKey)
		if err != nil {
			return err
		}
		fmt.Fprintf(context.Stdout, "Your new API key is: %s\n", APIKey)
	}
	return nil
}

func (c *RegenerateAPIToken) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.StringVar(&c.user, "user", "", "Generates a new API token for the given user email")
		c.fs.StringVar(&c.user, "u", "", "Generates a new API token for the given user email")
	}
	return c.fs
}

type ListUsers struct {
	userEmail string
	role      string
	context   string
	fs        *gnuflag.FlagSet
}

func (c *ListUsers) Run(ctx *cmd.Context, cli *cmd.Client) error {
	if c.userEmail != "" && c.role != "" {
		return errors.New("You cannot filter by user email and role at same time. Enter <tsuru user-list --help> for more information.")
	}
	if c.context != "" && c.role == "" {
		return errors.New("You should provide a role to filter by context value.")
	}

	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	users, _, err := apiClient.UserApi.UsersList(context.TODO(), c.userEmail, &tsuru.UsersListOpts{
		Context: optional.NewString(c.context),
		Role:    optional.NewString(c.role),
	})
	if err != nil {
		return err
	}

	table := tablecli.NewTable()
	table.Headers = tablecli.Row([]string{"User", "Roles"})
	for _, u := range users {
		table.AddRow(tablecli.Row([]string{
			u.Email,
			strings.Join(formatRoleInstances(u.Roles), "\n"),
		}))
	}
	table.LineSeparator = true
	table.Sort()
	ctx.Stdout.Write(table.Bytes())
	return nil
}

func (c *ListUsers) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "user-list",
		MinArgs: 0,
		Usage:   "user list [--user/-u useremail] [--role/-r role [-c/--context-value value]]",
		Desc:    "List all users in tsuru. It may also filter users by user email or role name with context value.",
	}
}

func (c *ListUsers) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.StringVar(&c.userEmail, "user", "", "Filter user by user email")
		c.fs.StringVar(&c.userEmail, "u", "", "Filter user by user email")
		c.fs.StringVar(&c.role, "r", "", "Filter user by role")
		c.fs.StringVar(&c.role, "role", "", "Filter user by role")
		c.fs.StringVar(&c.context, "c", "", "Filter user by role context value")
		c.fs.StringVar(&c.context, "context-value", "", "Filter user by role context value")
	}
	return c.fs
}

type UserInfo struct{}

func (UserInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "user-info",
		Usage: "user info",
		Desc:  "Displays information about the current user.",
	}
}

func (UserInfo) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	u, _, err := apiClient.UserApi.UserGet(context.TODO())
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Email: %s\n", u.Email)
	roles := formatRoleInstances(u.Roles)
	if len(roles) > 0 {
		fmt.Fprintf(ctx.Stdout, "Roles:\n\t%s\n", strings.Join(roles, "\n\t"))
	}
	perms := formatPermissionInstances(u.Permissions)
	if len(perms) > 0 {
		fmt.Fprintf(ctx.Stdout, "Permissions:\n\t%s\n", strings.Join(perms, "\n\t"))
	}
	if len(u.Groups) > 0 {
		fmt.Fprintf(ctx.Stdout, "Groups:\n\t%s\n", strings.Join(u.Groups, "\n\t"))
	}
	return nil
}

func parseErrBody(err error) error {
	type httpErr interface {
		Body() []byte
	}
	if hErr, ok := err.(httpErr); ok {
		return fmt.Errorf("%s", hErr.Body())
	}
	return err
}
