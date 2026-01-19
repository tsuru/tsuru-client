// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/antihax/optional"
	"github.com/mattn/go-shellwords"
	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/completions"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
)

type JobCreate struct {
	schedule       string
	teamOwner      string
	plan           string
	pool           string
	description    string
	manual         bool
	maxRunningTime int64
	tags           cmd.StringSliceFlag

	fs *pflag.FlagSet
}

func (c *JobCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "job-create",
		Usage: "job create <jobname> <image> \"<commands>\" [--plan/-p plan name] [--schedule/-s schedule name] [--team/-t team owner] [--pool/-o pool name] [--description/-d description] [--tag/-g tag] [--max-running-time/-m seconds] [--manual bool]...",
		Desc: `Creates a new job using the given name and platform

In order to create an job, you need to be member of at least one team. All
teams that you are member (see [[tsuru team-list]]) will be able to access the
job

The [[--plan]] parameter defines the plan to be used. The plan specifies how
computational resources are allocated to your job execution. Typically this
means limits for cpu and memory usage is allocated.
The list of available plans can be found running [[tsuru plan list]]

If this parameter is not informed, tsuru will choose the plan with the
[[default]] flag set to true

The [[--schedule]] parameter defines how often this job will be executed. This string follows the unix-cron format,
if you need to test the cron expressions visit the site: https://crontab.guru/

The [[--team]] parameter describes which team is responsible for the created
app, this is only needed if the current user belongs to more than one team, in
which case this parameter will be mandatory

The [[--pool]] parameter defines which pool your app will be deployed.
This is only needed if you have more than one pool associated with your teams

The [[--description]] parameter sets a description for your job.
It is an optional parameter, and if its not set the job will only not have a
description associated

The [[--manual]] parameter sets your job as a manual job.
A manual job is only run when explicitly triggered by the user i.e: tsuru job trigger <job-name> 

The [[--tag]] parameter sets a tag to your job. You can set multiple [[--tag]] parameters

The [[--max-running-time]] sets maximum amount of time (in seconds) that the job
can run. If the job exceeds this limit, it will be automatically stopped. If
this parameter is not informed, default value is 3600s`,
		MinArgs: 1,
	}
}

func (c *JobCreate) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)
		c.fs.SortFlags = false
		infoMessage := "The plan used to create the job"
		c.fs.StringVarP(&c.plan, standards.FlagPlan, standards.ShortFlagPlan, "", infoMessage)

		schedule := "Schedule string"
		c.fs.StringVarP(&c.schedule, "schedule", "s", "", schedule)
		teamMessage := "Team owner job"
		c.fs.StringVarP(&c.teamOwner, standards.FlagTeam, standards.ShortFlagTeam, "", teamMessage)
		poolMessage := "Pool to deploy your job"
		c.fs.StringVarP(&c.pool, standards.FlagPool, standards.ShortFlagPool, "", poolMessage)
		descriptionMessage := "Job description"
		c.fs.StringVarP(&c.description, standards.FlagDescription, standards.ShortFlagDescription, "", descriptionMessage)
		tagMessage := "Job tag"
		c.fs.VarP(&c.tags, standards.FlagTag, standards.ShortFlagTag, tagMessage)
		manualMessage := "Manual job"
		c.fs.BoolVar(&c.manual, "manual", false, manualMessage)
		maxRunningTime := "Maximum running time in seconds for the job"
		c.fs.Int64VarP(&c.maxRunningTime, "max-running-time", "m", 0, maxRunningTime)
	}
	return c.fs
}

func parseJobCommands(commands []string) ([]string, error) {
	if len(commands) != 1 {
		return commands, nil
	}
	quotedCommands := commands[0]
	jsonCommands := []string{}
	if err := json.Unmarshal([]byte(quotedCommands), &jsonCommands); err == nil {
		return jsonCommands, nil
	}
	shellCommands, err := shellwords.Parse(quotedCommands)
	if err != nil {
		return nil, err
	}
	return shellCommands, nil
}

