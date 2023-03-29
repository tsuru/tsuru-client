package client

import (
	"context"
	"fmt"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
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
