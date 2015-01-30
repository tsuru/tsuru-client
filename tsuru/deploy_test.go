// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/testing"
	tsuruIo "github.com/tsuru/tsuru/io"
	"launchpad.net/gocheck"
)

func (s *S) TestDeployInfo(c *gocheck.C) {
	desc := `Deploys set of files and/or directories to tsuru server. Some examples of calls are:

tsuru app-deploy .
tsuru app-deploy myfile.jar Procfile
`
	expected := &cmd.Info{
		Name:    "app-deploy",
		Usage:   "app-deploy [-a/--app <appname>] <file-or-dir-1> [file-or-dir-2] ... [file-or-dir-n]",
		Desc:    desc,
		MinArgs: 1,
	}
	var cmd appDeploy
	c.Assert(cmd.Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestDeployRun(c *gocheck.C) {
	var called bool
	var buf bytes.Buffer
	err := targz(nil, &buf, "testdata")
	c.Assert(err, gocheck.IsNil)
	trans := testing.ConditionalTransport{
		Transport: testing.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			defer req.Body.Close()
			called = true
			file, _, err := req.FormFile("file")
			c.Assert(err, gocheck.IsNil)
			content, err := ioutil.ReadAll(file)
			c.Assert(err, gocheck.IsNil)
			c.Assert(content, gocheck.DeepEquals, buf.Bytes())
			return req.Method == "POST" && req.URL.Path == "/apps/secret/deploy"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := testing.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	cmd := appDeploy{GuessingCommand: guessCommand}
	err = cmd.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
}

func (s *S) TestDeployRunNotOK(c *gocheck.C) {
	trans := testing.Transport{Message: "deploy worked\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := testing.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := appDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.Equals, cmd.ErrAbortCommand)
}

func (s *S) TestDeployRunFileNotFound(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"/tmp/something/that/doesnt/really/exist/im/sure"},
	}
	fake := testing.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := appDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
}

func (s *S) TestDeployRunRequestFailure(c *gocheck.C) {
	trans := testing.Transport{Message: "app not found\n", Status: http.StatusNotFound}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := testing.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := appDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "app not found\n")
}

func (s *S) TestTargz(c *gocheck.C) {
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: &buf}
	var gzipBuf, tarBuf bytes.Buffer
	err := targz(&ctx, &gzipBuf, "testdata", "..")
	c.Assert(err, gocheck.IsNil)
	gzipReader, err := gzip.NewReader(&gzipBuf)
	c.Assert(err, gocheck.IsNil)
	_, err = io.Copy(&tarBuf, gzipReader)
	c.Assert(err, gocheck.IsNil)
	tarReader := tar.NewReader(&tarBuf)
	var headers []string
	var contents []string
	for header, err := tarReader.Next(); err == nil; header, err = tarReader.Next() {
		headers = append(headers, header.Name)
		if !header.FileInfo().IsDir() {
			content, err := ioutil.ReadAll(tarReader)
			c.Assert(err, gocheck.IsNil)
			contents = append(contents, string(content))
		}
	}
	expected := []string{
		"testdata", "testdata/directory", "testdata/directory/file.txt",
		"testdata/file1.txt", "testdata/file2.txt",
	}
	sort.Strings(expected)
	sort.Strings(headers)
	c.Assert(headers, gocheck.DeepEquals, expected)
	expectedContents := []string{"wat\n", "something happened\n", "twice\n"}
	sort.Strings(expectedContents)
	sort.Strings(contents)
	c.Assert(contents, gocheck.DeepEquals, expectedContents)
	c.Assert(buf.String(), gocheck.Equals, `Warning: skipping ".."`)
}

func (s *S) TestTargzFailure(c *gocheck.C) {
	var stderr bytes.Buffer
	ctx := cmd.Context{Stderr: &stderr}
	var buf bytes.Buffer
	err := targz(&ctx, &buf, "/tmp/something/that/definitely/doesnt/exist/right", "testdata")
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "stat /tmp/something/that/definitely/doesnt/exist/right: no such file or directory")
}

