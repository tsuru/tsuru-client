// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	tsuruIo "github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
)

func (s *S) TestDeployInfo(c *check.C) {
	var cmd AppDeploy
	c.Assert(cmd.Info(), check.NotNil)
}

func (s *S) TestDeployRun(c *check.C) {
	calledTimes := 0
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: bytes.NewBufferString("")}
	err := targz(&ctx, &buf, false, "testdata", "..")
	c.Assert(err, check.IsNil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			if req.Body != nil {
				defer req.Body.Close()
			}
			if calledTimes == 1 {
				return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/secret")
			}
			file, _, transErr := req.FormFile("file")
			c.Assert(transErr, check.IsNil)
			content, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(content, check.DeepEquals, buf.Bytes())
			c.Assert(req.Header.Get("Content-Type"), check.Matches, "multipart/form-data; boundary=.*")
			c.Assert(req.FormValue("origin"), check.Equals, "app-deploy")
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/secret/deploy")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	cmd := AppDeploy{GuessingCommand: guessCommand}
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(calledTimes, check.Equals, 2)
}

func (s *S) TestDeployImage(c *check.C) {
	calledTimes := 0
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			if req.Body != nil {
				defer req.Body.Close()
			}
			if calledTimes == 1 {
				return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/secret")
			}
			image := req.FormValue("image")
			c.Assert(image, check.Equals, "registr.com/image-to-deploy")
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
			c.Assert(req.FormValue("origin"), check.Equals, "image")
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/secret/deploy")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	cmd := AppDeploy{GuessingCommand: guessCommand}
	cmd.Flags().Parse(true, []string{"-i", "registr.com/image-to-deploy"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(calledTimes, check.Equals, 2)
}

func (s *S) TestDeployRunWithMessage(c *check.C) {
	calledTimes := 0
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: bytes.NewBufferString("")}
	err := targz(&ctx, &buf, false, "testdata", "..")
	c.Assert(err, check.IsNil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			if req.Body != nil {
				defer req.Body.Close()
			}
			if calledTimes == 1 {
				return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/secret")
			}
			file, _, transErr := req.FormFile("file")
			c.Assert(transErr, check.IsNil)
			content, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(content, check.DeepEquals, buf.Bytes())
			c.Assert(req.Header.Get("Content-Type"), check.Matches, "multipart/form-data; boundary=.*")
			c.Assert(req.FormValue("origin"), check.Equals, "app-deploy")
			c.Assert(req.FormValue("message"), check.Equals, "my awesome deploy")
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/secret/deploy")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	cmd := AppDeploy{GuessingCommand: guessCommand}
	cmd.Flags().Parse(true, []string{"-m", "my awesome deploy"})
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(calledTimes, check.Equals, 2)
}

func (s *S) TestDeployAuthNotOK(c *check.C) {
	calledTimes := 0
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Forbidden", Status: http.StatusForbidden},
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/secret")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.ErrorMatches, "Forbidden")
	c.Assert(calledTimes, check.Equals, 1)
}

func (s *S) TestDeployRunNotOK(c *check.C) {
	trans := cmdtest.Transport{Message: "deploy worked\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.Equals, cmd.ErrAbortCommand)
}

func (s *S) TestDeployRunFileNotFound(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"/tmp/something/that/doesn't/really/exist/im/sure"},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
}

func (s *S) TestDeployRunWithoutArgsAndImage(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "You should provide at least one file or a docker image to deploy.\n")
}

func (s *S) TestDeployRunWithArgsAndImage(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	command.Flags().Parse(true, []string{"-i", "registr.com/image-to-deploy"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "You can't deploy files and docker image at the same time.\n")
}

func (s *S) TestDeployRunRequestFailure(c *check.C) {
	trans := cmdtest.Transport{Message: "app not found\n", Status: http.StatusNotFound}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "app not found\n")
}

func (s *S) TestTargzSymlink(c *check.C) {
	if runtime.GOOS == "windows" {
		c.Skip("no symlink support on windows")
	}
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: &buf}
	var gzipBuf, tarBuf bytes.Buffer
	err := targz(&ctx, &gzipBuf, false, "testdata-symlink", "..")
	c.Assert(err, check.IsNil)
	gzipReader, err := gzip.NewReader(&gzipBuf)
	c.Assert(err, check.IsNil)
	_, err = io.Copy(&tarBuf, gzipReader)
	c.Assert(err, check.IsNil)
	tarReader := tar.NewReader(&tarBuf)
	var headers [][]string
	for header, err := tarReader.Next(); err == nil; header, err = tarReader.Next() {
		if header.Linkname != "" {
			headers = append(headers, []string{header.Name, header.Linkname})
		}
	}
	expected := [][]string{{"testdata-symlink/link", "test"}}
	c.Assert(headers, check.DeepEquals, expected)
}

