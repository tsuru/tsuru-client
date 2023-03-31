package client

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
)

type JobCreate struct {
	schedule    string
	teamOwner   string
	plan        string
	pool        string
	description string
	tags        cmd.StringSliceFlag

	fs *gnuflag.FlagSet
}

func (c *JobCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "job-create",
		Usage: "job create <jobname> <image> <commands> [--plan/-p plan name] [--schedule/-s schedule name] [--team/-t team owner] [--pool/-o pool name] [--description/-d description] [--tag/-g tag]...",
		Desc: `Creates a new job using the given name and platform.

In order to create an job, you need to be member of at least one team. All
teams that you are member (see [[tsuru team-list]]) will be able to access the
job.

The [[--plan]] parameter defines the plan to be used. The plan specifies how
computational resources are allocated to your job execution. Typically this
means limits for cpu and memory usage is allocated.
The list of available plans can be found running [[tsuru plan list]].

If this parameter is not informed, tsuru will choose the plan with the
[[default]] flag set to true.

The [[--schedule]] parameter defines how often this job will be executed. This string follows the unix-cron format,
if you need to test the cron expressions visit the site: https://crontab.guru/.

The [[--team]] parameter describes which team is responsible for the created
app, this is only needed if the current user belongs to more than one team, in
which case this parameter will be mandatory.

The [[--pool]] parameter defines which pool your app will be deployed.
This is only needed if you have more than one pool associated with your teams.

The [[--description]] parameter sets a description for your job.
It is an optional parameter, and if its not set the job will only not have a
description associated.

The [[--tag]] parameter sets a tag to your job. You can set multiple [[--tag]] parameters.`,
		MinArgs: 2,
	}
}

func (c *JobCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		infoMessage := "The plan used to create the job"
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.StringVar(&c.plan, "plan", "", infoMessage)
		c.fs.StringVar(&c.plan, "p", "", infoMessage)
		schedule := "schedule string"
		c.fs.StringVar(&c.schedule, "schedule", "", schedule)
		c.fs.StringVar(&c.schedule, "s", "", schedule)
		teamMessage := "Team owner job"
		c.fs.StringVar(&c.teamOwner, "team", "", teamMessage)
		c.fs.StringVar(&c.teamOwner, "t", "", teamMessage)
		poolMessage := "Pool to deploy your job"
		c.fs.StringVar(&c.pool, "pool", "", poolMessage)
		c.fs.StringVar(&c.pool, "o", "", poolMessage)
		descriptionMessage := "Job description"
		c.fs.StringVar(&c.description, "description", "", descriptionMessage)
		c.fs.StringVar(&c.description, "d", "", descriptionMessage)
		tagMessage := "Job tag"
		c.fs.Var(&c.tags, "tag", tagMessage)
		c.fs.Var(&c.tags, "g", tagMessage)
	}
	return c.fs
}

func (c *JobCreate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}

	jobName := ctx.Args[0]
	image := ctx.Args[1]
	commands := ctx.Args[2:]

	_, err = apiClient.JobApi.CreateJob(context.Background(), tsuru.InputJob{
		Name:        jobName,
		Tags:        c.tags,
		Schedule:    c.schedule,
		Plan:        c.plan,
		Pool:        c.pool,
		Description: c.description,
		TeamOwner:   c.teamOwner,
		Container: tsuru.InputJobContainer{
			Image:   image,
			Command: commands,
		},
	})

	if err == nil {
		fmt.Fprintf(ctx.Stdout, "Job %q has been created!\n", jobName)
		fmt.Fprintln(ctx.Stdout, "Use job info to check the status of the job.")
	}

	return err
}

type JobInfo struct {
	fs   *gnuflag.FlagSet
	json bool
}

func (c *JobInfo) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("job-info", gnuflag.ContinueOnError)
		c.fs.BoolVar(&c.json, "json", false, "Show JSON")
	}
	return c.fs
}

