// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/cezarsa/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/api"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
)

const EnvSetValidationMessage = `You must specify environment variables in the form "NAME=value".

Example:

  tsuru env-set NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'

For private variables like passwords you can use -p or --private.

Example:

  tsuru env-set NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes' -p`

type EnvGet struct {
	cmd.GuessingCommand
}

func (c *EnvGet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "env-get",
		Usage:   "env-get [-a/--app appname] [ENVIRONMENT_VARIABLE1] [ENVIRONMENT_VARIABLE2] ...",
		Desc:    `Retrieves environment variables for an application.`,
		MinArgs: 0,
	}
}

func (c *EnvGet) Run(context *cmd.Context, client *cmd.Client) error {
	b, err := requestEnvGetURL(c.GuessingCommand, context.Args, client)
	if err != nil {
		return err
	}
	var variables []map[string]interface{}
	err = json.Unmarshal(b, &variables)
	if err != nil {
		return err
	}
	formatted := make([]string, 0, len(variables))
	for _, v := range variables {
		value := "*** (private variable)"
		if v["public"].(bool) {
			value = v["value"].(string)
		}
		formatted = append(formatted, fmt.Sprintf("%s=%s", v["name"], value))
	}
	sort.Strings(formatted)
	fmt.Fprintln(context.Stdout, strings.Join(formatted, "\n"))
	return nil
}

type EnvSet struct {
	cmd.GuessingCommand
	fs        *gnuflag.FlagSet
	private   bool
	noRestart bool
}

func (c *EnvSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "env-set",
		Usage:   "env-set <NAME=value> [NAME=value] ... [-a/--app appname] [-p/--private] [--no-restart]",
		Desc:    `Sets environment variables for an application.`,
		MinArgs: 1,
	}
}

type env struct {
	Name  string
	Value string
}

func (c *EnvSet) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	raw := strings.Join(context.Args, "\n")
	regex := regexp.MustCompile(`(\w+=[^\n]+)(\n|$)`)
	decls := regex.FindAllStringSubmatch(raw, -1)
	if len(decls) < 1 || len(decls) != len(context.Args) {
		return errors.New(EnvSetValidationMessage)
	}
	envs := make([]struct{ Name, Value string }, len(decls))
	for i := range decls {
		parts := strings.SplitN(decls[i][1], "=", 2)
		envs[i] = struct{ Name, Value string }{Name: parts[0], Value: parts[1]}
	}
	e := api.Envs{
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
	w := tsuruIo.NewStreamWriter(context.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, response.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}

func (c *EnvSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.BoolVar(&c.private, "private", false, "Private environment variables")
		c.fs.BoolVar(&c.private, "p", false, "Private environment variables")
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Sets environment varibles without restart the application")
	}
	return c.fs
}

type EnvUnset struct {
	cmd.GuessingCommand
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (c *EnvUnset) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Unset environment variables without restart the application")
	}
	return c.fs
}

func (c *EnvUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "env-unset",
		Usage:   "env-unset <ENVIRONMENT_VARIABLE1> [ENVIRONMENT_VARIABLE2] ... [ENVIRONMENT_VARIABLEN] [-a/--app appname] [--no-restart]",
		Desc:    `Unset environment variables for an application.`,
		MinArgs: 1,
	}
}

func (c *EnvUnset) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
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
	request, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	w := tsuruIo.NewStreamWriter(context.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, response.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}

func requestEnvGetURL(g cmd.GuessingCommand, args []string, client *cmd.Client) ([]byte, error) {
	appName, err := g.Guess()
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
