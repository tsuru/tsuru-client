// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/safe"
	"gopkg.in/check.v1"
)

func (s *S) TestBuildInfo(c *check.C) {
	var cmd AppBuild
	c.Assert(cmd.Info(), check.NotNil)
}

func (s *S) TestBuildRun(c *check.C) {
	calledTimes := 0
	var buf bytes.Buffer
	err := Archive(&buf, false, []string{"testdata", ".."}, DefaultArchiveOptions(io.Discard))
	c.Assert(err, check.IsNil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "\nOK\n", Status: http.StatusOK},
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
			content, transErr := io.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(content, check.DeepEquals, buf.Bytes())
			c.Assert(req.Header.Get("Content-Type"), check.Matches, "multipart/form-data; boundary=.*")
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/myapp/build")
		},
	}
	s.setupFakeTransport(&trans)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	command := AppBuild{}
	command.Flags().Parse([]string{"-a", "myapp", "-t", "mytag"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(calledTimes, check.Equals, 2)
}

func (s *S) TestBuildFail(c *check.C) {
	var buf bytes.Buffer
	err := Archive(&buf, false, []string{"testdata", ".."}, DefaultArchiveOptions(io.Discard))
	c.Assert(err, check.IsNil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Failed", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			if req.Body != nil {
				defer req.Body.Close()
			}
			if req.Method == "GET" {
				return strings.HasSuffix(req.URL.Path, "/apps/myapp")
			}
			file, _, transErr := req.FormFile("file")
			c.Assert(transErr, check.IsNil)
			content, transErr := io.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(content, check.DeepEquals, buf.Bytes())
			c.Assert(req.Header.Get("Content-Type"), check.Matches, "multipart/form-data; boundary=.*")
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/myapp/build")
		},
	}
	s.setupFakeTransport(&trans)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	command := AppBuild{}
	command.Flags().Parse([]string{"-a", "myapp", "-t", "mytag"})
	err = command.Run(&context)
	c.Assert(err, check.Equals, cmd.ErrAbortCommand)
}

func (s *S) TestBuildRunWithoutArgs(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	s.setupFakeTransport(&trans)
	command := AppBuild{}
	command.Flags().Parse([]string{"-a", "myapp", "-t", "mytag"})
	err := command.Run(&ctx)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "you should provide at least one file to build the image")
}

func (s *S) TestBuildRunWithoutTag(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", "..", "-a", "myapp"},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	s.setupFakeTransport(&trans)
	command := AppBuild{}
	command.Flags().Parse([]string{"-a", "myapp"})
	err := command.Run(&ctx)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "you should provide one tag to build the image")
}

