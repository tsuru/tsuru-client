// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/app"
	"github.com/tsuru/tsuru-client/tsuru/config"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/types/quota"
)

type UserQuotaView struct{}

func (*UserQuotaView) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "user-quota-view",
		MinArgs: 1,
		Usage:   "user-quota-view <user-email>",
		Desc:    "Displays the current usage and limit of the user.",
	}
}

func (*UserQuotaView) Run(context *cmd.Context) error {
	url, err := config.GetURL("/users/" + context.Args[0] + "/quota")
	if err != nil {
		return err
	}
	request, _ := http.NewRequest("GET", url, nil)
	resp, err := tsuruHTTP.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var quota quota.Quota
	err = json.NewDecoder(resp.Body).Decode(&quota)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "User: %s\n", context.Args[0])
	fmt.Fprintf(context.Stdout, "Apps usage: %d/%d\n", quota.InUse, quota.Limit)
	return nil
}

type UserChangeQuota struct{}

func (*UserChangeQuota) Info() *cmd.Info {
	desc := `Changes the limit of apps that a user can create.

The new limit must be an integer, it may also be "unlimited".`
	return &cmd.Info{
		Name:    "user-quota-change",
		MinArgs: 2,
		Usage:   "user-quota-change <user-email> <new-limit>",
		Desc:    desc,
	}
}

func (*UserChangeQuota) Run(context *cmd.Context) error {
	u, err := config.GetURL("/users/" + context.Args[0] + "/quota")
	if err != nil {
		return err
	}
	limit, err := parseLimit(context.Args[1])
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("limit", limit)
	request, _ := http.NewRequest("PUT", u, bytes.NewBufferString(v.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = tsuruHTTP.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Quota successfully updated.")
	return nil
}

type AppQuotaView struct {
	app.AppNameMixIn

	flagsApplied bool
	json         bool
}

func (*AppQuotaView) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-quota-view",
		MinArgs: 0,
		Usage:   "app-quota-view [-a/--app appname]",
		Desc:    "Displays the current usage and limit of the given app.",
	}
}

func (c *AppQuotaView) Flags() *gnuflag.FlagSet {
	fs := c.AppNameMixIn.Flags()
	if !c.flagsApplied {
		fs.BoolVar(&c.json, "json", false, "Show JSON")

		c.flagsApplied = true
	}
	return fs
}

func (c *AppQuotaView) Run(context *cmd.Context) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	url, err := config.GetURL(fmt.Sprintf("/apps/%s/quota", appName))
	if err != nil {
		return err
	}
	request, _ := http.NewRequest("GET", url, nil)
	resp, err := tsuruHTTP.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var quota quota.Quota
	err = json.NewDecoder(resp.Body).Decode(&quota)
	if err != nil {
		return err
	}

	if c.json {
		return formatter.JSON(context.Stdout, quota)
	}

	fmt.Fprintf(context.Stdout, "App: %s\n", appName)
	fmt.Fprintf(context.Stdout, "Units usage: %d/%d\n", quota.InUse, quota.Limit)
	return nil
}

type AppQuotaChange struct {
	app.AppNameMixIn
}

func (*AppQuotaChange) Info() *cmd.Info {
	desc := `Changes the limit of units that an app can have.

The new limit must be an integer, it may also be "unlimited".`
	return &cmd.Info{
		Name:    "app-quota-change",
		MinArgs: 1,
		Usage:   "app-quota-change [-a/--app appname] <new-limit>",
		Desc:    desc,
	}
}

func (c *AppQuotaChange) Run(context *cmd.Context) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/quota", appName))
	if err != nil {
		return err
	}
	limit, err := parseLimit(c.Flags().Arg(0))
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("limit", limit)
	request, _ := http.NewRequest("PUT", u, bytes.NewBufferString(v.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = tsuruHTTP.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Quota successfully updated.")
	return nil
}

type TeamQuotaView struct{}

func (*TeamQuotaView) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "team-quota-view",
		MinArgs: 1,
		Usage:   "team-quota-view <team-name>",
		Desc:    "Displays the current usage and limit of the team.",
	}
}

func (*TeamQuotaView) Run(context *cmd.Context) error {
	url, err := config.GetURLVersion("1.12", "/teams/"+context.Args[0]+"/quota")
	if err != nil {
		return err
	}
	request, _ := http.NewRequest("GET", url, nil)
	resp, err := tsuruHTTP.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var quota quota.Quota
	err = json.NewDecoder(resp.Body).Decode(&quota)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Team: %s\n", context.Args[0])
	fmt.Fprintf(context.Stdout, "Apps usage: %d/%d\n", quota.InUse, quota.Limit)
	return nil
}

type TeamChangeQuota struct{}

func (*TeamChangeQuota) Info() *cmd.Info {
	desc := `Changes the limit of apps that a team can create.

The new limit must be an integer, it may also be "unlimited".`
	return &cmd.Info{
		Name:    "team-quota-change",
		MinArgs: 2,
		Usage:   "team-quota-change <team-name> <new-limit>",
		Desc:    desc,
	}
}

func (*TeamChangeQuota) Run(context *cmd.Context) error {
	u, err := config.GetURLVersion("1.12", "/teams/"+context.Args[0]+"/quota")
	if err != nil {
		return err
	}
	limit, err := parseLimit(context.Args[1])
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("limit", limit)
	request, _ := http.NewRequest("PUT", u, bytes.NewBufferString(v.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = tsuruHTTP.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Quota successfully updated.")
	return nil
}

func parseLimit(value string) (string, error) {
	if value == "unlimited" {
		return "-1", nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return "", errors.New(`invalid limit. It must be either an integer or "unlimited"`)
	}
	return strconv.Itoa(n), nil
}
