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
	"github.com/tsuru/tsuru/service"
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
	if strings.HasSuffix(req.URL.Path, "/services/mongodb-broker") {
		message = `[{"Name":"mymongo", "Apps":["myapp"], "Id":0, "Info":{"key": "value", "key2": "value2"}, "PlanName":"small", "ServiceName":"mongoservice", "Teams":["mongoteam"]}]`
	}
	if strings.HasSuffix(req.URL.Path, "/services/multicluster") {
		message = `[{"Name":"mymongo", "Apps":["myapp"], "Id":0, "Info":{"key": "value", "key2": "value2"}, "PlanName":"small", "ServiceName":"mongoservice", "Teams":["mongoteam"],"Pool":"my-pool-01"}]`
	}
	if strings.HasSuffix(req.URL.Path, "/services/mongodb-broker/plans") {
		if t.includePlans {
			message = `[{"Name":"default","Description":"Plan with parameter and response schemas","Schemas":{"service_instance":{"create":{"parameters":{"$schema":"http://json-schema.org/draft-04/schema#", "required": ["param-2"], "properties":{"param-1":{"description":"First input parameter","type":"string", "default":"value1"},"param-2":{"description":"Second input parameter","type":"string"}},"type":"object"}},"update":{"parameters":{"$schema":"http://json-schema.org/draft-04/schema#","properties":{"param-1":{"description":"First input parameter","type":"string"},"param-2":{"description":"Second input parameter","type":"string"}},"type":"object"}}},"service_binding":{"create":{"parameters":{"$schema":"http://json-schema.org/draft-04/schema#","properties":{"param-1":{"description":"First input parameter","type":"string"},"param-2":{"description":"Second input parameter","type":"string"}},"type":"object"}}}}}]`
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
	if strings.HasSuffix(req.URL.Path, "/services/multicluster/plans") {
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
			message = `{"Apps": ["app", "app2"], "Teams": ["admin", "admin2"], "TeamOwner": "admin", "CustomInfo" : {"key4": "value8", "key2": "value9", "key3":"value3"},"Description": "description", "PlanName": "small", "PlanDescription": "another plan", "Tags": ["tag 1", "tag 2"], "Parameters": {"param1": "{\"some\": \"custom-data\"}", "param2": "value2", "param3": 3}, "Pool": "my-pool"}`
		} else {
			message = `{"Apps": ["app", "app2"], "Teams": ["admin", "admin2"], "TeamOwner": "admin", "CustomInfo" : {},"Description": "", "PlanName": "", "PlanDescription": "", "Tags": []}`
		}
	}
	if strings.HasSuffix(req.URL.Path, "/status") {
		message = `Service instance "mongo" is up`
	}
	resp = &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString(message)),
		StatusCode: http.StatusOK,
	}
	return resp, nil
}

func (s *S) TestServiceList(c *check.C) {
	var stdout, stderr bytes.Buffer
	output, err := json.Marshal([]service.ServiceModel{
		{
			Service: "mysql",
			ServiceInstances: []service.ServiceInstance{
				{
					Name: "mysql01",
				},
				{
					Name: "mysql02",
				},
			},
		},
		{
			Service:          "oracle",
			ServiceInstances: []service.ServiceInstance{},
		},
	})
	c.Assert(err, check.IsNil)
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(output), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/instances")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = (&ServiceList{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	table := stdout.String()

	c.Assert(table, check.Equals, `+----------+-----------+
| Services | Instances |
+----------+-----------+
| mysql    | mysql01   |
| mysql    | mysql02   |
| oracle   |           |
+----------+-----------+
`)

}

