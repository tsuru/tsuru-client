// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
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
