// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"launchpad.net/gocheck"
)

type infoTransport struct{}

func (t *infoTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	var message string
	if req.URL.Path == "/services/mongodb" {
		message = `[{"Name":"mymongo", "Apps":["myapp"], "Info":{"key": "value", "key2": "value2"}}]`
	}
	if req.URL.Path == "/services/mongodb/plans" {
		message = `[{"Name": "small", "Description": "another plan"}]`
	}
	resp = &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString(message)),
		StatusCode: http.StatusOK,
	}
	return resp, nil
}

func (s *S) TestServiceList(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	output := `[{"service": "mysql", "instances": ["mysql01", "mysql02"]}, {"service": "oracle", "instances": []}]`
	expectedPrefix := `+---------+------------------+
| Service | Instances        |`
	lineMysql := "| mysql   | mysql01, mysql02 |"
	lineOracle := "| oracle  |                  |"
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: output, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/services/instances"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&serviceList{}).Run(&ctx, client)
	c.Assert(err, gocheck.IsNil)
	table := stdout.String()
	c.Assert(table, gocheck.Matches, "^"+expectedPrefix+".*")
	c.Assert(table, gocheck.Matches, "^.*"+lineMysql+".*")
	c.Assert(table, gocheck.Matches, "^.*"+lineOracle+".*")
}

