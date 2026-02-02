// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"gopkg.in/check.v1"
)

func (s *S) TestFormatterUsesCurrentTimeZone(c *check.C) {
	t := time.Now()
	logs := []log{
		{Date: t, Message: "Something happened", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "Something happened again", Source: "tsuru"},
	}
	data, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	var writer bytes.Buffer
	old := formatter.LocalTZ
	formatter.LocalTZ = time.UTC
	defer func() {
		formatter.LocalTZ = old
	}()
	logFmt := logFormatter{}
	err = logFmt.Format(&writer, json.NewDecoder(bytes.NewReader(data)))
	c.Assert(err, check.IsNil)
	tfmt := "2006-01-02 15:04:05 -0700"
	t = t.In(time.UTC)
	expected := color.BlueString(t.Format(tfmt)+" [tsuru]:") + " Something happened\n"
	expected = expected + color.BlueString(t.Add(2*time.Hour).Format(tfmt)+" [tsuru]:") + " Something happened again\n"
	c.Assert(writer.String(), check.Equals, expected)
}

func (s *S) TestAppLog(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "app", Unit: "abcdef"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = formatter.Local(t)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := color.BlueString(t.Format(tfmt)+" [tsuru]:") + " creating app lost\n"
	expected = expected + color.BlueString(t.Add(2*time.Hour).Format(tfmt)+" [app][abcdef]:") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	transport := cmdtest.Transport{
		Message: string(result),
		Status:  http.StatusOK,
	}
	s.setupFakeTransport(transport)
	command.Flags().Parse([]string{"--app", "appName"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithUnparsableData(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = formatter.Local(t)
	tfmt := "2006-01-02 15:04:05 -0700"

	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	transport := cmdtest.Transport{
		Message: string(result) + "\nunparseable data",
		Status:  http.StatusOK,
	}
	s.setupFakeTransport(&transport)
	command.Flags().Parse([]string{"--app", "appName"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	expected := color.BlueString(t.Format(tfmt)+" [tsuru]:") + " creating app lost\n"
	expected += "Error: unable to parse json: invalid character 'u' looking for beginning of value: \"\\nunparseable data\""
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithoutTheFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "app"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = formatter.Local(t)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := color.BlueString(t.Format(tfmt)+" [tsuru]:") + " creating app lost\n"
	expected = expected + color.BlueString(t.Add(2*time.Hour).Format(tfmt)+" [app]:") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	command.Flags().Parse([]string{"-a", "hitthelights"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/hitthelights/log") && req.Method == "GET" &&
				req.URL.Query().Get("lines") == "10"
		},
	}
	s.setupFakeTransport(trans)
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogShouldReturnNilIfHasNoContent(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	s.setupFakeTransport(&cmdtest.Transport{Message: "", Status: http.StatusNoContent})
	command.Flags().Parse([]string{"--app", "appName"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestAppLogInfo(c *check.C) {
	c.Assert((&AppLog{}).Info(), check.NotNil)
}

func (s *S) TestAppLogBySource(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "tsuru"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = formatter.Local(t)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := color.BlueString(t.Format(tfmt)+" [tsuru]:") + " creating app lost\n"
	expected = expected + color.BlueString(t.Add(2*time.Hour).Format(tfmt)+" [tsuru]:") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	command.Flags().Parse([]string{"-a", "hitthelights", "--source", "mysource"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("source") == "mysource"
		},
	}
	s.setupFakeTransport(trans)
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogByUnit(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru", Unit: "api"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "tsuru", Unit: "api"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = formatter.Local(t)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := color.BlueString(t.Format(tfmt)+" [tsuru][api]:") + " creating app lost\n"
	expected = expected + color.BlueString(t.Add(2*time.Hour).Format(tfmt)+" [tsuru][api]:") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	command.Flags().Parse([]string{"-a", "hitthelights", "--unit", "api"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("unit") == "api"
		},
	}
	s.setupFakeTransport(trans)
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithLines(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "tsuru"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = formatter.Local(t)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := color.BlueString(t.Format(tfmt)+" [tsuru]:") + " creating app lost\n"
	expected = expected + color.BlueString(t.Add(2*time.Hour).Format(tfmt)+" [tsuru]:") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	command.Flags().Parse([]string{"-a", "hitthelights", "--lines", "12"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("lines") == "12"
		},
	}
	s.setupFakeTransport(trans)
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithFollow(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "tsuru"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = formatter.Local(t)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := color.BlueString(t.Format(tfmt)+" [tsuru]:") + " creating app lost\n"
	expected = expected + color.BlueString(t.Add(2*time.Hour).Format(tfmt)+" [tsuru]:") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	command.Flags().Parse([]string{"-a", "hitthelights", "--lines", "12", "-f"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("lines") == "12" && req.URL.Query().Get("follow") == "1"
		},
	}
	s.setupFakeTransport(trans)
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithNoDateAndNoSource(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "GET /", Source: "web"},
		{Date: t.Add(2 * time.Hour), Message: "POST /", Source: "web"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	expected := "GET /\n"
	expected = expected + "POST /\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	command.Flags().Parse([]string{"-a", "hitthelights", "--lines", "12", "-f", "--no-date", "--no-source"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("lines") == "12" && req.URL.Query().Get("follow") == "1"
		},
	}
	s.setupFakeTransport(trans)
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithNoSource(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "GET /", Source: "web"},
		{Date: t.Add(2 * time.Hour), Message: "POST /", Source: "web"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = formatter.Local(t)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := color.BlueString(t.Format(tfmt)+":") + " GET /\n"
	expected = expected + color.BlueString(t.Add(2*time.Hour).Format(tfmt)+":") + " POST /\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	command.Flags().Parse([]string{"-a", "hitthelights", "--lines", "12", "-f", "--no-source"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("lines") == "12" && req.URL.Query().Get("follow") == "1"
		},
	}
	s.setupFakeTransport(trans)
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogFlagSet(c *check.C) {
	command := AppLog{}
	flagset := command.Flags()
	flagset.Parse([]string{"--source", "tsuru", "--unit", "abcdef", "--lines", "12", "--app", "ashamed", "--follow", "--no-date", "--no-source"})
	source := flagset.Lookup("source")
	c.Check(source, check.NotNil)
	c.Check(source.Name, check.Equals, "source")
	c.Check(source.Usage, check.Equals, "The log from the given source")
	c.Check(source.Value.String(), check.Equals, "tsuru")
	c.Check(source.DefValue, check.Equals, "")
	c.Check(source.Shorthand, check.Equals, "s")

	unit := flagset.Lookup("unit")
	c.Check(unit, check.NotNil)
	c.Check(unit.Name, check.Equals, "unit")
	c.Check(unit.Usage, check.Equals, "The log from the given unit")
	c.Check(unit.Value.String(), check.Equals, "abcdef")
	c.Check(unit.DefValue, check.Equals, "")
	c.Check(unit.Shorthand, check.Equals, "u")

	lines := flagset.Lookup("lines")
	c.Check(lines, check.NotNil)
	c.Check(lines.Name, check.Equals, "lines")
	c.Check(lines.Usage, check.Equals, "The number of log lines to display")
	c.Check(lines.Value.String(), check.Equals, "12")
	c.Check(lines.DefValue, check.Equals, "10")
	c.Check(lines.Shorthand, check.Equals, "l")

	app := flagset.Lookup("app")
	c.Check(app, check.NotNil)
	c.Check(app.Name, check.Equals, "app")
	c.Check(app.Usage, check.Equals, "The name of the app.")
	c.Check(app.Value.String(), check.Equals, "ashamed")
	c.Check(app.DefValue, check.Equals, "")
	c.Check(app.Shorthand, check.Equals, "a")

	follow := flagset.Lookup("follow")
	c.Check(follow, check.NotNil)
	c.Check(follow.Name, check.Equals, "follow")
	c.Check(follow.Usage, check.Equals, "Follow logs")
	c.Check(follow.Value.String(), check.Equals, "true")
	c.Check(follow.DefValue, check.Equals, "false")
	c.Check(follow.Shorthand, check.Equals, "f")

	noDate := flagset.Lookup("no-date")
	c.Check(noDate, check.NotNil)
	c.Check(noDate.Name, check.Equals, "no-date")
	c.Check(noDate.Usage, check.Equals, "No date information")
	c.Check(noDate.Value.String(), check.Equals, "true")
	c.Check(noDate.DefValue, check.Equals, "false")

	noSource := flagset.Lookup("no-source")
	c.Check(noSource, check.NotNil)
	c.Check(noSource.Name, check.Equals, "no-source")
	c.Check(noSource.Usage, check.Equals, "No source information")
	c.Check(noSource.Value.String(), check.Equals, "true")
	c.Check(noSource.DefValue, check.Equals, "false")
}
