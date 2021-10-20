// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	tsuruClient "github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
	volumeTypes "github.com/tsuru/tsuru/types/volume"
)

type VolumeCreate struct {
	fs   *gnuflag.FlagSet
	pool string
	team string
	opt  cmd.MapFlag
}

func (c *VolumeCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-create",
		Usage:   "volume create <volume-name> <plan-name> [-p/--pool <pool>] [-t/--team <team>] [-o/--opt key=value]...",
		Desc:    `Creates a new persistent volume based on a volume plan.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *VolumeCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("volume-create", gnuflag.ExitOnError)
		desc := "the pool that owns the service (mandatory if the user has access to more than one pool)"
		c.fs.StringVar(&c.pool, "pool", "", desc)
		c.fs.StringVar(&c.pool, "p", "", desc)
		desc = "the team that owns the service (mandatory if the user has access to more than one team)"
		c.fs.StringVar(&c.team, "team", "", desc)
		c.fs.StringVar(&c.team, "t", "", desc)
		desc = "backend specific volume options"
		c.fs.Var(&c.opt, "opt", desc)
		c.fs.Var(&c.opt, "o", desc)
	}
	return c.fs
}
func (c *VolumeCreate) volumeCreate(volumeName, planName string) tsuru.Volume {
	volumeCreate := tsuru.Volume{
		Name:      volumeName,
		Plan:      tsuru.VolumePlan{Name: planName},
		Pool:      c.pool,
		TeamOwner: c.team,
		Opts:      map[string]string(c.opt),
	}
	return volumeCreate
}
func (c *VolumeCreate) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	volumeName, planName := ctx.Args[0], ctx.Args[1]
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
	if err != nil {
		return err
	}
	volumeCreate := c.volumeCreate(volumeName, planName)
	response, err := apiClient.VolumeApi.VolumeCreate(context.TODO(), volumeCreate)

	if err != nil {
		return err
	}
	err = cmd.StreamJSONResponse(ctx.Stdout, response)

	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Volume successfully created.\n")
	return nil
}

type VolumeUpdate struct {
	fs   *gnuflag.FlagSet
	pool string
	team string
	opt  cmd.MapFlag
}

func (c *VolumeUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-update",
		Usage:   "volume update <volume name> <plan-name> [-p/--pool <pool>] [-t/--team <team>] [-o/--opt key=value]...",
		Desc:    `Update an existing persistent volume.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *VolumeUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("volume-update", gnuflag.ExitOnError)
		desc := "the pool that owns the service (mandatory if the user has access to more than one pool)"
		c.fs.StringVar(&c.pool, "pool", "", desc)
		c.fs.StringVar(&c.pool, "p", "", desc)
		desc = "the team that owns the service (mandatory if the user has access to more than one team)"
		c.fs.StringVar(&c.team, "team", "", desc)
		c.fs.StringVar(&c.team, "t", "", desc)
		desc = "backend specific volume options"
		c.fs.Var(&c.opt, "opt", desc)
		c.fs.Var(&c.opt, "o", desc)
	}
	return c.fs
}
func (c *VolumeUpdate) volumeUpdate(volumeName, planName string) tsuru.VolumeUpdateData {
	volumeUpdate := tsuru.VolumeUpdateData{
		Name:      volumeName,
		Plan:      tsuru.VolumePlan{Name: planName},
		Pool:      c.pool,
		TeamOwner: c.team,
		Opts:      map[string]string(c.opt),
	}
	return volumeUpdate
}

func (c *VolumeUpdate) Run(ctx *cmd.Context, client *cmd.Client) error {
	volumeName, planName := ctx.Args[0], ctx.Args[1]

	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient,
	})
	if err != nil {
		return err
	}
	volumeUpdate := c.volumeUpdate(volumeName, planName)
	_, err = apiClient.VolumeApi.VolumeUpdate(context.TODO(), volumeName, volumeUpdate)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Volume successfully updated.\n")
	return nil
}

