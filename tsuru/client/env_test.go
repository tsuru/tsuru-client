// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cezarsa/form"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	apiTypes "github.com/tsuru/tsuru/types/api"
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
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context)
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
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context)
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
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestEnvGetManagedByVariables(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_USER", "value": "someuser", "public": false, "managedBy": "my-service/instance"}, {"name": "DATABASE_HOST", "value": "somehost", "public": true, "managedBy": "my-service/instance"}]`
	result := "DATABASE_HOST=somehost (managed by my-service/instance)\nDATABASE_USER=*** (private variable managed by my-service/instance)\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	cmd := &EnvGet{}
	err := cmd.Flags().Parse(true, []string{"-a", "seek"})
	c.Assert(err, check.IsNil)

	err = cmd.Run(&context)
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
			var e apiTypes.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			name := e.Envs[0].Name == "DATABASE_HOST"
			value := e.Envs[0].Value == "somehost"
			return path && method && contentType && name && value
		},
	}
	s.setupFakeTransport(trans)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context)
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
	s.setupFakeTransport(&cmdtest.Transport{Message: string(result), Status: http.StatusOK})
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context)
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
			want := []apiTypes.Env{
				{Name: "LINE1", Value: `multiline
variable 1`, Alias: "", Private: &private},
				{Name: "LINE2", Value: `multiline
variable 2`, Alias: "", Private: &private},
			}
			err = req.ParseForm()
			c.Assert(err, check.IsNil)
			var e apiTypes.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			private = !e.Private
			noRestart := !e.NoRestart
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && contentType && method && private && noRestart
		},
	}
	s.setupFakeTransport(trans)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context)
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
			want := []apiTypes.Env{
				{Name: "DATABASE_HOST", Value: "some host", Alias: "", Private: &private},
				{Name: "DATABASE_USER", Value: "root", Alias: "", Private: &private},
				{Name: "DATABASE_PASSWORD", Value: ".1234..abc", Alias: "", Private: &private},
				{Name: "http_proxy", Value: "http://myproxy.com:3128/", Alias: "", Private: &private},
				{Name: "VALUE_WITH_EQUAL_SIGN", Value: "http://wholikesquerystrings.me/?tsuru=awesome", Alias: "", Private: &private},
				{Name: "BASE64_STRING", Value: "t5urur0ck5==", Alias: "", Private: &private},
				{Name: "SOME_PASSWORD", Value: "js87$%32??", Alias: "", Private: &private},
			}
			err = req.ParseForm()
			c.Assert(err, check.IsNil)
			var e apiTypes.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			private = !e.Private
			noRestart := !e.NoRestart
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && contentType && method && private && noRestart
		},
	}
	s.setupFakeTransport(trans)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	output := stdout.String()
	c.Assert(strings.Contains(output, "Warning: The environment variable 'DATABASE_PASSWORD' looks like a sensitive variable"), check.Equals, true)
	c.Assert(strings.Contains(output, "Warning: The environment variable 'SOME_PASSWORD' looks like a sensitive variable"), check.Equals, true)
	c.Assert(strings.Contains(output, expectedOut), check.Equals, true)
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
			want := []apiTypes.Env{
				{Name: "DATABASE_HOST", Value: "some host", Alias: "", Private: &private},
				{Name: "DATABASE_USER", Value: "root", Alias: "", Private: &private},
				{Name: "DATABASE_PASSWORD", Value: ".1234..abc", Alias: "", Private: &private},
				{Name: "http_proxy", Value: "http://myproxy.com:3128/", Alias: "", Private: &private},
				{Name: "VALUE_WITH_EQUAL_SIGN", Value: "http://wholikesquerystrings.me/?tsuru=awesome", Private: &private},
				{Name: "BASE64_STRING", Value: "t5urur0ck5==", Alias: "", Private: &private},
			}
			err = req.ParseForm()
			c.Assert(err, check.IsNil)
			var e apiTypes.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			private = e.Private
			noRestart := e.NoRestart
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && contentType && method && private && noRestart
		},
	}
	s.setupFakeTransport(trans)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-p", "1", "--no-restart"})
	err = command.Run(&context)
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
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := EnvUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context)
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
	s.setupFakeTransport(trans)
	command := EnvUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp", "--no-restart"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestRequestEnvURL(c *check.C) {
	result := "DATABASE_HOST=somehost"
	s.setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})
	args := []string{"DATABASE_HOST"}
	g := &EnvGet{}
	g.Flags().Parse(true, []string{"-a", "someapp"})
	b, err := requestEnvGetURL(g, args)
	c.Assert(err, check.IsNil)
	c.Assert(b, check.DeepEquals, []byte(result))
}