func (c *JobCreate) Run(ctx *cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	if !c.manual && c.schedule == "" {
		return errors.New("schedule or manual option must be set")
	}
	if c.manual && c.schedule != "" {
		return errors.New("cannot set both manual job and schedule options")
	}

	var image string
	var parsedCommands []string
	jobName := ctx.Args[0]
	if len(ctx.Args) > 1 {
		fmt.Fprintf(ctx.Stdout, "Job creation with image is being deprecated. You should use 'tsuru job deploy' to set a job`s image\n")
		image = ctx.Args[1]
		commands := ctx.Args[2:]

		parsedCommands, err = parseJobCommands(commands)
		if err != nil {
			return err
		}
	}

	var activeDeadlineSecondsResult *int64
	if c.fs != nil {
		c.fs.Visit(func(f *pflag.Flag) {
			if (f.Name == "max-running-time" || f.Name == "m") && c.maxRunningTime == 0 {
				activeDeadlineSecondsResult = &c.maxRunningTime
			}
		})
	}
	if c.maxRunningTime > 0 {
		activeDeadlineSecondsResult = &c.maxRunningTime
	}
	j := tsuru.InputJob{
		Name:                  jobName,
		Tags:                  c.tags,
		Schedule:              c.schedule,
		Plan:                  c.plan,
		Pool:                  c.pool,
		Description:           c.description,
		TeamOwner:             c.teamOwner,
		Manual:                c.manual,
		ActiveDeadlineSeconds: activeDeadlineSecondsResult,
		Container: tsuru.JobSpecContainer{
			Image:   image,
			Command: parsedCommands,
		},
	}
	if _, err := apiClient.JobApi.CreateJob(context.Background(), j); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Job created\nUse \"tsuru job info %s\" to check the status of the job\n", jobName)
	return nil
}

type JobInfo struct {
	fs   *pflag.FlagSet
	json bool
}

func (c *JobInfo) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("job-info", pflag.ContinueOnError)
		c.fs.BoolVar(&c.json, standards.FlagJSON, false, "Show JSON")
	}
	return c.fs
}

func (c *JobInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "job-info",
		Usage:   "job info <job>",
		Desc:    "Retrieve useful information from a job",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

const jobInfoFormat = `Job: {{.Job.Name}}
{{- with .Job.Description }}
Description: {{.}}
{{- end }}
{{- with .DashboardURL }}
Dashboard: {{.}}
{{- end }}
Teams: {{.Teams}}
Created by: {{.Job.Owner}}
Cluster: {{.Cluster}}
Pool: {{.Job.Pool}}
Plan: {{.Job.Plan.Name}}
{{- if and .Job.Spec.Schedule (not .Job.Spec.Manual) }}
Schedule: {{.Job.Spec.Schedule}}
{{- end }}
Image: {{.Job.Spec.Container.Image}}
Command: {{.Job.Spec.Container.Command}}
{{- if .Job.Spec.ActiveDeadlineSeconds }}
Max Running Time: {{.Job.Spec.ActiveDeadlineSeconds}}s
{{- end }}
{{- if .Job.Spec.ConcurrencyPolicy }}
Concurrency Policy: {{.Job.Spec.ConcurrencyPolicy}}
{{- end }}`

func (c *JobInfo) Run(ctx *cmd.Context) error {
	jobName := ctx.Args[0]
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
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

	teams := renderTeams(jobInfo.Job)
	err = tmpl.Execute(&buf, struct {
		Job          tsuru.Job
		DashboardURL string
		Cluster      string
		Teams        string
	}{Job: jobInfo.Job, DashboardURL: jobInfo.DashboardURL, Cluster: jobInfo.Cluster, Teams: teams})
	if err != nil {
		return err
	}

	renderJobUnits(&buf, jobInfo.Units)
	renderServiceInstanceBinds(&buf, jobInfo.ServiceInstanceBinds)
	fmt.Fprintln(ctx.Stdout, buf.String())
	return nil
}

var _ cmd.AutoCompleteCommand = &JobInfo{}

func (c *JobInfo) Complete(args []string, toComplete string) ([]string, error) {
	return completions.JobNameCompletionFunc(toComplete)
}

func renderTeams(job tsuru.Job) string {
	teams := []string{}
	if job.TeamOwner != "" {
		teams = append(teams, job.TeamOwner+" (owner)")
	}

	for _, t := range job.Teams {
		if t != job.TeamOwner {
			teams = append(teams, t)
		}
	}

	return strings.Join(teams, ", ")
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

		fmt.Fprintf(buf, "Units: %d\n", unitsTable.Rows())
		fmt.Fprint(buf, unitsTable.String())
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
	fs         *pflag.FlagSet
	filter     jobFilter
	json       bool
	simplified bool
}

func (c *JobList) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("job-list", pflag.ContinueOnError)
		c.fs.SortFlags = false

		c.fs.StringVarP(&c.filter.name, standards.FlagName, standards.ShortFlagName, "", "Filter jobs by name")
		c.fs.StringVarP(&c.filter.pool, standards.FlagPool, standards.ShortFlagPool, "", "Filter jobs by pool")
		c.fs.StringVarP(&c.filter.plan, standards.FlagPlan, standards.ShortFlagPlan, "", "Filter jobs by plan")
		c.fs.StringVarP(&c.filter.teamOwner, standards.FlagTeam, standards.ShortFlagTeam, "", "Filter jobs by team owner")
		c.fs.BoolVarP(&c.simplified, standards.FlagOnlyName, standards.ShortFlagOnlyName, false, "Display only jobs name")
		c.fs.BoolVar(&c.json, standards.FlagJSON, false, "Show JSON")
	}
	return c.fs
}

