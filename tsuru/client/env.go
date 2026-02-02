// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/cezarsa/form"
	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	apiTypes "github.com/tsuru/tsuru/types/api"
)

const EnvSetValidationMessage = `you must specify environment variables in the form "NAME=value".

Example:

  tsuru env-set NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'

For private variables like passwords you can use -p or --private.

Example:

  tsuru env-set NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes' -p`

const ErrMissingAppOrJob = "you must pass an application or job"
const ErrAppAndJobNotAllowedTogether = "you must pass an application or job, not both"

type EnvGet struct {
	appName string
	jobName string

	fs   *pflag.FlagSet
	json bool
}

func (c *EnvGet) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)

		c.fs.StringVarP(&c.appName, standards.FlagApp, standards.ShortFlagApp, "", "The name of the app.")
		c.fs.StringVarP(&c.jobName, standards.FlagJob, standards.ShortFlagJob, "", "The name of the job.")
		c.fs.BoolVar(&c.json, standards.FlagJSON, false, "Display JSON format")
	}
	return c.fs
}

func (c *EnvGet) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "env-get",
		Usage: "[-a/--app appname] [-j/--job jobname] [ENVIRONMENT_VARIABLE1] [ENVIRONMENT_VARIABLE2] ...",
		Desc:  `Retrieves environment variables for an application or job.`,
	}
}

func (c *EnvGet) Run(context *cmd.Context) error {
	context.RawOutput()

	err := checkAppAndJobInputs(c.appName, c.jobName)
	if err != nil {
		return err
	}

	b, err := requestEnvGetURL(c, context.Args)
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
		value := v["value"].(string)
		public := v["public"].(bool)
		managedBy, _ := v["managedBy"].(string)
		observation := ""

		if public && managedBy != "" {
			observation = fmt.Sprintf("(managed by %s)", managedBy)
		} else if !public && managedBy != "" {
			value = "***"
			observation = fmt.Sprintf("(private variable managed by %s)", managedBy)
		} else if !public {
			value = "***"
			observation = "(private variable)"
		}

		if observation != "" {
			value = value + " " + color.New(color.FgHiBlack, color.Bold).Sprint(observation)
		}
		formatted = append(formatted, fmt.Sprintf("%s=%s", v["name"], value))
	}
	sort.Strings(formatted)
	fmt.Fprintln(context.Stdout, strings.Join(formatted, "\n"))
	return nil
}

func (c *EnvGet) renderJSON(context *cmd.Context, variables []map[string]interface{}) error {
	type envJSON struct {
		Name      string `json:"name"`
		Value     string `json:"value"`
		Private   bool   `json:"private"`
		ManagedBy string `json:"managedBy,omitempty"`
	}

	data := make([]envJSON, 0, len(variables))

	for _, v := range variables {
		private := true
		value := "*** (private variable)"
		if v["public"].(bool) {
			value = v["value"].(string)
			private = false
		}
		managedBy, _ := v["managedBy"].(string)
		data = append(data, envJSON{
			Name:      v["name"].(string),
			Value:     value,
			Private:   private,
			ManagedBy: managedBy,
		})
	}

	return formatter.JSON(context.Stdout, data)
}

type EnvSet struct {
	appName   string
	jobName   string
	fs        *pflag.FlagSet
	private   bool
	noRestart bool
}

func (c *EnvSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "env-set",
		Usage:   "<NAME=value> [NAME=value] ... [-a/--app appname] [-j/--job jobname] [-p/--private] [--no-restart]",
		Desc:    `Sets environment variables for an application or job.`,
		MinArgs: 1,
	}
}

func (c *EnvSet) Run(context *cmd.Context) error {
	context.RawOutput()

	err := checkAppAndJobInputs(c.appName, c.jobName)
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

	var path, apiVersion string
	switch c.appName {
	case "":
		path = fmt.Sprintf("/jobs/%s/env", c.jobName)
		apiVersion = "1.13"
	default:
		path = fmt.Sprintf("/apps/%s/env", c.appName)
		apiVersion = "1.0"
	}

	for _, env := range envs {
		if isSensitiveName(env.Name) && !c.private {
			fmt.Fprintf(context.Stdout, "Warning: The environment variable '%s' looks like a sensitive variable. It is recommended to set it as private using the -p or --private flag.\n", env.Name)
		}
	}

	url, err := config.GetURLVersion(apiVersion, path)
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
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(context.Stdout, response)
}

