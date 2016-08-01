// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

type S struct{}

var manager *cmd.Manager

func (s *S) SetUpSuite(c *check.C) {
	var stdout, stderr bytes.Buffer
	manager = cmd.NewManager("glb", "1.0.0", "Supported-Tsuru-Version", &stdout, &stderr, os.Stdin, nil)
	os.Setenv("TSURU_TARGET", "http://localhost")
}

func (s *S) TearDownSuite(c *check.C) {
	os.Unsetenv("TSURU_TARGET")
}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) TestAddPoolToTheSchedulerCmd(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/pools")
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := AddPoolToSchedulerCmd{}
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAddPublicPool(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools")
			name := req.FormValue("name") == "test"
			public := req.FormValue("public") == "true"
			def := req.FormValue("default") == "false"
			return url && name && public && def
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := AddPoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"-p"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAddDefaultPool(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools")
			name := req.FormValue("name") == "test"
			public := req.FormValue("public") == "false"
			def := req.FormValue("default") == "true"
			return url && name && public && def
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"-d"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestFailToAddMoreThanOneDefaultPool(c *check.C) {
	var buf bytes.Buffer
	stdin := bytes.NewBufferString("no")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			name := req.FormValue("name") == "test"
			public := req.FormValue("public") == "false"
			def := req.FormValue("default") == "true"
			url := strings.HasSuffix(req.URL.Path, "/pools")
			return name && public && def && url
		},
	}
	manager := cmd.Manager{}
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf, Stdin: stdin}
	client := cmd.NewClient(&http.Client{Transport: &transportError}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"-d"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := "WARNING: Default pool already exist. Do you want change to test pool? (y/n) Pool add aborted.\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestForceToOverwriteDefaultPool(c *check.C) {
	var buf bytes.Buffer
	stdin := bytes.NewBufferString("no")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			name := req.FormValue("name") == "test"
			public := req.FormValue("public") == "false"
			def := req.FormValue("default") == "true"
			force := req.FormValue("force") == "true"
			return name && public && def && force
		},
	}
	manager := cmd.Manager{}
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf, Stdin: stdin}
	client := cmd.NewClient(&http.Client{Transport: &transportError}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"-d"})
	command.Flags().Parse(true, []string{"-f"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAskOverwriteDefaultPool(c *check.C) {
	var buf bytes.Buffer
	var called int
	stdin := bytes.NewBufferString("yes")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			called++
			name := req.FormValue("name") == "test"
			public := req.FormValue("public") == "false"
			def := req.FormValue("default") == "true"
			url := req.FormValue("force") == "false"
			return url && name && public && def
		},
	}
	transportOk := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK, Message: ""},
		CondFunc: func(req *http.Request) bool {
			called++
			return req.FormValue("force") == "true"
		},
	}
	multiTransport := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{transportError, transportOk},
	}
	context := cmd.Context{
		Args:   []string{"test"},
		Stdout: &buf,
		Stdin:  stdin,
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: &multiTransport}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"-d"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, 2)
	expected := "WARNING: Default pool already exist. Do you want change to test pool? (y/n) Pool successfully registered.\n"
	c.Assert(buf.String(), check.Equals, expected)
}
