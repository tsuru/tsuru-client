// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/provision/pool"
)

type AddPoolToSchedulerCmd struct {
	public       bool
	defaultPool  bool
	forceDefault bool
	provisioner  string
	fs           *gnuflag.FlagSet
}

func (AddPoolToSchedulerCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "pool-add",
		Usage: "pool-add <pool> [-p/--public] [-d/--default] [--provisioner <name>] [-f/--force]",
		Desc: `Adds a new pool.

Each docker node added using [[node-add]] command belongs to one pool.
Also, when creating a new application a pool must be chosen and this means
that all units of the created application will be spawned in nodes belonging
to the chosen pool.`,
		MinArgs: 1,
	}
}

func (c *AddPoolToSchedulerCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		msg := "Make pool public (all teams can use it)"
		c.fs.BoolVar(&c.public, "public", false, msg)
		c.fs.BoolVar(&c.public, "p", false, msg)
		msg = "Make pool default (when none is specified during [[app-create]] this pool will be used)"
		c.fs.BoolVar(&c.defaultPool, "default", false, msg)
		c.fs.BoolVar(&c.defaultPool, "d", false, msg)
		msg = "Force overwrite default pool"
		c.fs.BoolVar(&c.forceDefault, "force", false, msg)
		c.fs.BoolVar(&c.forceDefault, "f", false, msg)
		msg = "Provisioner associated to the pool (empty for default docker provisioner)"
		c.fs.StringVar(&c.provisioner, "provisioner", "", msg)
	}
	return c.fs
}

func (c *AddPoolToSchedulerCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	v := url.Values{}
	v.Set("name", ctx.Args[0])
	v.Set("public", strconv.FormatBool(c.public))
	v.Set("default", strconv.FormatBool(c.defaultPool))
	v.Set("force", strconv.FormatBool(c.forceDefault))
	v.Set("provisioner", c.provisioner)
	u, err := cmd.GetURL("/pools")
	if err != nil {
		return err
	}
	err = doRequest(client, u, "POST", v.Encode())
	if err != nil {
		if e, ok := err.(*errors.HTTP); ok && e.Code == http.StatusPreconditionFailed {
			retryMessage := "WARNING: Default pool already exist. Do you want change to %s pool? (y/n) "
			v.Set("force", "true")
			url, _ := cmd.GetURL("/pools")
			successMessage := "Pool successfully registered.\n"
			failMessage := "Pool add aborted.\n"
			return confirmAction(ctx, client, url, "POST", v.Encode(), retryMessage, failMessage, successMessage)
		}
		return err
	}
	ctx.Stdout.Write([]byte("Pool successfully registered.\n"))
	return nil
}

type UpdatePoolToSchedulerCmd struct {
	public       pointerBoolFlag
	defaultPool  pointerBoolFlag
	forceDefault bool
	fs           *gnuflag.FlagSet
}

func (UpdatePoolToSchedulerCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "pool-update",
		Usage:   "pool-update <pool> [--public=true/false] [--default=true/false] [-f/--force]",
		Desc:    `Updates attributes for a pool.`,
		MinArgs: 1,
	}
}

func (c *UpdatePoolToSchedulerCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		msg := "Make pool public (all teams can use it)"
		c.fs.Var(&c.public, "public", msg)
		msg = "Make pool default (when none is specified during [[app-create]] this pool will be used)"
		c.fs.Var(&c.defaultPool, "default", msg)
		c.fs.BoolVar(&c.forceDefault, "force", false, "Force pool to be default.")
		c.fs.BoolVar(&c.forceDefault, "f", false, "Force pool to be default.")
	}
	return c.fs
}

func (c *UpdatePoolToSchedulerCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	v := url.Values{}
	if c.public.value == nil {
		v.Set("public", "")
	} else {
		v.Set("public", strconv.FormatBool(*c.public.value))
	}
	if c.defaultPool.value == nil {
		v.Set("default", "")
	} else {
		v.Set("default", strconv.FormatBool(*c.defaultPool.value))
	}
	v.Set("force", strconv.FormatBool(c.forceDefault))
	u, err := cmd.GetURL(fmt.Sprintf("/pools/%s", ctx.Args[0]))
	if err != nil {
		return err
	}
	err = doRequest(client, u, "PUT", v.Encode())
	if err != nil {
		if e, ok := err.(*errors.HTTP); ok && e.Code == http.StatusPreconditionFailed {
			retryMessage := "WARNING: Default pool already exist. Do you want change to %s pool? (y/n) "
			failMessage := "Pool update aborted.\n"
			successMessage := "Pool successfully updated.\n"
			v.Set("force", "true")
			u, err = cmd.GetURL(fmt.Sprintf("/pools/%s", ctx.Args[0]))
			if err != nil {
				return err
			}
			return confirmAction(ctx, client, u, "PUT", v.Encode(), retryMessage, failMessage, successMessage)
		}
		return err
	}
	ctx.Stdout.Write([]byte("Pool successfully updated.\n"))
	return nil
}

type RemovePoolFromSchedulerCmd struct {
	cmd.ConfirmationCommand
}

func (c *RemovePoolFromSchedulerCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "pool-remove",
		Usage:   "pool-remove <pool> [-y]",
		Desc:    "Remove an existing pool.",
		MinArgs: 1,
	}
}

func (c *RemovePoolFromSchedulerCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	if !c.Confirm(ctx, fmt.Sprintf("Are you sure you want to remove \"%s\" pool?", ctx.Args[0])) {
		return nil
	}
	url, err := cmd.GetURL(fmt.Sprintf("/pools/%s", ctx.Args[0]))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte("Pool successfully removed.\n"))
	return nil
}

