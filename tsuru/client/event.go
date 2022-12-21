// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ajg/form"
	"github.com/ghodss/yaml"
	"github.com/iancoleman/orderedmap"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/event"
)

type EventList struct {
	fs     *gnuflag.FlagSet
	filter eventFilter
	json   bool
}

type eventFilter struct {
	filter    event.Filter
	kindNames cmd.StringSliceFlag
	running   bool
}

func (f *eventFilter) queryString(client *cmd.Client) (url.Values, error) {
	if f.running {
		f.filter.Running = &f.running
	}
	values, err := form.EncodeToValues(f.filter)
	if err != nil {
		return nil, err
	}
	for k, v := range values {
		values.Del(k)
		values[strings.ToLower(k)] = v
	}
	if f.filter.Running == nil {
		values.Del("running")
	}
	for _, k := range f.kindNames {
		values.Add("kindname", k)
	}
	return values, nil
}

func (f *eventFilter) flags(fs *gnuflag.FlagSet) {
	name := "Filter events by kind name"
	fs.Var(&f.kindNames, "kind", name)
	fs.Var(&f.kindNames, "k", name)
	name = "Filter events by target type"
	ptr := (*string)(&f.filter.Target.Type)
	fs.StringVar(ptr, "target", "", name)
	fs.StringVar(ptr, "t", "", name)
	name = "Filter events by target value"
	fs.StringVar(&f.filter.Target.Value, "target-value", "", name)
	fs.StringVar(&f.filter.Target.Value, "v", "", name)
	name = "Filter events by owner name"
	fs.StringVar(&f.filter.OwnerName, "owner", "", name)
	fs.StringVar(&f.filter.OwnerName, "o", "", name)
	name = "Shows only currently running events"
	fs.BoolVar(&f.running, "running", false, name)
	fs.BoolVar(&f.running, "r", false, name)
}

func (c *EventList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "event-list",
		Usage: "event list [--kind/-k kind name]... [--owner/-o owner] [--running/-r] [--include-removed/-i] [--target/-t target type] [--target-value/-v target value]",
		Desc: `Lists events that you have permission to see.

		Flags can be used to filter the list of events.`,
	}
}

func (c *EventList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.filter.flags(c.fs)
		c.fs.BoolVar(&c.json, "json", false, "Show JSON")
	}
	return c.fs
}

func (c *EventList) Run(context *cmd.Context, client *cmd.Client) error {
	qs, err := c.filter.queryString(client)
	if err != nil {
		return err
	}
	u, err := cmd.GetURLVersion("1.1", fmt.Sprintf("/events?%s", qs.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var evts []event.Event
	err = json.Unmarshal(result, &evts)
	if err != nil {
		return fmt.Errorf("unable to unmarshal %q: %s", string(result), err)
	}

	if c.json {
		result := []*orderedmap.OrderedMap{}
		for i := range evts {
			o, err := eventJSONFriendly(&evts[i])
			if err != nil {
				return err
			}
			result = append(result, o)
		}

		return formatter.JSON(context.Stdout, result)
	}

	return c.Show(evts, context)
}

var reEmailShort = regexp.MustCompile(`@.*$`)

func (c *EventList) Show(evts []event.Event, context *cmd.Context) error {
	tbl := tablecli.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = tablecli.Row{"ID", "Start (duration)", "Success", "Owner", "Kind", "Target"}
	for i := range evts {
		evt := &evts[i]
		targets := []event.Target{evt.Target}
		for _, et := range evt.ExtraTargets {
			targets = append(targets, et.Target)
		}
		targetsStr := make([]string, len(targets))
		for i, t := range targets {
			if t.Type == "container" {
				t.Value = ShortID(t.Value)
			}
			targetsStr[i] = fmt.Sprintf("%s: %s", t.Type, t.Value)
		}
		owner := reEmailShort.ReplaceAllString(evt.Owner.Name, "@…")
		var success string
		var duration *time.Duration
		if evt.Running {
			success = "…"
		} else {
			timeDiff := evt.EndTime.Sub(evt.StartTime)
			duration = &timeDiff
			success = fmt.Sprintf("%v", evt.Error == "")
			if evt.CancelInfo.Canceled {
				success += " ✗"
			}
		}
		ts := formatter.FormatDateAndDuration(evt.StartTime, duration)
		row := tablecli.Row{evt.UniqueID.Hex(), ts, success, owner, evt.Kind.Name, strings.Join(targetsStr, "\n")}
		var color string
		if evt.Running {
			color = "yellow"
		} else if evt.CancelInfo.Canceled {
			color = "magenta"
		} else if evt.Error != "" {
			color = "red"
		}
		if color != "" {
			for i, v := range row {
				if v != "" {
					row[i] = cmd.Colorfy(v, color, "", "")
				}
			}
		}
		tbl.AddRow(row)
	}
	fmt.Fprintf(context.Stdout, "%s", tbl.String())
	return nil
}

type EventInfo struct {
	fs   *gnuflag.FlagSet
	json bool
}

func (c *EventInfo) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("event-info", gnuflag.ContinueOnError)
		c.fs.BoolVar(&c.json, "json", false, "Show JSON")
	}
	return c.fs
}