func (s *S) TestDeployListInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "app-deploy-list",
		Usage:   "app-deploy-list [-a/--app <appname>]",
		Desc:    "List information about deploys for an application.",
		MinArgs: 0,
	}
	var cmd appDeployList
	c.Assert(cmd.Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestAppDeployList(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	result := `
[
  {
    "ID": "54c92d91a46ec0e78501d86b",
    "App": "test",
    "Timestamp": "2015-01-28T18:42:25.725Z",
    "Duration": 18709653486,
    "Commit": "54c92d91a46ec0e78501d86b54c92d91a46ec0e78501d86b",
    "Error": "",
    "Image": "tsuru/app-test:v3",
    "User": "admin@example.com",
    "Origin": "git",
    "CanRollback": true
  },
  {
    "ID": "54c922d0a46ec0e78501d84e",
    "App": "test",
    "Timestamp": "2015-01-28T17:56:32.583Z",
    "Duration": 18781564759,
    "Commit": "",
    "Error": "",
    "Image": "tsuru/app-test:v2",
    "User": "admin@example.com",
    "Origin": "app-deploy",
    "CanRollback": true
  },
  {
    "ID": "54c918a7a46ec0e78501d831",
    "App": "test",
    "Timestamp": "2015-01-28T17:13:11.498Z",
    "Duration": 26064205176,
    "Commit": "",
    "Error": "my-error",
    "Image": "tsuru/app-test:v1",
    "User": "",
    "Origin": "rollback",
    "CanRollback": false
  }
]
`
	timestamps := []string{
		"2015-01-28T18:42:25.725Z",
		"2015-01-28T17:56:32.583Z",
		"2015-01-28T17:13:11.498Z",
	}
	var formatted []string
	for _, t := range timestamps {
		parsed, _ := time.Parse(time.RFC3339, t)
		formatted = append(formatted, parsed.Local().Format(time.Stamp))
	}
	red := "\x1b[0;31;10m"
	reset := "\x1b[0m"
	expected := `+-----------------------+---------------+-------------------+-------------------------+----------+
| Image (Rollback)      | Origin        | User              | Date (Duration)         | Error    |
+-----------------------+---------------+-------------------+-------------------------+----------+
| tsuru/app-test:v3 (*) | git (54c92d9) | admin@example.com | ` + formatted[0] + ` (00:18) |          |
+-----------------------+---------------+-------------------+-------------------------+----------+
| tsuru/app-test:v2 (*) | app-deploy    | admin@example.com | ` + formatted[1] + ` (00:18) |          |
+-----------------------+---------------+-------------------+-------------------------+----------+
| ` + red + `tsuru/app-test:v1` + reset + `     | ` + red + `rollback` + reset + `      |                   | ` + red + formatted[2] + ` (00:26)` + reset + ` | ` + red + `my-error` + reset + ` |
+-----------------------+---------------+-------------------+-------------------------+----------+
`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &testing.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appDeployList{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestAppDeployRollbackInfo(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "app-deploy-rollback",
		Usage:   "app-deploy-rollback [-a/--app appname] [-y/--assume-yes] <image-name>",
		Desc:    "Deploys an existing image for an app. You can list available images with `tsuru app-deploy-list`.",
		MinArgs: 1,
	}
	c.Assert((&appDeployRollback{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestAppDeployRollback(c *gocheck.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"my-image"},
	}
	expectedOut := "-- deployed --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, gocheck.IsNil)
	trans := &testing.ConditionalTransport{
		Transport: testing.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			body, _ := ioutil.ReadAll(req.Body)
			return req.URL.Path == "/apps/arrakis/deploy/rollback" &&
				req.Method == "POST" && string(body) == "image=my-image"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := appDeployRollback{}
	command.Flags().Parse(true, []string{"--app", "arrakis", "-y"})
	err = command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(stdout.String(), gocheck.Equals, expectedOut)
}