type AddTeamsToPoolCmd struct{}

func (AddTeamsToPoolCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "pool-teams-add",
		Usage: "pool-teams-add <pool> <teams>...",
		Desc: `Adds teams to a pool. This will make the specified pool available when
creating a new application for one of the added teams.`,
		MinArgs: 2,
	}
}

func (AddTeamsToPoolCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	v := url.Values{}
	for _, team := range ctx.Args[1:] {
		v.Add("team", team)
	}
	u, err := cmd.GetURL(fmt.Sprintf("/pools/%s/team", ctx.Args[0]))
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte("Teams successfully registered.\n"))
	return nil
}

type RemoveTeamsFromPoolCmd struct{}

func (RemoveTeamsFromPoolCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "pool-teams-remove",
		Usage: "pool-teams-remove <pool> <teams>...",
		Desc: `Removes teams from a pool. Listed teams will be no longer able to use this
pool when creating a new application.`,
		MinArgs: 2,
	}
}

func (RemoveTeamsFromPoolCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	v := url.Values{}
	for _, team := range ctx.Args[1:] {
		v.Add("team", team)
	}
	u, err := cmd.GetURL(fmt.Sprintf("/pools/%s/team?%s", ctx.Args[0], v.Encode()))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte("Teams successfully removed.\n"))
	return nil
}

type pointerBoolFlag struct {
	value *bool
}

func (p *pointerBoolFlag) String() string {
	if p.value == nil {
		return "not set"
	}
	return fmt.Sprintf("%v", *p.value)
}

func (p *pointerBoolFlag) Set(value string) error {
	if value == "" {
		return nil
	}
	v, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	p.value = &v
	return nil
}

func doRequest(client *cmd.Client, url, method, body string) error {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(req)
	return err
}

func confirmAction(ctx *cmd.Context, client *cmd.Client, url, method, body string, retryMessage, failMessage, successMessage string) error {
	var answer string
	fmt.Fprintf(ctx.Stdout, retryMessage, ctx.Args[0])
	fmt.Fscanf(ctx.Stdin, "%s", &answer)
	if answer == "y" || answer == "yes" {
		err := doRequest(client, url, method, body)
		if err != nil {
			return err
		}
		ctx.Stdout.Write([]byte(successMessage))
		return nil

	}
	ctx.Stdout.Write([]byte(failMessage))
	return nil
}

type PoolConstraintList struct{}

func (c *PoolConstraintList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "pool-constraint-list",
		Usage: "pool-constraint-list",
		Desc:  "List pool constraints.",
	}
}

func (c *PoolConstraintList) Run(ctx *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURLVersion("1.3", "/constraints")
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var constraints []pool.PoolConstraint
	err = json.NewDecoder(resp.Body).Decode(&constraints)
	if err != nil {
		return err
	}
	tbl := tablecli.NewTable()
	tbl.Headers = tablecli.Row{"Pool Expression", "Field", "Values", "Blacklist"}
	for _, c := range constraints {
		tbl.AddRow(tablecli.Row{c.PoolExpr, string(c.Field), strings.Join(c.Values, ","), strconv.FormatBool(c.Blacklist)})
	}
	tbl.SortByColumn(0)
	ctx.Stdout.Write([]byte(tbl.String()))
	return nil
}

type PoolConstraintSet struct {
	append    bool
	blacklist bool
	fs        *gnuflag.FlagSet
}

func (c *PoolConstraintSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.append, "append", false, "Append to existing constraint.")
		c.fs.BoolVar(&c.append, "a", false, "Append to existing constraint.")
		c.fs.BoolVar(&c.blacklist, "b", false, "Blacklist constraint.")
		c.fs.BoolVar(&c.blacklist, "blacklist", false, "Blacklist constraint.")
	}
	return c.fs
}
func (c *PoolConstraintSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "pool-constraint-set",
		Usage: "pool-constraint-set <poolExpression> <field> [<values>]... [-b/--blacklist] [-a/--append]",
		Desc: `Set a constraint on a pool expression.

Examples:

[[tsuru pool-constraint-set dev_pool team "*" # allows every team to use the pool "dev_pool" ]]
[[tsuru pool-constraint-set "dev_*" router prod_router --blacklist # disallows "prod_router" to be used on every pool with "dev_" prefix ]]
[[tsuru pool-constraint-set prod_pool team team2 team3 --append # adds "team2" and "team3" to the list of teams allowed to use pool "prod_pool"]]
[[tsuru pool-constraint-set prod_pool service service1 service2 --append # adds "service1" and "service2" to the list of services allowed to be used on pool "prod_pool"]]
[[tsuru pool-constraint-set prod_pool service service1 --blacklist # disallows "service1" to be used on pool "prod_pool"]]`,
		MinArgs: 2,
	}
}

func (c *PoolConstraintSet) Run(ctx *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURLVersion("1.3", "/constraints")
	if err != nil {
		return err
	}
	values := ctx.Args[2:]
	var allValues []string
	for _, v := range values {
		allValues = append(allValues, strings.Split(v, ",")...)
	}
	constraintType, err := pool.ToConstraintType(ctx.Args[1])
	if err != nil {
		return err
	}
	constraint := pool.PoolConstraint{
		PoolExpr:  ctx.Args[0],
		Field:     constraintType,
		Blacklist: c.blacklist,
		Values:    allValues,
	}
	v, err := form.EncodeToValues(constraint)
	if err != nil {
		return err
	}
	v.Set("append", strconv.FormatBool(c.append))
	err = doRequest(client, u, http.MethodPut, v.Encode())
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte("Constraint successfully set.\n"))
	return nil
}