func (s *S) TestServiceListWithEmptyResponse(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	output := "[]"
	expected := ""
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: output, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/services/instances"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&serviceList{}).Run(&ctx, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestInfoServiceList(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "service-list",
		Usage:   "service-list",
		Desc:    "Get all available services, and user's instances for this services",
		MinArgs: 0,
	}
	command := &serviceList{}
	c.Assert(command.Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestServiceListShouldBeCommand(c *gocheck.C) {
	var _ cmd.Command = &serviceList{}
}

func (s *S) TestServiceBind(c *gocheck.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	ctx := cmd.Context{
		Args:   []string{"my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := `["DATABASE_HOST","DATABASE_USER","DATABASE_PASSWORD"]`
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "PUT" && req.URL.Path == "/services/instances/my-mysql/g1"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := serviceBind{}
	command.Flags().Parse(true, []string{"-a", "g1"})
	err = command.Run(&ctx, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestServiceBindWithoutFlag(c *gocheck.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	ctx := cmd.Context{
		Args:   []string{"my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := `["DATABASE_HOST","DATABASE_USER","DATABASE_PASSWORD"]`
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: string(result),
			Status:  http.StatusOK,
		},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "PUT" && req.URL.Path == "/services/instances/my-mysql/ge"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "ge"}
	err = (&serviceBind{cmd.GuessingCommand{G: fake}}).Run(&ctx, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestServiceBindWithoutEnvironmentVariables(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := `something`
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "PUT" && req.URL.Path == "/services/instances/my-mysql/g1"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := serviceBind{}
	command.Flags().Parse(true, []string{"-a", "g1"})
	err = command.Run(&ctx, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestServiceBindWithRequestFailure(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.Transport{Message: "This user does not have access to this app.", Status: http.StatusForbidden}

	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := serviceBind{}
	command.Flags().Parse(true, []string{"-a", "g1"})
	err := command.Run(&ctx, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, trans.Message)
}

func (s *S) TestServiceBindInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:  "service-bind",
		Usage: "service-bind <instancename> [-a/--app appname]",
		Desc: `bind a service instance to an app

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
	c.Assert((&serviceBind{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestServiceBindIsAFlaggedCommand(c *gocheck.C) {
	var _ cmd.FlaggedCommand = &serviceBind{}
}

func (s *S) TestServiceUnbind(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	ctx := cmd.Context{
		Args:   []string{"hand"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := `something`
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && req.URL.Path == "/services/instances/hand/pocket"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := serviceUnbind{}
	command.Flags().Parse(true, []string{"-a", "pocket"})
	err = command.Run(&ctx, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestServiceUnbindWithoutFlag(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	ctx := cmd.Context{
		Args:   []string{"hand"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := `something`
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && req.URL.Path == "/services/instances/hand/sleeve"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "sleeve"}
	err = (&serviceUnbind{cmd.GuessingCommand{G: fake}}).Run(&ctx, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}

func (s *S) TestServiceUnbindWithRequestFailure(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"hand"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.Transport{Message: "This app is not bound to this service.", Status: http.StatusPreconditionFailed}

	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := serviceUnbind{}
	command.Flags().Parse(true, []string{"-a", "pocket"})
	err := command.Run(&ctx, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, trans.Message)
}

func (s *S) TestServiceUnbindInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:  "service-unbind",
		Usage: "service-unbind <instancename> [-a/--app appname]",
		Desc: `unbind a service instance from an app

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 1,
	}
	c.Assert((&serviceUnbind{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestServiceUnbindIsAFlaggedComand(c *gocheck.C) {
	var _ cmd.FlaggedCommand = &serviceUnbind{}
}

func (s *S) TestServiceAddInfo(c *gocheck.C) {
	usage := `service-add <servicename> <serviceinstancename> [plan] [-t/--owner-team <team>]
e.g.:

    $ tsuru service-add mongodb tsuru_mongodb small -t myteam

Will add a new instance of the "mongodb" service, named "tsuru_mongodb" with the plan "small".`
	expected := &cmd.Info{
		Name:    "service-add",
		Usage:   usage,
		Desc:    "Create a service instance to one or more apps make use of.",
		MinArgs: 2,
		MaxArgs: 3,
	}
	command := &serviceAdd{}
	c.Assert(command.Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestServiceAddRun(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	result := "Service successfully added.\n"
	args := []string{
		"my_app_db",
		"mysql",
		"small",
	}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	err := (&serviceAdd{}).Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, gocheck.Equals, result)
}

func (s *S) TestServiceAddFlags(c *gocheck.C) {
	flagDesc := "the team that owns te service (mandatory if the user is member of more than one team)"
	command := serviceAdd{}
	flagset := command.Flags()
	c.Assert(flagset, gocheck.NotNil)
	flagset.Parse(true, []string{"-t", "wat"})
	assume := flagset.Lookup("team-owner")
	c.Check(assume, gocheck.NotNil)
	c.Check(assume.Name, gocheck.Equals, "team-owner")
	c.Check(assume.Usage, gocheck.Equals, flagDesc)
	c.Check(assume.Value.String(), gocheck.Equals, "wat")
	c.Check(assume.DefValue, gocheck.Equals, "")
	sassume := flagset.Lookup("t")
	c.Check(sassume, gocheck.NotNil)
	c.Check(sassume.Name, gocheck.Equals, "t")
	c.Check(sassume.Usage, gocheck.Equals, flagDesc)
	c.Check(sassume.Value.String(), gocheck.Equals, "wat")
	c.Check(sassume.DefValue, gocheck.Equals, "")
	c.Check(command.teamOwner, gocheck.Equals, "wat")
}

func (s *S) TestServiceInstanceStatusInfo(c *gocheck.C) {
	usg := `service-status <serviceinstancename>
e.g.:

    $ tsuru service-status my_mongodb
`
	expected := &cmd.Info{
		Name:    "service-status",
		Usage:   usg,
		Desc:    "Check status of a given service instance.",
		MinArgs: 1,
	}
	got := (&serviceInstanceStatus{}).Info()
	c.Assert(got, gocheck.DeepEquals, expected)
}

func (s *S) TestServiceInstanceStatusRun(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	result := `Service instance "foo" is up`
	args := []string{"fooBar"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	err := (&serviceInstanceStatus{}).Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	obtained := stdout.String()
	obtained = strings.Replace(obtained, "\n", "", -1)
	c.Assert(obtained, gocheck.Equals, result)
}

func (s *S) TestServiceInfoInfo(c *gocheck.C) {
	usg := `service-info <service>
e.g.:

    $ tsuru service-info mongodb
`
	expected := &cmd.Info{
		Name:    "service-info",
		Usage:   usg,
		Desc:    "List all instances of a service",
		MinArgs: 1,
	}
	got := (&serviceInfo{}).Info()
	c.Assert(got, gocheck.DeepEquals, expected)
}

func (s *S) TestServiceInfoExtraHeaders(c *gocheck.C) {
	result := []byte(`[{"Name":"mymongo", "Apps":["myapp"], "Info":{"key": "value", "key2": "value2"}}]`)
	var instances []ServiceInstanceModel
	json.Unmarshal(result, &instances)
	expected := []string{"key", "key2"}
	headers := (&serviceInfo{}).ExtraHeaders(instances)
	c.Assert(headers, gocheck.DeepEquals, expected)
}

func (s *S) TestServiceInfoRun(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	expected := `Info for "mongodb"

Instances
+-----------+-------+-------+--------+
| Instances | Apps  | key   | key2   |
+-----------+-------+-------+--------+
| mymongo   | myapp | value | value2 |
+-----------+-------+-------+--------+

Plans
+-------+--------------+
| Name  | Description  |
+-------+--------------+
| small | another plan |
+-------+--------------+
`
	args := []string{"mongodb"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &infoTransport{}}, nil, manager)
	err := (&serviceInfo{}).Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, gocheck.Equals, expected)
}

func (s *S) TestServiceDocInfo(c *gocheck.C) {
	i := (&serviceDoc{}).Info()
	expected := &cmd.Info{
		Name:    "service-doc",
		Usage:   "service-doc <servicename>",
		Desc:    "Show documentation of a service",
		MinArgs: 1,
	}
	c.Assert(i, gocheck.DeepEquals, expected)
}

func (s *S) TestServiceDocRun(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	result := `This is a test doc for a test service.
Service test is foo bar.
`
	expected := result
	ctx := cmd.Context{
		Args:   []string{"foo"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: result,
			Status:  http.StatusOK,
		},
		CondFunc: func(r *http.Request) bool {
			return r.Method == "GET" && r.URL.Path == "/services/foo/doc"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	err := (&serviceDoc{}).Run(&ctx, client)
	c.Assert(err, gocheck.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, gocheck.Equals, expected)
}

func (s *S) TestServiceRemoveInfo(c *gocheck.C) {
	i := (&serviceRemove{}).Info()
	expected := &cmd.Info{
		Name:    "service-remove",
		Usage:   "service-remove <serviceinstancename> [--assume-yes]",
		Desc:    "Removes a service instance",
		MinArgs: 1,
	}
	c.Assert(i, gocheck.DeepEquals, expected)
}

func (s *S) TestServiceRemoveRun(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"some-service-instance"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	expected := `Are you sure you want to remove service "some-service-instance"? (y/n) `
	expected += `Service "some-service-instance" successfully removed!` + "\n"
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusOK,
		},
		CondFunc: func(r *http.Request) bool {
			return r.URL.Path == "/services/instances/some-service-instance" &&
				r.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	err := (&serviceRemove{}).Run(&ctx, client)
	c.Assert(err, gocheck.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, gocheck.Equals, expected)
}

func (s *S) TestServiceRemoveWithoutAsking(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	expected := `Service "ble" successfully removed!` + "\n"
	context := cmd.Context{
		Args:   []string{"ble"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	command := serviceRemove{}
	command.Flags().Parse(true, []string{"ble", "-y"})
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestServiceRemoveFlags(c *gocheck.C) {
	command := serviceRemove{}
	flagset := command.Flags()
	c.Assert(flagset, gocheck.NotNil)
	flagset.Parse(true, []string{"-y"})
	assume := flagset.Lookup("assume-yes")
	c.Check(assume, gocheck.NotNil)
	c.Check(assume.Name, gocheck.Equals, "assume-yes")
	c.Check(assume.Usage, gocheck.Equals, "Don't ask for confirmation, just remove the service.")
	c.Check(assume.Value.String(), gocheck.Equals, "true")
	c.Check(assume.DefValue, gocheck.Equals, "false")
	sassume := flagset.Lookup("y")
	c.Check(sassume, gocheck.NotNil)
	c.Check(sassume.Name, gocheck.Equals, "y")
	c.Check(sassume.Usage, gocheck.Equals, "Don't ask for confirmation, just remove the service.")
	c.Check(sassume.Value.String(), gocheck.Equals, "true")
	c.Check(sassume.DefValue, gocheck.Equals, "false")
	c.Check(command.yes, gocheck.Equals, true)
}
