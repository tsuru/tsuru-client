package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	tsuruClientApp "github.com/tsuru/tsuru-client/tsuru/app"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
)

const metadataSetValidationMessage = `you must specify metadata in the form "NAME=value" with the specified type.

Example:

  tsuru metadata-set <-a APPNAME | -j JOBNAME> -t label NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'
  tsuru metadata-set <-a APPNAME | -j JOBNAME> -t annotation NAME=value OTHER_NAME="value with spaces" ANOTHER_NAME='using single quotes'`

var allowedTypes = []string{"label", "annotation"}

func (c *JobOrApp) getMetadata(apiClient *tsuru.APIClient) (*tsuru.Metadata, map[string]tsuru.Metadata, error) {
	if c.Type == "job" {
		job, _, err := apiClient.JobApi.GetJob(context.Background(), c.val)
		if err != nil {
			return nil, nil, err
		}
		return &job.Job.Metadata, nil, nil
	}
	app, _, err := apiClient.AppApi.AppGet(context.Background(), c.val)
	if err != nil {
		return nil, nil, err
	}

	processMetadata := map[string]tsuru.Metadata{}

	for _, p := range app.Processes {
		processMetadata[p.Name] = p.Metadata
	}

	return &app.Metadata, processMetadata, nil
}

func (c *JobOrApp) setMetadata(apiClient *tsuru.APIClient, metadata tsuru.Metadata, noRestart bool) (*http.Response, error) {
	if c.Type == "job" {
		j := tsuru.InputJob{
			Name:     c.val,
			Metadata: metadata,
		}
		return apiClient.JobApi.UpdateJob(context.Background(), c.val, j)
	}

	a := tsuru.UpdateApp{
		NoRestart: noRestart,
	}

	if c.appProcess == "" {
		a.Metadata = metadata
	} else {
		a.Processes = append(a.Processes, tsuru.AppProcess{
			Name:     c.appProcess,
			Metadata: metadata,
		})
	}
	return apiClient.AppApi.AppUpdate(context.Background(), c.val, a)
}

type MetadataGet struct {
	tsuruClientApp.AppNameMixIn
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

func (c *MetadataGet) Run(context *cmd.Context) error {
	joa := JobOrApp{fs: c.fs}
	err := joa.validate()
	if err != nil {
		return err
	}
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	metadata, metadataByProcess, err := joa.getMetadata(apiClient)
	if err != nil {
		return err
	}

	if c.json {
		return formatter.JSON(context.Stdout, metadata)
	}

	if len(metadataByProcess) > 0 {
		fmt.Fprintln(context.Stdout, "Metadata for app:")
	}
	outputMetadata(context.Stdout, metadata)

	for process, processMetadata := range metadataByProcess {
		if len(metadataByProcess) > 0 {
			fmt.Fprintf(context.Stdout, "\nMetadata for process: %q\n", process)
		}
		outputMetadata(context.Stdout, &processMetadata)
	}

	return nil
}

func outputMetadata(w io.Writer, metadata *tsuru.Metadata) {
	if len(metadata.Labels) > 0 {
		formatted := make([]string, 0, len(metadata.Labels))
		for _, v := range metadata.Labels {
			formatted = append(formatted, fmt.Sprintf("\t%s: %s", v.Name, v.Value))
		}
		sort.Strings(formatted)
		fmt.Fprintln(w, "Labels:")
		fmt.Fprintln(w, strings.Join(formatted, "\n"))
	}

	if len(metadata.Annotations) > 0 {
		formatted := make([]string, 0, len(metadata.Annotations))
		for _, v := range metadata.Annotations {
			formatted = append(formatted, fmt.Sprintf("\t%s: %s", v.Name, v.Value))
		}
		sort.Strings(formatted)
		fmt.Fprintln(w, "Annotations:")
		fmt.Fprintln(w, strings.Join(formatted, "\n"))
	}
}

type MetadataSet struct {
	tsuruClientApp.AppNameMixIn
	job          string
	processName  string
	fs           *gnuflag.FlagSet
	metadataType string
	noRestart    bool
}

func (c *MetadataSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "metadata-set",
		Usage:   "metadata set <NAME=value> [NAME=value] ... <-a/--app appname | -j/--job jobname> <-p process> [-t/--type type]",
		Desc:    `Sets metadata such as labels and annotations for an application or job.`,
		MinArgs: 1,
	}
}

func (c *MetadataSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.job, "job", "", "The name of the job.")
		c.fs.StringVar(&c.job, "j", "", "The name of the job.")
		c.fs.StringVar(&c.processName, "process", "", "The name of process of app (optional).")
		c.fs.StringVar(&c.processName, "p", "", "The name of process of app (optional).")
		c.fs.StringVar(&c.metadataType, "type", "", "Metadata type: annotation or label")
		c.fs.StringVar(&c.metadataType, "t", "", "Metadata type: annotation or label")
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Sets metadata without restarting the application")
	}
	return c.fs
}

func (c *MetadataSet) Run(ctx *cmd.Context) error {
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

	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}

	response, err := joa.setMetadata(apiClient, metadata, c.noRestart)
	if err != nil {
		return err
	}

	err = formatter.StreamJSONResponse(ctx.Stdout, response)
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
	return errors.New("a type is required: label or annotation")
}

type MetadataUnset struct {
	tsuruClientApp.AppNameMixIn
	job          string
	processName  string
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
		c.fs.StringVar(&c.processName, "process", "", "The name of process of app (optional).")
		c.fs.StringVar(&c.processName, "p", "", "The name of process of app (optional).")
		c.fs.StringVar(&c.metadataType, "type", "", "Metadata type: annotation or label")
		c.fs.StringVar(&c.metadataType, "t", "", "Metadata type: annotation or label")
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "Sets metadata without restarting the application")
	}
	return c.fs
}

func (c *MetadataUnset) Run(ctx *cmd.Context) error {
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
		items[i] = tsuru.MetadataItem{Name: parts[0], Delete: true}
	}

	var metadata tsuru.Metadata
	switch c.metadataType {
	case "label":
		metadata.Labels = items
	case "annotation":
		metadata.Annotations = items
	}

	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}

	response, err := joa.setMetadata(apiClient, metadata, c.noRestart)
	if err != nil {
		return err
	}

	err = formatter.StreamJSONResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "%s %q has been updated!\n", joa.Type, joa.val)
	return nil
}
