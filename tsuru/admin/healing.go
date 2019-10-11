// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/provision/docker/types"
)

type ListHealingHistoryCmd struct {
	fs            *gnuflag.FlagSet
	nodeOnly      bool
	containerOnly bool
}

func (c *ListHealingHistoryCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "healing-list",
		Usage: "healing-list [--node] [--container]",
		Desc:  "List healing history for nodes or containers.",
	}
}

func renderHistoryTable(history []types.HealingEvent, filter string, ctx *cmd.Context) {
	fmt.Fprintln(ctx.Stdout, strings.ToUpper(filter[:1])+filter[1:]+":")
	headers := tablecli.Row([]string{"Start", "Finish", "Success", "Failing", "Created", "Error"})
	t := tablecli.Table{Headers: headers}
	for i := range history {
		event := history[i]
		if event.Action != filter+"-healing" {
			continue
		}
		data := make([]string, 2)
		if filter == "node" {
			data[0] = event.FailingNode.Address
			data[1] = event.CreatedNode.Address
		} else {
			data[0] = event.FailingContainer.ID
			data[1] = event.CreatedContainer.ID
			if len(data[0]) > 10 {
				data[0] = data[0][:10]
			}
			if len(data[1]) > 10 {
				data[1] = data[1][:10]
			}
		}
		var endTime string
		if event.EndTime.IsZero() {
			endTime = "in progress"
		} else {
			endTime = formatter.FormatStamp(event.EndTime)
		}
		t.AddRow(tablecli.Row([]string{
			formatter.FormatStamp(event.StartTime),
			endTime,
			fmt.Sprintf("%t", event.Successful),
			data[0],
			data[1],
			event.Error,
		}))
	}
	t.LineSeparator = true
	ctx.Stdout.Write(t.Bytes())
}

func (c *ListHealingHistoryCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	var filter string
	if c.nodeOnly && !c.containerOnly {
		filter = "node"
	}
	if c.containerOnly && !c.nodeOnly {
		filter = "container"
	}
	url, err := cmd.GetURLVersion("1.3", fmt.Sprintf("/healing?filter=%s", filter))
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var history []types.HealingEvent
	if resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&history)
		if err != nil {
			return err
		}
	}
	if filter != "" {
		renderHistoryTable(history, filter, ctx)
	} else {
		renderHistoryTable(history, "node", ctx)
		renderHistoryTable(history, "container", ctx)
	}
	return nil
}

func (c *ListHealingHistoryCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ContinueOnError)
		c.fs.BoolVar(&c.nodeOnly, "node", false, "List only healing process started for nodes")
		c.fs.BoolVar(&c.containerOnly, "container", false, "List only healing process started for containers")
	}
	return c.fs
}