func (c *JobList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "job-list",
		Usage: "job list",
		Desc:  "List jobs",
	}
}

func (c *JobList) Run(ctx *cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}

	jobs, resp, err := apiClient.JobApi.ListJob(context.Background())

	if resp != nil && resp.StatusCode == http.StatusNoContent {
		fmt.Fprint(ctx.Stdout, "No jobs found\n")
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
		schedule := j.Spec.Schedule
		if j.Spec.Manual {
			schedule = "manual"
		}
		tbl.AddRow(tablecli.Row{
			j.Name,
			schedule,
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
		Desc:    "Delete an existing job",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *JobDelete) Run(ctx *cmd.Context) error {
	jobName := ctx.Args[0]

	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}

	_, err = apiClient.JobApi.DeleteJob(context.Background(), jobName)
	if err != nil {
		return err
	}

	fmt.Fprint(ctx.Stdout, "Job successfully deleted\n")
	return nil
}

var _ cmd.AutoCompleteCommand = &JobDelete{}

func (c *JobDelete) Complete(args []string, toComplete string) ([]string, error) {
	return completions.JobNameCompletionFunc(toComplete)
}

type JobTrigger struct{}

func (c *JobTrigger) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "job-trigger",
		Usage:   "job trigger <job-name>",
		Desc:    "Trigger an existing job",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *JobTrigger) Run(ctx *cmd.Context) error {
	jobName := ctx.Args[0]

	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}

	_, err = apiClient.JobApi.TriggerJob(context.Background(), jobName)
	if err != nil {
		return err
	}

	fmt.Fprint(ctx.Stdout, "Job successfully triggered\n")
	return nil
}

var _ cmd.AutoCompleteCommand = &JobTrigger{}

func (c *JobTrigger) Complete(args []string, toComplete string) ([]string, error) {
	return completions.JobNameCompletionFunc(toComplete)
}

type JobUpdate struct {
	schedule       string
	teamOwner      string
	plan           string
	pool           string
	description    string
	image          string
	manual         bool
	maxRunningTime int64
	tags           cmd.StringSliceFlag

	fs *pflag.FlagSet
}

func (c *JobUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "job-update",
		Usage:   "job update <job-name> [--image/-i <image>] [--plan/-p plan name] [--schedule/-s schedule name] [--manual] [--team/-t team owner] [--pool/-o pool name] [--description/-d description] [--max-running-time/-m seconds] [--tag/-g tag]... -- [commands]",
		Desc:    "Updates a job",
		MinArgs: 1,
	}
}

