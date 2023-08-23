package client

import (
	"context"
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
)

const metadataSetValidationMessage = `You must specify metadata in the form "NAME=value" with the specified type.

Example:

  tsuru metadata-set <-a APPNAME | -j JOBNAME> -t label NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'
  tsuru metadata-set <-a APPNAME | -j JOBNAME> -t annotation NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'

`

var allowedTypes = []string{"label", "annotation"}

type JobOrApp struct {
	Type string
	val  string
	fs   *gnuflag.FlagSet
}

func (c *JobOrApp) validate() error {
	appName := c.fs.Lookup("app").Value.String()
	jobName := c.fs.Lookup("job").Value.String()
	if appName == "" && jobName == "" {
		return errors.New("job name or app name is required")
	}
	if appName != "" && jobName != "" {
		return errors.New("please use only one of the -a/--app and -j/--job flags")
	}
	if appName != "" {
		c.Type = "app"
		c.val = appName
		return nil
	}
	c.Type = "job"
	c.val = jobName
	return nil
}

func (c *JobOrApp) getMetdata(apiClient *tsuru.APIClient) (tsuru.Metadata, error) {
	if c.Type == "job" {
		job, _, err := apiClient.JobApi.GetJob(context.Background(), c.val)
		if err != nil {
			return tsuru.Metadata{}, err
		}
		return job.Job.Metadata, nil
	}
	app, _, err := apiClient.AppApi.AppGet(context.Background(), c.val)
	if err != nil {
		return tsuru.Metadata{}, err
	}
	return app.Metadata, nil
}

func (c *JobOrApp) setMetdata(apiClient *tsuru.APIClient, metadata tsuru.Metadata, noRestart bool) (*http.Response, error) {
	if c.Type == "job" {
		j := tsuru.InputJob{
			Name:     c.val,
			Metadata: metadata,
		}
		return apiClient.JobApi.UpdateJob(context.Background(), c.val, j)
	}
	a := tsuru.UpdateApp{
		Metadata:  metadata,
		NoRestart: noRestart,
	}
	return apiClient.AppApi.AppUpdate(context.Background(), c.val, a)
}

type MetadataGet struct {
	cmd.AppNameMixIn
	jobName      string
	flagsApplied bool
	json         bool
	fs           *gnuflag.FlagSet
}

func (c *MetadataGet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.jobName, "job", "", "The name of the job.")
		c.fs.StringVar(&c.jobName, "j", "", "The name of the job.")
		if !c.flagsApplied {
			c.fs.BoolVar(&c.json, "json", false, "Show JSON")
			c.flagsApplied = true
		}
	}
	return c.fs
}

func (c *MetadataGet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "metadata-get",
		Usage:   "metadata get <-a/--app appname | -j/--job jobname>",
		Desc:    `Retrieves metadata for an application or job.`,
		MinArgs: 0,
	}
}

func (c *MetadataGet) Run(context *cmd.Context, cli *cmd.Client) error {
	joa := JobOrApp{fs: c.fs}
	err := joa.validate()
	if err != nil {
		return err
	}
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	metadata, err := joa.getMetdata(apiClient)
	if err != nil {
		return err
	}

	if c.json {
		return formatter.JSON(context.Stdout, metadata)
	}

	formatted := make([]string, 0, len(metadata.Labels))
	for _, v := range metadata.Labels {
		formatted = append(formatted, fmt.Sprintf("\t%s: %s", v.Name, v.Value))
	}
	sort.Strings(formatted)
	fmt.Fprintln(context.Stdout, "Labels:")
	fmt.Fprintln(context.Stdout, strings.Join(formatted, "\n"))

	formatted = make([]string, 0, len(metadata.Annotations))
	for _, v := range metadata.Annotations {
		formatted = append(formatted, fmt.Sprintf("\t%s: %s", v.Name, v.Value))
	}
	sort.Strings(formatted)
	fmt.Fprintln(context.Stdout, "Annotations:")
	fmt.Fprintln(context.Stdout, strings.Join(formatted, "\n"))
	return nil
}

type MetadataSet struct {
	cmd.AppNameMixIn
	job          string
	fs           *gnuflag.FlagSet
	metadataType string
	noRestart    bool
}

func (c *MetadataSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "metadata-set",
		Usage:   "metadata set <NAME=value> [NAME=value] ... <-a/--app appname | -j/--job jobname> [-t/--type type]",
		Desc:    `Sets metadata such as labels and annotations for an application or job.`,
		MinArgs: 1,
	}
}

func (c *MetadataSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.job, "job", "", "The name of the job.")
		c.fs.StringVar(&c.job, "j", "", "The name of the job.")
		c.fs.StringVar(&c.metadataType, "type", "", "Metadata type: annotation or label")
		c.fs.StringVar(&c.metadataType, "t", "", "Metadata type: annotation or label")
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Sets metadata without restarting the application")
	}
	return c.fs
}

func (c *MetadataSet) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()
	joa := JobOrApp{fs: c.fs}
	err := joa.validate()
	if err != nil {
		return err
	}
	if len(ctx.Args) < 1 {
		return errors.New(metadataSetValidationMessage)
	}

	if err = validateType(c.metadataType); err != nil {
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

	response, err := joa.setMetdata(apiClient, metadata, c.noRestart)
	if err != nil {
		return err
	}

	err = cmd.StreamJSONResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "%s %q has been updated!\n", joa.Type, joa.val)
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
	job          string
	fs           *gnuflag.FlagSet
	metadataType string
	noRestart    bool
}

func (c *MetadataUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "metadata-unset",
		Usage:   "metadata unset <NAME> [NAME] ... <-a/--app appname | -j--job jobname> [-t/--type type]",
		Desc:    `Unsets metadata such as labels and annotations for an application or job.`,
		MinArgs: 1,
	}
}

func (c *MetadataUnset) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.job, "job", "", "The name of the job.")
		c.fs.StringVar(&c.job, "j", "", "The name of the job.")
		c.fs.StringVar(&c.metadataType, "type", "", "Metadata type: annotation or label")
		c.fs.StringVar(&c.metadataType, "t", "", "Metadata type: annotation or label")
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Sets metadata without restarting the application")
	}
	return c.fs
}

func (c *MetadataUnset) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()
	joa := JobOrApp{fs: c.fs}
	err := joa.validate()
	if err != nil {
		return err
	}
	if len(ctx.Args) < 1 {
		return errors.New(metadataSetValidationMessage)
	}

	if err = validateType(c.metadataType); err != nil {
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

	response, err := joa.setMetdata(apiClient, metadata, c.noRestart)
	if err != nil {
		return err
	}

	err = cmd.StreamJSONResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "%s %q has been updated!\n", joa.Type, joa.val)
	return nil
}