var SENSITIVE_KEYWORDS = map[string]struct{}{
	"KEY":        {},
	"TOKEN":      {},
	"SECRET":     {},
	"PASSWORD":   {},
	"CREDENTIAL": {},
	"API_KEY":    {},
	"APIKEY":     {},
}

func isSensitiveName(name string) bool {
	upperName := strings.ToUpper(name)
	for keyword := range SENSITIVE_KEYWORDS {
		if strings.Contains(upperName, keyword) {
			return true
		}
	}
	return false
}

func (c *EnvSet) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)

		c.fs.StringVarP(&c.appName, standards.FlagApp, standards.ShortFlagApp, "", "The name of the app.")
		c.fs.StringVarP(&c.jobName, standards.FlagJob, standards.ShortFlagJob, "", "The name of the job.")

		c.fs.BoolVarP(&c.private, "private", "p", false, "Private environment variables")
		c.fs.BoolVar(&c.noRestart, standards.FlagNoRestart, false, "Sets environment varibles without restart the application")
	}
	return c.fs
}

type EnvUnset struct {
	appName   string
	jobName   string
	fs        *pflag.FlagSet
	noRestart bool
}

func (c *EnvUnset) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)

		c.fs.StringVarP(&c.appName, standards.FlagApp, standards.ShortFlagApp, "", "The name of the app.")
		c.fs.StringVarP(&c.jobName, standards.FlagJob, standards.ShortFlagJob, "", "The name of the job.")
		c.fs.BoolVar(&c.noRestart, standards.FlagNoRestart, false, "Unset environment variables without restart the application")
	}
	return c.fs
}

func (c *EnvUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "env-unset",
		Usage:   "<ENVIRONMENT_VARIABLE1> [ENVIRONMENT_VARIABLE2] ... [ENVIRONMENT_VARIABLEN] [-a/--app appname] [-j/--job jobname] [--no-restart]",
		Desc:    `Unset environment variables for an application or job.`,
		MinArgs: 1,
	}
}

func (c *EnvUnset) Run(context *cmd.Context) error {
	context.RawOutput()

	err := checkAppAndJobInputs(c.appName, c.jobName)
	if err != nil {
		return err
	}

	v := url.Values{}
	for _, e := range context.Args {
		v.Add("env", e)
	}
	v.Set("noRestart", strconv.FormatBool(c.noRestart))

	var path, apiVersion string
	switch c.appName {
	case "":
		path = fmt.Sprintf("/jobs/%s/env?%s", c.jobName, v.Encode())
		apiVersion = "1.13"
	default:
		path = fmt.Sprintf("/apps/%s/env?%s", c.appName, v.Encode())
		apiVersion = "1.0"
	}

	url, err := config.GetURLVersion(apiVersion, path)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(context.Stdout, response)
}

func requestEnvGetURL(c *EnvGet, args []string) ([]byte, error) {
	v := url.Values{}
	for _, e := range args {
		v.Add("env", e)
	}

	var path, apiVersion string
	switch c.appName {
	case "":
		path = fmt.Sprintf("/jobs/%s/env?%s", c.jobName, v.Encode())
		apiVersion = "1.16"
	default:
		path = fmt.Sprintf("/apps/%s/env?%s", c.appName, v.Encode())
		apiVersion = "1.0"
	}

	url, err := config.GetURLVersion(apiVersion, path)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	r, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func checkAppAndJobInputs(appName string, jobName string) error {
	if appName == "" && jobName == "" {
		return errors.New(ErrMissingAppOrJob)
	}

	if appName != "" && jobName != "" {
		return errors.New(ErrAppAndJobNotAllowedTogether)
	}

	return nil
}
