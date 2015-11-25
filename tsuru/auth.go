// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"launchpad.net/gnuflag"
)

type userCreate struct{}

func (c *userCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "user-create",
		Usage:   "user-create <email>",
		Desc:    "Creates a user within tsuru remote server. It will ask for the password before issue the request.",
		MinArgs: 1,
	}
}

func (c *userCreate) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	url, err := cmd.GetURL("/users")
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
	b := bytes.NewBufferString(`{"email":"` + email + `", "password":"` + password + `"}`)
	request, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
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

type userRemove struct{}

func (c *userRemove) currentUserEmail(client *cmd.Client) (string, error) {
	url, err := cmd.GetURL("/users/info")
	if err != nil {
		return "", err
	}
	request, _ := http.NewRequest("GET", url, nil)
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

func (c *userRemove) Run(context *cmd.Context, client *cmd.Client) error {
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
	url, err := cmd.GetURL("/users")
	if err != nil {
		return err
	}
	var qs string
	if email != "" {
		qs = "?user=" + email
	}
	request, err := http.NewRequest("DELETE", url+qs, nil)
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

func (c *userRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "user-remove",
		Usage: "user-remove [email]",
		Desc: `Remove currently authenticated user from remote tsuru
server. Since there cannot exist any orphan teams, tsuru will refuse to remove
a user that is the last member of some team. If this is your case, make sure
you remove the team using ` + "`team-remove`" + ` before removing the user.`,
		MinArgs: 0,
		MaxArgs: 1,
	}
}

type teamCreate struct{}

func (c *teamCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "team-create",
		Usage: "team-create <teamname>",
		Desc: `Create a team for the user. tsuru requires a user to be a member of at least
one team in order to create an app or a service instance.

When you create a team, you're automatically member of this team.`,
		MinArgs: 1,
	}
}

func (c *teamCreate) Run(context *cmd.Context, client *cmd.Client) error {
	team := context.Args[0]
	b := bytes.NewBufferString(fmt.Sprintf(`{"name":"%s"}`, team))
	url, err := cmd.GetURL("/teams")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `Team "%s" successfully created!`+"\n", team)
	return nil
}

type teamRemove struct {
	cmd.ConfirmationCommand
}

func (c *teamRemove) Run(context *cmd.Context, client *cmd.Client) error {
	team := context.Args[0]
	question := fmt.Sprintf("Are you sure you want to remove team %q?", team)
	if !c.Confirm(context, question) {
		return nil
	}
	url, err := cmd.GetURL(fmt.Sprintf("/teams/%s", team))
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
	fmt.Fprintf(context.Stdout, `Team "%s" successfully removed!`+"\n", team)
	return nil
}

func (c *teamRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "team-remove",
		Usage: "team-remove <team-name>",
		Desc: `Removes a team from tsuru server. You're able to remove teams that you're
member of. A team that has access to any app cannot be removed. Before
removing a team, make sure it does not have access to any app (see "app-grant"
and "app-revoke" commands for details).`,
		MinArgs: 1,
	}
}

type teamUserAdd struct{}

func (c *teamUserAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-user-add",
		Usage:   "team-user-add <teamname> <useremail>",
		Desc:    "Adds a user to a team. You need to be a member of the team to be able to add another user to it.",
		MinArgs: 2,
	}
}

func (c *teamUserAdd) Run(context *cmd.Context, client *cmd.Client) error {
	teamName, userName := context.Args[0], context.Args[1]
	url, err := cmd.GetURL(fmt.Sprintf("/teams/%s/%s", teamName, userName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `User "%s" was added to the "%s" team`+"\n", userName, teamName)
	return nil
}

type teamUserRemove struct{}

func (c *teamUserRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-user-remove",
		Usage:   "team-user-remove <teamname> <useremail>",
		Desc:    "Removes a user from a team. You need to be a member of the team to be able to remove a user from it.",
		MinArgs: 2,
	}
}

