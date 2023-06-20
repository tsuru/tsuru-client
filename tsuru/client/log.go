// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
)

type AppLog struct {
	cmd.AppNameMixIn
	fs       *gnuflag.FlagSet
	source   string
	unit     string
	lines    int
	follow   bool
	noDate   bool
	noSource bool
}

func (c *AppLog) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-log",
		Usage: "app log [-a/--app appname] [-l/--lines numberOfLines] [-s/--source source] [-u/--unit unit] [-f/--follow]",
		Desc: `Shows log entries for an application. These logs include everything the
application send to stdout and stderr, alongside with logs from tsuru server
(deployments, restarts, etc.)

The [[--lines]] flag is optional and by default its value is 10.

The [[--source]] flag is optional and allows filtering logs by log source
(e.g. application, tsuru api).

The [[--unit]] flag is optional and allows filtering by unit. It's useful if
your application has multiple units and you want logs from a single one.

The [[--follow]] flag is optional and makes the command wait for additional
log output

The [[--no-date]] flag is optional and makes the log output without date.

The [[--no-source]] flag is optional and makes the log output without source
information, useful to very dense logs.
`,
		MinArgs: 0,
	}
}

type logFormatter struct {
	noDate   bool
	noSource bool
}

func (f logFormatter) Format(out io.Writer, dec *json.Decoder) error {
	var logs []log
	err := dec.Decode(&logs)
	if err != nil {
		if err == io.EOF {
			return err
		}
		buffered := dec.Buffered()
		bufferedData, _ := io.ReadAll(buffered)
		return fmt.Errorf("unable to parse json: %v: %q", err, string(bufferedData))
	}
	for _, l := range logs {
		prefix := f.prefix(l)

		if prefix == "" {
			fmt.Fprintf(out, "%s\n", l.Message)
		} else {
			fmt.Fprintf(out, "%s %s\n", cmd.Colorfy(prefix, "blue", "", ""), l.Message)
		}
	}
	return nil
}

func (f logFormatter) prefix(l log) string {
	parts := make([]string, 0, 2)
	if !f.noDate {
		parts = append(parts, formatter.Local(l.Date).Format("2006-01-02 15:04:05 -0700"))
	}
	if !f.noSource {
		if l.Unit != "" && l.Source != "" {
			parts = append(parts, fmt.Sprintf("[%s][%s]", l.Source, l.Unit))
		} else if l.Unit != "" {
			parts = append(parts, fmt.Sprintf("[%s]", l.Unit))
		} else {
			parts = append(parts, fmt.Sprintf("[%s]", l.Source))
		}
	}
	prefix := strings.Join(parts, " ")
	if prefix != "" {
		prefix = prefix + ":"
	}
	return prefix
}

type log struct {
	Date    time.Time
	Message string
	Source  string
	Unit    string
}

func (c *AppLog) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/log?lines=%d", appName, c.lines))
	if err != nil {
		return err
	}
	if c.source != "" {
		url = fmt.Sprintf("%s&source=%s", url, c.source)
	}
	if c.unit != "" {
		url = fmt.Sprintf("%s&unit=%s", url, c.unit)
	}
	if c.follow {
		url += "&follow=1"
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	defer response.Body.Close()
	formatter := logFormatter{
		noDate:   c.noDate,
		noSource: c.noSource,
	}
	dec := json.NewDecoder(response.Body)
	for {
		err = formatter.Format(context.Stdout, dec)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(context.Stdout, "Error: %v", err)
			}
			break
		}
	}
	return nil
}

func (c *AppLog) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.IntVar(&c.lines, "lines", 10, "The number of log lines to display")
		c.fs.IntVar(&c.lines, "l", 10, "The number of log lines to display")
		c.fs.StringVar(&c.source, "source", "", "The log from the given source")
		c.fs.StringVar(&c.source, "s", "", "The log from the given source")
		c.fs.StringVar(&c.unit, "unit", "", "The log from the given unit")
		c.fs.StringVar(&c.unit, "u", "", "The log from the given unit")
		c.fs.BoolVar(&c.follow, "follow", false, "Follow logs")
		c.fs.BoolVar(&c.follow, "f", false, "Follow logs")
		c.fs.BoolVar(&c.noDate, "no-date", false, "No date information")
		c.fs.BoolVar(&c.noSource, "no-source", false, "No source information")
	}
	return c.fs
}