type VolumeList struct{}

func (c *VolumeList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-list",
		Usage:   "volume list",
		Desc:    `Lists existing persistent volumes.`,
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *VolumeList) Run(ctx *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURLVersion("1.4", "/volumes")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	rsp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode == http.StatusNoContent {
		fmt.Fprintln(ctx.Stdout, "No volumes available.")
		return nil
	}
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	var volumes []volumeTypes.Volume
	err = json.Unmarshal(data, &volumes)
	if err != nil {
		return err
	}
	return c.render(ctx, volumes)
}

func (c *VolumeList) render(ctx *cmd.Context, volumes []volumeTypes.Volume) error {
	tbl := tablecli.NewTable()
	tbl.Headers = tablecli.Row{"Name", "Plan", "Pool", "Team"}
	tbl.LineSeparator = true
	for _, v := range volumes {
		tbl.AddRow(tablecli.Row{
			v.Name,
			v.Plan.Name,
			v.Pool,
			v.TeamOwner,
		})
	}
	tbl.Sort()
	fmt.Fprint(ctx.Stdout, tbl.String())
	return nil
}

type VolumeInfo struct{}

func (c *VolumeInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-info",
		Usage:   "volume info <volume>",
		Desc:    `Get a volume.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *VolumeInfo) Run(ctx *cmd.Context, client *cmd.Client) error {
	volumeName := ctx.Args[0]
	u, err := cmd.GetURLVersion("1.4", "/volumes/"+volumeName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	rsp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode == http.StatusNoContent {
		fmt.Fprintln(ctx.Stdout, "No volumes available.")
		return nil
	}
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	var volume volumeTypes.Volume
	err = json.Unmarshal(data, &volume)
	if err != nil {
		return err
	}
	return c.render(ctx, volume)
}

func (c *VolumeInfo) render(ctx *cmd.Context, volume volumeTypes.Volume) error {
	fmt.Fprintf(ctx.Stdout, "Name: %s\nPlan: %s\nPool: %s\nTeam: %s\n",
		volume.Name,
		volume.Plan.Name,
		volume.Pool,
		volume.TeamOwner,
	)
	bindTable := tablecli.NewTable()
	bindTable.Headers = tablecli.Row([]string{"App", "MountPoint", "Mode"})
	bindTable.LineSeparator = true
	for _, b := range volume.Binds {
		mode := "rw"
		if b.ReadOnly {
			mode = "ro"
		}
		bindTable.AddRow(tablecli.Row([]string{b.ID.App, b.ID.MountPoint, mode}))
	}
	fmt.Fprintf(ctx.Stdout, "\nBinds:\n")
	fmt.Fprint(ctx.Stdout, bindTable.String())
	planOptsTable := tablecli.NewTable()
	planOptsTable.Headers = []string{"Key", "Value"}
	planOptsTable.LineSeparator = true
	for k, v := range volume.Plan.Opts {
		planOptsTable.AddRow([]string{k, fmt.Sprintf("%v", v)})
	}
	planOptsTable.Sort()
	fmt.Fprint(ctx.Stdout, "\nPlan Opts:\n")
	fmt.Fprint(ctx.Stdout, planOptsTable.String())
	optsTable := tablecli.NewTable()
	optsTable.Headers = []string{"Key", "Value"}
	optsTable.LineSeparator = true
	for k, v := range volume.Opts {
		optsTable.AddRow([]string{k, fmt.Sprintf("%v", v)})
	}
	optsTable.Sort()
	fmt.Fprintf(ctx.Stdout, "\nOpts:\n")
	fmt.Fprint(ctx.Stdout, optsTable.String())
	return nil
}

type VolumePlansList struct{}

func (c *VolumePlansList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-plan-list",
		Usage:   "volume plan list",
		Desc:    `Lists existing volume plans.`,
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *VolumePlansList) Run(ctx *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURLVersion("1.4", "/volumeplans")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	rsp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	var plans map[string][]volumeTypes.VolumePlan
	if rsp.StatusCode != http.StatusNoContent {
		data, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, &plans)
		if err != nil {
			return err
		}
	}
	return c.render(ctx, plans)
}

func (c *VolumePlansList) render(ctx *cmd.Context, plans map[string][]volumeTypes.VolumePlan) error {
	tbl := tablecli.NewTable()
	tbl.Headers = tablecli.Row{"Plan", "Provisioner", "Opts"}
	tbl.LineSeparator = true
	for provisioner, provPlans := range plans {
		for _, p := range provPlans {
			var opts []string
			for k, v := range p.Opts {
				opts = append(opts, fmt.Sprintf("%s: %v", k, v))
			}
			sort.Strings(opts)
			tbl.AddRow(tablecli.Row{
				p.Name,
				provisioner,
				strings.Join(opts, "\n"),
			})
		}
	}
	tbl.SortByColumn(0, 1)
	fmt.Fprint(ctx.Stdout, tbl.String())
	return nil
}

type VolumeDelete struct{}

func (c *VolumeDelete) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-delete",
		Usage:   "volume delete <volume-name>",
		Desc:    `Delete an existing persistent volume.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *VolumeDelete) Run(ctx *cmd.Context, client *cmd.Client) error {
	volumeName := ctx.Args[0]
	u, err := cmd.GetURLVersion("1.4", "/volumes/"+volumeName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Volume successfully deleted.\n")
	return nil
}

type VolumeBind struct {
	cmd.AppNameMixIn
	fs        *gnuflag.FlagSet
	readOnly  bool
	noRestart bool
}

func (c *VolumeBind) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-bind",
		Usage:   "volume bind <volume-name> <mount point> [-a/--app <appname>] [-r/--readonly] [--no-restart]",
		Desc:    `Binds an existing volume to an application.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *VolumeBind) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		desc := "the volume will be available only for reading"
		c.fs.BoolVar(&c.readOnly, "readonly", false, desc)
		c.fs.BoolVar(&c.readOnly, "r", false, desc)
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "prevents restarting the application")
	}
	return c.fs
}
func (c *VolumeBind) volumeBind(appName string, ctx *cmd.Context) tsuru.VolumeBindData {
	bind := tsuru.VolumeBindData{
		App:        appName,
		Mountpoint: ctx.Args[1],
		Readonly:   c.readOnly,
		Norestart:  c.noRestart,
	}
	return bind
}
func (c *VolumeBind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	volumeName := ctx.Args[0]
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	bind := c.volumeBind(appName, ctx)
	apiClient, err := tsuruClient.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: client.HTTPClient})
	if err != nil {
		return err
	}
	response, err := apiClient.VolumeApi.VolumeBind(context.TODO(), volumeName, bind)
	if err != nil {
		return err
	}
	err = cmd.StreamJSONResponse(ctx.Stdout, response)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Volume successfully bound.\n")
	return nil
}

type VolumeUnbind struct {
	cmd.AppNameMixIn
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (c *VolumeUnbind) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-unbind",
		Usage:   "volume unbind <volume-name> <mount point> [-a/--app <appname>]",
		Desc:    `Unbinds a volume from an application.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *VolumeUnbind) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "prevents restarting the application")
	}
	return c.fs
}

func (c *VolumeUnbind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	volumeName := ctx.Args[0]
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	bind := struct {
		App        string
		MountPoint string
		NoRestart  bool
	}{
		App:        appName,
		MountPoint: ctx.Args[1],
		NoRestart:  c.noRestart,
	}
	val, err := form.EncodeToValues(bind)
	if err != nil {
		return err
	}
	u, err := cmd.GetURLVersion("1.4", fmt.Sprintf("/volumes/%s/bind?%s", volumeName, val.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	err = cmd.StreamJSONResponse(ctx.Stdout, resp)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Stdout, "Volume successfully unbound.\n")
	return nil
}
