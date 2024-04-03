// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	tsuruerr "github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/fs/fstest"
	check "gopkg.in/check.v1"
)

type S struct{}

func (s *S) SetUpTest(c *check.C) {
	var stdout, stderr bytes.Buffer
	globalManager = cmd.NewManager("glb", &stdout, &stderr, os.Stdin, nil)
	//var exiter recordingExiter
	//globalManager.e = &exiter TODO
	os.Setenv("TSURU_TARGET", "http://localhost")
	os.Setenv("TSURU_TOKEN", "abc123")
	if env := os.Getenv("TERM"); env == "" {
		os.Setenv("TERM", "tsuruterm")
	}
}

func (s *S) TearDownTest(c *check.C) {
	os.Unsetenv("TSURU_TARGET")
	os.Unsetenv("TSURU_TOKEN")
}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

var globalManager *cmd.Manager

func (s *S) TestShouldSetCloseToTrue(c *check.C) {
	os.Setenv("TSURU_VERBOSITY", "2")
	defer func() {
		os.Unsetenv("TSURU_VERBOSITY")
	}()

	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	transport := cmdtest.Transport{
		Status:  http.StatusOK,
		Message: "OK",
	}
	var buf bytes.Buffer
	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper:  &transport,
		Stdout:        &buf,
		ClientName:    "test",
		ClientVersion: "0.1.0",
	})
	client.Do(request)
	c.Assert(request.Close, check.Equals, true)
	c.Assert(buf.String(), check.Matches,
		`(?s)`+
			`.*<Request uri="/">.*`+
			`GET / HTTP/1.1\r\n.*`+
			`<Response uri="/">.*`+
			`HTTP/0.0 200 OK.*`)
}

func (s *S) TestShouldReturnBodyMessageOnError(c *check.C) {
	os.Setenv("TSURU_VERBOSITY", "2")
	defer func() {
		os.Unsetenv("TSURU_VERBOSITY")
	}()

	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	var buf bytes.Buffer

	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper:  &cmdtest.Transport{Message: "You can't do this", Status: http.StatusForbidden},
		Stdout:        &buf,
		ClientName:    "test",
		ClientVersion: "0.1.0",
	})
	response, err := client.Do(request)
	c.Assert(response, check.IsNil)
	c.Assert(err, check.NotNil)
	urlErr, ok := err.(*url.Error)
	c.Assert(ok, check.Equals, true)
	httpErr, ok := urlErr.Err.(*tsuruerr.HTTP)
	c.Assert(ok, check.Equals, true)
	c.Assert(httpErr.Code, check.Equals, http.StatusForbidden)
	expectedMsg := "You can't do this"
	c.Assert(httpErr.Message, check.Equals, expectedMsg)
	c.Assert(buf.String(), check.Matches,
		`(?s)`+
			`.*<Request uri="/">.*`+
			`GET / HTTP/1.1\r\n.*`+
			`<Response uri="/">.*`+
			`HTTP/0.0 403 Forbidden.*`+
			`You can't do this.*`)
}

func (s *S) TestShouldReturnStatusMessageOnErrorWhenBodyIsEmpty(c *check.C) {
	os.Setenv("TSURU_VERBOSITY", "2")
	defer func() {
		os.Unsetenv("TSURU_VERBOSITY")
	}()

	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	var buf bytes.Buffer
	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper: &cmdtest.Transport{
			Message: "",
			Status:  http.StatusServiceUnavailable,
		},
		Stdout:        &buf,
		ClientName:    "test",
		ClientVersion: "0.1.0",
	})
	response, err := client.Do(request)
	c.Assert(err, check.NotNil)
	c.Assert(response, check.IsNil)
	expectedMsg := "503 Service Unavailable"
	urlErr, ok := err.(*url.Error)
	c.Assert(ok, check.Equals, true)
	httpErr := urlErr.Err.(*tsuruerr.HTTP)
	c.Assert(httpErr.Code, check.Equals, http.StatusServiceUnavailable)
	c.Assert(httpErr.Message, check.Equals, expectedMsg)
	c.Assert(buf.String(), check.Matches,
		`(?s)`+
			`.*<Request uri="/">.*`+
			`GET / HTTP/1.1\r\n.*`+
			`<Response uri="/">.*`+
			`HTTP/0.0 503 Service Unavailable\r\n`+
			`Content-Length: 0\r\n`+
			`\r\n`+
			`\*+ </Response uri="/">.*`)
}

func (s *S) TestShouldHandleUnauthorizedErrorSpecially(c *check.C) {
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	var buf bytes.Buffer
	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper: &cmdtest.Transport{Message: "unauthorized", Status: http.StatusUnauthorized},

		Stdout:        &buf,
		ClientName:    "test",
		ClientVersion: "0.1.0",
	})
	response, err := client.Do(request)
	c.Assert(response, check.IsNil)
	c.Assert(err.Error(), check.Equals, "Get \"/\": unauthorized")
}