func (c *teamUserRemove) Run(context *cmd.Context, client *cmd.Client) error {
	teamName, userName := context.Args[0], context.Args[1]
	url, err := cmd.GetURL(fmt.Sprintf("/teams/%s/%s", teamName, userName))
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
	fmt.Fprintf(context.Stdout, `User "%s" was removed from the "%s" team`+"\n", userName, teamName)
	return nil
}

type teamUserList struct{}

func (teamUserList) Run(context *cmd.Context, client *cmd.Client) error {
	teamName := context.Args[0]
	url, err := cmd.GetURL("/teams/" + teamName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var t struct{ Users []string }
	err = json.NewDecoder(resp.Body).Decode(&t)
	if err != nil {
		return err
	}
	sort.Strings(t.Users)
	for _, user := range t.Users {
		fmt.Fprintf(context.Stdout, "- %s\n", user)
	}
	return nil
}

func (teamUserList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-user-list",
		Usage:   "team-user-list <teamname>",
		Desc:    "List members of a team.",
		MinArgs: 1,
	}
}

type teamList struct{}

func (c *teamList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-list",
		Usage:   "team-list",
		Desc:    "List all teams that you are member.",
		MinArgs: 0,
	}
}

type teamItem struct {
	Name        string
	Permissions []string
}

func (c *teamList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/teams")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var teams []teamItem
		err = json.Unmarshal(b, &teams)
		if err != nil {
			return err
		}
		table := cmd.NewTable()
		table.Headers = cmd.Row{"Team", "Permissions"}
		table.LineSeparator = true
		for _, team := range teams {
			table.AddRow(cmd.Row{team.Name, strings.Join(team.Permissions, "\n")})
		}
		fmt.Fprint(context.Stdout, table.String())
	}
	return nil
}

type changePassword struct{}

func (c *changePassword) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users/password")
	if err != nil {
		return err
	}
	var body bytes.Buffer
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
	if new != confirm {
		return errors.New("New password and password confirmation didn't match.")
	}
	jsonBody := map[string]string{
		"old": old,
		"new": new,
	}
	err = json.NewEncoder(&body).Encode(jsonBody)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, &body)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Password successfully updated!")
	return nil
}

func (c *changePassword) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "change-password",
		Usage: "change-password",
		Desc: `Changes the password of the logged in user. It will ask for the current
password, the new and the confirmation.`,
	}
}

type resetPassword struct {
	token string
}

func (c *resetPassword) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "reset-password",
		Usage: "reset-password <email> [--token|-t <token>]",
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

func (c *resetPassword) msg() string {
	if c.token == "" {
		return `You've successfully started the password reset process.

Please check your email.`
	}
	return `Your password has been reset and mailed to you.

Please check your email.`
}

func (c *resetPassword) Run(context *cmd.Context, client *cmd.Client) error {
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

func (c *resetPassword) Flags() *gnuflag.FlagSet {
	fs := gnuflag.NewFlagSet("reset-password", gnuflag.ExitOnError)
	fs.StringVar(&c.token, "token", "", "Token to reset the password")
	fs.StringVar(&c.token, "t", "", "Token to reset the password")
	return fs
}

type showAPIToken struct{}

func (c *showAPIToken) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "token-show",
		Usage: "token-show",
		Desc: `Shows API token for the user. This token allow authenticated API calls to
tsuru and will never expire. This is useful for integrating CI servers with
tsuru.

The key will be generated the first time this command is called. See [[tsuru token-regenerate]]
if you need to invalidate an existing token.`,
		MinArgs: 0,
	}
}

func (c *showAPIToken) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users/api-key")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
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

type regenerateAPIToken struct{}

func (c *regenerateAPIToken) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "token-regenerate",
		Usage:   "token-regenerate",
		Desc:    `Generates a new API token. This invalidates previously generated API tokens.`,
		MinArgs: 0,
	}
}

func (c *regenerateAPIToken) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users/api-key")
	if err != nil {
		return err
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
		b, err := ioutil.ReadAll(resp.Body)
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
