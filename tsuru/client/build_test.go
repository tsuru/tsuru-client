// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestBuildInfo(c *check.C) {
	var cmd AppBuild
	c.Assert(cmd.Info(), check.NotNil)
}

func (s *S) TestBuildRun(c *check.C) {
	calledTimes := 0
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: bytes.NewBufferString("")}
	err := targz(&ctx, &buf, nil, "testdata", "..")
	c.Assert(err, check.IsNil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "OK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			if req.Body != nil {
				defer req.Body.Close()
			}
			if calledTimes == 1 {
				return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/myapp")
			}
			file, _, transErr := req.FormFile("file")
			c.Assert(transErr, check.IsNil)
			content, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(content, check.DeepEquals, buf.Bytes())
			c.Assert(req.Header.Get("Content-Type"), check.Matches, "multipart/form-data; boundary=.*")
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/myapp/build")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "myapp"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppBuild{GuessingCommand: guessCommand}
	command.Flags().Parse(true, []string{"-t", "mytag"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(calledTimes, check.Equals, 2)
}

func (s *S) TestBuildRunWithoutArgs(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "myapp"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppBuild{GuessingCommand: guessCommand}
	command.Flags().Parse(true, []string{"-t", "mytag"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "You should provide at least one file to build the image.\n")
}

func (s *S) TestBuildRunWithoutTag(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "myapp"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppBuild{GuessingCommand: guessCommand}
	err := command.Run(&ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "You should provide one tag to build the image.\n")
}