func (s *S) TestShouldReturnErrorWhenServerIsDown(c *check.C) {
	os.Setenv("TSURU_VERBOSITY", "2")
	os.Unsetenv("TSURU_TARGET")
	config.SetFileSystem(&fstest.RecordingFs{FileContent: "http://tsuru.abc.xyz"})
	defer func() {
		os.Unsetenv("TSURU_VERBOSITY")
		config.ResetFileSystem()
	}()
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	var buf bytes.Buffer

	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper:  nil,
		Stdout:        &buf,
		ClientName:    "test",
		ClientVersion: "0.1.0",
	})
	_, err = client.Do(request)
	c.Assert(err, check.NotNil)
	c.Assert(strings.Contains(err.Error(), "Failed to connect to tsuru server (http://tsuru.abc.xyz), it's probably down"), check.Equals, true)
	c.Assert(strings.Replace(buf.String(), "\n", "\\n", -1), check.Matches,
		``+
			`.*<Request uri="/">.*`+
			`GET / HTTP/1.1\r\\n.*`)
}

func (s *S) TestShouldNotIncludeTheHeaderAuthorizationWhenTheTsuruTokenFileIsMissing(c *check.C) {
	os.Unsetenv("TSURU_TOKEN")
	os.Setenv("TSURU_VERBOSITY", "2")

	config.SetFileSystem(&fstest.FileNotFoundFs{})
	defer func() {
		os.Unsetenv("TSURU_VERBOSITY")
		config.ResetFileSystem()
	}()
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	trans := cmdtest.Transport{Message: "", Status: http.StatusOK}
	var buf bytes.Buffer
	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper:  &trans,
		Stdout:        &buf,
		ClientName:    "test",
		ClientVersion: "0.1.0",
	})
	_, err = client.Do(request)
	c.Assert(err, check.IsNil)
	header := map[string][]string(request.Header)
	_, ok := header["Authorization"]
	c.Assert(ok, check.Equals, false)
	c.Assert(strings.Replace(buf.String(), "\n", "\\n", -1), check.Matches,
		``+
			`.*<Request uri="/">.*`+
			`GET / HTTP/1.1\r\\n.*`)
}

func (s *S) TestShouldValidateVersion(c *check.C) {
	var buf bytes.Buffer
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	trans := cmdtest.Transport{
		Message: "",
		Status:  http.StatusOK,
		Headers: map[string][]string{"Supported-Tsuru": {"0.3"}},
	}
	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper:  &trans,
		Stderr:        &buf,
		ClientName:    "glb",
		ClientVersion: "0.2.1",
	})
	_, err = client.Do(request)
	c.Assert(err, check.IsNil)
	expected := `#####################################################################

WARNING: You're using an unsupported version of glb.

You must have at least version 0.3, your current
version is 0.2.1.

Please go to http://docs.tsuru.io/en/latest/using/install-client.html
and download the last version.

#####################################################################

`
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestShouldSkipValidationIfThereIsNoSupportedHeaderDeclared(c *check.C) {
	var buf bytes.Buffer
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	trans := cmdtest.Transport{Message: "", Status: http.StatusOK, Headers: map[string][]string{}}
	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper:  &trans,
		Stdout:        &buf,
		ClientName:    "glb",
		ClientVersion: "0.2.1",
	})
	_, err = client.Do(request)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "")
}

func (s *S) TestShouldSkupValidationIfServerDoesNotReturnSupportedHeader(c *check.C) {
	var buf bytes.Buffer
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	trans := cmdtest.Transport{Message: "", Status: http.StatusOK}
	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper:  &trans,
		Stdout:        &buf,
		ClientName:    "glb",
		ClientVersion: "0.2.1",
	})
	_, err = client.Do(request)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "")
}

func (s *S) TestShouldIncludeVerbosityHeader(c *check.C) {
	os.Setenv("TSURU_VERBOSITY", "2")
	defer func() {
		os.Unsetenv("TSURU_VERBOSITY")
	}()
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	trans := cmdtest.Transport{Message: "", Status: http.StatusOK}
	var buf bytes.Buffer
	client := NewTerminalClient(TerminalClientOptions{
		RoundTripper:  &trans,
		Stdout:        &buf,
		ClientName:    "glb",
		ClientVersion: "0.2.1",
	})
	_, err = client.Do(request)
	c.Assert(err, check.IsNil)
	c.Assert(request.Header.Get(verbosityHeader), check.Equals, "2")
}
