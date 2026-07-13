// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	apiTypes "github.com/tsuru/tsuru/types/app"
)

const configSetValidationMessage = `You must specify both a source and a target filename.

Example:

	tsuru app-config-set -a APPNAME source target
`

type ConfigGet struct {
	cmd.AppNameMixIn
}

func (c *ConfigGet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-config-get",
		Usage:   "app config get [-a/--app appname] [--show-content]",
		Desc:    `Retrieves configuration files for an application.`,
		MinArgs: 0,
	}
}

func (c *ConfigGet) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
	if err != nil {
		return err
	}

	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/config", appName))
	if err != nil {
		return err
	}

	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	defer response.Body.Close()

	var a map[string]string
	err = json.NewDecoder(response.Body).Decode(&a)
	if err != nil {
		return err
	}

	formatted := make([]string, 0, len(a))
	for file, content := range a {
		formatted = append(formatted, fmt.Sprintf("\t%s:\n%s", file, content))
	}
	fmt.Fprintln(context.Stdout, "Files:")
	fmt.Fprintln(context.Stdout, strings.Join(formatted, "\n"))

	return nil
}

type ConfigSet struct {
	cmd.AppNameMixIn
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (c *ConfigSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-config-set",
		Usage:   "app config set <source> <target> [-a/--app appname]",
		Desc:    `Sets a configuration file for an application.`,
		MinArgs: 2,
	}
}

func (c *ConfigSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Sets a configuration file without restarting the application")
	}
	return c.fs
}

func (c *ConfigSet) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()
	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("Please use the -a/--app flag to specify which app you want to update.")
	}

	if len(ctx.Args) < 2 {
		return errors.New(configSetValidationMessage)
	}

	sourceContent, err := ioutil.ReadFile(ctx.Args[0])
	if err != nil {
		return err
	}

	args := apiTypes.Config{
		Filename:  ctx.Args[1],
		Content:   string(sourceContent),
		NoRestart: c.noRestart,
	}

	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/config", appName))
	if err != nil {
		return err
	}
	v, err := json.Marshal(args)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, bytes.NewReader(v))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	_, err = cli.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Configuration has been set for app %s!\n", appName)
	return nil
}

type ConfigUnset struct {
	cmd.AppNameMixIn
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (c *ConfigUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-config-unset",
		Usage:   "app config unset <target> [-a/--app appname]",
		Desc:    `Unsets a configuration file an application.`,
		MinArgs: 1,
	}
}

func (c *ConfigUnset) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Unsets a configuration file without restarting the application")
	}
	return c.fs
}

func (c *ConfigUnset) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()
	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("Please use the -a/--app flag to specify which app you want to update.")
	}

	if len(ctx.Args) < 1 {
		return errors.New(metadataSetValidationMessage)
	}

	args := apiTypes.Config{
		Filename:  ctx.Args[0],
		NoRestart: c.noRestart,
	}

	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/config", appName))
	if err != nil {
		return err
	}
	v, err := json.Marshal(args)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, bytes.NewReader(v))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	_, err = cli.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Configuration has been unset for app %s!\n", appName)
	return nil
}
