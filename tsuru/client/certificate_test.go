// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestCertificateSetRunSuccessfully(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args: []string{
			"./testdata/cert/server.crt",
			"./testdata/cert/server.key",
		},
	}

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/secret/certificate")
			method := req.Method == "POST"
			cname := req.FormValue("cname") == "app.io"
			certificate := req.FormValue("certificate") == s.mustReadFileString(c, "./testdata/cert/server.crt")
			key := req.FormValue("key") == s.mustReadFileString(c, "./testdata/cert/server.key")
			return url && method && cname && certificate && key
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := CertificateSet{GuessingCommand: guessCommand}
	command.Flags().Parse(true, []string{"-c", "app.io"})
	c.Assert(command.cname, check.Equals, "app.io")
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Succesfully created the certificated.\n")
}

func (s *S) TestCertificateSetRunCerticateNotFound(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args: []string{
			"./testdata/cert/not-found.crt",
			"./testdata/cert/server.key",
		},
	}
	trans := &cmdtest.Transport{Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := CertificateSet{GuessingCommand: guessCommand}
	command.Flags().Parse(true, []string{"-c", "app.io"})
	c.Assert(command.cname, check.Equals, "app.io")
	err := command.Run(&context, client)
	c.Assert(os.IsNotExist(err), check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) mustReadFileString(c *check.C, path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		c.Fatal(err)
		return ""
	}
	return string(data)
}
