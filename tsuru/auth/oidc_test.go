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
	"time"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec"
	"github.com/tsuru/tsuru/fs/fstest"
	"golang.org/x/oauth2"

	"github.com/tsuru/tsuru/types/auth"
	"gopkg.in/check.v1"
)

func (s *S) TestOIDChLogin(c *check.C) {

	config.SetFileSystem(&fstest.RecordingFs{})

	execut = &fakeExecutor{
		DoExecute: func(opts exec.ExecuteOptions) error {

			go func() {
				time.Sleep(time.Second)
				_, err := http.Get("http://localhost:41000/?code=321")
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

type fakeTokenSource struct{}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: "access-token-321",
		Expiry:      time.Now().Add(time.Hour),
	}, nil
}

func (s *S) TestTokenSourceFSStorage(c *check.C) {

	config.SetFileSystem(&fstest.RecordingFs{})

	defer func() {
		config.ResetFileSystem()
	}()

	fts := &fakeTokenSource{}
	tokenSourceFSStorage := &TokenSourceFSStorage{
		BaseTokenSource: fts,
		LastToken: &config.TokenV2{
			OAuth2Token: &oauth2.Token{
				AccessToken: "access-token-123",
			},
		},
	}

	token, err := tokenSourceFSStorage.Token()
	c.Assert(err, check.IsNil)

	c.Assert(token.AccessToken, check.Equals, "access-token-321")

	tokenV1fromConfig, err := config.ReadTokenV1()
	c.Assert(err, check.IsNil)
	c.Assert(tokenV1fromConfig, check.Equals, "access-token-321")

	tokenV2fromConfig, err := config.ReadTokenV2()
	c.Assert(err, check.IsNil)
	c.Assert(tokenV2fromConfig.OAuth2Token.AccessToken, check.Equals, "access-token-321")

}
