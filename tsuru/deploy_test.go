// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"sort"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/testing"
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