func (s *S) TestServiceListWithPool(c *check.C) {
	var stdout, stderr bytes.Buffer
	output, err := json.Marshal([]service.ServiceModel{
		{
			Service: "mysql",
			ServiceInstances: []service.ServiceInstance{
				{
					Name: "mysql01",
					Pool: "cluster-pool-01",
				},
				{
					Name: "mysql02",
					Pool: "cluster-pool-02",
				},
			},
		},
		{
			Service:          "oracle",
			ServiceInstances: []service.ServiceInstance{},
		},
	})
	c.Assert(err, check.IsNil)

	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(output), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/instances")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = (&ServiceList{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	table := stdout.String()

	c.Assert(table, check.Equals, `+----------+-----------+-----------------+
| Services | Instances | Pool            |
+----------+-----------+-----------------+
| mysql    | mysql01   | cluster-pool-01 |
| mysql    | mysql02   | cluster-pool-02 |
| oracle   |           |                 |
+----------+-----------+-----------------+
`)
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
			var bindResult map[string]interface{}
			err := json.NewDecoder(req.Body).Decode(&bindResult)
			c.Assert(err, check.IsNil)
			c.Assert(bindResult, check.DeepEquals, map[string]interface{}{
				"noRestart": true,
			})
			return method && path
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
			var bindResult map[string]interface{}
			err := json.NewDecoder(req.Body).Decode(&bindResult)
			c.Assert(err, check.IsNil)
			c.Assert(bindResult, check.DeepEquals, map[string]interface{}{})
			return method && path
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
	c.Assert(err.Error(), check.Equals, "403 Forbidden: "+trans.Message)
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
			c.Assert(req.URL.Query().Get("noRestart"), check.Equals, "true")
			c.Assert(req.URL.Query().Get("force"), check.Equals, "true")
			c.Assert(req.URL.Path, check.Equals, "/1.0/services/service/instances/hand/pocket")
			c.Assert(req.Method, check.Equals, http.MethodDelete)
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceUnbind{}
	command.Flags().Parse(true, []string{"-a", "pocket", "--no-restart", "--force"})
	err = command.Run(&ctx, client)
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
	result := "Service instance successfully added.\nFor additional information use: tsuru service instance info mysql my_app_db\n"
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
			var result map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&result)
			c.Assert(err, check.IsNil)
			c.Assert(result, check.DeepEquals, map[string]interface{}{
				"name":        "my_app_db",
				"plan_name":   "small",
				"team_owner":  "my team",
				"description": "desc",
				"tags": []interface{}{
					"my tag 1", "my tag 2"},
				"parameters": map[string]interface{}{
					"param1": "value1",
					"param2": "value2",
				},
				"pool": "pool-one",
			})

			c.Assert(r.Method, check.DeepEquals, "POST")
			c.Assert(r.Header.Get("Content-Type"), check.DeepEquals, "application/json")
			return strings.HasSuffix(r.URL.Path, "/services/mysql/instances")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ServiceInstanceAdd{}
	command.Flags().Parse(true, []string{
		"--team-owner", "my team",
		"--description", "desc",
		"--tag", "my tag 1", "--tag", "my tag 2",
		"--plan-param", "param1=value1", "--plan-param", "param2=value2",
		"--pool", "pool-one",
	})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, result)
}

func (s *S) TestServiceInstanceAddRunWithEmptyTag(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := "Service instance successfully added.\nFor additional information use: tsuru service instance info mysql my_app_db\n"
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
			var result map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&result)
			c.Assert(err, check.IsNil)
			c.Assert(result, check.DeepEquals, map[string]interface{}{
				"name":      "my_app_db",
				"plan_name": "small",
				"tags":      []interface{}{""},
			})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ServiceInstanceAdd{}
	command.Flags().Parse(true, []string{"--tag", ""})
	err := command.Run(&context, client)
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
	flagDesc = "service instance tag"
	flagset.Parse(true, []string{"-g", "my tag"})
	assume = flagset.Lookup("tag")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "tag")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "[\"my tag\"]")
	c.Check(assume.DefValue, check.Equals, "[]")
	sassume = flagset.Lookup("g")
	c.Check(sassume, check.NotNil)
	c.Check(sassume.Name, check.Equals, "g")
	c.Check(sassume.Usage, check.Equals, flagDesc)
	c.Check(sassume.Value.String(), check.Equals, "[\"my tag\"]")
	c.Check(sassume.DefValue, check.Equals, "[]")
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
			r.ParseForm()
			c.Check(r.FormValue("description"), check.Equals, "desc")
			c.Check(r.Form["tag"], check.HasLen, 2)
			c.Check(r.Form["tag"][0], check.Equals, "tag1")
			c.Check(r.Form["tag"][1], check.Equals, "tag2")
			c.Check(r.FormValue("plan"), check.Equals, "new-plan")
			c.Check(r.Method, check.Equals, http.MethodPut)
			c.Check(r.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
			c.Check(strings.HasSuffix(r.URL.Path, "/services/service/instances/service-instance"), check.Equals, true)
			c.Check(r.FormValue("teamowner"), check.Equals, "new-team")
			c.Check(r.FormValue("parameters.param1"), check.Equals, "value1")
			c.Check(r.FormValue("parameters.param2"), check.Equals, "value2")
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ServiceInstanceUpdate{}
	command.Flags().Parse(true, []string{"--description", "desc", "--tag", "tag1", "--tag", "tag2", "--team-owner", "new-team", "--plan", "new-plan", "--plan-param", "param1=value1", "--plan-param", "param2=value2"})
	err := (&command).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, result)
}

