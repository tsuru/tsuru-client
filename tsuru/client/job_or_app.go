// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"errors"

	"github.com/tsuru/gnuflag"
)

type JobOrApp struct {
	Type       string
	val        string
	appProcess string
	fs         *gnuflag.FlagSet
}

func (c *JobOrApp) validate() error {
	appName := c.fs.Lookup("app").Value.String()
	jobName := c.fs.Lookup("job").Value.String()
	var processName string

	if flag := c.fs.Lookup("process"); flag != nil {
		processName = flag.Value.String()
	}

	if appName == "" && jobName == "" {
		return errors.New("job name or app name is required")
	}
	if appName != "" && jobName != "" {
		return errors.New("please use only one of the -a/--app and -j/--job flags")
	}
	if processName != "" && jobName != "" {
		return errors.New("please specify process just for an app")
	}
	if appName != "" {
		c.Type = "app"
		c.val = appName
		c.appProcess = processName
		return nil
	}
	c.Type = "job"
	c.val = jobName
	return nil
}
