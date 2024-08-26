// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestCertificateSetInfo(c *check.C) {
	c.Assert((&CertificateSet{}).Info(), check.NotNil)
}

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
	s.setupFakeTransport(trans)
	command := CertificateSet{}
	command.Flags().Parse(true, []string{"-a", "secret", "-c", "app.io"})
	c.Assert(command.cname, check.Equals, "app.io")
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Successfully created the certificate.\n")
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
	s.setupFakeTransport(trans)
	command := CertificateSet{}
	command.Flags().Parse(true, []string{"-a", "secret", "-c", "app.io"})
	c.Assert(command.cname, check.Equals, "app.io")
	err := command.Run(&context)
	c.Assert(os.IsNotExist(err), check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestCertificateUnsetInfo(c *check.C) {
	c.Assert((&CertificateUnset{}).Info(), check.NotNil)
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
	s.setupFakeTransport(trans)
	command := CertificateUnset{}
	command.Flags().Parse(true, []string{"-a", "secret", "-c", "app.io"})
	c.Assert(command.cname, check.Equals, "app.io")
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Certificate removed.\n")
	c.Assert(requestCount, check.Equals, 1)
}

func (s *S) mustReadFileString(c *check.C, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		c.Fatal(err)
		return ""
	}
	return string(data)
}

func (s *S) TestCertificateListInfo(c *check.C) {
	c.Assert((&CertificateList{}).Info(), check.NotNil)
}

func (s *S) TestCertificateListRunSuccessfully(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	requestCount := 0
	certData := s.mustReadFileString(c, "./testdata/cert/server.crt")
	appCert := appCertificate{
		RouterCertificates: map[string]routerCertificate{
			"ingress-router": {
				CNameCertificates: map[string]cnameCertificate{
					"myapp.io": {
						Certificate: certData,
						Issuer: "lets-encrypt",
					},
					"myapp.other.io": {
						Certificate: "",
					},
				},
			},
			"a-new-router": {
				CNameCertificates: map[string]cnameCertificate{
					"myapp.io": {
						Certificate: certData,
					},
				},
			},
		},
	}
	data, err := json.Marshal(appCert)
	c.Assert(err, check.IsNil)
	expectedNotBefore, err := time.Parse("2006-01-02 15:04:05", "2017-01-12 20:33:11")
	expectedNotAfter, err := time.Parse("2006-01-02 15:04:05", "2027-01-10 20:33:11")
	c.Assert(err, check.IsNil)
	notBeforeStr := expectedNotBefore.UTC().Format(time.RFC3339)
	notAfterStr := expectedNotAfter.UTC().Format(time.RFC3339)
	expected := `+----------------+----------------------------+-----------------------+----------------------+
| Router         | CName                      | Public Key Info       | Certificate Validity |
+----------------+----------------------------+-----------------------+----------------------+
| a-new-router   | myapp.io                   | Algorithm             | Not before           |
|                |                            | RSA                   | ` + notBeforeStr + ` |
|                |                            |                       |                      |
|                |                            | Key size (in bits)    | Not after            |
|                |                            | 2048                  | ` + notAfterStr + ` |
+----------------+----------------------------+-----------------------+----------------------+
| ingress-router | myapp.io                   | Algorithm             | Not before           |
|                |   managed by: cert-manager | RSA                   | ` + notBeforeStr + ` |
|                |   issuer: lets-encrypt     |                       |                      |
|                |                            | Key size (in bits)    | Not after            |
|                |                            | 2048                  | ` + notAfterStr +` |
+----------------+----------------------------+-----------------------+----------------------+
| ingress-router | myapp.other.io             | failed to decode data | -                    |
+----------------+----------------------------+-----------------------+----------------------+
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
	s.setupFakeTransport(trans)
	command := CertificateList{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err = command.Run(&context)
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
	appCert := appCertificate{
		RouterCertificates: map[string]routerCertificate{
			"ingress-router": {
				CNameCertificates: map[string]cnameCertificate{
					"myapp.io": {
						Certificate: certData,
						Issuer: "lets-encrypt",
					},
					"myapp.other.io": {
						Certificate: "",
					},
				},
			},
		},
	}
	data, err := json.Marshal(appCert)
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
	s.setupFakeTransport(trans)
	command := CertificateList{}
	command.Flags().Parse(true, []string{"-a", "myapp", "-r"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(stdout.String(), "myapp.other.io:\nNo certificate."), check.Equals, true)
	c.Assert(strings.Contains(stdout.String(), "myapp.io:\n"+certData), check.Equals, true)
	c.Assert(requestCount, check.Equals, 1)
}

func (s *S) TestCertificateIssuerSetInfo(c *check.C) {
	c.Assert((&CertificateIssuerSet{}).Info(), check.NotNil)
}

func (s *S) TestCertificateIssuerSetRunSuccessfully(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args: []string{
			"lets-encrypt",
		},
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/secret/certissuer")
			method := req.Method == http.MethodPut
			cname := req.FormValue("cname") == "app.io"
			issuer := req.FormValue("issuer") == "lets-encrypt"
			return url && method && cname && issuer
		},
	}
	s.setupFakeTransport(trans)
	command := CertificateIssuerSet{}
	command.Flags().Parse(true, []string{"-a", "secret", "-c", "app.io"})
	c.Assert(command.cname, check.Equals, "app.io")
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Successfully created the certificate issuer.\n")
}

func (s *S) TestCertificateIssuerUnsetInfo(c *check.C) {
	c.Assert((&CertificateIssuerUnset{}).Info(), check.NotNil)
}

func (s *S) TestCertificateIssuerUnsetRunSuccessfully(c *check.C) {
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
			url := strings.HasSuffix(req.URL.Path, "/apps/secret/certissuer")
			method := req.Method == http.MethodDelete
			cname := req.FormValue("cname") == "app.io"

			return url && method && cname
		},
	}
	s.setupFakeTransport(trans)
	command := CertificateIssuerUnset{}
	command.Flags().Parse(true, []string{"-a", "secret", "-c", "app.io"})
	c.Assert(command.cname, check.Equals, "app.io")
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Certificate issuer removed.\n")
	c.Assert(requestCount, check.Equals, 1)
}
