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
	//"fmt"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
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

func (s *S) TestServiceList(c *check.C) {
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
	c.Assert(err, check.IsNil)
	table := stdout.String()
	c.Assert(table, check.Matches, "^"+expectedPrefix+".*")
	c.Assert(table, check.Matches, "^.*"+lineMysql+".*")
	c.Assert(table, check.Matches, "^.*"+lineOracle+".*")
}

func (s *S) TestServiceListWithEmptyResponse(c *check.C) {
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
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestInfoServiceList(c *check.C) {
	command := &serviceList{}
	c.Assert(command.Info(), check.NotNil)
}

func (s *S) TestServiceListShouldBeCommand(c *check.C) {
	var _ cmd.Command = &serviceList{}
}

func (s *S) TestServiceBind(c *check.C) {
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
	c.Assert(err, check.IsNil)
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
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceBindWithoutFlag(c *check.C) {
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
	c.Assert(err, check.IsNil)
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
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceBindWithoutEnvironmentVariables(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := `something`
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
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
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceBindWithRequestFailure(c *check.C) {
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
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, trans.Message)
}

func (s *S) TestServiceBindInfo(c *check.C) {
	c.Assert((&serviceBind{}).Info(), check.NotNil)
}

func (s *S) TestServiceBindIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &serviceBind{}
}

func (s *S) TestServiceUnbind(c *check.C) {
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
	c.Assert(err, check.IsNil)
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
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceUnbindWithoutFlag(c *check.C) {
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
	c.Assert(err, check.IsNil)
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
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceUnbindWithRequestFailure(c *check.C) {
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
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, trans.Message)
}

func (s *S) TestServiceUnbindInfo(c *check.C) {
	c.Assert((&serviceUnbind{}).Info(), check.NotNil)
}

func (s *S) TestServiceUnbindIsAFlaggedComand(c *check.C) {
	var _ cmd.FlaggedCommand = &serviceUnbind{}
}

func (s *S) TestServiceAddInfo(c *check.C) {
	command := &serviceAdd{}
	c.Assert(command.Info(), check.NotNil)
}

func (s *S) TestServiceAddRun(c *check.C) {
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
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, result)
}

func (s *S) TestServiceAddFlags(c *check.C) {
	flagDesc := "the team that owns the service (mandatory if the user is member of more than one team)"
	command := serviceAdd{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse(true, []string{"-t", "wat"})
	assume := flagset.Lookup("team-owner")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "team-owner")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "wat")
	c.Check(assume.DefValue, check.Equals, "")
	sassume := flagset.Lookup("t")
	c.Check(sassume, check.NotNil)
	c.Check(sassume.Name, check.Equals, "t")
	c.Check(sassume.Usage, check.Equals, flagDesc)
	c.Check(sassume.Value.String(), check.Equals, "wat")
	c.Check(sassume.DefValue, check.Equals, "")
	c.Check(command.teamOwner, check.Equals, "wat")
}

func (s *S) TestServiceInstanceStatusInfo(c *check.C) {
	got := (&serviceInstanceStatus{}).Info()
	c.Assert(got, check.NotNil)
}

func (s *S) TestServiceInstanceStatusRun(c *check.C) {
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
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	obtained = strings.Replace(obtained, "\n", "", -1)
	c.Assert(obtained, check.Equals, result)
}

func (s *S) TestServiceInfoInfo(c *check.C) {
	got := (&serviceInfo{}).Info()
	c.Assert(got, check.NotNil)
}

func (s *S) TestServiceInfoExtraHeaders(c *check.C) {
	result := []byte(`[{"Name":"mymongo", "Apps":["myapp"], "Info":{"key": "value", "key2": "value2"}}]`)
	var instances []ServiceInstanceModel
	json.Unmarshal(result, &instances)
	expected := []string{"key", "key2"}
	headers := (&serviceInfo{}).ExtraHeaders(instances)
	c.Assert(headers, check.DeepEquals, expected)
}

func (s *S) TestServiceInfoRun(c *check.C) {
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
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceDocInfo(c *check.C) {
	i := (&serviceDoc{}).Info()
	c.Assert(i, check.NotNil)
}

func (s *S) TestServiceDocRun(c *check.C) {
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
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceRemoveInfo(c *check.C) {
	i := (&serviceRemove{}).Info()
	c.Assert(i, check.NotNil)
}

func (s *S) TestServiceRemoveRun(c *check.C) {
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
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceRemoveWithoutAsking(c *check.C) {
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
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestServiceRemoveFlags(c *check.C) {
	command := serviceRemove{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse(true, []string{"-y"})
	assume := flagset.Lookup("assume-yes")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "assume-yes")
	c.Check(assume.Usage, check.Equals, "Don't ask for confirmation, just remove the service.")
	c.Check(assume.Value.String(), check.Equals, "true")
	c.Check(assume.DefValue, check.Equals, "false")
	sassume := flagset.Lookup("y")
	c.Check(sassume, check.NotNil)
	c.Check(sassume.Name, check.Equals, "y")
	c.Check(sassume.Usage, check.Equals, "Don't ask for confirmation, just remove the service.")
	c.Check(sassume.Value.String(), check.Equals, "true")
	c.Check(sassume.DefValue, check.Equals, "false")
	c.Check(command.yes, check.Equals, true)
}

func (s *S) TestServiceRemoveWithAppBindNoUnbind(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Are you sure you want to remove service "mongodb"? (y/n) `
	expected += `Do you want unbind all apps? (y/n) `
	expected += `This service instance is bound to at least one app. Unbind them before removing it` + "\n"
	ctx := cmd.Context{
		Args:   []string{"mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\tn\n"),
	}
	msg := "This service instance is bound to at least one app. Unbind them before removing it"
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: msg, Status: http.StatusConflict},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/services/instances/mongodb" && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&serviceRemove{}).Run(&ctx, client)
	c.Assert(err, check.NotNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceRemoveWithAppBindYesUnbind(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Are you sure you want to remove service "mongodb"? (y/n) `
	expected += `Do you want unbind all apps? (y/n) `
	expected += `Service "mongodb" successfully removed!` + "\n"
	ctx := cmd.Context{
		Args:   []string{"mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\ty\n"),
	}
	msg := "This service instance is bound to at least one app. Unbind them before removing it"
	instanceTransport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: msg, Status: http.StatusConflict},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/services/instances/mongodb" && req.Method == "DELETE"
		},
	}
	msg1 := `Service "mongodb" successfully removed!` + "\n"
	appTransport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: msg1, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "DELETE" && req.URL.Path == "/services/instances/mongodb"
		},
	}
	trans := &cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{instanceTransport, appTransport},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&serviceRemove{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceRemoveWithAppBindWithFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Service "mongodb" successfully removed!` + "\n"
	ctx := cmd.Context{
		Args:   []string{"mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	msg := "Service mongodb successfully removed!"
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: msg, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/services/instances/mongodb" && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := serviceRemove{}
	command.Flags().Parse(true, []string{"-y", "-u"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceInstanceGrantInfo(c *check.C) {
	info := (&serviceInstanceGrant{}).Info()
	c.Assert(info, check.NotNil)
}

func (s *S) TestServiceInstanceGrantRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	command := serviceInstanceGrant{}
	ctx := cmd.Context{
		Args:   []string{"test-service-instance", "team"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transp := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			return "/services/instances/permission/test-service-instance/team" == r.URL.Path &&
				"PUT" == r.Method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transp}, nil, manager)
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestServiceInstanceRevokeInfo(c *check.C) {
	info := (&serviceInstanceRevoke{}).Info()
	c.Assert(info, check.NotNil)
}

func (s *S) TestServiceInstanceRevokeRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	command := serviceInstanceRevoke{}
	ctx := cmd.Context{
		Args:   []string{"test-service-instance", "team"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transp := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			return "/services/instances/permission/test-service-instance/team" == r.URL.Path &&
				"DELETE" == r.Method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transp}, nil, manager)
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}
