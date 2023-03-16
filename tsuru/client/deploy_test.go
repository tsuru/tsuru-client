// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/tsuru/tsuru-client/tsuru/formatter"
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
	var buf bytes.Buffer
	err := Archive(&buf, false, []string{"testdata", ".."}, DefaultArchiveOptions(io.Discard))
	c.Assert(err, check.IsNil)

	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			if req.Body != nil {
				defer req.Body.Close()
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
	}
	cmd := AppDeploy{}
	err = cmd.Flags().Parse(true, []string{"testdata", "..", "-a", "secret"})
	c.Assert(err, check.IsNil)
	context.Args = cmd.Flags().Args()
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

type slowReader struct {
	io.ReadCloser
	Latency time.Duration
}

func (s *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(s.Latency)
	return s.ReadCloser.Read(p)
}

func (s *slowReader) Close() error {
	return s.ReadCloser.Close()
}

func (s *S) TestDeployRunCancel(c *check.C) {
	var buf bytes.Buffer
	err := Archive(&buf, false, []string{"testdata", ".."}, DefaultArchiveOptions(io.Discard))
	c.Assert(err, check.IsNil)
	deploy := make(chan struct{}, 1)
	trans := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				Transport: &cmdtest.BodyTransport{
					Status:  http.StatusOK,
					Headers: map[string][]string{"X-Tsuru-Eventid": {"5aec54d93195b20001194951"}},
					Body:    &slowReader{ReadCloser: ioutil.NopCloser(bytes.NewBufferString("deploy worked\nOK\n")), Latency: time.Second * 5},
				},
				CondFunc: func(req *http.Request) bool {
					deploy <- struct{}{}
					if req.Body != nil {
						defer req.Body.Close()
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
			},
			{
				Transport: cmdtest.Transport{Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					c.Assert(req.Method, check.Equals, "POST")
					c.Assert(req.URL.Path, check.Equals, "/1.1/events/5aec54d93195b20001194951/cancel")
					return true
				},
			},
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  bytes.NewReader([]byte("y")),
		Args:   []string{"testdata", "..", "-a", "secret"},
	}
	cmd := AppDeploy{}

	err = cmd.Flags().Parse(true, context.Args)
	c.Assert(err, check.IsNil)
	context.Args = cmd.Flags().Args()

	go func() {
		err = cmd.Run(&context, client)
		c.Assert(err, check.IsNil)
	}()
	<-deploy
	err = cmd.Cancel(context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestDeployImage(c *check.C) {
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			if req.Body != nil {
				defer req.Body.Close()
			}
			c.Assert(req.Header.Get("Content-Type"), check.Matches, "application/x-www-form-urlencoded")
			c.Assert(req.FormValue("image"), check.Equals, "registr.com/image-to-deploy")
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
	cmd := AppDeploy{}
	err := cmd.Flags().Parse(true, []string{"-a", "secret", "-i", "registr.com/image-to-deploy"})
	c.Assert(err, check.IsNil)
	context.Args = cmd.Flags().Args()
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestDeployRunWithMessage(c *check.C) {
	var buf bytes.Buffer
	err := Archive(&buf, false, []string{"testdata", ".."}, DefaultArchiveOptions(io.Discard))
	c.Assert(err, check.IsNil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			if req.Body != nil {
				defer req.Body.Close()
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
	cmd := AppDeploy{}
	err = cmd.Flags().Parse(true, []string{"-a", "secret", "-m", "my awesome deploy"})
	c.Assert(err, check.IsNil)
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestDeployAuthNotOK(c *check.C) {
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Forbidden", Status: http.StatusForbidden},
		CondFunc: func(req *http.Request) bool {
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/secret/deploy")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", "..", "-a", "secret"},
	}
	command := AppDeploy{}
	err := command.Flags().Parse(true, context.Args)
	c.Assert(err, check.IsNil)
	context.Args = command.Flags().Args()
	err = command.Run(&context, client)
	c.Assert(err, check.ErrorMatches, "Forbidden")
}

func (s *S) TestDeployRunNotOK(c *check.C) {
	trans := cmdtest.Transport{Message: "deploy worked\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", "..", "-a", "secret"},
	}
	command := AppDeploy{}
	err := command.Flags().Parse(true, context.Args)
	c.Assert(err, check.IsNil)
	context.Args = command.Flags().Args()
	err = command.Run(&context, client)
	c.Assert(err, check.Equals, cmd.ErrAbortCommand)
}

func (s *S) TestDeployRunFileNotFound(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"/tmp/something/that/doesn't/really/exist/im/sure", "-a", "secret"},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppDeploy{}
	err := command.Flags().Parse(true, context.Args)
	c.Assert(err, check.IsNil)
	context.Args = command.Flags().Args()
	err = command.Run(&context, client)
	c.Assert(err, check.NotNil)
}

func (s *S) TestDeployRunWithoutArgsAndImage(c *check.C) {
	command := AppDeploy{}
	err := command.Flags().Parse(true, []string{"-a", "secret"})
	c.Assert(err, check.IsNil)
	ctx := &cmd.Context{Stdout: io.Discard, Stderr: io.Discard, Args: command.Flags().Args()}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Status: http.StatusInternalServerError}}, nil, manager)
	err = command.Run(ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "You should provide at least one file, Docker image name or Dockerfile to deploy.\n")
}

