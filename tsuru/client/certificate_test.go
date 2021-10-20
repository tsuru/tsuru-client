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

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
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
			var cert tsuru.CertificateSetData
			err := json.NewDecoder(req.Body).Decode(&cert)
			c.Assert(err, check.IsNil)
			certifica, err := ioutil.ReadFile("./testdata/cert/server.crt")
			if err != nil {
				return false
			}
			key, err := ioutil.ReadFile("./testdata/cert/server.key")
			if err != nil {
				return false
			}
			c.Assert(cert, check.DeepEquals, tsuru.CertificateSetData{
				Cname:       "app.io",
				Certificate: certifica,
				Key:         key,
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := CertificateSet{}
	command.Flags().Parse(true, []string{"-a", "secret", "-c", "app.io"})
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
	command := CertificateSet{}
	command.Flags().Parse(true, []string{"-a", "secret", "-c", "app.io"})
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
			//cname := req.FormValue("cname") == "app.io"

			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := CertificateUnset{}
	command.Flags().Parse(true, []string{"-a", "secret", "-c", "app.io"})
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
	certMap := map[string]map[string]string{
		"ingress-router": {
			"myapp.io":       s.mustReadFileString(c, "./testdata/cert/server.crt"),
			"myapp.other.io": "",
		},
		"a-new-router": {
			"myapp.io": s.mustReadFileString(c, "./testdata/cert/server.crt"),
		},
	}
	data, err := json.Marshal(certMap)
	c.Assert(err, check.IsNil)
	expectedDate, err := time.Parse("2006-01-02 15:04:05", "2027-01-10 20:33:11")
	c.Assert(err, check.IsNil)
	datestr := formatter.Local(expectedDate).Format("2006-01-02 15:04:05")
	expected := `+----------------+----------------+---------------------+----------------------------+----------------------------+
| Router         | CName          | Expires             | Issuer                     | Subject                    |
+----------------+----------------+---------------------+----------------------------+----------------------------+
| a-new-router   | myapp.io       | ` + datestr + ` | C=BR; ST=Rio de Janeiro;   | C=BR; ST=Rio de Janeiro;   |
|                |                |                     | L=Rio de Janeiro; O=Tsuru; | L=Rio de Janeiro; O=Tsuru; |
|                |                |                     | CN=app.io                  | CN=app.io                  |
+----------------+----------------+---------------------+----------------------------+----------------------------+
| ingress-router | myapp.io       | ` + datestr + ` | C=BR; ST=Rio de Janeiro;   | C=BR; ST=Rio de Janeiro;   |
|                |                |                     | L=Rio de Janeiro; O=Tsuru; | L=Rio de Janeiro; O=Tsuru; |
|                |                |                     | CN=app.io                  | CN=app.io                  |
+----------------+----------------+---------------------+----------------------------+----------------------------+
| ingress-router | myapp.other.io | -                   | -                          | -                          |
+----------------+----------------+---------------------+----------------------------+----------------------------+
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
	command := CertificateList{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
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
	certMap := map[string]map[string]string{
		"ingress-router": {
			"myapp.io":       certData,
			"myapp.other.io": "",
		},
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
	command := CertificateList{}
	command.Flags().Parse(true, []string{"-a", "myapp", "-r"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(stdout.String(), "myapp.other.io:\nNo certificate."), check.Equals, true)
	c.Assert(strings.Contains(stdout.String(), "myapp.io:\n"+certData), check.Equals, true)
	c.Assert(requestCount, check.Equals, 1)
}
