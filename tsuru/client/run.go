// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	tsuruClientApp "github.com/tsuru/tsuru-client/tsuru/app"
	"github.com/tsuru/tsuru-client/tsuru/config"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
)

type AppRun struct {
	tsuruClientApp.AppNameMixIn
	fs       *gnuflag.FlagSet
	once     bool
	isolated bool
}

func (c *AppRun) Info() *cmd.Info {
	desc := `Runs an arbitrary command in application's containers. The base directory for
all commands is the root of the application.

If you use the [[--once]] flag tsuru will run the command only in one unit.
Otherwise, it will run the command in all units.`
	return &cmd.Info{
		Name:    "app-run",
		Usage:   "app run <command> [commandarg1] [commandarg2] ... [commandargn] [-a/--app appname] [-o/--once] [-i/--isolated]",
		Desc:    desc,
		MinArgs: 1,
	}
}

func (c *AppRun) Run(context *cmd.Context) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/run", appName))
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("command", strings.Join(context.Args, " "))
	v.Set("once", strconv.FormatBool(c.once))
	v.Set("isolated", strconv.FormatBool(c.isolated))
	b := strings.NewReader(v.Encode())
	request, err := http.NewRequest("POST", u, b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	w := tsuruIo.NewStreamWriter(context.Stdout, &tsuruIo.SimpleJsonMessageFormatter{NoTimestamp: true})
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, r.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}

func (c *AppRun) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVar(&c.once, "once", false, "Running only one unit")
		c.fs.BoolVar(&c.once, "o", false, "Running only one unit")
		c.fs.BoolVar(&c.isolated, "isolated", false, "Running in ephemeral container")
		c.fs.BoolVar(&c.isolated, "i", false, "Running in ephemeral container")
	}
	return c.fs
}
