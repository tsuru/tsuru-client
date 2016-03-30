// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/cezarsa/form"
	"github.com/tsuru/tsuru/api"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
)

func (s *S) TestEnvGetInfo(c *check.C) {
	c.Assert((&envGet{}).Info(), check.NotNil)
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
	command := envGet{}
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
			want := `["DATABASE_HOST","DATABASE_USER"]` + "\n"
			defer req.Body.Close()
			got, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			return strings.HasSuffix(req.URL.Path, "/apps/someapp/env") && req.Method == "GET" && string(got) == want
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envGet{}
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
	command := envGet{}
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
	command := envGet{}
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
	fake := &cmdtest.FakeGuesser{Name: "seek"}
	err := (&envGet{cmd.GuessingCommand{G: fake}}).Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestEnvSetInfo(c *check.C) {
	c.Assert((&envSet{}).Info(), check.NotNil)
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
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			var e api.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envSet{}
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
	command := envSet{}
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
			want := []struct{ Name, Value string }{
				{Name: "DATABASE_HOST", Value: "some host"},
				{Name: "DATABASE_USER", Value: "root"},
				{Name: "DATABASE_PASSWORD", Value: ".1234..abc"},
				{Name: "http_proxy", Value: "http://myproxy.com:3128/"},
				{Name: "VALUE_WITH_EQUAL_SIGN", Value: "http://wholikesquerystrings.me/?tsuru=awesome"},
				{Name: "BASE64_STRING", Value: "t5urur0ck5=="},
				{Name: "SOME_PASSWORD", Value: "js87$%32??"},
			}
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			var e api.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			private := !e.Private
			noRestart := !e.NoRestart
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && contentType && method && private && noRestart
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envSet{}
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
			want := []struct{ Name, Value string }{
				{Name: "DATABASE_HOST", Value: "some host"},
				{Name: "DATABASE_USER", Value: "root"},
				{Name: "DATABASE_PASSWORD", Value: ".1234..abc"},
				{Name: "http_proxy", Value: "http://myproxy.com:3128/"},
				{Name: "VALUE_WITH_EQUAL_SIGN", Value: "http://wholikesquerystrings.me/?tsuru=awesome"},
				{Name: "BASE64_STRING", Value: "t5urur0ck5=="},
			}
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			var e api.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(e.Envs, check.DeepEquals, want)
			private := e.Private
			noRestart := e.NoRestart
			path := strings.HasSuffix(req.URL.Path, "/apps/someapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && contentType && method && private && noRestart
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-p", "1", "--no-restart"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestEnvSetWithoutFlag(c *check.C) {
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
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			var e api.Envs
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			err = dec.DecodeValues(&e, req.Form)
			c.Assert(err, check.IsNil)
			private := !e.Private
			noRestart := !e.NoRestart
			path := strings.HasSuffix(req.URL.Path, "/apps/otherapp/env")
			method := req.Method == "POST"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && contentType && method && private && noRestart
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "otherapp"}
	err = (&envSet{GuessingCommand: cmd.GuessingCommand{G: fake}}).Run(&context, client)
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
	command := envSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, nil)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, envSetValidationMessage)
}

func (s *S) TestEnvUnsetInfo(c *check.C) {
	c.Assert((&envUnset{}).Info(), check.NotNil)
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
			want := `["DATABASE_HOST"]` + "\n"
			defer req.Body.Close()
			got, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			return strings.HasSuffix(req.URL.Path, "/apps/someapp/env") && req.Method == "DELETE" && string(got) == want &&
				req.URL.RawQuery == "noRestart=false"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envUnset{}
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
			want := `["DATABASE_HOST"]` + "\n"
			defer req.Body.Close()
			got, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			c.Assert(req.URL.RawQuery, check.Equals, "noRestart=true")
			return strings.HasSuffix(req.URL.Path, "/apps/someapp/env") && req.Method == "DELETE" && string(got) == want
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := envUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp", "--no-restart"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestEnvUnsetWithoutFlag(c *check.C) {
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
			return strings.HasSuffix(req.URL.Path, "/apps/otherapp/env") && req.Method == "DELETE" &&
				req.URL.RawQuery == "noRestart=false"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "otherapp"}
	err = (&envUnset{GuessingCommand: cmd.GuessingCommand{G: fake}}).Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestRequestEnvURL(c *check.C) {
	result := "DATABASE_HOST=somehost"
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	args := []string{"DATABASE_HOST"}
	g := cmd.GuessingCommand{G: &cmdtest.FakeGuesser{Name: "someapp"}}
	b, err := requestEnvURL("GET", g, args, client)
	c.Assert(err, check.IsNil)
	c.Assert(b, check.DeepEquals, []byte(result))
}
