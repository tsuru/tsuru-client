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

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
)

func (s *S) TestEnvGetInfo(c *check.C) {
	c.Assert((&EnvGet{}).Info(), check.NotNil)
}

func (s *S) TestEnvGetRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_HOST", "value": "somehost", "public": true}]`
	result := "DATABASE_HOST=somehost\n"
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: jsonResult, Status: http.StatusOK}}, nil, manager)
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestEnvGetRunWithMultipleParams(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_HOST", "value": "somehost", "public": true}, {"name": "DATABASE_USER", "value": "someuser", "public": true}]`
	result := "DATABASE_HOST=somehost\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: jsonResult, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "GET"
			envs := req.URL.Query()["env"]
			c.Assert(envs, check.DeepEquals, []string{"DATABASE_HOST", "DATABASE_USER"})
			return path && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestEnvGetAlwaysPrintInAlphabeticalOrder(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_USER", "value": "someuser", "public": true}, {"name": "DATABASE_HOST", "value": "somehost", "public": true}]`
	result := "DATABASE_HOST=somehost\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: jsonResult, Status: http.StatusOK}}, nil, manager)
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestEnvGetPrivateVariables(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_USER", "value": "someuser", "public": true}, {"name": "DATABASE_HOST", "value": "somehost", "public": false}]`
	result := "DATABASE_HOST=*** (private variable)\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: jsonResult, Status: http.StatusOK}}, nil, manager)
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestEnvGetWithoutTheFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_HOST", "value": "somehost", "public": true}, {"name": "DATABASE_USER", "value": "someuser", "public": true}]`
	result := "DATABASE_HOST=somehost\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: jsonResult, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/seek/env") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	cmd := &EnvGet{}
	err := cmd.Flags().Parse(true, []string{"-a", "seek"})
	c.Assert(err, check.IsNil)

	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestEnvSetInfo(c *check.C) {
	c.Assert((&EnvSet{}).Info(), check.NotNil)
}

func (s *S) TestEnvSetRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST=somehost"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			err = req.ParseForm()
			c.Assert(err, check.IsNil)
			//var e tsuru.Env
			// dec := form.NewDecoder(nil)
			// dec.IgnoreUnknownKeys(true)
			// dec.UseJSONTags(false)
			c.Assert(err, check.IsNil)
			data, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			var envResult map[string]interface{}
			err = json.Unmarshal(data, &envResult)
			c.Assert(err, check.IsNil)
			c.Assert(envResult, check.DeepEquals, map[string]interface{}{"envs": []interface{}{map[string]interface{}{"name": "DATABASE_HOST",
				"value": "somehost"}}})
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/json"
			//name := e.Env[0].Name == "DATABASE_HOST"
			//value := e.Env[0].Value == "somehost"
			return path && method && contentType
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestEnvSetRunWithMultipleParams(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST=somehost", "DATABASE_USER=user"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: string(result), Status: http.StatusOK}}, nil, manager)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestEnvSetMultilineVariables(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args: []string{
			`LINE1=multiline
variable 1`,
			`LINE2=multiline
variable 2`},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			private := false
			want := []tsuru.Env{
				{Name: "LINE1", Value: "multiline\nvariable 1", Alias: "", Private: private},
				{Name: "LINE2", Value: "multiline\nvariable 2", Alias: "", Private: private},
			}
			err = req.ParseForm()
			// c.Assert(err, check.IsNil)
			var e tsuru.EnvSetData
			// dec := form.NewDecoder(nil)
			// dec.IgnoreUnknownKeys(true)
			// dec.UseJSONTags(false)
			// err = dec.DecodeValues(&e, req.Form)
			// c.Assert(err, check.IsNil)
			data, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			///var envResult map[string]interface{}
			err = json.Unmarshal(data, &e)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			//c.Assert(e.Envs, check.DeepEquals, want)
			private = !e.Private
			noRestart := !e.Norestart
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/json"
			return path && contentType && method && private && noRestart
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestEnvSetValues(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args: []string{
			"DATABASE_HOST=some host",
			"DATABASE_USER=root",
			"DATABASE_PASSWORD=.1234..abc",
			"http_proxy=http://myproxy.com:3128/",
			"VALUE_WITH_EQUAL_SIGN=http://wholikesquerystrings.me/?tsuru=awesome",
			"BASE64_STRING=t5urur0ck5==",
			"SOME_PASSWORD=js87$%32??",
		},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			private := false
			want := []tsuru.Env{
				{Name: "DATABASE_HOST", Value: "some host", Alias: "", Private: private},
				{Name: "DATABASE_USER", Value: "root", Alias: "", Private: private},
				{Name: "DATABASE_PASSWORD", Value: ".1234..abc", Alias: "", Private: private},
				{Name: "http_proxy", Value: "http://myproxy.com:3128/", Alias: "", Private: private},
				{Name: "VALUE_WITH_EQUAL_SIGN", Value: "http://wholikesquerystrings.me/?tsuru=awesome", Alias: "", Private: private},
				{Name: "BASE64_STRING", Value: "t5urur0ck5==", Alias: "", Private: private},
				{Name: "SOME_PASSWORD", Value: "js87$%32??", Alias: "", Private: private},
			}
			err = req.ParseForm()
			c.Assert(err, check.IsNil)
			var e tsuru.EnvSetData
			//dec := form.NewDecoder(nil)
			//dec.IgnoreUnknownKeys(true)
			//dec.UseJSONTags(false)
			///err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			data, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			//var envResult map[string]interface{}
			err = json.Unmarshal(data, &e)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			//c.Assert(err, check.IsNil)
			//c.Assert(e.Envs, check.DeepEquals, want)
			private = !e.Private
			noRestart := !e.Norestart
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/json"
			return path && contentType && method && private && noRestart
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestEnvSetValuesAndPrivateAndNoRestart(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args: []string{
			"DATABASE_HOST=some host",
			"DATABASE_USER=root",
			"DATABASE_PASSWORD=.1234..abc",
			"http_proxy=http://myproxy.com:3128/",
			"VALUE_WITH_EQUAL_SIGN=http://wholikesquerystrings.me/?tsuru=awesome",
			"BASE64_STRING=t5urur0ck5==",
		},

		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			private := false
			want := []tsuru.Env{
				{Name: "DATABASE_HOST", Value: "some host", Alias: "", Private: private},
				{Name: "DATABASE_USER", Value: "root", Alias: "", Private: private},
				{Name: "DATABASE_PASSWORD", Value: ".1234..abc", Alias: "", Private: private},
				{Name: "http_proxy", Value: "http://myproxy.com:3128/", Alias: "", Private: private},
				{Name: "VALUE_WITH_EQUAL_SIGN", Value: "http://wholikesquerystrings.me/?tsuru=awesome", Private: private},
				{Name: "BASE64_STRING", Value: "t5urur0ck5==", Alias: "", Private: private},
			}
			err = req.ParseForm()
			c.Assert(err, check.IsNil)
			var e tsuru.EnvSetData
			// dec := form.NewDecoder(nil)
			// dec.IgnoreUnknownKeys(true)
			// dec.UseJSONTags(false)
			// err = dec.DecodeValues(&e, req.Form)
			data, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &e)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			private = e.Private
			noRestart := e.Norestart
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/json"
			return path && contentType && method && private && noRestart
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-p", "1", "--no-restart"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestEnvSetInvalidParameters(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST", "somehost"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, nil)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, EnvSetValidationMessage)
}

func (s *S) TestEnvUnsetInfo(c *check.C) {
	c.Assert((&EnvUnset{}).Info(), check.NotNil)
}

func (s *S) TestEnvUnsetRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully unset\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == http.MethodDelete
			noRestart := req.URL.Query().Get("noRestart") == "false"
			env := req.URL.Query().Get("env") == "DATABASE_HOST"
			return path && method && noRestart && env
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := EnvUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestEnvUnsetWithNoRestartFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully unset\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == http.MethodDelete
			noRestart := req.URL.Query().Get("noRestart") == "true"
			env := req.URL.Query().Get("env") == "DATABASE_HOST"
			return path && method && noRestart && env
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := EnvUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp", "--no-restart"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestRequestEnvURL(c *check.C) {
	result := "DATABASE_HOST=somehost"
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	args := []string{"DATABASE_HOST"}
	g := cmd.AppNameMixIn{}
	g.Flags().Parse(true, []string{"-a", "someapp"})
	b, err := requestEnvGetURL(g, args, client)
	c.Assert(err, check.IsNil)
	c.Assert(b, check.DeepEquals, []byte(result))
}
