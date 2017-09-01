// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
	"github.com/tsuru/tsuru/volume"
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
		Usage:   "volume-create <volume-name> <plan-name> [-p/--pool <pool>] [-t/--team <team>] [-o/--opt key=value]...",
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

func (c *VolumeCreate) Run(ctx *cmd.Context, client *cmd.Client) error {
	volumeName, planName := ctx.Args[0], ctx.Args[1]
	vol := volume.Volume{
		Name:      volumeName,
		Plan:      volume.VolumePlan{Name: planName},
		Pool:      c.pool,
		TeamOwner: c.team,
		Opts:      map[string]string(c.opt),
	}
	val, err := form.EncodeToValues(vol)
	if err != nil {
		return err
	}
	body := strings.NewReader(val.Encode())
	u, err := cmd.GetURLVersion("1.4", "/volumes")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
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
		Usage:   "volume-update <volume-name> <plan-name> [-p/--pool <pool>] [-t/--team <team>] [-o/--opt key=value]...",
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

func (c *VolumeUpdate) Run(ctx *cmd.Context, client *cmd.Client) error {
	volumeName, planName := ctx.Args[0], ctx.Args[1]
	vol := volume.Volume{
		Name:      volumeName,
		Plan:      volume.VolumePlan{Name: planName},
		Pool:      c.pool,
		TeamOwner: c.team,
		Opts:      map[string]string(c.opt),
	}
	val, err := form.EncodeToValues(vol)
	if err != nil {
		return err
	}
	body := strings.NewReader(val.Encode())
	u, err := cmd.GetURLVersion("1.4", "/volumes/"+volumeName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
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
		Usage:   "volume-list",
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
	var volumes []volume.Volume
	err = json.Unmarshal(data, &volumes)
	if err != nil {
		return err
	}
	return c.render(ctx, volumes)
}

func (c *VolumeList) render(ctx *cmd.Context, volumes []volume.Volume) error {
	tbl := cmd.NewTable()
	tbl.Headers = cmd.Row{"Name", "Plan", "Pool", "Team", "Plan Opts", "Opts", "Binds"}
	tbl.LineSeparator = true
	for _, v := range volumes {
		var bindsStr []string
		for _, b := range v.Binds {
			mode := "rw"
			if b.ReadOnly {
				mode = "ro"
			}
			bindsStr = append(bindsStr, fmt.Sprintf("%s:%s:%s", b.ID.App, b.ID.MountPoint, mode))
		}
		var planOpts []string
		for k, v := range v.Plan.Opts {
			planOpts = append(planOpts, fmt.Sprintf("%s: %v", k, v))
		}
		sort.Strings(planOpts)
		var opts []string
		for k, v := range v.Opts {
			opts = append(opts, k+": "+v)
		}
		sort.Strings(opts)
		tbl.AddRow(cmd.Row{
			v.Name,
			v.Plan.Name,
			v.Pool,
			v.TeamOwner,
			strings.Join(planOpts, "\n"),
			strings.Join(opts, "\n"),
			strings.Join(bindsStr, "\n"),
		})
	}
	tbl.Sort()
	fmt.Fprint(ctx.Stdout, tbl.String())
	return nil
}

type VolumePlansList struct{}

func (c *VolumePlansList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-plan-list",
		Usage:   "volume-plan-list",
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
	var plans map[string][]volume.VolumePlan
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

func (c *VolumePlansList) render(ctx *cmd.Context, plans map[string][]volume.VolumePlan) error {
	tbl := cmd.NewTable()
	tbl.Headers = cmd.Row{"Plan", "Provisioner", "Opts"}
	tbl.LineSeparator = true
	for provisioner, provPlans := range plans {
		for _, p := range provPlans {
			var opts []string
			for k, v := range p.Opts {
				opts = append(opts, fmt.Sprintf("%s: %v", k, v))
			}
			sort.Strings(opts)
			tbl.AddRow(cmd.Row{
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
		Usage:   "volume-delete <volume-name>",
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
	cmd.GuessingCommand
	fs        *gnuflag.FlagSet
	readOnly  bool
	noRestart bool
}

func (c *VolumeBind) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-bind",
		Usage:   "volume-bind <volume-name> <mount point> [-a/--app <appname>] [-r/--readonly] [--no-restart]",
		Desc:    `Binds an existing volume to an application.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *VolumeBind) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		desc := "the volume will be available only for reading"
		c.fs.BoolVar(&c.readOnly, "readonly", false, desc)
		c.fs.BoolVar(&c.readOnly, "r", false, desc)
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "prevents restarting the application")
	}
	return c.fs
}

func (c *VolumeBind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	volumeName := ctx.Args[0]
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	bind := struct {
		App        string
		MountPoint string
		ReadOnly   bool
		NoRestart  bool
	}{
		App:        appName,
		MountPoint: ctx.Args[1],
		ReadOnly:   c.readOnly,
		NoRestart:  c.noRestart,
	}
	val, err := form.EncodeToValues(bind)
	if err != nil {
		return err
	}
	body := strings.NewReader(val.Encode())
	u, err := cmd.GetURLVersion("1.4", fmt.Sprintf("/volumes/%s/bind", volumeName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	w := tsuruIo.NewStreamWriter(ctx.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, resp.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	fmt.Fprint(ctx.Stdout, "Volume successfully bound.\n")
	return nil
}

type VolumeUnbind struct {
	cmd.GuessingCommand
	fs        *gnuflag.FlagSet
	noRestart bool
}

func (c *VolumeUnbind) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "volume-unbind",
		Usage:   "volume-unbind <volume-name> <mount point> [-a/--app <appname>]",
		Desc:    `Unbinds a volume from an application.`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (c *VolumeUnbind) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.BoolVar(&c.noRestart, "no-restart", false, "prevents restarting the application")
	}
	return c.fs
}

func (c *VolumeUnbind) Run(ctx *cmd.Context, client *cmd.Client) error {
	ctx.RawOutput()
	volumeName := ctx.Args[0]
	appName, err := c.Guess()
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
	defer resp.Body.Close()
	w := tsuruIo.NewStreamWriter(ctx.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, resp.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	fmt.Fprint(ctx.Stdout, "Volume successfully unbound.\n")
	return nil
}
