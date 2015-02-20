// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestAutoScaleEnable(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "PUT" && req.URL.Path == "/autoscale/ble/enable"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := autoScaleEnable{}
	command.Flags().Parse(true, []string{"-a", "ble"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAutoScaleDisable(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "PUT" && req.URL.Path == "/autoscale/ble/disable"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := autoScaleDisable{}
	command.Flags().Parse(true, []string{"-a", "ble"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAutoScaleConfig(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			var config AutoScaleConfig
			err := json.NewDecoder(req.Body).Decode(&config)
			c.Assert(err, check.IsNil)
			c.Assert(config.MaxUnits, check.Equals, 10)
			c.Assert(config.MinUnits, check.Equals, 2)
			c.Assert(config.Enabled, check.Equals, true)
			c.Assert(config.Increase.Wait, check.Equals, 300000000000)
			c.Assert(config.Decrease.Wait, check.Equals, 300000000000)
			c.Assert(config.Increase.Expression, check.Equals, "{cpu_max} > 90")
			c.Assert(config.Decrease.Expression, check.Equals, "{cpu_max} < 10")
			c.Assert(config.Increase.Units, check.Equals, 2)
			c.Assert(config.Decrease.Units, check.Equals, 1)
			return req.Method == "PUT" && req.URL.Path == "/autoscale/ble"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := autoScaleConfig{}
	command.Flags().Parse(true, []string{"-a", "ble", "--max-units", "10", "--min-units", "2", "--increase-step", "2", "--increase-wait-time", "300", "--increase-expression", "{cpu_max} > 90", "--decrease-step", "1", "--decrease-wait-time", "300", "--decrease-expression", "{cpu_max} < 10", "--enabled"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}
