// Copyright 2016 crane authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"gopkg.in/check.v1"
)

func (s *S) TestServiceCreateInfo(c *check.C) {
	desc := "Creates a service based on a passed manifest. The manifest format should be a yaml and follow the standard described in the documentation (should link to it here)"
	cmd := ServiceCreate{}
	i := cmd.Info()
	c.Assert(i.Name, check.Equals, "service-create")
	c.Assert(i.Usage, check.Equals, "service create path/to/manifest [- for stdin]")
	c.Assert(i.Desc, check.Equals, desc)
	c.Assert(i.MinArgs, check.Equals, 1)
}

func (s *S) TestServiceCreateRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	args := []string{"testdata/manifest.yml"}
	context := cmd.Context{
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "success",
			Status:  http.StatusCreated,
		},
		CondFunc: func(req *http.Request) bool {
			method := req.Method == "POST"
			url := strings.HasSuffix(req.URL.Path, "/services")
			id := req.FormValue("id") == "mysqlapi"
			endpoint := req.FormValue("endpoint") == "mysqlapi.com"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return method && url && id && endpoint && contentType
		},
	}
	s.setupFakeTransport(&trans)
	err := (&ServiceCreate{}).Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Service successfully created\n")
}

func (s *S) TestServiceDestroyRun(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	stdin := bytes.NewBufferString("y\n")
	context := cmd.Context{
		Args:   []string{"my-service"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  stdin,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusNoContent,
		},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && strings.HasSuffix(req.URL.Path, "/services/my-service")
		},
	}
	s.setupFakeTransport(&trans)
	err := (&ServiceDestroy{}).Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := `Are you sure you want to remove the service "my-service"? This will remove the service and NOT a service instance. (y/n) Service successfully removed.`
	c.Assert(stdout.String(), check.Equals, expected+"\n")
}

func (s *S) TestServiceDestroyRunWithRequestFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"my-service"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  bytes.NewBufferString("y\n"),
	}
	trans := cmdtest.Transport{
		Message: "This service cannot be removed because it has instances.\nPlease remove these instances before removing the service.",
		Status:  http.StatusForbidden,
	}
	s.setupFakeTransport(trans)
	err := (&ServiceDestroy{}).Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(tsuruHTTP.UnwrapErr(err).Error(), check.Equals, trans.Message)
}

func (s *S) TestServiceDestroyIsACommand(c *check.C) {
	var _ cmd.Command = &ServiceDestroy{}
}

func (s *S) TestServiceDestroyInfo(c *check.C) {
	expected := &cmd.Info{
		Name:    "service-destroy",
		Usage:   "service destroy <servicename>",
		Desc:    "removes a service from catalog",
		MinArgs: 1,
		MaxArgs: 1,
	}
	c.Assert((&ServiceDestroy{}).Info(), check.DeepEquals, expected)
}

func (s *S) TestServiceUpdate(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusOK,
		},
		CondFunc: func(req *http.Request) bool {
			called = true
			method := req.Method == "PUT"
			url := strings.HasSuffix(req.URL.Path, "/services/mysqlapi")
			id := req.FormValue("id") == "mysqlapi"
			endpoint := req.FormValue("endpoint") == "mysqlapi.com"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return method && url && id && endpoint && contentType
		},
	}
	s.setupFakeTransport(&trans)
	context := cmd.Context{
		Args:   []string{"testdata/manifest.yml"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	err := (&ServiceUpdate{}).Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "Service successfully updated.\n")
}

func (s *S) TestServiceUpdateIsACommand(c *check.C) {
	var _ cmd.Command = &ServiceUpdate{}
}

func (s *S) TestServiceUpdateInfo(c *check.C) {
	expected := &cmd.Info{
		Name:    "service-update",
		Usage:   "service update <path/to/manifest>",
		Desc:    "Update service data, extracting it from the given manifest file.",
		MinArgs: 1,
	}
	c.Assert((&ServiceUpdate{}).Info(), check.DeepEquals, expected)
}

func (s *S) TestServiceDocAdd(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			called = true
			method := req.Method == "PUT"
			path := strings.HasSuffix(req.URL.Path, "/services/serv/doc")
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return method && path && contentType
		},
	}
	s.setupFakeTransport(&trans)
	context := cmd.Context{
		Args:   []string{"serv", "testdata/doc.md"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	err := (&ServiceDocAdd{}).Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "Documentation for 'serv' successfully updated.\n")
}

func (s *S) TestServiceDocAddInfo(c *check.C) {
	expected := &cmd.Info{
		Name:    "service-doc-add",
		Usage:   "service doc add <service> <path/to/docfile>",
		Desc:    "Update service documentation, extracting it from the given file.",
		MinArgs: 2,
	}
	c.Assert((&ServiceDocAdd{}).Info(), check.DeepEquals, expected)
}

func (s *S) TestServiceDocGet(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some doc", Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/services/serv/doc")
		},
	}
	s.setupFakeTransport(&trans)
	context := cmd.Context{
		Args:   []string{"serv"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	err := (&ServiceDocGet{}).Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(context.Stdout.(*bytes.Buffer).String(), check.Equals, "some doc")
}

func (s *S) TestServiceDocGetInfo(c *check.C) {
	expected := &cmd.Info{
		Name:    "service-doc-get",
		Usage:   "service doc get <service>",
		Desc:    "Shows service documentation.",
		MinArgs: 1,
	}
	c.Assert((&ServiceDocGet{}).Info(), check.DeepEquals, expected)
}

func (s *S) TestServiceTemplateInfo(c *check.C) {
	got := (&ServiceTemplate{}).Info()
	usg := `service template
e.g.: $ tsuru service template template`
	expected := &cmd.Info{
		Name:  "service-template",
		Usage: usg,
		Desc:  "Generates a manifest template file and places it in current directory",
	}
	c.Assert(got, check.DeepEquals, expected)
}

func (s *S) TestServiceTemplateRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	trans := cmdtest.Transport{Message: "", Status: http.StatusOK}
	s.setupFakeTransport(trans)
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	err := (&ServiceTemplate{}).Run(&ctx)
	defer os.Remove("./manifest.yaml")
	c.Assert(err, check.IsNil)
	expected := "Generated file \"manifest.yaml\" in current directory\n"
	c.Assert(stdout.String(), check.Equals, expected)
	f, err := os.Open("./manifest.yaml")
	c.Assert(err, check.IsNil)
	fc, err := io.ReadAll(f)
	c.Assert(err, check.IsNil)
	manifest := `id: servicename
username: username_to_auth
password: .{16}
team: team_responsible_to_provide_service
endpoint:
  production: production-endpoint.com
multi-cluster: false`
	c.Assert(string(fc), check.Matches, manifest)
}
