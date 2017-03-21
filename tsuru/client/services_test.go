// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
)

type infoTransport struct {
	includePlans bool
	includeAll   bool
}

func (t *infoTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	var message string
	if strings.HasSuffix(req.URL.Path, "/services/mongodb") {
		message = `[{"Name":"mymongo", "Apps":["myapp"], "Id":0, "Info":{"key": "value", "key2": "value2"}, "PlanName":"small", "ServiceName":"mongoservice", "Teams":["mongoteam"]}]`
	}
	if strings.HasSuffix(req.URL.Path, "/services/mongodb/plans") {
		if t.includePlans {
			message = `[{"Name": "small", "Description": "another plan"}]`
		} else {
			message = `[]`
		}
	}
	if strings.HasSuffix(req.URL.Path, "/services/mongodbnoplan") {
		message = `[{"Name":"mymongo", "Apps":["myapp"], "Id":0, "Info":{"key": "value", "key2": "value2"}, "PlanName":"", "ServiceName":"noplanservice", "Teams":["noplanteam"]}]`
	}
	if strings.HasSuffix(req.URL.Path, "/services/mongodbnoplan/plans") {
		if t.includePlans {
			message = `[{"Name": "small", "Description": "another plan"}]`
		} else {
			message = `[]`
		}
	}

	if strings.HasSuffix(req.URL.Path, "/services/mongo") {
		message = `[{"Name":"mymongo", "Apps":["myapp"], "Id":0, "Info":{"key": "value", "key2": "value2"}, "PlanName":"small", "ServiceName":"mongoservice", "Teams":["mongoteam"]}]`
	}
	if strings.HasSuffix(req.URL.Path, "/services/mongo/plans") {
		if t.includePlans {
			message = `[{"Name": "small", "Description": "another plan"}]`
		} else {
			message = `[]`
		}
	}
	if strings.HasSuffix(req.URL.Path, "/services/mongo/doc") {
		message = `This is a test doc for a test service.
Service test is foo bar.
`
	}
	if strings.HasSuffix(req.URL.Path, "/services/mymongo/plans") {
		if t.includePlans {
			message = `[{"Name": "small", "Description": "another plan"}]`
		} else {
			message = `[]`
		}
	}
	if strings.HasSuffix(req.URL.Path, "/services/mymongo/instances/mongo") {
		if t.includeAll {
			message = `{"Apps": ["app", "app2"], "Teams": ["admin", "admin2"], "TeamOwner": "admin", "CustomInfo" : {"key4": "value8", "key2": "value9", "key3":"value3"},"Description": "description", "PlanName": "small", "PlanDescription": "another plan", "Tags": ["tag 1", "tag 2"]}`
		} else {
			message = `{"Apps": ["app", "app2"], "Teams": ["admin", "admin2"], "TeamOwner": "admin", "CustomInfo" : {},"Description": "", "PlanName": "", "PlanDescription": "", "Tags": []}`
		}
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
			return strings.HasSuffix(req.URL.Path, "/services/instances")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&ServiceList{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	table := stdout.String()
	c.Assert(table, check.Matches, "^"+expectedPrefix+".*")
	c.Assert(table, check.Matches, "^.*"+lineMysql+".*")
	c.Assert(table, check.Matches, "^.*"+lineOracle+".*")
}

func (s *S) TestServiceListWithEmptyResponse(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := ""
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/instances")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&ServiceList{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestServiceListInfo(c *check.C) {
	command := &ServiceList{}
	c.Assert(command.Info(), check.NotNil)
}

func (s *S) TestServiceListShouldBeCommand(c *check.C) {
	var _ cmd.Command = &ServiceList{}
}

func (s *S) TestServiceInstanceBind(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	ctx := cmd.Context{
		Args:   []string{"mysql", "my-mysql"},
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
			method := req.Method == "PUT"
			path := strings.HasSuffix(req.URL.Path, "/services/mysql/instances/my-mysql/g1")
			noRestart := req.FormValue("noRestart") == "true"
			return method && path && noRestart
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceBind{}
	command.Flags().Parse(true, []string{"-a", "g1", "--no-restart"})
	err = command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceInstanceBindWithoutFlag(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	ctx := cmd.Context{
		Args:   []string{"mysql", "my-mysql"},
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
			method := req.Method == "PUT"
			path := strings.HasSuffix(req.URL.Path, "/services/mysql/instances/my-mysql/ge")
			noRestart := req.FormValue("noRestart") == "false"
			return method && path && noRestart
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "ge"}
	err = (&ServiceInstanceBind{GuessingCommand: cmd.GuessingCommand{G: fake}}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceInstanceBindWithoutEnvironmentVariables(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"mysql", "my-mysql"},
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
			method := req.Method == "PUT"
			path := strings.HasSuffix(req.URL.Path, "/services/mysql/instances/my-mysql/g1")
			noRestart := req.FormValue("noRestart") == "false"
			return method && path && noRestart
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceBind{}
	command.Flags().Parse(true, []string{"-a", "g1"})
	err = command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceInstanceBindWithRequestFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"mysql", "my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.Transport{Message: "This user does not have access to this app.", Status: http.StatusForbidden}

	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceBind{}
	command.Flags().Parse(true, []string{"-a", "g1"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, trans.Message)
}

func (s *S) TestServiceInstanceBindInfo(c *check.C) {
	c.Assert((&ServiceInstanceBind{}).Info(), check.NotNil)
}

func (s *S) TestServiceInstanceBindIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &ServiceInstanceBind{}
}

func (s *S) TestServiceInstanceUnbind(c *check.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	ctx := cmd.Context{
		Args:   []string{"service", "hand"},
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
			return req.Method == http.MethodDelete && strings.HasSuffix(req.URL.Path, "/services/service/instances/hand/pocket") &&
				req.URL.RawQuery == "noRestart=true"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceUnbind{}
	command.Flags().Parse(true, []string{"-a", "pocket", "--no-restart"})
	err = command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceInstanceUnbindWithoutFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	ctx := cmd.Context{
		Args:   []string{"service", "hand"},
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
			return req.Method == http.MethodDelete && strings.HasSuffix(req.URL.Path, "/services/service/instances/hand/sleeve") &&
				req.URL.RawQuery == "noRestart=false"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "sleeve"}
	err = (&ServiceInstanceUnbind{GuessingCommand: cmd.GuessingCommand{G: fake}}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestServiceInstanceUnbindWithRequestFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"service", "hand"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.Transport{Message: "This app is not bound to this service.", Status: http.StatusPreconditionFailed}

	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceUnbind{}
	command.Flags().Parse(true, []string{"-a", "pocket"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, trans.Message)
}

func (s *S) TestServiceInstanceUnbindInfo(c *check.C) {
	c.Assert((&ServiceInstanceUnbind{}).Info(), check.NotNil)
}

func (s *S) TestServiceInstanceUnbindIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &ServiceInstanceUnbind{}
}

func (s *S) TestServiceInstanceAddInfo(c *check.C) {
	command := &ServiceInstanceAdd{}
	c.Assert(command.Info(), check.NotNil)
}

func (s *S) TestServiceInstanceAddRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := "Service successfully added.\n"
	args := []string{
		"mysql",
		"my_app_db",
		"small",
	}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			name := r.FormValue("name") == "my_app_db"
			plan := r.FormValue("plan") == "small"
			owner := r.FormValue("owner ") == ""
			description := r.FormValue("description") == ""
			method := r.Method == "POST"
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(r.URL.Path, "/services/mysql/instances")
			return method && url && name && owner && plan && description && contentType
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	err := (&ServiceInstanceAdd{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, result)
}

func (s *S) TestServiceInstanceAddFlags(c *check.C) {
	flagDesc := "the team that owns the service (mandatory if the user is member of more than one team)"
	command := ServiceInstanceAdd{}
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
	flagDesc = "service instance description"
	flagset.Parse(true, []string{"-d", "description"})
	assume = flagset.Lookup("description")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "description")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "description")
	c.Check(assume.DefValue, check.Equals, "")
	sassume = flagset.Lookup("d")
	c.Check(sassume, check.NotNil)
	c.Check(sassume.Name, check.Equals, "d")
	c.Check(sassume.Usage, check.Equals, flagDesc)
	c.Check(sassume.Value.String(), check.Equals, "description")
	c.Check(sassume.DefValue, check.Equals, "")
	c.Check(command.description, check.Equals, "description")
}

func (s *S) TestServiceInstanceUpdateInfo(c *check.C) {
	command := &ServiceInstanceUpdate{}
	c.Assert(command.Info(), check.NotNil)
}

func (s *S) TestServiceInstanceUpdateRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := "Service successfully updated.\n"
	args := []string{
		"service",
		"service-instance",
	}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			description := r.FormValue("description") == ""
			method := r.Method == "PUT"
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(r.URL.Path, "/services/service/instances/service-instance")
			return method && url && description && contentType
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	err := (&ServiceInstanceUpdate{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, result)
}

func (s *S) TestServiceInstanceUpdateFlags(c *check.C) {
	flagDesc := "service instance description"
	command := ServiceInstanceUpdate{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse(true, []string{"-d", "description"})
	assume := flagset.Lookup("description")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "description")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "description")
	c.Check(assume.DefValue, check.Equals, "")
	sassume := flagset.Lookup("d")
	c.Check(sassume, check.NotNil)
	c.Check(sassume.Name, check.Equals, "d")
	c.Check(sassume.Usage, check.Equals, flagDesc)
	c.Check(sassume.Value.String(), check.Equals, "description")
	c.Check(sassume.DefValue, check.Equals, "")
	c.Check(command.description, check.Equals, "description")
}

func (s *S) TestServiceInstanceStatusInfo(c *check.C) {
	got := (&ServiceInstanceStatus{}).Info()
	c.Assert(got, check.NotNil)
}

func (s *S) TestServiceInstanceStatusRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `Service instance "foo" is up`
	args := []string{"foo", "fooBar"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	err := (&ServiceInstanceStatus{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	obtained = strings.Replace(obtained, "\n", "", -1)
	c.Assert(obtained, check.Equals, result)
}

func (s *S) TestServiceInfoInfo(c *check.C) {
	got := (&ServiceInfo{}).Info()
	c.Assert(got, check.NotNil)
}

func (s *S) TestServiceInfoExtraHeaders(c *check.C) {
	result := []byte(`[{"Name":"mymongo", "Apps":["myapp"], "Info":{"key": "value", "key2": "value2"}}]`)
	var instances []ServiceInstanceModel
	json.Unmarshal(result, &instances)
	expected := []string{"key", "key2"}
	headers := (&ServiceInfo{}).ExtraHeaders(instances)
	c.Assert(headers, check.DeepEquals, expected)
}

func (s *S) TestServiceInstanceInfoRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Service: mymongo
Instance: mongo
Apps: app, app2
Teams: admin, admin2
Team Owner: admin
Description: description
Tags: tag 1, tag 2
Plan: small
Plan description: another plan

Custom Info for "mongo"
key2:
value9

key3:
value3

key4:
value8
`
	args := []string{"mymongo", "mongo"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &infoTransport{includeAll: true}}, nil, manager)
	err := (&ServiceInstanceInfo{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceInstanceInfoRunWithoutPlansAndCustomInfoAndDescription(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Service: mymongo
Instance: mongo
Apps: app, app2
Teams: admin, admin2
Team Owner: admin
Description:
Tags:
Plan:
Plan description:
`
	args := []string{"mymongo", "mongo"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &infoTransport{includeAll: false}}, nil, manager)
	err := (&ServiceInstanceInfo{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := strings.Replace(stdout.String(), " \n", "\n", -1)
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceInfoRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Info for "mongodb"

Instances
+-----------+-------+-------+-------+--------+
| Instances | Plan  | Apps  | key   | key2   |
+-----------+-------+-------+-------+--------+
| mymongo   | small | myapp | value | value2 |
+-----------+-------+-------+-------+--------+

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
	client := cmd.NewClient(&http.Client{Transport: &infoTransport{includePlans: true}}, nil, manager)
	err := (&ServiceInfo{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceInfoNoPlans(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Info for "mongodbnoplan"

Instances
+-----------+-------+-------+--------+
| Instances | Apps  | key   | key2   |
+-----------+-------+-------+--------+
| mymongo   | myapp | value | value2 |
+-----------+-------+-------+--------+
`
	args := []string{"mongodbnoplan"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &infoTransport{includePlans: false}}, nil, manager)
	err := (&ServiceInfo{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceInfoWithDoc(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Info for "mongo"

Instances
+-----------+-------+-------+-------+--------+
| Instances | Plan  | Apps  | key   | key2   |
+-----------+-------+-------+-------+--------+
| mymongo   | small | myapp | value | value2 |
+-----------+-------+-------+-------+--------+

Plans
+-------+--------------+
| Name  | Description  |
+-------+--------------+
| small | another plan |
+-------+--------------+

Documentation:
This is a test doc for a test service.
Service test is foo bar.
`
	args := []string{"mongo"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &infoTransport{includePlans: true}}, nil, manager)
	err := (&ServiceInfo{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceInstanceRemoveInfo(c *check.C) {
	i := (&ServiceInstanceRemove{}).Info()
	c.Assert(i, check.NotNil)
}

func (s *S) TestServiceInstanceRemoveRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"some-service-name", "some-service-instance"},
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
			return strings.HasSuffix(r.URL.Path, "/services/some-service-name/instances/some-service-instance") &&
				r.Method == http.MethodDelete
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	err := (&ServiceInstanceRemove{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceInstanceRemoveWithoutAsking(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Service "ble" successfully removed!` + "\n"
	context := cmd.Context{
		Args:   []string{"service", "ble"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	command := ServiceInstanceRemove{}
	command.Flags().Parse(true, []string{"-y"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestServiceInstanceRemoveFlags(c *check.C) {
	command := ServiceInstanceRemove{}
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

func (s *S) TestServiceInstanceRemoveUnbindFlag(c *check.C) {
	command := ServiceInstanceRemove{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse(true, []string{"-u"})
	assume := flagset.Lookup("unbind")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "unbind")
	c.Check(assume.Usage, check.Equals, "Don't ask for confirmation, just remove all applications bound.")
	c.Check(assume.Value.String(), check.Equals, "true")
	c.Check(assume.DefValue, check.Equals, "false")
	sassume := flagset.Lookup("u")
	c.Check(sassume, check.NotNil)
	c.Check(sassume.Name, check.Equals, "u")
	c.Check(sassume.Usage, check.Equals, "Don't ask for confirmation, just remove all applications bound.")
	c.Check(sassume.Value.String(), check.Equals, "true")
	c.Check(sassume.DefValue, check.Equals, "false")
	c.Check(command.yesUnbind, check.Equals, true)
}

func (s *S) TestServiceInstanceRemoveWithAppBindNoUnbind(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Are you sure you want to remove service "mongodb"? (y/n) `
	expected += `Applications bound to the service "mongodb": "app1,app2"` + "\n"
	expected += `Do you want unbind all apps? (y/n) `
	expected += `Abort.` + "\n"
	ctx := cmd.Context{
		Args:   []string{"some-service-name", "mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\tn\n"),
	}

	expectedError := "This service instance is bound to at least one app. Unbind them before removing it"
	expectedMsg := "app1,app2"
	msg1 := io.SimpleJsonMessage{Message: expectedMsg, Error: expectedError}
	result, err := json.Marshal(msg1)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/some-service-name/instances/mongodb") &&
				req.Method == http.MethodDelete
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = (&ServiceInstanceRemove{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected)
}

func (s *S) TestServiceInstanceRemoveWithAppBindYesUnbind(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Are you sure you want to remove service "mongodb"? (y/n) `
	expected += `Applications bound to the service "mongodb": "app1,app2"` + "\n"
	expected += `Do you want unbind all apps? (y/n) `
	expected2 := `Service "mongodb" successfully removed!` + "\n"
	ctx := cmd.Context{
		Args:   []string{"some-service-name", "mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\ty\n"),
	}
	expectedError := "This service instance is bound to at least one app. Unbind them before removing it"
	expectedMsg := "app1,app2"
	msg := io.SimpleJsonMessage{Message: expectedMsg, Error: expectedError}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	instanceTransport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/some-service-name/instances/mongodb") &&
				req.Method == http.MethodDelete
		},
	}
	expectedOut1 := "-- mongodb removed --"
	msg1 := io.SimpleJsonMessage{Message: expectedOut1}
	result, err = json.Marshal(msg1)
	c.Assert(err, check.IsNil)
	appTransport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == http.MethodDelete &&
				strings.HasSuffix(req.URL.Path, "/services/some-service-name/instances/mongodb") &&
				req.URL.RawQuery == "unbindall=true"
		},
	}
	trans := &cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{instanceTransport, appTransport},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = (&ServiceInstanceRemove{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected+expectedOut1+expected2)
}

func (s *S) TestServiceInstanceRemoveWithAppBindWithFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Service "mongodb" successfully removed!` + "\n"
	expectedOut := "-- service remove --"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	ctx := cmd.Context{
		Args:   []string{"service-name", "mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/service-name/instances/mongodb") && req.Method == http.MethodDelete && req.URL.RawQuery == "unbindall=true"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceRemove{}
	command.Flags().Parse(true, []string{"-y", "-u"})
	err = command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expectedOut+expected)
}

func (s *S) TestServiceInstanceRemoveWithAppBindShowAppsBound(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Are you sure you want to remove service "mongodb"? (y/n) `
	expected += `Applications bound to the service "mongodb": "app1,app2"` + "\n"
	expected += `Do you want unbind all apps? (y/n) `
	expected2 := `Service "mongodb" successfully removed!` + "\n"
	ctx := cmd.Context{
		Args:   []string{"service-name", "mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\ty\n"),
	}
	expectedError := "This service instance is bound to at least one app. Unbind them before removing it"
	expectedMsg := "app1,app2"
	msg := io.SimpleJsonMessage{Message: expectedMsg, Error: expectedError}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	instanceTransport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/service-name/instances/mongodb") && req.Method == http.MethodDelete
		},
	}
	expectedOut1 := "-- mongodb removed --"
	msg1 := io.SimpleJsonMessage{Message: expectedOut1}
	result, err = json.Marshal(msg1)
	c.Assert(err, check.IsNil)
	appTransport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.Method == http.MethodDelete && strings.HasSuffix(req.URL.Path, "/services/service-name/instances/mongodb") && req.URL.RawQuery == "unbindall=true"
		},
	}
	trans := &cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{instanceTransport, appTransport},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = (&ServiceInstanceRemove{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, expected+expectedOut1+expected2)
}

func (s *S) TestServiceInstanceGrantInfo(c *check.C) {
	info := (&ServiceInstanceGrant{}).Info()
	c.Assert(info, check.NotNil)
}

func (s *S) TestServiceInstanceGrantRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	command := ServiceInstanceGrant{}
	ctx := cmd.Context{
		Args:   []string{"test-service", "test-service-instance", "team"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transp := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			path := "/services/test-service/instances/permission/test-service-instance/team"
			return strings.HasSuffix(r.URL.Path, path) && "PUT" == r.Method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transp}, nil, manager)
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestServiceInstanceRevokeInfo(c *check.C) {
	info := (&ServiceInstanceRevoke{}).Info()
	c.Assert(info, check.NotNil)
}

func (s *S) TestServiceInstanceRevokeRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	command := ServiceInstanceRevoke{}
	ctx := cmd.Context{
		Args:   []string{"test-service", "test-service-instance", "team"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transp := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			path := "/services/test-service/instances/permission/test-service-instance/team"
			return strings.HasSuffix(r.URL.Path, path) && http.MethodDelete == r.Method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transp}, nil, manager)
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}
