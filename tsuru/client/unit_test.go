// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	tsuruIo "github.com/tsuru/tsuru/io"
	check "gopkg.in/check.v1"
)

func (s *S) TestUnitAdd(c *check.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	context := cmd.Context{
		Args:   []string{"3"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "-- added unit --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.FormValue("process"), check.Equals, "p1")
			c.Assert(req.FormValue("units"), check.Equals, "3")
			return strings.HasSuffix(req.URL.Path, "/apps/radio/units") && req.Method == "PUT"
		},
	}
	s.setupFakeTransport(trans)
	command := UnitAdd{}
	command.Flags().Parse(true, []string{"-a", "radio", "-p", "p1"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestUnitAddWithVersion(c *check.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	context := cmd.Context{
		Args:   []string{"3"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "-- added unit --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.FormValue("process"), check.Equals, "p1")
			c.Assert(req.FormValue("units"), check.Equals, "3")
			c.Assert(req.FormValue("version"), check.Equals, "9")
			return strings.HasSuffix(req.URL.Path, "/apps/radio/units") && req.Method == "PUT"
		},
	}
	s.setupFakeTransport(trans)
	command := UnitAdd{}
	command.Flags().Parse(true, []string{"-a", "radio", "-p", "p1", "--version", "9"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestUnitAddFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"3"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	msg := tsuruIo.SimpleJsonMessage{Error: "errored msg"}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	s.setupFakeTransport(&cmdtest.Transport{Message: string(result), Status: 200})
	command := UnitAdd{}
	command.Flags().Parse(true, []string{"-a", "radio"})
	err = command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "errored msg")
}

func (s *S) TestUnitAddInfo(c *check.C) {
	c.Assert((&UnitAdd{}).Info(), check.NotNil)
}

func (s *S) TestUnitAddIsFlaggedACommand(c *check.C) {
	var _ cmd.FlaggedCommand = &UnitAdd{}
}

func (s *S) TestUnitRemove(c *check.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	context := cmd.Context{
		Args:   []string{"2"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "-- removed unit --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.FormValue("process"), check.Equals, "web1")
			c.Assert(req.FormValue("units"), check.Equals, "2")
			return strings.HasSuffix(req.URL.Path, "/apps/vapor/units") && req.Method == http.MethodDelete
		},
	}
	s.setupFakeTransport(trans)
	command := UnitRemove{}
	command.Flags().Parse(true, []string{"-a", "vapor", "-p", "web1"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "-- removed unit --")
}

func (s *S) TestUnitRemoveFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Message: "Failed to remove.", Status: 500})
	command := UnitRemove{}
	command.Flags().Parse(true, []string{"-a", "vapor"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "Failed to remove.")
}

func (s *S) TestUnitRemoveInfo(c *check.C) {
	c.Assert((&UnitRemove{}).Info(), check.NotNil)
}

func (s *S) TestUnitRemoveIsACommand(c *check.C) {
	var _ cmd.Command = &UnitRemove{}
}

func (s *S) TestUnitSetAddUnits(c *check.C) {
	var stdout, stderr bytes.Buffer
	var calledGet bool
	var calledPut bool
	context := cmd.Context{
		Args:   []string{"10"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	resultGet := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","ProcessName":"web"},{"Ip":"9.9.9.9","ID":"app1/1","Status":"started","ProcessName":"web"},{"Ip":"","ID":"app1/2","Status":"pending","ProcessName":"web"},{"Ip":"8.8.8.8","ID":"app1/3","Status":"started","ProcessName":"worker"}],"teams":["tsuruteam","crane"],"owner":"myapp_owner","deploys":7,"router":"planb"}`

	expectedOut := "-- added unit --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	resultPut, _ := json.Marshal(msg)

	transport := &cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				CondFunc: func(req *http.Request) bool {
					calledGet = true
					return strings.HasSuffix(req.URL.Path, "/apps/app1") && req.Method == http.MethodGet
				},
				Transport: cmdtest.Transport{Message: resultGet, Status: http.StatusOK},
			},
			{
				CondFunc: func(req *http.Request) bool {
					calledPut = true
					c.Assert(req.FormValue("process"), check.Equals, "web")
					c.Assert(req.FormValue("units"), check.Equals, "7")
					return strings.HasSuffix(req.URL.Path, "/apps/app1/units") && req.Method == http.MethodPut
				},
				Transport: cmdtest.Transport{Message: string(resultPut), Status: http.StatusOK},
			},
		},
	}

	s.setupFakeTransport(transport)
	command := UnitSet{}
	command.Flags().Parse(true, []string{"-a", "app1", "-p", "web"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(calledGet, check.Equals, true)
	c.Assert(calledPut, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestUnitSetAddUnitsFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	var calledGet bool
	var calledPut bool
	context := cmd.Context{
		Args:   []string{"10"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	resultGet := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","ProcessName":"web"},{"Ip":"9.9.9.9","ID":"app1/1","Status":"started","ProcessName":"web"},{"Ip":"","ID":"app1/2","Status":"pending","ProcessName":"web"},{"Ip":"8.8.8.8","ID":"app1/3","Status":"started","ProcessName":"worker"}],"teams":["tsuruteam","crane"],"owner":"myapp_owner","deploys":7,"router":"planb"}`

	transport := &cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				CondFunc: func(req *http.Request) bool {
					calledGet = true
					return strings.HasSuffix(req.URL.Path, "/apps/app1") && req.Method == http.MethodGet
				},
				Transport: cmdtest.Transport{Message: resultGet, Status: http.StatusOK},
			},
			{
				CondFunc: func(req *http.Request) bool {
					calledPut = true
					c.Assert(req.FormValue("process"), check.Equals, "web")
					c.Assert(req.FormValue("units"), check.Equals, "7")
					return strings.HasSuffix(req.URL.Path, "/apps/app1/units") && req.Method == http.MethodPut
				},
				Transport: cmdtest.Transport{Message: "Failed to put.", Status: http.StatusInternalServerError},
			},
		},
	}

	s.setupFakeTransport(transport)
	command := UnitSet{}
	command.Flags().Parse(true, []string{"-a", "app1", "-p", "web"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "Failed to put.")
	c.Assert(calledGet, check.Equals, true)
	c.Assert(calledPut, check.Equals, true)
}

func (s *S) TestUnitSetRemoveUnits(c *check.C) {
	var stdout, stderr bytes.Buffer
	var calledGet bool
	var calledDelete bool
	context := cmd.Context{
		Args:   []string{"1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	resultGet := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","ProcessName":"web"},{"Ip":"9.9.9.9","ID":"app1/1","Status":"started","ProcessName":"web"},{"Ip":"","ID":"app1/2","Status":"pending","ProcessName":"web"},{"Ip":"8.8.8.8","ID":"app1/3","Status":"started","ProcessName":"worker"}],"teams":["tsuruteam","crane"],"owner":"myapp_owner","deploys":7,"router":"planb"}`

	expectedOut := "-- removed unit --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	resultDelete, _ := json.Marshal(msg)

	transport := &cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				CondFunc: func(req *http.Request) bool {
					calledGet = true
					return strings.HasSuffix(req.URL.Path, "/apps/app1") && req.Method == http.MethodGet
				},
				Transport: cmdtest.Transport{Message: resultGet, Status: http.StatusOK},
			},
			{
				CondFunc: func(req *http.Request) bool {
					calledDelete = true
					c.Assert(req.FormValue("process"), check.Equals, "web")
					c.Assert(req.FormValue("units"), check.Equals, "2")
					return strings.HasSuffix(req.URL.Path, "/apps/app1/units") && req.Method == http.MethodDelete
				},
				Transport: cmdtest.Transport{Message: string(resultDelete), Status: http.StatusOK},
			},
		},
	}

	s.setupFakeTransport(transport)
	command := UnitSet{}
	command.Flags().Parse(true, []string{"-a", "app1", "-p", "web"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(calledGet, check.Equals, true)
	c.Assert(calledDelete, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestUnitSetRemoveUnitsFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	var calledGet bool
	var calledDelete bool
	context := cmd.Context{
		Args:   []string{"1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	resultGet := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","ProcessName":"web"},{"Ip":"9.9.9.9","ID":"app1/1","Status":"started","ProcessName":"web"},{"Ip":"","ID":"app1/2","Status":"pending","ProcessName":"web"},{"Ip":"8.8.8.8","ID":"app1/3","Status":"started","ProcessName":"worker"}],"teams":["tsuruteam","crane"],"owner":"myapp_owner","deploys":7,"router":"planb"}`

	transport := &cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				CondFunc: func(req *http.Request) bool {
					calledGet = true
					return strings.HasSuffix(req.URL.Path, "/apps/app1") && req.Method == http.MethodGet
				},
				Transport: cmdtest.Transport{Message: resultGet, Status: http.StatusOK},
			},
			{
				CondFunc: func(req *http.Request) bool {
					calledDelete = true
					c.Assert(req.FormValue("process"), check.Equals, "web")
					c.Assert(req.FormValue("units"), check.Equals, "2")
					return strings.HasSuffix(req.URL.Path, "/apps/app1/units") && req.Method == http.MethodDelete
				},
				Transport: cmdtest.Transport{Message: "Failed to delete.", Status: http.StatusInternalServerError},
			},
		},
	}

	s.setupFakeTransport(transport)
	command := UnitSet{}
	command.Flags().Parse(true, []string{"-a", "app1", "-p", "web"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "Failed to delete.")
	c.Assert(calledGet, check.Equals, true)
	c.Assert(calledDelete, check.Equals, true)
}

func (s *S) TestUnitSetNoChanges(c *check.C) {
	var stdout, stderr bytes.Buffer
	var calledGet bool
	context := cmd.Context{
		Args:   []string{"3"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	resultGet := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","ProcessName":"web"},{"Ip":"9.9.9.9","ID":"app1/1","Status":"started","ProcessName":"web"},{"Ip":"","ID":"app1/2","Status":"pending","ProcessName":"web"},{"Ip":"8.8.8.8","ID":"app1/3","Status":"started","ProcessName":"worker"}],"teams":["tsuruteam","crane"],"owner":"myapp_owner","deploys":7,"router":"planb"}`
	transport := &cmdtest.ConditionalTransport{
		CondFunc: func(req *http.Request) bool {
			calledGet = true
			return strings.HasSuffix(req.URL.Path, "/apps/app1") && req.Method == http.MethodGet
		},
		Transport: cmdtest.Transport{Message: resultGet, Status: http.StatusOK},
	}

	s.setupFakeTransport(transport)
	command := UnitSet{}
	command.Flags().Parse(true, []string{"-a", "app1", "-p", "web"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(calledGet, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "The process web, version 0 already has 3 units.\n")
}

func (s *S) TestUnitSetFailedGet(c *check.C) {
	var stdout, stderr bytes.Buffer
	calledTimes := 0
	context := cmd.Context{
		Args:   []string{"3"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	transport := &cmdtest.ConditionalTransport{
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			return strings.HasSuffix(req.URL.Path, "/apps/app1") && req.Method == http.MethodGet
		},
		Transport: cmdtest.Transport{Message: "Failed to get.", Status: http.StatusInternalServerError},
	}

	s.setupFakeTransport(transport)
	command := UnitSet{}
	command.Flags().Parse(true, []string{"-a", "app1", "-p", "web"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, "Failed to get.")
	c.Assert(calledTimes, check.Equals, 1)
}

func (s *S) TestUnitSetNoProcessSpecifiedAndMultipleExist(c *check.C) {
	var stdout, stderr bytes.Buffer
	var calledGet bool
	context := cmd.Context{
		Args:   []string{"3"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	resultGet := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","ProcessName":"web"},{"Ip":"9.9.9.9","ID":"app1/1","Status":"started","ProcessName":"web"},{"Ip":"","ID":"app1/2","Status":"pending","ProcessName":"web"},{"Ip":"8.8.8.8","ID":"app1/3","Status":"started","ProcessName":"worker"}],"teams":["tsuruteam","crane"],"owner":"myapp_owner","deploys":7,"router":"planb"}`
	transport := &cmdtest.ConditionalTransport{
		CondFunc: func(req *http.Request) bool {
			calledGet = true
			return strings.HasSuffix(req.URL.Path, "/apps/app1") && req.Method == http.MethodGet
		},
		Transport: cmdtest.Transport{Message: resultGet, Status: http.StatusOK},
	}

	s.setupFakeTransport(transport)
	command := UnitSet{}
	command.Flags().Parse(true, []string{"-a", "app1"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "Please use the -p/--process flag to specify which process you want to set units for.")
	c.Assert(calledGet, check.Equals, true)
}

func (s *S) TestUnitSetNoProcessSpecifiedAndSingleExists(c *check.C) {
	var stdout, stderr bytes.Buffer
	var calledGet bool
	var calledPut bool
	context := cmd.Context{
		Args:   []string{"10"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	resultGet := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"","ID":"app1/2","Status":"pending","ProcessName":"worker"},{"Ip":"8.8.8.8","ID":"app1/3","Status":"started","ProcessName":"worker"}],"teams":["tsuruteam","crane"],"owner":"myapp_owner","deploys":7,"router":"planb"}`

	expectedOut := "-- added unit --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	resultPut, _ := json.Marshal(msg)

	transport := &cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				CondFunc: func(req *http.Request) bool {
					calledGet = true
					return strings.HasSuffix(req.URL.Path, "/apps/app1") && req.Method == http.MethodGet
				},
				Transport: cmdtest.Transport{Message: resultGet, Status: http.StatusOK},
			},
			{
				CondFunc: func(req *http.Request) bool {
					calledPut = true
					c.Assert(req.FormValue("process"), check.Equals, "worker")
					c.Assert(req.FormValue("units"), check.Equals, "8")
					return strings.HasSuffix(req.URL.Path, "/apps/app1/units") && req.Method == http.MethodPut
				},
				Transport: cmdtest.Transport{Message: string(resultPut), Status: http.StatusOK},
			},
		},
	}

	s.setupFakeTransport(transport)
	command := UnitSet{}
	command.Flags().Parse(true, []string{"-a", "app1"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(calledGet, check.Equals, true)
	c.Assert(calledPut, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestUnitSetInfo(c *check.C) {
	c.Assert((&UnitSet{}).Info(), check.NotNil)
}

func (s *S) TestUnitSetIsACommand(c *check.C) {
	var _ cmd.Command = &UnitSet{}
}

func (s *S) TestUnitKill(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"unit1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := cmdtest.Transport{
		Message: "",
		Status:  http.StatusOK,
	}
	s.setupFakeTransport(transport)
	command := UnitKill{}
	command.Flags().Parse(true, []string{"-a", "app1", "-f"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)

	stdout.Reset()
	stderr.Reset()

	context = cmd.Context{
		Args:   []string{"unit1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command = UnitKill{}
	command.Flags().Parse(true, []string{"-j", "job1", "-f"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
}

func (s *S) TestUnitKillMissingUnit(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := cmdtest.Transport{
		Message: "",
		Status:  http.StatusOK,
	}
	s.setupFakeTransport(transport)
	command := UnitKill{}
	command.Flags().Parse(true, []string{"-a", "app1", "-f"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "you must provide the unit name.")
}

func (s *S) TestUnitKillAppAndJobMutuallyExclusive(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"app1", "job1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	transport := cmdtest.Transport{
		Message: "",
		Status:  http.StatusOK,
	}
	s.setupFakeTransport(transport)
	command := UnitKill{}
	command.Flags().Parse(true, []string{"-a", "app1", "-j", "job1"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "please use only one of the -a/--app and -j/--job flags")
}
