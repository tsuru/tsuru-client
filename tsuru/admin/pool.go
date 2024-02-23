// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/config"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/provision/pool"
)

type AddPoolToSchedulerCmd struct {
	public       bool
	defaultPool  bool
	forceDefault bool
	provisioner  string
	labels       cmd.MapFlag
	fs           *gnuflag.FlagSet
}

func (AddPoolToSchedulerCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "pool-add",
		Usage: "pool-add <pool> [-p/--public] [-d/--default] [--provisioner <name>] [-f/--force] [-l/--labels \"{\"key\":\"value\"}\"]",
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
		msg = "LabelSet that integrates with kubernetes, i.e could be used to define a podAffinity rule for the pool"
		c.fs.Var(&c.labels, "labels", msg)
	}
	return c.fs
}

type addOpts struct {
	Name         string            `json:"name"`
	Public       bool              `json:"public,omitempty"`
	DefaultPool  bool              `json:"default,omitempty"`
	ForceDefault bool              `json:"force,omitempty"`
	Provisioner  string            `json:"provisioner,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

func (c *AddPoolToSchedulerCmd) marshalAddPoolOpts(poolName string) ([]byte, error) {
	opts := addOpts{
		Name:         poolName,
		Public:       c.public,
		DefaultPool:  c.defaultPool,
		ForceDefault: c.forceDefault,
		Provisioner:  c.provisioner,
		Labels:       c.labels,
	}

	return json.Marshal(opts)
}

func (c *AddPoolToSchedulerCmd) Run(ctx *cmd.Context) error {
	poolName := ctx.Args[0]
	body, err := c.marshalAddPoolOpts(poolName)
	if err != nil {
		return err
	}
	u, err := config.GetURL("/pools")
	if err != nil {
		return err
	}
	err = doRequest(u, "POST", body)
	if err != nil {
		err = tsuruHTTP.UnwrapErr(err)
		if e, ok := err.(*errors.HTTP); ok && e.Code == http.StatusPreconditionFailed {
			retryMessage := "WARNING: Default pool already exist. Do you want change to %s pool? (y/n) "
			c.forceDefault = true
			body, err = c.marshalAddPoolOpts(poolName)
			if err != nil {
				return err
			}
			url, _ := config.GetURL("/pools")
			successMessage := "Pool successfully registered.\n"
			failMessage := "Pool add aborted.\n"
			return confirmAction(ctx, url, "POST", body, retryMessage, failMessage, successMessage)
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
	labelsAdd    cmd.MapFlag
	labelsRemove cmd.StringSliceFlag
	fs           *gnuflag.FlagSet
}

func (UpdatePoolToSchedulerCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "pool-update",
		Usage:   "pool-update <pool> [--public=true/false] [--default=true/false] [-f/--force] [--add-labels key=value]... [--remove-labels key]...",
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
		c.fs.Var(&c.labelsAdd, "add-labels", "group of key/value pairs that specify a kubernetes object label, this option adds the specified labels to the pool")
		c.fs.Var(&c.labelsRemove, "remove-labels", "group of keys from a kubernetes object label, this option removes the specified labels from the pool")
	}
	return c.fs
}

type updateOpts struct {
	Public       *bool `json:"public,omitempty"`
	DefaultPool  *bool `json:"default,omitempty"`
	ForceDefault bool  `json:"force,omitempty"`

	Labels map[string]string `json:"labels"`
}

func removeKeys(m map[string]string, toRemove []string) (map[string]string, error) {
	for _, k := range toRemove {
		if _, ok := m[k]; !ok {
			return nil, &errors.ValidationError{Message: fmt.Sprintf("key %s does not exist in pool labelset, can't delete an unexisting key", k)}
		}
		delete(m, k)
	}

	return m, nil
}

func (c *UpdatePoolToSchedulerCmd) checkLabels(poolName string) (map[string]string, error) {
	api, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return nil, err
	}

	pool, _, err := api.PoolApi.PoolGet(context.TODO(), poolName)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string)
	if len(c.labelsRemove) > 0 {
		labels, err = removeKeys(pool.Labels, c.labelsRemove)
		if err != nil {
			return nil, err
		}
	}

	for k, v := range c.labelsAdd {
		labels[k] = v
	}

	return labels, nil
}

func (c *UpdatePoolToSchedulerCmd) marshalUpdateOpts(poolName string) ([]byte, error) {
	var labels map[string]string
	var err error
	if len(c.labelsAdd) > 0 || len(c.labelsRemove) > 0 {
		labels, err = c.checkLabels(poolName)
		if err != nil {
			return nil, err
		}
	}
	opts := updateOpts{
		Public:       c.public.value,
		DefaultPool:  c.defaultPool.value,
		ForceDefault: c.forceDefault,
		Labels:       labels,
	}
	return json.Marshal(opts)
}

func (c *UpdatePoolToSchedulerCmd) Run(ctx *cmd.Context) error {
	poolName := ctx.Args[0]
	body, err := c.marshalUpdateOpts(poolName)
	if err != nil {
		return err
	}

	u, err := config.GetURL(fmt.Sprintf("/pools/%s", poolName))
	if err != nil {
		return err
	}
	err = doRequest(u, "PUT", body)
	if err != nil {
		err = tsuruHTTP.UnwrapErr(err)
		if e, ok := err.(*errors.HTTP); ok && e.Code == http.StatusPreconditionFailed {
			retryMessage := "WARNING: Default pool already exist. Do you want change to %s pool? (y/n) "
			failMessage := "Pool update aborted.\n"
			successMessage := "Pool successfully updated.\n"
			c.forceDefault = true
			body, err = c.marshalUpdateOpts(poolName)
			if err != nil {
				return err
			}

			u, err = config.GetURL(fmt.Sprintf("/pools/%s", poolName))
			if err != nil {
				return err
			}
			return confirmAction(ctx, u, "PUT", body, retryMessage, failMessage, successMessage)
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

func (c *RemovePoolFromSchedulerCmd) Run(ctx *cmd.Context) error {
	if !c.Confirm(ctx, fmt.Sprintf("Are you sure you want to remove \"%s\" pool?", ctx.Args[0])) {
		return nil
	}
	url, err := config.GetURL(fmt.Sprintf("/pools/%s", ctx.Args[0]))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = tsuruHTTP.AuthenticatedClient.Do(req)
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

func (AddTeamsToPoolCmd) Run(ctx *cmd.Context) error {
	v := url.Values{}
	for _, team := range ctx.Args[1:] {
		v.Add("team", team)
	}
	u, err := config.GetURL(fmt.Sprintf("/pools/%s/team", ctx.Args[0]))
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = tsuruHTTP.AuthenticatedClient.Do(req)
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

func (RemoveTeamsFromPoolCmd) Run(ctx *cmd.Context) error {
	v := url.Values{}
	for _, team := range ctx.Args[1:] {
		v.Add("team", team)
	}
	u, err := config.GetURL(fmt.Sprintf("/pools/%s/team?%s", ctx.Args[0], v.Encode()))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	_, err = tsuruHTTP.AuthenticatedClient.Do(req)
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

func doRequest(url, method string, body []byte) error {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	_, err = tsuruHTTP.AuthenticatedClient.Do(req)
	return err
}

func confirmAction(ctx *cmd.Context, url, method string, body []byte, retryMessage, failMessage, successMessage string) error {
	var answer string
	fmt.Fprintf(ctx.Stdout, retryMessage, ctx.Args[0])
	fmt.Fscanf(ctx.Stdin, "%s", &answer)
	if answer == "y" || answer == "yes" {
		err := doRequest(url, method, body)
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

func (c *PoolConstraintList) Run(ctx *cmd.Context) error {
	url, err := config.GetURLVersion("1.3", "/constraints")
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := tsuruHTTP.AuthenticatedClient.Do(req)
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

func (c *PoolConstraintSet) Run(ctx *cmd.Context) error {
	u, err := config.GetURLVersion("1.3", "/constraints")
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

	body, err := json.Marshal(constraint)
	if err != nil {
		return err
	}

	if c.append {
		u = fmt.Sprintf("%s?append=true", u)
	}
	err = doRequest(u, http.MethodPut, body)
	if err != nil {
		return err
	}
	ctx.Stdout.Write([]byte("Constraint successfully set.\n"))
	return nil
}
