// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ajg/form"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/event"
)

type EventBlockList struct {
	fs     *gnuflag.FlagSet
	active bool
}

func (c *EventBlockList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-block-list",
		Usage:   "event-block-list [-a/--active]",
		Desc:    "Lists all event blocks",
		MinArgs: 0,
	}
}

func (c *EventBlockList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.active, "active", false, "Display only active blocks.")
		c.fs.BoolVar(&c.active, "a", false, "Display only active blocks.")
	}
	return c.fs
}

func (c *EventBlockList) Run(context *cmd.Context, client *cmd.Client) error {
	path := "/events/blocks"
	if c.active {
		path += "?active=true"
	}
	url, err := cmd.GetURLVersion("1.3", path)
	if err != nil {
		return err
	}
	request, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(request)
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
	tbl := cmd.NewTable()
	tbl.Headers = cmd.Row{"ID", "Start (duration)", "Kind", "Owner", "Target (Type: Value)", "Reason"}
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
		row := cmd.Row{b.ID.Hex(), ts, kind, owner, fmt.Sprintf("%s: %s", targetType, targetValue), b.Reason}
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

type EventBlockAdd struct {
	fs          *gnuflag.FlagSet
	kind        string
	owner       string
	targetType  string
	targetValue string
}

func (c *EventBlockAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-block-add",
		Usage:   "event-block-add <reason> [-k/--kind kindName] [-o/--owner ownerName] [-t/--target targetType] [-v/--value targetValue]",
		Desc:    "Block events.",
		MinArgs: 1,
	}
}

func (c *EventBlockAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.fs.StringVar(&c.kind, "kind", "", "Event kind to be blocked.")
		c.fs.StringVar(&c.kind, "k", "", "Event kind to be blocked.")
		c.fs.StringVar(&c.owner, "owner", "", "Block this owner's events.")
		c.fs.StringVar(&c.owner, "o", "", "Block this owner's events.")
		c.fs.StringVar(&c.targetType, "target", "", "Block events with this target type.")
		c.fs.StringVar(&c.targetType, "t", "", "Block events with this target type.")
		c.fs.StringVar(&c.targetValue, "value", "", "Block events with this target value.")
		c.fs.StringVar(&c.targetValue, "v", "", "Block events with this target value.")
	}
	return c.fs
}

func (c *EventBlockAdd) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURLVersion("1.3", "/events/blocks")
	if err != nil {
		return err
	}
	target := event.Target{}
	if c.targetType != "" {
		var targetType event.TargetType
		targetType, err = event.GetTargetType(c.targetType)
		if err != nil {
			return err
		}
		target.Type = targetType
	}
	target.Value = c.targetValue
	block := event.Block{
		Reason:    context.Args[0],
		KindName:  c.kind,
		OwnerName: c.owner,
		Target:    target,
	}
	v, err := form.EncodeToValues(&block)
	if err != nil {
		return err
	}
	err = doRequest(client, url, http.MethodPost, v.Encode())
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
		Usage:   "event-block-remove <ID>",
		Desc:    "Removes an event block.",
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *EventBlockRemove) Run(context *cmd.Context, client *cmd.Client) error {
	uuid := context.Args[0]
	url, err := cmd.GetURLVersion("1.3", fmt.Sprintf("/events/blocks/%s", uuid))
	if err != nil {
		return err
	}
	request, _ := http.NewRequest(http.MethodDelete, url, nil)
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	context.Stdout.Write([]byte(fmt.Sprintf("Block %s successfully removed.\n", uuid)))
	return nil
}