func (s *S) TestServiceInstanceUpdateRunWithEmptyTag(c *check.C) {
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
			r.ParseForm()
			return len(r.Form["tag"]) == 1 && r.Form["tag"][0] == ""
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := ServiceInstanceUpdate{}
	command.Flags().Parse(true, []string{"--tag", ""})
	err := (&command).Run(&context, client)
	c.Assert(err, check.IsNil)
	obtained := stdout.String()
	c.Assert(obtained, check.Equals, result)
}

func (s *S) TestServiceInstanceUpdateFlags(c *check.C) {
	command := ServiceInstanceUpdate{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	err := flagset.Parse(true, []string{"-t", "the new owner"})
	c.Assert(err, check.IsNil)
	flagDesc := "service instance team owner"
	assume := flagset.Lookup("team-owner")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "team-owner")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "the new owner")
	c.Check(assume.DefValue, check.Equals, "")
	assume = flagset.Lookup("t")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "t")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "the new owner")
	c.Check(assume.DefValue, check.Equals, "")
	c.Check(command.teamOwner, check.Equals, "the new owner")

	err = flagset.Parse(true, []string{"-d", "description"})
	c.Assert(err, check.IsNil)
	flagDesc = "service instance description"
	assume = flagset.Lookup("description")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "description")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "description")
	c.Check(assume.DefValue, check.Equals, "")
	assume = flagset.Lookup("d")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "d")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "description")
	c.Check(assume.DefValue, check.Equals, "")
	c.Check(command.description, check.Equals, "description")

	err = flagset.Parse(true, []string{"-g", "my tag"})
	c.Assert(err, check.IsNil)
	flagDesc = "service instance tag"
	assume = flagset.Lookup("tag")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "tag")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "[\"my tag\"]")
	c.Check(assume.DefValue, check.Equals, "[]")
	assume = flagset.Lookup("g")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "g")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "[\"my tag\"]")
	c.Check(assume.DefValue, check.Equals, "[]")

	err = flagset.Parse(true, []string{"-p", "my plan"})
	c.Assert(err, check.IsNil)
	flagDesc = "service instance plan"
	assume = flagset.Lookup("plan")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "plan")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "my plan")
	c.Check(assume.DefValue, check.Equals, "")
	assume = flagset.Lookup("p")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "p")
	c.Check(assume.Usage, check.Equals, flagDesc)
	c.Check(assume.Value.String(), check.Equals, "my plan")
	c.Check(assume.DefValue, check.Equals, "")
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
Pool: my-pool
Apps: app, app2
Teams: admin, admin2
Team Owner: admin
Description: description
Tags: tag 1, tag 2
Plan: small
Plan description: another plan
Plan parameters:
	param1 = {"some": "custom-data"}
	param2 = value2
	param3 = 3

Custom Info for "mongo"
key2:
	value9

key3:
	value3

key4:
	value8
Status: Service instance "mongo" is up
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
Status: Service instance "mongo" is up
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
+-------+--------------+-----------------+----------------+
| Name  | Description  | Instance Params | Binding Params |
+-------+--------------+-----------------+----------------+
| small | another plan |                 |                |
+-------+--------------+-----------------+----------------+
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

func (s *S) TestServiceInfoRunWithPools(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Info for "multicluster"

Instances
+-----------+-------+------------+-------+-------+--------+
| Instances | Plan  | Pool       | Apps  | key   | key2   |
+-----------+-------+------------+-------+-------+--------+
| mymongo   | small | my-pool-01 | myapp | value | value2 |
+-----------+-------+------------+-------+-------+--------+

Plans
+-------+--------------+-----------------+----------------+
| Name  | Description  | Instance Params | Binding Params |
+-------+--------------+-----------------+----------------+
| small | another plan |                 |                |
+-------+--------------+-----------------+----------------+
`
	args := []string{"multicluster"}
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

func (s *S) TestServiceInfoRunWithPoolSelected(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Info for "multicluster" in pool "my-pool-01"

Instances
+-----------+-------+-------+-------+--------+
| Instances | Plan  | Apps  | key   | key2   |
+-----------+-------+-------+-------+--------+
| mymongo   | small | myapp | value | value2 |
+-----------+-------+-------+-------+--------+

Plans
+-------+--------------+-----------------+----------------+
| Name  | Description  | Instance Params | Binding Params |
+-------+--------------+-----------------+----------------+
| small | another plan |                 |                |
+-------+--------------+-----------------+----------------+
`
	args := []string{"multicluster"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &infoTransport{includePlans: true}}, nil, manager)
	err := (&ServiceInfo{
		pool: "my-pool-01",
	}).Run(&context, client)
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
+-------+--------------+-----------------+----------------+
| Name  | Description  | Instance Params | Binding Params |
+-------+--------------+-----------------+----------------+
| small | another plan |                 |                |
+-------+--------------+-----------------+----------------+

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