func (c *JobInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "job-info",
		Usage:   "job info <job>",
		Desc:    `Get a job.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

const jobInfoFormat = `Job: {{.Job.Name}}
{{- if .Job.Description }}
Description: {{.Job.Description}}
{{- end }}
Teams: {{.Job.Teams}}
Created by: {{.Job.Owner}}
Pool: {{.Job.Pool}}
Plan: {{.Job.Plan.Name}}
{{- if .Job.Spec.Schedule }}
Schedule: {{.Job.Spec.Schedule}}
{{- end }}
Image: {{.Job.Spec.Container.Image}}
Command: {{.Job.Spec.Container.Command}}`

func (c *JobInfo) Run(ctx *cmd.Context, cli *cmd.Client) error {
	jobName := ctx.Args[0]
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}

	jobInfo, _, err := apiClient.JobApi.GetJob(context.Background(), jobName)

	if err != nil {
		return err
	}
	if c.json {
		return formatter.JSON(ctx.Stdout, jobInfo)
	}

	var buf bytes.Buffer
	tmpl := template.Must(template.New("job").Parse(jobInfoFormat))

	err = tmpl.Execute(&buf, jobInfo)
	if err != nil {
		return err
	}

	renderJobUnits(&buf, jobInfo.Units)
	fmt.Println(buf.String())
	return nil
}

func renderJobUnits(buf *bytes.Buffer, units []tsuru.Unit) {

	titles := []string{"Name", "Status", "Restarts", "Age"}
	unitsTable := tablecli.NewTable()
	tablecli.TableConfig.ForceWrap = false
	unitsTable.Headers = tablecli.Row(titles)

	for _, unit := range units {
		row := tablecli.Row{
			unit.Name,
			jobUnitReadyAndStatus(unit),
			countValue(unit.Restarts),
			jobAge(unit.CreatedAt),
		}

		unitsTable.AddRow(row)
	}
	if unitsTable.Rows() > 0 {
		unitsTable.SortByColumn(2)
		buf.WriteString("\n")

		buf.WriteString(fmt.Sprintf("Units: %d\n", unitsTable.Rows()))
		buf.WriteString(unitsTable.String())
	}
}

func jobUnitReadyAndStatus(u tsuru.Unit) string {
	if u.Ready != nil && *u.Ready {
		return "ready"
	}

	return u.Status
}

func jobAge(createdAt string) string {
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return ""
	}
	return translateTimestampSince(&t)
}

type jobFilter struct {
	name      string
	pool      string
	plan      string
	teamOwner string
}

type JobList struct {
	fs         *gnuflag.FlagSet
	filter     jobFilter
	json       bool
	simplified bool
}

func (c *JobList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("job-list", gnuflag.ContinueOnError)
		c.fs.StringVar(&c.filter.name, "name", "", "Filter jobs by name")
		c.fs.StringVar(&c.filter.name, "n", "", "Filter jobs by name")
		c.fs.StringVar(&c.filter.pool, "pool", "", "Filter jobs by pool")
		c.fs.StringVar(&c.filter.pool, "o", "", "Filter jobs by pool")
		c.fs.StringVar(&c.filter.plan, "plan", "", "Filter jobs by plan")
		c.fs.StringVar(&c.filter.plan, "p", "", "Filter jobs by plan")
		c.fs.StringVar(&c.filter.teamOwner, "team", "", "Filter jobs by team owner")
		c.fs.StringVar(&c.filter.teamOwner, "t", "", "Filter jobs by team owner")
		c.fs.BoolVar(&c.simplified, "q", false, "Display only jobs name")
		c.fs.BoolVar(&c.json, "json", false, "Show JSON")
	}
	return c.fs
}

func (c *JobList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "job-list",
		Usage:   "job list",
		Desc:    `List jobs.`,
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *JobList) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}

	jobs, resp, err := apiClient.JobApi.ListJob(context.Background())

	if resp != nil && resp.StatusCode == http.StatusNoContent {
		fmt.Fprint(ctx.Stdout, "No jobs found.\n")
		return nil
	}
	if err != nil {
		return err
	}

	jobs = c.clientSideFilter(jobs)
	if c.json {
		return formatter.JSON(ctx.Stdout, jobs)
	}

	if c.simplified {
		for _, j := range jobs {
			fmt.Fprintln(ctx.Stdout, j.Name)
		}
		return nil
	}

	tbl := tablecli.NewTable()
	tbl.Headers = tablecli.Row{"Name", "Schedule", "Image", "Command"}
	tbl.LineSeparator = true
	for _, j := range jobs {
		tbl.AddRow(tablecli.Row{
			j.Name,
			j.Spec.Schedule,
			j.Spec.Container.Image,
			strings.Join(j.Spec.Container.Command, " "),
		})
	}
	tbl.Sort()
	fmt.Fprint(ctx.Stdout, tbl.String())

	return nil
}

func (c *JobList) clientSideFilter(jobs []tsuru.Job) []tsuru.Job {
	result := make([]tsuru.Job, 0, len(jobs))

	for _, j := range jobs {
		insert := true
		if c.filter.name != "" && !strings.Contains(j.Name, c.filter.name) {
			insert = false
		}

		if c.filter.pool != "" && j.Pool != c.filter.pool {
			insert = false
		}

		if c.filter.plan != "" && j.Plan.Name != c.filter.plan {
			insert = false
		}

		if c.filter.teamOwner != "" && j.TeamOwner != c.filter.teamOwner {
			insert = false
		}

		if insert {
			result = append(result, j)
		}
	}

	return result
}

type JobDelete struct{}

func (c *JobDelete) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "job-delete",
		Usage:   "job delete <job-name>",
		Desc:    `Delete an existing job volume.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *JobDelete) Run(ctx *cmd.Context, cli *cmd.Client) error {
	jobName := ctx.Args[0]

	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}

	_, err = apiClient.JobApi.DeleteJob(context.Background(), jobName)
	if err != nil {
		return err
	}

	fmt.Fprint(ctx.Stdout, "Job successfully deleted.\n")
	return nil
}
