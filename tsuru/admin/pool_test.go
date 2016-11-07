// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

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

func (s *S) TestAddPoolWithProvisioner(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools")
			name := req.FormValue("name") == "test"
			public := req.FormValue("public") == "false"
			def := req.FormValue("default") == "false"
			prov := req.FormValue("provisioner") == "kub"
			return url && name && public && def && prov
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"--provisioner", "kub"})
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

func (s *S) TestUpdatePoolToTheSchedulerCmd(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			def := req.FormValue("default") == ""
			public := req.FormValue("public") == "true"
			url := strings.HasSuffix(req.URL.Path, "/pools/poolTest")
			method := req.Method == "PUT"
			force := req.FormValue("force") == "false"
			return public && method && url && force && def
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := UpdatePoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"--public", "true"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestFailToUpdateMoreThanOneDefaultPool(c *check.C) {
	var buf bytes.Buffer
	stdin := bytes.NewBufferString("no")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			def := req.FormValue("default") == "true"
			public := req.FormValue("public") == ""
			force := req.FormValue("force") == "false"
			url := strings.HasSuffix(req.URL.Path, "/pools/test")
			return def && url && public && force
		},
	}
	manager := cmd.Manager{}
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf, Stdin: stdin}
	client := cmd.NewClient(&http.Client{Transport: &transportError}, nil, &manager)
	command := UpdatePoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"--default=true"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := "WARNING: Default pool already exist. Do you want change to test pool? (y/n) Pool update aborted.\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestForceToOverwriteDefaultPoolInUpdate(c *check.C) {
	var buf bytes.Buffer
	stdin := bytes.NewBufferString("no")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			def := req.FormValue("default") == "true"
			public := req.FormValue("public") == ""
			url := strings.HasSuffix(req.URL.Path, "/pools/test")
			force := req.FormValue("force") == "true"
			method := req.Method == "PUT"
			return url && force && def && public && method
		},
	}
	manager := cmd.Manager{}
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf, Stdin: stdin}
	client := cmd.NewClient(&http.Client{Transport: &transportError}, nil, &manager)
	command := UpdatePoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"--default=true"})
	command.Flags().Parse(true, []string{"-f"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAskOverwriteDefaultPoolInUpdate(c *check.C) {
	var buf bytes.Buffer
	var called int
	stdin := bytes.NewBufferString("yes")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			called++
			def := req.FormValue("default") == "true"
			public := req.FormValue("public") == ""
			url := strings.HasSuffix(req.URL.Path, "/pools/test")
			force := req.FormValue("force") == "false"
			return url && def && public && force
		},
	}
	transportOk := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK, Message: ""},
		CondFunc: func(req *http.Request) bool {
			called++
			url := strings.HasSuffix(req.URL.Path, "/pools/test")
			force := req.FormValue("force") == "true"
			return url && force
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
	command := UpdatePoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"--default=true"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, 2)
	expected := "WARNING: Default pool already exist. Do you want change to test pool? (y/n) Pool successfully updated.\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestRemovePoolFromTheSchedulerCmd(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools/poolTest")
			method := req.Method == "DELETE"
			return method && url
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := RemovePoolFromSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"-y"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestRemovePoolFromTheSchedulerCmdConfirmation(c *check.C) {
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{"poolX"},
		Stdout: &stdout,
		Stdin:  strings.NewReader("n\n"),
	}
	command := RemovePoolFromSchedulerCmd{}
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Are you sure you want to remove \"poolX\" pool? (y/n) Abort.\n")
}

func (s *S) TestAddTeamsToPoolCmdRun(c *check.C) {
	var buf bytes.Buffer
	ctx := cmd.Context{Stdout: &buf, Args: []string{"pool1", "team1", "team2"}}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools/pool1/team")
			method := req.Method == "POST"
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			teams := req.Form["team"]
			c.Assert(teams, check.DeepEquals, []string{"team1", "team2"})
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return url && method && contentType
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	err := AddTeamsToPoolCmd{}.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestRemoveTeamsFromPoolCmdRun(c *check.C) {
	var buf bytes.Buffer
	ctx := cmd.Context{Stdout: &buf, Args: []string{"pool1", "team1"}}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools/pool1/team")
			method := req.Method == "DELETE"
			rq := req.URL.RawQuery == "team=team1"
			return url && method && rq
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	err := RemoveTeamsFromPoolCmd{}.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}