func (s *S) TestServiceInfoWithSchemas(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Info for "mongodb-broker"

Instances
+-----------+-------+-------+-------+--------+
| Instances | Plan  | Apps  | key   | key2   |
+-----------+-------+-------+-------+--------+
| mymongo   | small | myapp | value | value2 |
+-----------+-------+-------+-------+--------+

Plans
+---------+------------------------------------------+---------------------------------------+---------------------------------------+
| Name    | Description                              | Instance Params                       | Binding Params                        |
+---------+------------------------------------------+---------------------------------------+---------------------------------------+
| default | Plan with parameter and response schemas | param-1:                              | param-1:                              |
|         |                                          |   description: First input parameter  |   description: First input parameter  |
|         |                                          |   type: string                        |   type: string                        |
|         |                                          |   default: value1                     | param-2:                              |
|         |                                          | param-2:                              |   description: Second input parameter |
|         |                                          |   description: Second input parameter |   type: string                        |
|         |                                          |   type: string                        |                                       |
|         |                                          |   required: true                      |                                       |
|         |                                          |                                       |                                       |
+---------+------------------------------------------+---------------------------------------+---------------------------------------+
`
	args := []string{"mongodb-broker"}
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

func (s *S) TestServiceInstanceRemoveRunWithForce(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"some-service-name", "some-service-instance"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			return strings.HasSuffix(r.URL.Path, "/services/some-service-name/instances/some-service-instance") &&
				r.Method == http.MethodDelete
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	cmd := ServiceInstanceRemove{}
	cmd.Flags().Parse(true, []string{"-f", "-y"})
	err := cmd.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestServiceInstanceRemoveFlags(c *check.C) {
	command := ServiceInstanceRemove{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse(true, []string{"-f"})
	c.Check(command.force, check.Equals, true)
	assume := flagset.Lookup("f")
	c.Check(assume.Name, check.Equals, "f")
	c.Check(assume.Usage, check.Equals, "Forces the removal of a service instance binded to apps.")
	c.Check(assume.Value.String(), check.Equals, "true")
	c.Check(assume.DefValue, check.Equals, "false")
}

func (s *S) TestServiceInstanceRemoveWithoutForce(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"some-service-name", "mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	msg := "Applications bound to the service \"some-service-name\": \"app1\"\n: This service instance is bound to at least one app. Unbind them before removing it"
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: msg, Status: http.StatusBadRequest},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/some-service-name/instances/mongodb") &&
				req.Method == http.MethodDelete &&
				req.URL.Query().Get("unbindall") == "false" &&
				req.URL.Query().Get("ignoreerrors") == "false"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceRemove{}
	command.Flags().Parse(true, []string{"-y"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, trans.Transport.(cmdtest.Transport).Message)
}

func (s *S) TestServiceInstanceRemoveWithAppBindWithFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"service-name", "mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/service-name/instances/mongodb") &&
				req.Method == http.MethodDelete &&
				req.URL.Query().Get("unbindall") == "true" &&
				req.URL.Query().Get("ignoreerrors") == "false"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceRemove{}
	command.Flags().Parse(true, []string{"-f", "-y"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestServiceInstanceRemoveWithAppBindWithFlagsIgnoreErros(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"service-name", "mongodb"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/services/service-name/instances/mongodb") &&
				req.Method == http.MethodDelete &&
				req.URL.Query().Get("unbindall") == "true" &&
				req.URL.Query().Get("ignoreerrors") == "true"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ServiceInstanceRemove{}
	command.Flags().Parse(true, []string{"-f", "-y", "--ignore-errors"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
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
