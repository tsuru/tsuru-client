// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/testing"
	"launchpad.net/gocheck"
)

func (s *S) TestAutoScaleEnable(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := testing.ConditionalTransport{
		Transport: testing.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "PUT" && req.URL.Path == "/autoscale/ble/enable"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := autoScaleEnable{}
	command.Flags().Parse(true, []string{"-a", "ble"})
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestAutoScaleDisable(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := testing.ConditionalTransport{
		Transport: testing.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "PUT" && req.URL.Path == "/autoscale/ble/disable"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := autoScaleDisable{}
	command.Flags().Parse(true, []string{"-a", "ble"})
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestAutoScaleConfig(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := testing.ConditionalTransport{
		Transport: testing.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			var config AutoScaleConfig
			err := json.NewDecoder(req.Body).Decode(&config)
			c.Assert(err, gocheck.IsNil)
			c.Assert(config.MaxUnits, gocheck.Equals, 10)
			c.Assert(config.MinUnits, gocheck.Equals, 2)
			c.Assert(config.Enabled, gocheck.Equals, true)
			c.Assert(config.Increase.Wait, gocheck.Equals, 300)
			c.Assert(config.Decrease.Wait, gocheck.Equals, 300)
			c.Assert(config.Increase.Expression, gocheck.Equals, "{cpu_max} > 90")
			c.Assert(config.Decrease.Expression, gocheck.Equals, "{cpu_max} < 10")
			c.Assert(config.Increase.Units, gocheck.Equals, 2)
			c.Assert(config.Decrease.Units, gocheck.Equals, 1)
			return req.Method == "PUT" && req.URL.Path == "/autoscale/ble"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := autoScaleConfig{}
	command.Flags().Parse(true, []string{"-a", "ble", "--max-units", "10", "--min-units", "2", "--increase-step", "2", "--increase-wait-time", "300", "--increase-expression", "{cpu_max} > 90", "--decrease-step", "1", "--decrease-wait-time", "300", "--decrease-expression", "{cpu_max} < 10", "--enabled"})
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
}