func (s *S) TestJobEnvGetRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_HOST", "value": "somehost", "public": true}]`
	result := "DATABASE_HOST=somehost\n"
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestJobEnvGetRunWithMultipleParams(c *check.C) {
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
			path := strings.HasSuffix(req.URL.Path, "/jobs/sample-job/env")
			method := req.Method == "GET"
			envs := req.URL.Query()["env"]
			c.Assert(envs, check.DeepEquals, []string{"DATABASE_HOST", "DATABASE_USER"})
			return path && method
		},
	}
	s.setupFakeTransport(trans)
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestJobEnvGetAlwaysPrintInAlphabeticalOrder(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_USER", "value": "someuser", "public": true}, {"name": "DATABASE_HOST", "value": "somehost", "public": true}]`
	result := "DATABASE_HOST=somehost\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestJobEnvGetPrivateVariables(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `[{"name": "DATABASE_USER", "value": "someuser", "public": true}, {"name": "DATABASE_HOST", "value": "somehost", "public": false}]`
	result := "DATABASE_HOST=*** (private variable)\nDATABASE_USER=someuser\n"
	params := []string{"DATABASE_HOST", "DATABASE_USER"}
	context := cmd.Context{
		Args:   params,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := EnvGet{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestJobEnvGetWithoutTheFlag(c *check.C) {
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
			return strings.HasSuffix(req.URL.Path, "/jobs/sample-job/env") && req.Method == "GET"
		},
	}
	s.setupFakeTransport(trans)
	cmd := &EnvGet{}
	err := cmd.Flags().Parse(true, []string{"-j", "sample-job"})
	c.Assert(err, check.IsNil)

	err = cmd.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestJobEnvSetRun(c *check.C) {
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
			var e apiTypes.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			path := strings.HasSuffix(req.URL.Path, "/jobs/sample-job/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			name := e.Envs[0].Name == "DATABASE_HOST"
			value := e.Envs[0].Value == "somehost"
			return path && method && contentType && name && value
		},
	}
	s.setupFakeTransport(trans)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestJobEnvSetRunWithMultipleParams(c *check.C) {
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
	s.setupFakeTransport(&cmdtest.Transport{Message: string(result), Status: http.StatusOK})
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestJobEnvSetMultilineVariables(c *check.C) {
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
			want := []apiTypes.Env{
				{Name: "LINE1", Value: `multiline
variable 1`, Alias: "", Private: &private},
				{Name: "LINE2", Value: `multiline
variable 2`, Alias: "", Private: &private},
			}
			err = req.ParseForm()
			c.Assert(err, check.IsNil)
			var e apiTypes.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			private = !e.Private
			path := strings.HasSuffix(req.URL.Path, "/jobs/sample-job/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && contentType && method && private
		},
	}
	s.setupFakeTransport(trans)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestJobEnvSetValues(c *check.C) {
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
			want := []apiTypes.Env{
				{Name: "DATABASE_HOST", Value: "some host", Alias: "", Private: &private},
				{Name: "DATABASE_USER", Value: "root", Alias: "", Private: &private},
				{Name: "DATABASE_PASSWORD", Value: ".1234..abc", Alias: "", Private: &private},
				{Name: "http_proxy", Value: "http://myproxy.com:3128/", Alias: "", Private: &private},
				{Name: "VALUE_WITH_EQUAL_SIGN", Value: "http://wholikesquerystrings.me/?tsuru=awesome", Alias: "", Private: &private},
				{Name: "BASE64_STRING", Value: "t5urur0ck5==", Alias: "", Private: &private},
				{Name: "SOME_PASSWORD", Value: "js87$%32??", Alias: "", Private: &private},
			}
			err = req.ParseForm()
			c.Assert(err, check.IsNil)
			var e apiTypes.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			private = !e.Private
			path := strings.HasSuffix(req.URL.Path, "/jobs/sample-job/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && contentType && method && private
		},
	}
	s.setupFakeTransport(trans)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	output := stdout.String()
	c.Assert(strings.Contains(output, "Warning: The environment variable 'DATABASE_PASSWORD' looks like a sensitive variable"), check.Equals, true)
	c.Assert(strings.Contains(output, "Warning: The environment variable 'SOME_PASSWORD' looks like a sensitive variable"), check.Equals, true)
	c.Assert(strings.Contains(output, expectedOut), check.Equals, true)
}

func (s *S) TestJobEnvSetValuesAndPrivate(c *check.C) {
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
			want := []apiTypes.Env{
				{Name: "DATABASE_HOST", Value: "some host", Alias: "", Private: &private},
				{Name: "DATABASE_USER", Value: "root", Alias: "", Private: &private},
				{Name: "DATABASE_PASSWORD", Value: ".1234..abc", Alias: "", Private: &private},
				{Name: "http_proxy", Value: "http://myproxy.com:3128/", Alias: "", Private: &private},
				{Name: "VALUE_WITH_EQUAL_SIGN", Value: "http://wholikesquerystrings.me/?tsuru=awesome", Private: &private},
				{Name: "BASE64_STRING", Value: "t5urur0ck5==", Alias: "", Private: &private},
			}
			err = req.ParseForm()
			c.Assert(err, check.IsNil)
			var e apiTypes.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			private = e.Private
			path := strings.HasSuffix(req.URL.Path, "/jobs/sample-job/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && contentType && method && private
		},
	}
	s.setupFakeTransport(trans)
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-j", "sample-job", "-p"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestJobEnvSetInvalidParameters(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"DATABASE_HOST", "somehost"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, EnvSetValidationMessage)
}

func (s *S) TestJobEnvUnsetRun(c *check.C) {
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
			path := strings.HasSuffix(req.URL.Path, "/jobs/sample-job/env")
			method := req.Method == http.MethodDelete
			env := req.URL.Query().Get("env") == "DATABASE_HOST"
			return path && method && env
		},
	}
	s.setupFakeTransport(trans)
	command := EnvUnset{}
	command.Flags().Parse(true, []string{"-j", "sample-job"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestJobRequestEnvURL(c *check.C) {
	result := "DATABASE_HOST=somehost"
	s.setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})
	args := []string{"DATABASE_HOST"}
	g := &EnvGet{}
	g.Flags().Parse(true, []string{"-j", "sample-job"})
	b, err := requestEnvGetURL(g, args)
	c.Assert(err, check.IsNil)
	c.Assert(b, check.DeepEquals, []byte(result))
}