func (s *S) TestDeployRunWithArgsAndImage(c *check.C) {
	command := AppDeploy{}
	err := command.Flags().Parse(true, []string{"-i", "registr.com/image-to-deploy", "./path/to/dir"})
	c.Assert(err, check.IsNil)
	ctx := &cmd.Context{Stdout: io.Discard, Stderr: io.Discard, Args: command.Flags().Args()}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Status: http.StatusInternalServerError}}, nil, manager)
	err = command.Run(ctx, client)
	c.Assert(err, check.ErrorMatches, "You can't deploy files and docker image at the same time.\n")
}

func (s *S) TestDeployRunRequestFailure(c *check.C) {
	trans := cmdtest.Transport{Message: "app not found\n", Status: http.StatusNotFound}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppDeploy{}
	err := command.Flags().Parse(true, []string{"testdata", "..", "-a", "secret"})
	c.Assert(err, check.IsNil)
	ctx := &cmd.Context{Stdout: io.Discard, Stderr: io.Discard, Args: command.Flags().Args()}
	err = command.Run(ctx, client)
	c.Assert(err, check.ErrorMatches, "app not found\n")
}

func (s *S) TestDeploy_Run_DockerfileAndDockerImage(c *check.C) {
	command := AppDeploy{}
	err := command.Flags().Parse(true, []string{"-i", "registry.example.com/my-team/my-app:v42", "--dockerfile", "."})
	c.Assert(err, check.IsNil)
	ctx := &cmd.Context{Stdout: io.Discard, Stderr: io.Discard, Args: command.Flags().Args()}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Status: http.StatusInternalServerError}}, nil, manager)
	err = command.Run(ctx, client)
	c.Assert(err, check.ErrorMatches, "You can't deploy container image and container file at same time.\n")
}

func (s *S) TestDeploy_Run_UsingDockerfile(c *check.C) {
	command := AppDeploy{}
	err := command.Flags().Parse(true, []string{"-a", "my-app", "--dockerfile", "./testdata/deploy4/"})
	c.Assert(err, check.IsNil)

	ctx := &cmd.Context{Stdout: io.Discard, Stderr: io.Discard, Args: command.Flags().Args()}

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deployed\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			if req.Body != nil {
				defer req.Body.Close()
			}
			c.Assert(req.Header.Get("Content-Type"), check.Matches, "multipart/form-data; boundary=.*")
			c.Assert(req.FormValue("dockerfile"), check.Equals, "FROM busybox:latest\n\nCOPY ./app.sh /usr/local/bin/\n")

			file, _, nerr := req.FormFile("file")
			c.Assert(nerr, check.IsNil)
			defer file.Close()
			files := extractFiles(s.t, c, file)
			c.Assert(files, check.DeepEquals, []miniFile{
				{Name: filepath.Join("Dockerfile"), Type: tar.TypeReg, Data: []byte("FROM busybox:latest\n\nCOPY ./app.sh /usr/local/bin/\n")},
				{Name: filepath.Join("app.sh"), Type: tar.TypeReg, Data: []byte("echo \"Starting my application :P\"\n")},
			})

			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/my-app/deploy")
		},
	}

	err = command.Run(ctx, cmd.NewClient(&http.Client{Transport: trans}, nil, manager))
	c.Assert(err, check.IsNil)
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
		formatted = append(formatted, formatter.Local(parsed).Format(time.RFC822))
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
	err := command.Flags().Parse(true, []string{"--app", "test"})
	c.Assert(err, check.IsNil)
	context.Args = command.Flags().Args()
	err = command.Run(&context, client)
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
	command := AppDeployList{}
	command.Flags().Parse(true, []string{"-a", "secret"})
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
