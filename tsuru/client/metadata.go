package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
	appTypes "github.com/tsuru/tsuru/types/app"
)

const metadataSetValidationMessage = `You must specify metadata in the form "NAME=value" with the specified type.

Example:

  tsuru app-metadata-set -a APPNAME -t label NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'
  tsuru app-metadata-set -a APPNAME -t annotation NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'

`

var allowedTypes = []string{"label", "annotation"}

type MetadataGet struct {
	cmd.AppNameMixIn

	flagsApplied bool
	json         bool
}

func (c *MetadataGet) Flags() *gnuflag.FlagSet {
	fs := c.AppNameMixIn.Flags()
	if !c.flagsApplied {
		fs.BoolVar(&c.json, "json", false, "Show JSON")

		c.flagsApplied = true
	}
	return fs
}

func (c *MetadataGet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-metadata-get",
		Usage:   "app metadata get [-a/--app appname]",
		Desc:    `Retrieves metadata for an application.`,
		MinArgs: 0,
	}
}

func (c *MetadataGet) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
	if err != nil {
		return err
	}

	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s", appName))
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

	var a struct {
		Metadata appTypes.Metadata
	}
	err = json.NewDecoder(response.Body).Decode(&a)
	if err != nil {
		return err
	}

	if c.json {
		return formatter.JSON(context.Stdout, a.Metadata)
	}

	formatted := make([]string, 0, len(a.Metadata.Labels))
	for _, v := range a.Metadata.Labels {
		formatted = append(formatted, fmt.Sprintf("\t%s: %s", v.Name, v.Value))
	}
	sort.Strings(formatted)
	fmt.Fprintln(context.Stdout, "Labels:")
	fmt.Fprintln(context.Stdout, strings.Join(formatted, "\n"))

	formatted = make([]string, 0, len(a.Metadata.Annotations))
	for _, v := range a.Metadata.Annotations {
		formatted = append(formatted, fmt.Sprintf("\t%s: %s", v.Name, v.Value))
	}
	sort.Strings(formatted)
	fmt.Fprintln(context.Stdout, "Annotations:")
	fmt.Fprintln(context.Stdout, strings.Join(formatted, "\n"))
	return nil
}

type MetadataSet struct {
	cmd.AppNameMixIn
	fs           *gnuflag.FlagSet
	metadataType string
	noRestart    bool
}

func (c *MetadataSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-metadata-set",
		Usage:   "app metadata set <NAME=value> [NAME=value] ... [-a/--app appname] [-t/--type type]",
		Desc:    `Sets metadata such as labels and annotations for an application.`,
		MinArgs: 1,
	}
}

func (c *MetadataSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.metadataType, "type", "", "Metadata type: annotation or label")
		c.fs.StringVar(&c.metadataType, "t", "", "Metadata type: annotation or label")
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Sets metadata without restarting the application")
	}
	return c.fs
}

func (c *MetadataSet) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()
	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("Please use the -a/--app flag to specify which app you want to update.")
	}

	if len(ctx.Args) < 1 {
		return errors.New(metadataSetValidationMessage)
	}

	if err := validateType(c.metadataType); err != nil {
		return err
	}

	items := make([]tsuru.MetadataItem, len(ctx.Args))
	for i := range ctx.Args {
		parts := strings.SplitN(ctx.Args[i], "=", 2)
		if len(parts) != 2 {
			return errors.New(metadataSetValidationMessage)
		}
		items[i] = tsuru.MetadataItem{Name: parts[0], Value: parts[1]}
	}

	var metadata tsuru.Metadata
	switch c.metadataType {
	case "label":
		metadata.Labels = items
	case "annotation":
		metadata.Annotations = items
	}

	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}

	updateApp := tsuru.UpdateApp{
		Metadata:  metadata,
		NoRestart: c.noRestart,
	}

	response, err := apiClient.AppApi.AppUpdate(context.TODO(), appName, updateApp)
	if err != nil {
		return err
	}

	err = cmd.StreamJSONResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "App %q has been updated!\n", appName)
	return nil
}

func validateType(t string) error {
	t = strings.ToLower(t)
	for _, allowed := range allowedTypes {
		if t == allowed {
			return nil
		}
	}
	return errors.New("A type is required: label or annotation")
}

type MetadataUnset struct {
	cmd.AppNameMixIn
	fs           *gnuflag.FlagSet
	metadataType string
	noRestart    bool
}

func (c *MetadataUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-metadata-unset",
		Usage:   "app metadata unset <NAME> [NAME] ... [-a/--app appname] [-t/--type type]",
		Desc:    `Unsets metadata such as labels and annotations for an application.`,
		MinArgs: 1,
	}
}

func (c *MetadataUnset) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.metadataType, "type", "", "Metadata type: annotation or label")
		c.fs.StringVar(&c.metadataType, "t", "", "Metadata type: annotation or label")
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Sets metadata without restarting the application")
	}
	return c.fs
}

func (c *MetadataUnset) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()
	appName := c.Flags().Lookup("app").Value.String()
	if appName == "" {
		return errors.New("Please use the -a/--app flag to specify which app you want to update.")
	}

	if len(ctx.Args) < 1 {
		return errors.New(metadataSetValidationMessage)
	}

	if err := validateType(c.metadataType); err != nil {
		return err
	}

	items := make([]tsuru.MetadataItem, len(ctx.Args))
	for i := range ctx.Args {
		items[i] = tsuru.MetadataItem{Name: ctx.Args[i], Delete: true}
	}

	var metadata tsuru.Metadata
	switch c.metadataType {
	case "label":
		metadata.Labels = items
	case "annotation":
		metadata.Annotations = items
	}

	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}

	updateApp := tsuru.UpdateApp{
		Metadata:  metadata,
		NoRestart: c.noRestart,
	}

	response, err := apiClient.AppApi.AppUpdate(context.TODO(), appName, updateApp)
	if err != nil {
		return err
	}

	err = cmd.StreamJSONResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "App %q has been updated!\n", appName)
	return nil
}
