// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	tsuruClientApp "github.com/tsuru/tsuru-client/tsuru/app"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	provTypes "github.com/tsuru/tsuru/types/provision"
)

type UnitAdd struct {
	tsuruClientApp.AppNameMixIn
	fs      *gnuflag.FlagSet
	process string
	version string
}

func (c *UnitAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-add",
		Usage: "unit add <# of units> [-a/--app appname] [-p/--process processname] [--version version]",
		Desc: `Adds new units to a process of an application. You need to have access to the
app to be able to add new units to it.`,
		MinArgs: 1,
	}
}

func (c *UnitAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

func (c *UnitAdd) Run(context *cmd.Context) error {
	context.RawOutput()
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/units", appName))
	if err != nil {
		return err
	}
	val := url.Values{}
	val.Add("units", context.Args[0])
	val.Add("process", c.process)
	val.Set("version", c.version)
	request, err := http.NewRequest("PUT", u, bytes.NewBufferString(val.Encode()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return formatter.StreamJSONResponse(context.Stdout, response)
}

type UnitRemove struct {
	tsuruClientApp.AppNameMixIn
	fs      *gnuflag.FlagSet
	process string
	version string
}

func (c *UnitRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-remove",
		Usage: "unit remove <# of units> [-a/--app appname] [-p/-process processname] [--version version]",
		Desc: `Removes units from a process of an application. You need to have access to the
app to be able to remove units from it.`,
		MinArgs: 1,
	}
}

func (c *UnitRemove) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.process, "process", "", "Process name")
		c.fs.StringVar(&c.process, "p", "", "Process name")
		c.fs.StringVar(&c.version, "version", "", "Version number")
	}
	return c.fs
}

func (c *UnitRemove) Run(context *cmd.Context) error {
	context.RawOutput()
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	val := url.Values{}
	val.Add("units", context.Args[0])
	val.Add("process", c.process)
	val.Set("version", c.version)
	url, err := config.GetURL(fmt.Sprintf("/apps/%s/units?%s", appName, val.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(context.Stdout, response)
}

type UnitKill struct {
	tsuruClientApp.AppNameMixIn
	jobName string
	fs      *gnuflag.FlagSet
	force   bool
}

func (c *UnitKill) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-kill",
		Usage: "unit kill <-a/--app appname|-j/--job jobname> [-f/--force] <unit>",
		Desc: `Kills units from a process of an application or job. You need to have access to the
app or job to be able to remove unit from it.`,
		MinArgs: 1,
	}
}

func (c *UnitKill) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.StringVar(&c.jobName, "job", "", "The name of the job.")
		c.fs.StringVar(&c.jobName, "j", "", "The name of the job.")
		c.fs.BoolVar(&c.force, "f", false, "Forces the termination of unit.")
	}
	return c.fs
}

func (c *UnitKill) Run(context *cmd.Context) error {
	context.RawOutput()
	joa := JobOrApp{fs: c.fs}
	err := joa.validate()
	if err != nil {
		return err
	}
	if len(context.Args) < 1 {
		return errors.New("you must provide the unit name.")
	}
	unit := context.Args[0]
	v := url.Values{}
	if c.force {
		v.Set("force", "true")
	}
	version := "1.12"
	if joa.Type == "job" {
		version = "1.13"
	}
	url, err := config.GetURLVersion(version, fmt.Sprintf("/%ss/%s/units/%s?%s", joa.Type, joa.val, unit, v.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(context.Stdout, response)
}

type UnitSet struct {
	tsuruClientApp.AppNameMixIn
	fs      *gnuflag.FlagSet
	process string
	version int
}

func (c *UnitSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "unit-set",
		Usage: "unit set <# of units> [-a/--app appname] [-p/--process processname] [--version version]",
		Desc: `Set the number of units for a process of an application, adding or removing units as needed. You need to have access to the
app to be able to set the number of units for it. The process flag is optional if the app has only 1 process.`,
		MinArgs: 1,
	}
}

func (c *UnitSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		processMessage := "Process name"
		c.fs.StringVar(&c.process, "process", "", processMessage)
		c.fs.StringVar(&c.process, "p", "", processMessage)
		c.fs.IntVar(&c.version, "version", 0, "Version number")
	}
	return c.fs
}

func (c *UnitSet) Run(context *cmd.Context) error {
	context.RawOutput()
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	result, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var a app
	err = json.Unmarshal(result, &a)
	if err != nil {
		return err
	}

	unitsByProcess := map[string][]provTypes.Unit{}
	unitsByVersion := map[int][]provTypes.Unit{}
	for _, u := range a.Units {
		unitsByProcess[u.ProcessName] = append(unitsByProcess[u.ProcessName], u)
		unitsByVersion[u.Version] = append(unitsByVersion[u.Version], u)
	}

	if len(unitsByProcess) != 1 && c.process == "" {
		return errors.New("Please use the -p/--process flag to specify which process you want to set units for.")
	}

	if len(unitsByVersion) != 1 && c.version == 0 {
		return errors.New("Please use the --version flag to specify which version you want to set units for.")
	}

	if c.process == "" {
		for p := range unitsByProcess {
			c.process = p
			break
		}
	}

	if c.version == 0 {
		for v := range unitsByVersion {
			c.version = v
			break
		}
	}

	existingUnits := 0
	for _, unit := range a.Units {
		if unit.ProcessName == c.process && unit.Version == c.version {
			existingUnits++
		}
	}

	desiredUnits, err := strconv.Atoi(context.Args[0])
	if err != nil {
		return err
	}

	if existingUnits < desiredUnits {
		u, err := config.GetURL(fmt.Sprintf("/apps/%s/units", appName))
		if err != nil {
			return err
		}

		unitsToAdd := desiredUnits - existingUnits
		val := url.Values{}
		val.Add("units", strconv.Itoa(unitsToAdd))
		val.Add("process", c.process)
		val.Add("version", strconv.Itoa(c.version))
		request, err := http.NewRequest(http.MethodPut, u, bytes.NewBufferString(val.Encode()))
		if err != nil {
			return err
		}

		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		response, err := tsuruHTTP.AuthenticatedClient.Do(request)
		if err != nil {
			return err
		}

		defer response.Body.Close()
		return formatter.StreamJSONResponse(context.Stdout, response)
	}

	if existingUnits > desiredUnits {
		unitsToRemove := existingUnits - desiredUnits
		val := url.Values{}
		val.Add("units", strconv.Itoa(unitsToRemove))
		val.Add("process", c.process)
		val.Add("version", strconv.Itoa(c.version))
		u, err := config.GetURL(fmt.Sprintf("/apps/%s/units?%s", appName, val.Encode()))
		if err != nil {
			return err
		}

		request, err := http.NewRequest(http.MethodDelete, u, nil)
		if err != nil {
			return err
		}

		response, err := tsuruHTTP.AuthenticatedClient.Do(request)
		if err != nil {
			return err
		}

		defer response.Body.Close()
		return formatter.StreamJSONResponse(context.Stdout, response)
	}

	fmt.Fprintf(context.Stdout, "The process %s, version %d already has %d units.\n", c.process, c.version, existingUnits)
	return nil
}
