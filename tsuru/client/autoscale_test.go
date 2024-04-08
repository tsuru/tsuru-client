// Copyright 2020 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestAutoScaleSet(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Unit auto scale successfully set.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.9/apps/myapp/units/autoscale")
			c.Assert(r.Method, check.Equals, "POST")
			var ret tsuru.AutoScaleSpec
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.AutoScaleSpec{
				AverageCPU: "30%",
				MinUnits:   2,
				MaxUnits:   5,
				Process:    "proc1",
			})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := AutoScaleSet{}
	command.Info()
	command.Flags().Parse(true, []string{"-a", "myapp", "-p", "proc1", "--min", "2", "--max", "5", "--cpu", "30%"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestKEDAScheduleAutoScaleSet(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Unit auto scale successfully set.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.9/apps/myapp/units/autoscale")
			c.Assert(r.Method, check.Equals, "POST")
			var ret tsuru.AutoScaleSpec
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.AutoScaleSpec{
				AverageCPU: "30%",
				MinUnits:   2,
				MaxUnits:   5,
				Process:    "proc1",
				Schedules: []tsuru.AutoScaleSchedule{
					{
						MinReplicas: 2,
						Start:       "0 6 * * *",
						End:         "0 18 * * *",
					},
					{
						MinReplicas: 1,
						Start:       "0 18 * * *",
						End:         "0 0 * * *",
					},
				},
			})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := AutoScaleSet{}
	command.Info()
	command.Flags().Parse(true, []string{
		"-a", "myapp", "-p", "proc1", "--min", "2", "--max", "5", "--cpu", "30%",
		"--schedule", "{\"minReplicas\": 2, \"start\": \"0 6 * * *\", \"end\": \"0 18 * * *\"}",
		"--schedule", "{\"minReplicas\": 1, \"start\": \"0 18 * * *\", \"end\": \"0 0 * * *\"}",
	})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestKEDAPrometheusAutoScaleSet(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Unit auto scale successfully set.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.9/apps/myapp/units/autoscale")
			c.Assert(r.Method, check.Equals, "POST")
			var ret tsuru.AutoScaleSpec
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.AutoScaleSpec{
				MinUnits: 1,
				MaxUnits: 5,
				Process:  "proc1",
				Prometheus: []tsuru.AutoScalePrometheus{
					{
						Name:      "prometheus_metric_1",
						Threshold: 1,
						Query:     "my_metric_1(app='my-app')",
					},
					{
						Name:              "prometheus_metric_2",
						Threshold:         5,
						Query:             "my_metric_2(app='my-app')",
						PrometheusAddress: "exemple.prometheus.com",
					},
				},
			})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := AutoScaleSet{}
	command.Info()
	command.Flags().Parse(true, []string{
		"-a", "myapp", "-p", "proc1", "--min", "1", "--max", "5",
		"--prometheus", "{\"name\": \"prometheus_metric_1\", \"threshold\": 1, \"query\": \"my_metric_1(app='my-app')\"}",
		"--prometheus", "{\"name\": \"prometheus_metric_2\", \"threshold\": 5, \"query\": \"my_metric_2(app='my-app')\", \"prometheusAddress\": \"exemple.prometheus.com\"}",
	})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAutoScaleUnset(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Unit auto scale successfully unset.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.9/apps/myapp/units/autoscale")
			c.Assert(r.Method, check.Equals, "DELETE")
			c.Assert(r.URL.Query().Get("process"), check.Equals, "proc1")
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := AutoScaleUnset{}
	command.Info()
	command.Flags().Parse(true, []string{"-a", "myapp", "-p", "proc1"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