func (c *EventInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-info",
		Usage:   "event info <event-id>",
		Desc:    `Show detailed information about one single event.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *EventInfo) Run(context *cmd.Context, client *cmd.Client) error {
	u, err := cmd.GetURLVersion("1.1", fmt.Sprintf("/events/%s", context.Args[0]))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var evt event.Event
	err = json.Unmarshal(result, &evt)
	if err != nil {
		return fmt.Errorf("unable to unmarshal %q: %s", string(result), err)
	}

	if c.json {
		o, err := eventJSONFriendly(&evt)
		if err != nil {
			return err
		}
		return formatter.JSON(context.Stdout, o)
	}
	return c.Show(&evt, context)
}

func eventJSONFriendly(evt *event.Event) (*orderedmap.OrderedMap, error) {
	var startData interface{}
	var endData interface{}
	var otherData interface{}

	evt.StartData(&startData)
	evt.EndData(&endData)
	evt.OtherData(&otherData)

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(evt)
	if err != nil {
		return nil, err
	}

	o := orderedmap.New()
	err = json.Unmarshal(buf.Bytes(), &o)
	if err != nil {
		return nil, err
	}

	o.Delete("StartCustomData")
	o.Delete("EndCustomData")
	o.Delete("OtherCustomData")

	o.Set("StartData", startData)
	o.Set("EndData", endData)
	o.Set("OtherData", otherData)

	return o, nil
}

func (c *EventInfo) Show(evt *event.Event, context *cmd.Context) error {
	type item struct {
		label string
		value string
	}
	startFmt := formatter.FormatDate(evt.StartTime)
	var endFmt string
	if evt.Running {
		duration := time.Since(evt.StartTime)
		endFmt = fmt.Sprintf("running (%s)", formatter.FormatDuration(&duration))
	} else {
		duration := evt.EndTime.Sub(evt.StartTime)
		endFmt = formatter.FormatDateAndDuration(evt.EndTime, &duration)
	}
	targets := []event.Target{evt.Target}
	for _, et := range evt.ExtraTargets {
		targets = append(targets, et.Target)
	}
	items := []item{
		{"ID", evt.UniqueID.Hex()},
		{"Start", startFmt},
		{"End", endFmt},
	}
	for i, t := range targets {
		var itemName string
		if i == 0 {
			itemName = "Targets"
		}
		items = append(items, item{itemName, fmt.Sprintf("%s(%s)", t.Type, t.Value)})
	}
	items = append(items,
		item{"Kind", fmt.Sprintf("%s(%s)", evt.Kind.Type, evt.Kind.Name)},
		item{"Owner", fmt.Sprintf("%s(%s)", evt.Owner.Type, evt.Owner.Name)},
	)

	if evt.SourceIP != "" {
		items = append(items, item{
			"Source IP", evt.SourceIP,
		})
	}

	successful := evt.Error == ""
	successfulStr := strconv.FormatBool(successful)
	if successful {
		if evt.Running {
			successfulStr = "…"
		}
		items = append(items, item{"Success", successfulStr})
	} else {
		parts := strings.Split(evt.Error, "\n")
		var redError []string
		for i, p := range parts {
			if i == 0 && p != "" {
				redError = append(redError, "")
			}
			if p == "" {
				redError = append(redError, "")
				continue
			}
			redError = append(redError, cmd.Colorfy(p, "red", "", ""))
		}
		fullError := strings.Join(redError, "\n")
		if !strings.HasSuffix(fullError, "\n") {
			fullError += "\n"
		}
		redSuccess := cmd.Colorfy(successfulStr, "red", "", "")
		items = append(items, []item{
			{"Success", redSuccess},
			{"Error", fullError},
		}...)
	}
	items = append(items, []item{
		{"Cancelable", strconv.FormatBool(evt.Cancelable)},
		{"Canceled", strconv.FormatBool(evt.CancelInfo.Canceled)},
	}...)
	if evt.CancelInfo.Canceled {
		items = append(items, []item{
			{"  Reason", evt.CancelInfo.Reason},
			{"  By", evt.CancelInfo.Owner},
			{"  At", evt.CancelInfo.AckTime.Format(time.RFC822Z)},
		}...)
	}
	labels := []string{"Start", "End", "Other"}
	for i, fn := range []func(interface{}) error{evt.StartData, evt.EndData, evt.OtherData} {
		var data interface{}
		err := fn(&data)
		if err == nil && data != nil {
			str, err := yaml.Marshal(data)
			if err == nil {
				padded := padLines(string(str), "    ")
				items = append(items, item{fmt.Sprintf("%s Custom Data", labels[i]), "\n" + padded})
			}
		}
	}
	log := evt.Log()
	if log != "" {
		items = append(items, item{"Log", "\n" + padLines(log, "    ")})
	}
	var maxSz int
	for _, item := range items {
		sz := len(item.label)
		if len(item.value) > 0 && item.value[0] != '\n' && sz > maxSz {
			maxSz = sz
		}
	}
	for _, item := range items {
		if item.label != "" {
			item.label += ":"
		}
		count := (maxSz - len(item.label)) + 2
		var pad string
		if count > 0 && len(item.value) > 0 && item.value[0] != '\n' {
			pad = strings.Repeat(" ", count)
		}
		label := cmd.Colorfy(item.label, "cyan", "", "")
		fmt.Fprintf(context.Stdout, "%s%s%s\n", label, pad, item.value)
	}
	return nil
}

var rePadLines = regexp.MustCompile(`(?m)^(.+)`)

func padLines(s string, pad string) string {
	return rePadLines.ReplaceAllString(s, pad+`$1`)
}

type EventCancel struct {
	cmd.ConfirmationCommand
}

func (c *EventCancel) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-cancel",
		Usage:   "event cancel <event-id> <reason> [-y]",
		Desc:    `Cancel a running event.`,
		MinArgs: 2,
	}
}

func (c *EventCancel) Run(context *cmd.Context, client *cmd.Client) error {
	if !c.Confirm(context, "Are you sure you want to cancel this event?") {
		return nil
	}
	u, err := cmd.GetURLVersion("1.1", fmt.Sprintf("/events/%s/cancel", context.Args[0]))
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("reason", strings.Join(context.Args[1:], " "))
	request, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Cancellation successfully requested.")
	return nil
}
