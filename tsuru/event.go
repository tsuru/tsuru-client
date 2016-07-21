package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cezarsa/form"
	"github.com/ghodss/yaml"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/event"
)

type eventList struct {
	fs     *gnuflag.FlagSet
	filter eventFilter
}

type eventFilter struct {
	KindName    string
	Target      string
	TargetValue string
	OwnerName   string
	Running     bool
}

func (f *eventFilter) queryString(client *cmd.Client) (url.Values, error) {
	values, err := form.EncodeToValues(f)
	if err != nil {
		return nil, err
	}
	for k, v := range values {
		values.Del(k)
		values[strings.ToLower(k)] = v
	}
	if !f.Running {
		values.Del("running")
	}
	return values, nil
}

func (f *eventFilter) flags(fs *gnuflag.FlagSet) {
	name := "Filter events by kind name"
	fs.StringVar(&f.KindName, "kind", "", name)
	fs.StringVar(&f.KindName, "k", "", name)
	name = "Filter events by target name"
	fs.StringVar(&f.Target, "target", "", name)
	fs.StringVar(&f.Target, "t", "", name)
	name = "Filter events by target value"
	fs.StringVar(&f.TargetValue, "target-value", "", name)
	fs.StringVar(&f.TargetValue, "v", "", name)
	name = "Filter events by owner name"
	fs.StringVar(&f.OwnerName, "owner", "", name)
	fs.StringVar(&f.OwnerName, "o", "", name)
	name = "Shows only currently running events"
	fs.BoolVar(&f.Running, "running", false, name)
	fs.BoolVar(&f.Running, "r", false, name)
}

func (c *eventList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "event-list",
		Usage: "event-list [-k kindName]",
		Desc:  `Lists events possibly filtering them.`,
	}
}

func (c *eventList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		c.filter.flags(c.fs)
	}
	return c.fs
}

func (c *eventList) Run(context *cmd.Context, client *cmd.Client) error {
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
	return c.Show(evts, context)
}

var reEmailShort = regexp.MustCompile(`@.*$`)

func (c *eventList) Show(evts []event.Event, context *cmd.Context) error {
	tbl := cmd.NewTable()
	tbl.Headers = cmd.Row{"ID", "Start (duration)", "Success", "Owner", "Kind", "Target"}
	for i := range evts {
		evt := &evts[i]
		if evt.Target.Type == "container" {
			evt.Target.Value = evt.Target.Value[:12]
		}
		fullTarget := fmt.Sprintf("%s: %s", evt.Target.Type, evt.Target.Value)
		startFmt := evt.StartTime.Format(time.RFC822Z)
		owner := reEmailShort.ReplaceAllString(evt.Owner.Name, "@…")
		var ts, success string
		if evt.Running {
			ts = fmt.Sprintf("%s (…)", startFmt)
			success = "…"
		} else {
			ts = fmt.Sprintf("%s (%v)", startFmt, evt.EndTime.Sub(evt.StartTime))
			success = fmt.Sprintf("%v", evt.Error == "")
			if evt.CancelInfo.Canceled {
				success += " ✗"
			}
		}
		row := cmd.Row{evt.UniqueID.Hex(), ts, success, owner, evt.Kind.Name, fullTarget}
		var color string
		if evt.CancelInfo.Canceled {
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

type eventInfo struct{}

func (c *eventInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-info",
		Usage:   "event-info <event-id>",
		Desc:    `Show detailed information about one single event.`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *eventInfo) Run(context *cmd.Context, client *cmd.Client) error {
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
	return c.Show(evt, context)
}

func (c *eventInfo) Show(evt event.Event, context *cmd.Context) error {
	type item struct {
		label string
		value string
	}
	startFmt := evt.StartTime.Format(time.RFC822Z)
	var endFmt string
	if evt.Running {
		endFmt = fmt.Sprintf("running (%v)", time.Now().Sub(evt.StartTime))
	} else {
		endFmt = fmt.Sprintf("%s (%v)", evt.EndTime.Format(time.RFC822Z), evt.EndTime.Sub(evt.StartTime))
	}
	items := []item{
		{"ID", evt.UniqueID.Hex()},
		{"Start", startFmt},
		{"End", endFmt},
		{"Target", fmt.Sprintf("%s(%s)", evt.Target.Type, evt.Target.Value)},
		{"Kind", fmt.Sprintf("%s(%s)", evt.Kind.Type, evt.Kind.Name)},
		{"Owner", fmt.Sprintf("%s(%s)", evt.Owner.Type, evt.Owner.Name)},
	}
	sucessful := evt.Error == ""
	sucessfulStr := strconv.FormatBool(sucessful)
	if sucessful {
		if evt.Running {
			sucessfulStr = "…"
		}
		items = append(items, item{"Success", sucessfulStr})
	} else {
		redError := cmd.Colorfy(fmt.Sprintf("%q", evt.Error), "red", "", "")
		redSuccess := cmd.Colorfy(sucessfulStr, "red", "", "")
		items = append(items, []item{
			{"Success", redSuccess},
			{"Error", redError},
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
	if evt.Log != "" {
		items = append(items, item{"Log", "\n" + padLines(evt.Log, "    ")})
	}
	var maxSz int
	for _, item := range items {
		sz := len(item.label)
		if len(item.value) > 0 && item.value[0] != '\n' && sz > maxSz {
			maxSz = sz
		}
	}
	for _, item := range items {
		count := (maxSz - len(item.label)) + 1
		var pad string
		if count > 0 && len(item.value) > 0 && item.value[0] != '\n' {
			pad = strings.Repeat(" ", count)
		}
		label := cmd.Colorfy(item.label+":", "cyan", "", "")
		fmt.Fprintf(context.Stdout, "%s%s%s\n", label, pad, item.value)
	}
	return nil
}

var rePadLines = regexp.MustCompile(`(?m)^(.+)`)

func padLines(s string, pad string) string {
	return rePadLines.ReplaceAllString(s, pad+`$1`)
}

type eventCancel struct {
	cmd.ConfirmationCommand
}

func (c *eventCancel) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-cancel",
		Usage:   "event-cancel <event-id> <reason> [-y]",
		Desc:    `Cancel a running event.`,
		MinArgs: 2,
	}
}

func (c *eventCancel) Run(context *cmd.Context, client *cmd.Client) error {
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
	fmt.Fprintln(context.Stdout, "Event successfully canceled.")
	return nil
}