func (c *JobUpdate) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)
		c.fs.SortFlags = false

		infoMessage := "The plan used to create the job"
		c.fs.StringVarP(&c.plan, standards.FlagPlan, standards.ShortFlagPlan, "", infoMessage)

		schedule := "Schedule string"
		c.fs.StringVarP(&c.schedule, "schedule", "s", "", schedule)

		manualMessage := "Manual job"
		c.fs.BoolVar(&c.manual, "manual", false, manualMessage)

		teamMessage := "Team owner job"
		c.fs.StringVarP(&c.teamOwner, standards.FlagTeam, standards.ShortFlagTeam, "", teamMessage)

		poolMessage := "Pool to deploy your job"
		c.fs.StringVarP(&c.pool, standards.FlagPool, standards.ShortFlagPool, "", poolMessage)

		descriptionMessage := "Job description"
		c.fs.StringVarP(&c.description, standards.FlagDescription, standards.ShortFlagDescription, "", descriptionMessage)

		tagMessage := "Job tag"
		c.fs.VarP(&c.tags, standards.FlagTag, standards.ShortFlagTag, tagMessage)

		imageMessage := "New image for the job to run"
		c.fs.StringVarP(&c.image, "image", "i", "", imageMessage)

		maxRunningTime := "Maximum running time in seconds for the job"
		c.fs.Int64VarP(&c.maxRunningTime, "max-running-time", "m", 0, maxRunningTime)
	}
	return c.fs
}

func (c *JobUpdate) Run(ctx *cmd.Context) error {
	jobName := ctx.Args[0]
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	if c.manual && c.schedule != "" {
		return errors.New("cannot set both manual job and schedule options")
	}
	if c.image != "" {
		fmt.Fprintf(ctx.Stdout, "Job update with image is being deprecated. You should use 'tsuru job deploy' to set a job`s image\n")
	}
	var jobUpdateCommands []string
	if len(ctx.Args) > 1 {
		jobUpdateCommands, err = parseJobCommands(ctx.Args[1:])
		if err != nil {
			return err
		}
	}
	var activeDeadlineSecondsResult *int64
	if c.fs != nil {
		c.fs.Visit(func(f *pflag.Flag) {
			if (f.Name == "max-running-time" || f.Name == "m") && c.maxRunningTime == 0 {
				activeDeadlineSecondsResult = &c.maxRunningTime
			}
		})
	}
	if c.maxRunningTime > 0 {
		activeDeadlineSecondsResult = &c.maxRunningTime
	}
	j := tsuru.InputJob{
		Name:                  jobName,
		Tags:                  c.tags,
		Schedule:              c.schedule,
		Manual:                c.manual,
		Plan:                  c.plan,
		Pool:                  c.pool,
		Description:           c.description,
		TeamOwner:             c.teamOwner,
		ActiveDeadlineSeconds: activeDeadlineSecondsResult,
		Container: tsuru.JobSpecContainer{
			Image:   c.image,
			Command: jobUpdateCommands,
		},
	}

	_, err = apiClient.JobApi.UpdateJob(context.Background(), jobName, j)
	if err != nil {
		return err
	}

	fmt.Fprintf(ctx.Stdout, "Job updated\nUse \"tsuru job info %s\" to check the status of the job\n", jobName)
	return nil
}

var _ cmd.AutoCompleteCommand = &JobUpdate{}

func (c *JobUpdate) Complete(args []string, toComplete string) ([]string, error) {
	return completions.JobNameCompletionFunc(toComplete)
}

type JobLog struct {
	follow bool
	fs     *pflag.FlagSet
}

func (c *JobLog) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "job-log",
		Usage:   "job log <job-name>",
		Desc:    "Retrieve logs a job",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *JobLog) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("job-log", pflag.ExitOnError)
		followMsg := "Follow logs"
		c.fs.BoolVarP(&c.follow, "follow", "f", false, followMsg)
	}
	return c.fs
}

var _ cmd.AutoCompleteCommand = &JobLog{}

func (c *JobLog) Complete(args []string, toComplete string) ([]string, error) {
	return completions.JobNameCompletionFunc(toComplete)
}

func (c *JobLog) Run(ctx *cmd.Context) error {
	jobName := ctx.Args[0]
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}

	resp, err := apiClient.JobApi.JobLog(context.Background(), jobName, &tsuru.JobLogOpts{
		Follow: optional.NewBool(c.follow),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	formatter := logFormatter{}
	dec := json.NewDecoder(resp.Body)
	ctx.RawOutput()
	for {
		err = formatter.Format(ctx.Stdout, dec)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(ctx.Stdout, "Error: %v", err)
			}
			break
		}
	}

	return nil
}
