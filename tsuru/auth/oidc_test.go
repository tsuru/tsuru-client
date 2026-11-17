// Copyright 2023 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package auth

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru/exec"
	"github.com/tsuru/tsuru/fs/fstest"
	"golang.org/x/oauth2"

	"github.com/tsuru/tsuru/types/auth"
	"gopkg.in/check.v1"
)

// openedURL returns the URL handed to the browser opener: the last argument on
// every OS (Windows runs `cmd /c start "" <url>`), with Windows' ^& escaping
// undone.
func openedURL(args []string) string {
	return strings.ReplaceAll(args[len(args)-1], "^&", "&")
}

func (s *S) TestOIDChLogin(c *check.C) {

	config.SetFileSystem(&fstest.RecordingFs{})

	execut = &fakeExecutor{
		DoExecute: func(opts exec.ExecuteOptions) error {

			authURL, err := url.Parse(openedURL(opts.Args))
			c.Assert(err, check.IsNil)
			state := authURL.Query().Get("state")
			c.Assert(len(state) >= 32, check.Equals, true)
			go func() {
				time.Sleep(time.Second)
				_, err := http.Get("http://localhost:41000/?code=321&state=" + url.QueryEscape(state))
				c.Assert(err, check.IsNil)
			}()

			return nil
		},
	}

	defer func() {
		config.ResetFileSystem()
		execut = nil
	}()

	fakeIDP := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		c.Assert(err, check.IsNil)
		body, err := url.ParseQuery(string(b))
		c.Assert(err, check.IsNil)

		c.Assert(body.Get("code"), check.Equals, "321")

		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"access_token":"mytoken", "refresh_token": "refreshtoken"}`))
	}))
	defer fakeIDP.Close()

	context := &cmd.Context{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	err := oidcLogin(context, &auth.SchemeInfo{
		Data: auth.SchemeData{
			Port:     "41000",
			TokenURL: fakeIDP.URL,
			ClientID: "test-tsuru",
			Scopes:   []string{"scope1"},
		},
	})

	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(context.Stderr.(*bytes.Buffer).String(), "The OIDC token will expire in"), check.Equals, true)
	tokenV1, err := config.ReadTokenV1()
	c.Assert(err, check.IsNil)
	c.Assert(tokenV1, check.Equals, "mytoken")

	tokenV2, err := config.ReadTokenV2()
	c.Assert(err, check.IsNil)
	c.Assert(tokenV2, check.DeepEquals, &config.TokenV2{
		Scheme: "oidc",
		OAuth2Token: &oauth2.Token{
			AccessToken:  "mytoken",
			RefreshToken: "refreshtoken",
		},
		OAuth2Config: &oauth2.Config{
			ClientID:    "test-tsuru",
			RedirectURL: "http://localhost:41000",
			Scopes:      []string{"scope1"},
			Endpoint: oauth2.Endpoint{
				TokenURL: fakeIDP.URL,
			},
		},
	})
}

func (s *S) TestOIDCLoginErrorRedirect(c *check.C) {
	config.SetFileSystem(&fstest.RecordingFs{})

	bodyCh := make(chan string, 1)
	execut = &fakeExecutor{
		DoExecute: func(opts exec.ExecuteOptions) error {
			go func() {
				time.Sleep(time.Second)
				resp, err := http.Get("http://localhost:41001/?error=invalid_request&error_description=Invalid+scopes:+openid")
				c.Assert(err, check.IsNil)
				defer resp.Body.Close()
				b, err := io.ReadAll(resp.Body)
				c.Assert(err, check.IsNil)
				bodyCh <- string(b)
			}()
			return nil
		},
	}

	defer func() {
		config.ResetFileSystem()
		execut = nil
	}()

	var tokenEndpointCalls int32
	fakeIDP := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&tokenEndpointCalls, 1)
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(`{"error":"invalid_grant","error_description":"Code not valid"}`))
	}))
	defer fakeIDP.Close()

	stderr := &bytes.Buffer{}
	context := &cmd.Context{
		Stdout: &bytes.Buffer{},
		Stderr: stderr,
	}

	err := oidcLogin(context, &auth.SchemeInfo{
		Data: auth.SchemeData{
			Port:     "41001",
			TokenURL: fakeIDP.URL,
			ClientID: "test-tsuru",
			Scopes:   []string{"scope1"},
		},
	})
	c.Assert(err, check.IsNil)

	body := <-bodyCh
	c.Assert(strings.Contains(body, "invalid_request"), check.Equals, true)
	c.Assert(strings.Contains(body, "Invalid scopes: openid"), check.Equals, true)
	c.Assert(strings.Contains(body, "invalid_grant"), check.Equals, false)
	c.Assert(strings.Contains(stderr.String(), "invalid_request"), check.Equals, true)
	c.Assert(strings.Contains(stderr.String(), "Invalid scopes: openid"), check.Equals, true)
	c.Assert(atomic.LoadInt32(&tokenEndpointCalls), check.Equals, int32(0))

	tokenV1, _ := config.ReadTokenV1()
	c.Assert(tokenV1, check.Equals, "")
}

func (s *S) TestOIDCLoginMissingCode(c *check.C) {
	config.SetFileSystem(&fstest.RecordingFs{})

	bodyCh := make(chan string, 1)
	execut = &fakeExecutor{
		DoExecute: func(opts exec.ExecuteOptions) error {
			go func() {
				time.Sleep(time.Second)
				resp, err := http.Get("http://localhost:41002/")
				c.Assert(err, check.IsNil)
				defer resp.Body.Close()
				b, err := io.ReadAll(resp.Body)
				c.Assert(err, check.IsNil)
				bodyCh <- string(b)
			}()
			return nil
		},
	}

	defer func() {
		config.ResetFileSystem()
		execut = nil
	}()

	var tokenEndpointCalls int32
	fakeIDP := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&tokenEndpointCalls, 1)
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(`{"error":"invalid_grant","error_description":"Code not valid"}`))
	}))
	defer fakeIDP.Close()

	stderr := &bytes.Buffer{}
	context := &cmd.Context{
		Stdout: &bytes.Buffer{},
		Stderr: stderr,
	}

	err := oidcLogin(context, &auth.SchemeInfo{
		Data: auth.SchemeData{
			Port:     "41002",
			TokenURL: fakeIDP.URL,
			ClientID: "test-tsuru",
			Scopes:   []string{"scope1"},
		},
	})
	c.Assert(err, check.IsNil)

	body := <-bodyCh
	c.Assert(strings.Contains(body, "missing 'code' parameter"), check.Equals, true)
	c.Assert(strings.Contains(stderr.String(), "missing 'code' parameter"), check.Equals, true)
	c.Assert(atomic.LoadInt32(&tokenEndpointCalls), check.Equals, int32(0))
}

func (s *S) TestOIDCLoginStateMismatch(c *check.C) {
	config.SetFileSystem(&fstest.RecordingFs{})

	bodyCh := make(chan string, 1)
	execut = &fakeExecutor{
		DoExecute: func(opts exec.ExecuteOptions) error {
			go func() {
				time.Sleep(time.Second)
				resp, err := http.Get("http://localhost:41003/?code=321&state=forged-state")
				c.Assert(err, check.IsNil)
				defer resp.Body.Close()
				b, err := io.ReadAll(resp.Body)
				c.Assert(err, check.IsNil)
				bodyCh <- string(b)
			}()
			return nil
		},
	}

	defer func() {
		config.ResetFileSystem()
		execut = nil
	}()

	var tokenEndpointCalls int32
	fakeIDP := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&tokenEndpointCalls, 1)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"access_token":"mytoken"}`))
	}))
	defer fakeIDP.Close()

	stderr := &bytes.Buffer{}
	context := &cmd.Context{
		Stdout: &bytes.Buffer{},
		Stderr: stderr,
	}

	err := oidcLogin(context, &auth.SchemeInfo{
		Data: auth.SchemeData{
			Port:     "41003",
			TokenURL: fakeIDP.URL,
			ClientID: "test-tsuru",
			Scopes:   []string{"scope1"},
		},
	})
	c.Assert(err, check.IsNil)

	body := <-bodyCh
	c.Assert(strings.Contains(body, "does not match this login"), check.Equals, true)
	c.Assert(strings.Contains(stderr.String(), "does not match this login"), check.Equals, true)
	c.Assert(atomic.LoadInt32(&tokenEndpointCalls), check.Equals, int32(0))

	tokenV1, _ := config.ReadTokenV1()
	c.Assert(tokenV1, check.Equals, "")
}

func (s *S) TestOpenedURL(c *check.C) {
	windows := []string{"/c", "start", "", "https://idp/authorize?client_id=tsuru^&state=abc123^&response_type=code"}
	u, err := url.Parse(openedURL(windows))
	c.Assert(err, check.IsNil)
	c.Assert(u.Query().Get("state"), check.Equals, "abc123")
	c.Assert(u.Query().Get("client_id"), check.Equals, "tsuru")

	darwin := []string{"https://idp/authorize?client_id=tsuru&state=abc123"}
	u, err = url.Parse(openedURL(darwin))
	c.Assert(err, check.IsNil)
	c.Assert(u.Query().Get("state"), check.Equals, "abc123")
}