func (s *S) TestCheckAppAndJobInputsMissingAppOrJob(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"mysql", "my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{})
	err := (&EnvGet{}).Run(&ctx)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "you must pass an application or job")
}

func (s *S) TestCheckAppAndJobInputsPassingBothAppAndJob(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"mysql", "my-mysql"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{})
	command := &EnvGet{}
	command.Flags().Parse(true, []string{"-j", "sample-job", "-a", "sample-app"})
	err := command.Run(&ctx)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "you must pass an application or job, not both")
}

func (s *S) TestIsSensitiveName(c *check.C) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"DATABASE_PASSWORD", true},
		{"PASSWORD", true},
		{"MY_PASSWORD", true},
		{"PASSWORD_HASH", true},
		{"API_KEY", true},
		{"APIKEY", true},
		{"MY_APIKEY", true},
		{"SECRET", true},
		{"MY_SECRET", true},
		{"SECRET_VALUE", true},
		{"TOKEN", true},
		{"ACCESS_TOKEN", true},
		{"MY_TOKEN", true},
		{"KEY", true},
		{"ENCRYPTION_KEY", true},
		{"PRIVATE_KEY", true},
		{"CREDENTIAL", true},
		{"AWS_CREDENTIAL", true},
		{"MY_CREDENTIAL", true},
		{"DATABASE_HOST", false},
		{"DATABASE_USER", false},
		{"PORT", false},
		{"ENVIRONMENT", false},
		{"MY_VARIABLE", false},
		{"lowercase_password", true},
		{"MixedCase_Token", true},
		{"my_api_key", true},
	}

	for _, tt := range tests {
		result := isSensitiveName(tt.name)
		c.Assert(result, check.Equals, tt.expected, check.Commentf("Expected %s to be %v", tt.name, tt.expected))
	}
}

func (s *S) TestEnvSetWithSensitiveVariableWarning(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args: []string{
			"DATABASE_HOST=localhost",
			"API_KEY=my-secret-key",
			"DATABASE_PASSWORD=secret123",
		},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	s.setupFakeTransport(&cmdtest.Transport{Message: string(result), Status: http.StatusOK})
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	output := stdout.String()
	c.Assert(strings.Contains(output, "Warning: The environment variable 'API_KEY' looks like a sensitive variable"), check.Equals, true)
	c.Assert(strings.Contains(output, "Warning: The environment variable 'DATABASE_PASSWORD' looks like a sensitive variable"), check.Equals, true)
	c.Assert(strings.Contains(output, "DATABASE_HOST"), check.Equals, false)
}

func (s *S) TestEnvSetWithSensitiveVariableAndPrivateFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args: []string{
			"API_KEY=my-secret-key",
			"DATABASE_PASSWORD=secret123",
		},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	s.setupFakeTransport(&cmdtest.Transport{Message: string(result), Status: http.StatusOK})
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-p"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	output := stdout.String()
	// Should not show warning when using -p flag
	c.Assert(strings.Contains(output, "Warning"), check.Equals, false)
	c.Assert(strings.Contains(output, expectedOut), check.Equals, true)
}

func (s *S) TestEnvSetWithNonSensitiveVariables(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args: []string{
			"DATABASE_HOST=localhost",
			"DATABASE_USER=root",
			"PORT=3000",
		},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "variable(s) successfully exported\n"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	s.setupFakeTransport(&cmdtest.Transport{Message: string(result), Status: http.StatusOK})
	command := EnvSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	output := stdout.String()
	// Should not show any warnings for non-sensitive variables
	c.Assert(strings.Contains(output, "Warning"), check.Equals, false)
	c.Assert(strings.Contains(output, expectedOut), check.Equals, true)
}
