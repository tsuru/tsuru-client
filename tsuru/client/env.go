// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruAPIApp "github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
	apiTypes "github.com/tsuru/tsuru/types/api"
)

const EnvSetValidationMessage = `You must specify environment variables in the form "NAME=value".

Example:

  tsuru env-set NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'

For private variables like passwords you can use -p or --private.

Example:

  tsuru env-set NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes' -p`

type EnvGet struct {
	cmd.AppNameMixIn

	fs   *gnuflag.FlagSet
	json bool
}

func (c *EnvGet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVar(&c.json, "json", false, "Display JSON format")

	}
	return c.fs
}

func (c *EnvGet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "env-get",
		Usage:   "env get [-a/--app appname] [ENVIRONMENT_VARIABLE1] [ENVIRONMENT_VARIABLE2] ...",
		Desc:    `Retrieves environment variables for an application.`,
		MinArgs: 0,
	}
}

func (c *EnvGet) Run(context *cmd.Context, client *cmd.Client) error {
	b, err := requestEnvGetURL(c.AppNameMixIn, context.Args, client)
	if err != nil {
		return err
	}
	var variables []map[string]interface{}
	err = json.Unmarshal(b, &variables)
	if err != nil {
		return err
	}

	if c.json {
		return c.renderJSON(context, variables)
	}

	formatted := make([]string, 0, len(variables))
	for _, v := range variables {
		value := tsuruAPIApp.SuppressedEnv
		if v["public"].(bool) {
			value = v["value"].(string)
		}
		formatted = append(formatted, fmt.Sprintf("%s=%s", v["name"], value))
	}
	sort.Strings(formatted)
	fmt.Fprintln(context.Stdout, strings.Join(formatted, "\n"))
	return nil
}

func (c *EnvGet) renderJSON(context *cmd.Context, variables []map[string]interface{}) error {
	type envJSON struct {
		Name    string `json:"name"`
		Value   string `json:"value"`
		Private bool   `json:"private"`
	}

	data := make([]envJSON, 0, len(variables))

	for _, v := range variables {
		private := true
		value := tsuruAPIApp.SuppressedEnv
		if v["public"].(bool) {
			value = v["value"].(string)
			private = false
		}
		data = append(data, envJSON{
			Name:    v["name"].(string),
			Value:   value,
			Private: private,
		})
	}

	return formatter.JSON(context.Stdout, data)
}

type EnvSet struct {
	cmd.AppNameMixIn
	fs        *gnuflag.FlagSet
	private   bool
	noRestart bool
}

func (c *EnvSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "env-set",
		Usage:   "env set <NAME=value> [NAME=value] ... [-a/--app appname] [-p/--private] [--no-restart]",
		Desc:    `Sets environment variables for an application.`,
		MinArgs: 1,
	}
}

func (c *EnvSet) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	if len(context.Args) < 1 {
		return errors.New(EnvSetValidationMessage)
	}
	envs := make([]apiTypes.Env, len(context.Args))
	for i := range context.Args {
		parts := strings.SplitN(context.Args[i], "=", 2)
		if len(parts) != 2 {
			return errors.New(EnvSetValidationMessage)
		}
		envs[i] = apiTypes.Env{Name: parts[0], Value: parts[1]}

	}
	e := apiTypes.Envs{
		Envs:      envs,
		NoRestart: c.noRestart,
		Private:   c.private,
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/env", appName))
	if err != nil {
		return err
	}
	v, err := form.EncodeToValues(&e)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(context.Stdout, response)
}

func (c *EnvSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVar(&c.private, "private", false, "Private environment variables")
		c.fs.BoolVar(&c.private, "p", false, "Private environment variables")
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Sets environment varibles without restart the application")
	}
	return c.fs
}

type EnvUnset struct {
	cmd.AppNameMixIn
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (c *EnvUnset) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Unset environment variables without restart the application")
	}
	return c.fs
}

func (c *EnvUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "env-unset",
		Usage:   "env unset <ENVIRONMENT_VARIABLE1> [ENVIRONMENT_VARIABLE2] ... [ENVIRONMENT_VARIABLEN] [-a/--app appname] [--no-restart]",
		Desc:    `Unset environment variables for an application.`,
		MinArgs: 1,
	}
}

func (c *EnvUnset) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	v := url.Values{}
	for _, e := range context.Args {
		v.Add("env", e)
	}
	v.Set("noRestart", strconv.FormatBool(c.noRestart))
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/env?%s", appName, v.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(context.Stdout, response)
}

func requestEnvGetURL(g cmd.AppNameMixIn, args []string, client *cmd.Client) ([]byte, error) {
	appName, err := g.AppName()
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	for _, e := range args {
		v.Add("env", e)
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/env?%s", appName, v.Encode()))
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	r, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}