func (s *S) TestTargzFailure(c *check.C) {
	var stderr bytes.Buffer
	ctx := cmd.Context{Stderr: &stderr}
	var buf bytes.Buffer
	err := targz(&ctx, &buf, false, "/tmp/something/that/definitely/doesn't/exist/right", "testdata")
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*(no such file or directory|cannot find the path specified).*")
}

func (s *S) TestDeployListInfo(c *check.C) {
	var cmd AppDeployList
	c.Assert(cmd.Info(), check.NotNil)
}

func (s *S) TestAppDeployList(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `
[
  {
    "ID": "54c92d91a46ec0e78501d86b",
    "App": "test",
    "Timestamp": "2015-01-27T18:42:25.725Z",
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
    "Timestamp": "2015-01-28T18:56:32.583Z",
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
    "Timestamp": "2015-01-28T19:13:11.498Z",
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
		"2015-01-27T18:42:25.725Z",
		"2015-01-28T18:56:32.583Z",
		"2015-01-28T19:13:11.498Z",
	}
	var formatted []string
	for _, t := range timestamps {
		parsed, _ := time.Parse(time.RFC3339, t)
		formatted = append(formatted, parsed.Local().Format(time.RFC822))
	}
	red := "\x1b[0;31;10m"
	reset := "\x1b[0m"
	expected := `+-----------------------+---------------+-------------------+-----------------------------+----------+
| Image (Rollback)      | Origin        | User              | Date (Duration)             | Error    |
+-----------------------+---------------+-------------------+-----------------------------+----------+
| ` + red + `tsuru/app-test:v1` + reset + `     | ` + red + `rollback` + reset + `      |                   | ` + red + formatted[2] + ` (00:26)` + reset + ` | ` + red + `my-error` + reset + ` |
+-----------------------+---------------+-------------------+-----------------------------+----------+
| tsuru/app-test:v2 (*) | app-deploy    | admin@example.com | ` + formatted[1] + ` (00:18) |          |
+-----------------------+---------------+-------------------+-----------------------------+----------+
| tsuru/app-test:v3 (*) | git (54c92d9) | admin@example.com | ` + formatted[0] + ` (00:18) |          |
+-----------------------+---------------+-------------------+-----------------------------+----------+
`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppDeployList{}
	command.Flags().Parse(true, []string{"--app", "test"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestDeployRunAppWithouDeploy(c *check.C) {
	trans := cmdtest.Transport{Message: "", Status: http.StatusNoContent}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeployList{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "App secret has no deploy.\n")
}

func (s *S) TestAppDeployRollbackInfo(c *check.C) {
	c.Assert((&AppDeployRollback{}).Info(), check.NotNil)
}

func (s *S) TestAppDeployRollback(c *check.C) {
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
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			method := req.Method == "POST"
			path := strings.HasSuffix(req.URL.Path, "/apps/arrakis/deploy/rollback")
			image := req.FormValue("image") == "my-image"
			rollback := req.FormValue("origin") == "rollback"
			return method && path && image && rollback
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppDeployRollback{}
	command.Flags().Parse(true, []string{"--app", "arrakis", "-y"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestAppDeployRollbackUpdateInfo(c *check.C) {
	c.Assert((&AppDeployRollbackUpdate{}).Info(), check.NotNil)
}

func (s *S) TestAppDeployRollbackUpdate(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			method := req.Method == http.MethodPut
			path := strings.HasSuffix(req.URL.Path, "/apps/zilean/deploy/rollback/update")
			image := req.FormValue("image") == "caitlyn"
			enable := req.FormValue("disable") == "true"
			reason := req.FormValue("reason") == "DEMACIA"
			rollback := req.FormValue("origin") == "rollback"
			return method && path && image && rollback && reason && enable
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppDeployRollbackUpdate{}
	command.Flags().Parse(true, []string{"--app", "zilean", "-i", "caitlyn", "-r", "DEMACIA", "-d"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestAppDeployRollbackUpdateDisabling(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			method := req.Method == http.MethodPut
			path := strings.HasSuffix(req.URL.Path, "/apps/xayah/deploy/rollback/update")
			image := req.FormValue("image") == "rakan"
			enable := req.FormValue("disable") == "false"
			reason := req.FormValue("reason") == "vastayan"
			rollback := req.FormValue("origin") == "rollback"
			return method && path && image && rollback && reason && enable
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppDeployRollbackUpdate{}
	command.Flags().Parse(true, []string{"--app", "xayah", "-i", "rakan", "-r", "vastayan"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestAppDeployRebuildInfo(c *check.C) {
	c.Assert((&AppDeployRebuild{}).Info(), check.NotNil)
}

func (s *S) TestAppDeployRebuild(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "---- rebuild ----"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			method := req.Method == "POST"
			path := strings.HasSuffix(req.URL.Path, "/apps/myapp/deploy/rebuild")
			rebuild := req.FormValue("origin") == "rebuild"
			return method && path && rebuild
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppDeployRebuild{}
	command.Flags().Parse(true, []string{"--app", "myapp"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestDeployRunTarGeneration(c *check.C) {
	var foundFiles []string
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			if req.Method == "GET" {
				return true
			}
			defer req.Body.Close()
			file, _, transErr := req.FormFile("file")
			c.Assert(transErr, check.IsNil)
			gzReader, transErr := gzip.NewReader(file)
			c.Assert(transErr, check.IsNil)
			tarReader := tar.NewReader(gzReader)
			foundFiles = nil
			for {
				header, transErr := tarReader.Next()
				if transErr == io.EOF {
					break
				}
				c.Assert(transErr, check.IsNil)
				foundFiles = append(foundFiles, header.Name)
			}
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	tests := []struct {
		files          []string
		ignored        []string
		deployArgs     []string
		flags          []string
		expected       []string
		expectedStderr string
		absPath        bool
	}{
		{
			files:      []string{"f1", "f2", "d1/f3", "d1/d2/f4"},
			deployArgs: []string{"."},
			expected:   []string{"f1", "f2", "d1", "d1/f3", "d1/d2", "d1/d2/f4"},
		},
		{
			files:      []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
			deployArgs: []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
			expected:   []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
		},
		{
			files:      []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
			deployArgs: []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
			flags:      []string{"-f"},
			expected:   []string{"file1.txt", "file2.txt"},
		},
		{
			files:      []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2/file3.txt", "testdata/deploy2/directory/file4.txt"},
			deployArgs: []string{"testdata/deploy", "testdata/deploy2"},
			expected:   []string{"testdata/deploy", "testdata/deploy2", "testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2/file3.txt", "testdata/deploy2/directory", "testdata/deploy2/directory/file4.txt"},
		},
		{
			files:      []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2/file3.txt", "testdata/deploy2/directory/file4.txt"},
			deployArgs: []string{"testdata/deploy", "testdata/deploy2"},
			flags:      []string{"-f"},
			expected:   []string{"file1.txt", "file2.txt", "file3.txt", "directory", "directory/file4.txt"},
		},
		{
			files:          []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy/directory/file.txt"},
			deployArgs:     []string{"testdata/deploy", ".."},
			expected:       []string{"testdata/deploy", "testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy/directory", "testdata/deploy/directory/file.txt"},
			expectedStderr: `Warning: skipping "\.\."`,
		},
		{
			files:      []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy/directory/file.txt"},
			deployArgs: []string{"testdata/deploy"},
			expected:   []string{"file1.txt", "file2.txt", "directory", "directory/file.txt"},
		},
		{
			files:      []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:    []string{"*.txt"},
			deployArgs: []string{"testdata/deploy2"},
			expected:   []string{"directory", "directory/dir2"},
		},
		{
			files:      []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2/file3.txt", "testdata/deploy2/directory/file4.txt"},
			deployArgs: []string{"testdata/deploy", "testdata/deploy2"},
			ignored:    []string{"*.txt"},
			expected:   []string{"testdata/deploy", "testdata/deploy2", "testdata/deploy2/directory"},
		},
		{
			files:      []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2/file3.txt", "testdata/deploy2/directory/file4.txt"},
			deployArgs: []string{"testdata/deploy", "testdata/deploy2"},
			ignored:    []string{"*.txt"},
			flags:      []string{"-f"},
			expected:   []string{"directory"},
		},
		{
			files:      []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:    []string{"*.txt"},
			deployArgs: []string{"testdata/deploy2"},
			expected:   []string{"directory", "directory/dir2"},
			absPath:    true,
		},
		{
			files:      []string{"file1.txt", "file2.txt", "directory/file.txt", "directory/dir2/file.txt"},
			ignored:    []string{"*.txt"},
			deployArgs: []string{"."},
			expected:   []string{".tsuruignore", "directory", "directory/dir2"},
		},
		{
			files:      []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:    []string{"directory"},
			deployArgs: []string{"testdata/deploy2"},
			expected:   []string{"file1.txt", "file2.txt"},
		},
		{
			files:      []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:    []string{"*/dir2"},
			deployArgs: []string{"testdata/deploy2"},
			expected:   []string{"directory", "directory/file.txt", "file1.txt", "file2.txt"},
		},
		{
			files:      []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:    []string{"directory/dir2/*"},
			deployArgs: []string{"testdata/deploy2"},
			expected:   []string{"directory", "directory/dir2", "directory/file.txt", "file1.txt", "file2.txt"},
		},
	}
	origDir, err := os.Getwd()
	c.Assert(err, check.IsNil)
	defer os.Chdir(origDir)
	for i, tt := range tests {
		tmpDir, err := ioutil.TempDir("", "integrarion")
		c.Assert(err, check.IsNil)
		defer os.RemoveAll(tmpDir)
		err = os.Chdir(tmpDir)
		c.Assert(err, check.IsNil)
		for _, f := range tt.files {
			err = os.MkdirAll(path.Dir(f), 0700)
			c.Assert(err, check.IsNil)
			err = ioutil.WriteFile(f, []byte{}, 0600)
			c.Assert(err, check.IsNil)
		}
		if len(tt.ignored) > 0 {
			err = ioutil.WriteFile(".tsuruignore", []byte(strings.Join(tt.ignored, "\n")), 0600)
			c.Assert(err, check.IsNil)
		}

		var stdout, stderr bytes.Buffer
		if tt.absPath {
			for i, f := range tt.deployArgs {
				tt.deployArgs[i], err = filepath.Abs(f)
				c.Assert(err, check.IsNil)
			}
		}
		context := cmd.Context{
			Stdout: &stdout,
			Stderr: &stderr,
			Args:   tt.deployArgs,
		}
		guessCommand := cmd.GuessingCommand{G: &cmdtest.FakeGuesser{Name: "secret"}}
		cmd := AppDeploy{GuessingCommand: guessCommand}
		cmd.Flags().Parse(true, tt.flags)
		err = cmd.Run(&context, client)
		c.Assert(err, check.IsNil)
		sort.Strings(foundFiles)
		sort.Strings(tt.expected)
		c.Assert(foundFiles, check.DeepEquals, tt.expected, check.Commentf("test %d", i))
		c.Assert(stderr.String(), check.Matches, tt.expectedStderr, check.Commentf("test %d", i))
	}
}