func (s *S) TestBuildRequestBodyWithProgressNilArchive(c *check.C) {
	body := safe.NewBuffer(nil)
	buf := safe.NewBuffer(nil)
	request, _ := http.NewRequest("POST", "/apps/myapp/build", body)
	values := url.Values{"tag": []string{"mytag"}}
	var stdout bytes.Buffer

	err := buildRequestBodyWithProgress(context.Background(), &stdout, request, buf, body, values, nil)
	c.Assert(err, check.IsNil)
	c.Assert(request.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
	c.Assert(body.String(), check.Equals, values.Encode())
}

func (s *S) TestBuildRequestBodyWithProgressWithArchive(c *check.C) {
	body := safe.NewBuffer(nil)
	buf := safe.NewBuffer(nil)
	request, _ := http.NewRequest("POST", "/apps/myapp/build", body)
	values := url.Values{"tag": []string{"mytag"}}
	stdout := safe.NewBuffer(nil)
	archive := bytes.NewBufferString("fake-archive-content")

	ctx, cancel := context.WithCancel(context.Background())
	err := buildRequestBodyWithProgress(ctx, stdout, request, buf, body, values, archive)
	c.Assert(err, check.IsNil)

	// stop the progress goroutine before reading stdout
	cancel()
	time.Sleep(100 * time.Millisecond)

	c.Assert(request.Header.Get("Content-Type"), check.Matches, "multipart/form-data; boundary=.*")
	c.Assert(body.Len() > 0, check.Equals, true)
	c.Assert(stdout.String(), check.Matches, `Uploading files \(\d+\.\d+MB\)\.\.\. .*`)
}

func (s *S) TestBuildRequestBodyWithProgressContextCancel(c *check.C) {
	body := safe.NewBuffer(nil)
	buf := safe.NewBuffer(nil)
	request, _ := http.NewRequest("POST", "/apps/myapp/build", body)
	values := url.Values{"tag": []string{"mytag"}}
	archive := bytes.NewBufferString("fake-archive-content")

	writerCh := make(chan struct{}, 100)
	pw := &probeWriter{ch: writerCh}

	ctx, cancel := context.WithCancel(context.Background())

	err := buildRequestBodyWithProgress(ctx, pw, request, buf, body, values, archive)
	c.Assert(err, check.IsNil)

	// wait for the goroutine to write at least once (proving it started)
	select {
	case <-writerCh:
	case <-time.After(5 * time.Second):
		c.Fatal("goroutine never started writing progress")
	}

	// cancel the context
	cancel()

	// drain any writes that were already in-flight
	time.Sleep(100 * time.Millisecond)
	for len(writerCh) > 0 {
		<-writerCh
	}

	// no more writes should happen after cancel
	select {
	case <-writerCh:
		c.Fatal("goroutine kept writing after context was cancelled")
	case <-time.After(5 * time.Second):
		// success: no writes after cancel
	}
}

type probeWriter struct {
	ch chan struct{}
}

func (w *probeWriter) Write(p []byte) (int, error) {
	select {
	case w.ch <- struct{}{}:
	default:
	}
	return len(p), nil
}

func (s *S) TestGuessingContainerFile(c *check.C) {
	cases := []struct {
		files         []string
		app           string
		expected      func(d string) string
		expectedError string
	}{
		{
			expectedError: "container file not found",
		},
		{
			app:      "my-app",
			files:    []string{"Containerfile"},
			expected: func(root string) string { return filepath.Join(root, "Containerfile") },
		},
		{
			app:      "my-app",
			files:    []string{"Containerfile", "Dockerfile"},
			expected: func(root string) string { return filepath.Join(root, "Dockerfile") },
		},
		{
			app:      "my-app",
			files:    []string{"Containerfile", "Dockerfile", "Containerfile.tsuru"},
			expected: func(root string) string { return filepath.Join(root, "Containerfile.tsuru") },
		},
		{
			app:      "my-app",
			files:    []string{"Containerfile", "Dockerfile", "Containerfile.tsuru", "Dockerfile.tsuru"},
			expected: func(root string) string { return filepath.Join(root, "Dockerfile.tsuru") },
		},
		{
			app:      "my-app",
			files:    []string{"Containerfile", "Dockerfile", "Containerfile.tsuru", "Dockerfile.tsuru", "Containerfile.my-app"},
			expected: func(root string) string { return filepath.Join(root, "Containerfile.my-app") },
		},
		{
			app:      "my-app",
			files:    []string{"Containerfile", "Dockerfile", "Containerfile.tsuru", "Dockerfile.tsuru", "Containerfile.my-app", "Dockerfile.my-app"},
			expected: func(root string) string { return filepath.Join(root, "Dockerfile.my-app") },
		},
	}

	for _, tt := range cases {
		dir := c.MkDir()

		for _, name := range tt.files {
			f, err := os.Create(filepath.Join(dir, name))
			c.Check(err, check.IsNil)
			c.Check(f.Close(), check.IsNil)
		}

		got, err := guessingContainerFile(tt.app, dir)
		if tt.expectedError != "" {
			c.Check(err, check.ErrorMatches, tt.expectedError)
		} else {
			c.Check(err, check.IsNil)
			c.Check(got, check.DeepEquals, tt.expected(dir))
		}
	}
}
