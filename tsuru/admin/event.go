// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/event"
	eventTypes "github.com/tsuru/tsuru/types/event"
)

type EventBlockList struct {
	fs     *pflag.FlagSet
	active bool
}

func (c *EventBlockList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-block-list",
		Usage:   "event block list [-a/--active]",
		Desc:    "Lists all event blocks",
		MinArgs: 0,
	}
}

func (c *EventBlockList) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)
		c.fs.BoolVar(&c.active, "active", false, "Display only active blocks.")
		c.fs.BoolVar(&c.active, "a", false, "Display only active blocks.")
	}
	return c.fs
}

func (c *EventBlockList) Run(context *cmd.Context) error {
	path := "/events/blocks"
	if c.active {
		path += "?active=true"
	}
	url, err := config.GetURLVersion("1.3", path)
	if err != nil {
		return err
	}
	request, _ := http.NewRequest("GET", url, nil)
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var blocks []event.Block
	if resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&blocks)
		if err != nil {
			return err
		}
	}
	tbl := tablecli.NewTable()
	tbl.Headers = tablecli.Row{"ID", "Start (duration)", "Kind", "Owner", "Target (Type: Value)", "Conditions", "Reason"}
	for _, b := range blocks {
		var duration *time.Duration
		if !b.EndTime.IsZero() {
			timeDiff := b.EndTime.Sub(b.StartTime)
			duration = &timeDiff
		}
		ts := formatter.FormatDateAndDuration(b.StartTime, duration)
		kind := valueOrWildcard(b.KindName)
		owner := valueOrWildcard(b.OwnerName)
		targetType := valueOrWildcard(string(b.Target.Type))
		targetValue := valueOrWildcard(b.Target.Value)
		conditions := mapValueOrWildcard(b.Conditions)
		row := tablecli.Row{b.ID.Hex(), ts, kind, owner, fmt.Sprintf("%s: %s", targetType, targetValue), conditions, b.Reason}
		color := "yellow"
		if !b.Active {
			color = "white"
		}
		for i, v := range row {
			row[i] = cmd.Colorfy(v, color, "", "")
		}
		tbl.AddRow(row)
	}
	context.Stdout.Write([]byte(tbl.String()))
	return nil
}

func valueOrWildcard(str string) string {
	if str == "" {
		return "all"
	}
	return str
}

func mapValueOrWildcard(m map[string]string) string {
	if len(m) == 0 {
		return "all"
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	var s string
	for _, k := range keys {
		s += fmt.Sprintf("%s=%s ", k, m[k])
	}

	return s[0 : len(s)-1] // trim last space
}

type EventBlockAdd struct {
	fs          *pflag.FlagSet
	kind        string
	owner       string
	targetType  string
	targetValue string
	conditions  cmd.MapFlag
}

func (c *EventBlockAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-block-add",
		Usage:   "event block add <reason> [-k/--kind kindName] [-o/--owner ownerName] [-t/--target targetType] [-v/--value targetValue] [-c/--conditions name=value]...",
		Desc:    "Block events.",
		MinArgs: 1,
	}
}

func (c *EventBlockAdd) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)
		c.fs.StringVar(&c.kind, "kind", "", "Event kind to be blocked.")
		c.fs.StringVar(&c.kind, "k", "", "Event kind to be blocked.")
		c.fs.StringVar(&c.owner, "owner", "", "Block this owner's events.")
		c.fs.StringVar(&c.owner, "o", "", "Block this owner's events.")
		c.fs.StringVar(&c.targetType, "target", "", "Block events with this target type.")
		c.fs.StringVar(&c.targetType, "t", "", "Block events with this target type.")
		c.fs.StringVar(&c.targetValue, "value", "", "Block events with this target value.")
		c.fs.StringVar(&c.targetValue, "v", "", "Block events with this target value.")
		c.fs.Var(&c.conditions, "conditions", "Conditions to apply on event kind to be blocked.")
		c.fs.Var(&c.conditions, "c", "Conditions to apply on event kind to be blocked.")
	}
	return c.fs
}

func (c *EventBlockAdd) Run(context *cmd.Context) error {
	url, err := config.GetURLVersion("1.3", "/events/blocks")
	if err != nil {
		return err
	}
	target := eventTypes.Target{}
	if c.targetType != "" {
		var targetType eventTypes.TargetType
		targetType, err = eventTypes.GetTargetType(c.targetType)
		if err != nil {
			return err
		}
		target.Type = targetType
	}
	target.Value = c.targetValue
	block := event.Block{
		Reason:     context.Args[0],
		KindName:   c.kind,
		OwnerName:  c.owner,
		Target:     target,
		Conditions: c.conditions,
	}

	body, err := json.Marshal(block)
	if err != nil {
		return err
	}
	err = doRequest(url, http.MethodPost, body)
	if err != nil {
		return err
	}
	context.Stdout.Write([]byte("Block successfully added.\n"))
	return nil
}

type EventBlockRemove struct{}

func (c *EventBlockRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-block-remove",
		Usage:   "event block remove <ID>",
		Desc:    "Removes an event block.",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *EventBlockRemove) Run(context *cmd.Context) error {
	uuid := context.Args[0]
	url, err := config.GetURLVersion("1.3", fmt.Sprintf("/events/blocks/%s", uuid))
	if err != nil {
		return err
	}
	request, _ := http.NewRequest(http.MethodDelete, url, nil)
	_, err = tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Block %s successfully removed.\n", uuid)
	return nil
}
