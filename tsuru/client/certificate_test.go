// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

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
			method := req.Method == http.MethodPut
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
	c.Assert(stdout.String(), check.Equals, "Successfully created the certificated.\n")
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

func (s *S) TestCertificateUnsetRunSuccessfully(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	requestCount := 0
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			requestCount++
			url := strings.HasSuffix(req.URL.Path, "/apps/secret/certificate")
			method := req.Method == http.MethodDelete
			cname := req.FormValue("cname") == "app.io"

			return url && method && cname
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := CertificateUnset{GuessingCommand: guessCommand}
	command.Flags().Parse(true, []string{"-c", "app.io"})
	c.Assert(command.cname, check.Equals, "app.io")
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Certificate removed.\n")
	c.Assert(requestCount, check.Equals, 1)
}

func (s *S) mustReadFileString(c *check.C, path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		c.Fatal(err)
		return ""
	}
	return string(data)
}

func (s *S) TestCertificateListRunSuccessfully(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	requestCount := 0
	certMap := map[string]string{
		"myapp.io":       s.mustReadFileString(c, "./testdata/cert/server.crt"),
		"myapp.other.io": "",
	}
	data, err := json.Marshal(certMap)
	c.Assert(err, check.IsNil)
	expectedDate, err := time.Parse("2006-01-02 15:04:05", "2027-01-10 20:33:11")
	c.Assert(err, check.IsNil)
	datestr := expectedDate.Local().Format("2006-01-02 15:04:05")
	expected := `+----------------+---------------------+----------------------------+----------------------------+
| CName          | Expires             | Issuer                     | Subject                    |
+----------------+---------------------+----------------------------+----------------------------+
| myapp.io       | ` + datestr + ` | C=BR; ST=Rio de Janeiro;   | C=BR; ST=Rio de Janeiro;   |
|                |                     | L=Rio de Janeiro; O=Tsuru; | L=Rio de Janeiro; O=Tsuru; |
|                |                     | CN=app.io                  | CN=app.io                  |
+----------------+---------------------+----------------------------+----------------------------+
| myapp.other.io | -                   | -                          | -                          |
+----------------+---------------------+----------------------------+----------------------------+
`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status:  http.StatusNoContent,
			Message: string(data),
		},
		CondFunc: func(req *http.Request) bool {
			requestCount++
			url := strings.HasSuffix(req.URL.Path, "/apps/myapp/certificate")
			method := req.Method == http.MethodGet
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "myapp"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := CertificateList{GuessingCommand: guessCommand}
	command.Flags().Parse(true, []string{})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
	c.Assert(requestCount, check.Equals, 1)
}

func (s *S) TestCertificateListRawRunSuccessfully(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	requestCount := 0
	certData := s.mustReadFileString(c, "./testdata/cert/server.crt")
	certMap := map[string]string{
		"myapp.io":       certData,
		"myapp.other.io": "",
	}
	data, err := json.Marshal(certMap)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status:  http.StatusNoContent,
			Message: string(data),
		},
		CondFunc: func(req *http.Request) bool {
			requestCount++
			url := strings.HasSuffix(req.URL.Path, "/apps/myapp/certificate")
			method := req.Method == http.MethodGet
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "myapp"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := CertificateList{GuessingCommand: guessCommand}
	command.Flags().Parse(true, []string{"-r"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(stdout.String(), "myapp.other.io:\nNo certificate."), check.Equals, true)
	c.Assert(strings.Contains(stdout.String(), "myapp.io:\n"+certData), check.Equals, true)
	c.Assert(requestCount, check.Equals, 1)
}
