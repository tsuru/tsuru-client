// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"time"

	"github.com/tsuru/tsuru/fs/fstest"
	"golang.org/x/oauth2"
	"gopkg.in/check.v1"
)

func (s *S) TestWriteTokenV2(c *check.C) {
	rfs := &fstest.RecordingFs{}
	SetFileSystem(rfs)
	defer func() {
		ResetFileSystem()
	}()
	err := WriteTokenV2(TokenV2{
		Scheme: "oidc",
		OAuth2Token: &oauth2.Token{
			AccessToken:  "123",
			Expiry:       time.Now(),
			RefreshToken: "321",
		},
	})
	c.Assert(err, check.IsNil)
	tokenPath := JoinWithUserDir(".tsuru", "token-v2.json")
	c.Assert(rfs.HasAction("create "+tokenPath), check.Equals, true)
	fil, err := Filesystem().Open(tokenPath)
	c.Assert(err, check.IsNil)

	t := TokenV2{}
	err = json.NewDecoder(fil).Decode(&t)
	c.Assert(err, check.IsNil)
	c.Assert(t.Scheme, check.Equals, "oidc")
	c.Assert(t.OAuth2Token.AccessToken, check.Equals, "123")
	c.Assert(t.OAuth2Token.Expiry, check.Not(check.IsNil))
	c.Assert(t.OAuth2Token.RefreshToken, check.Equals, "321")

}

func (s *S) TestWriteTokenV2WithTarget(c *check.C) {
	rfs := &fstest.RecordingFs{}
	SetFileSystem(rfs)
	initTestTarget()

	defer func() {
		ResetFileSystem()
	}()
	err := WriteTokenV2(TokenV2{
		Scheme: "oidc",
		OAuth2Token: &oauth2.Token{
			AccessToken:  "123",
			Expiry:       time.Now(),
			RefreshToken: "321",
		},
	})
	c.Assert(err, check.IsNil)
	tokenPath1 := JoinWithUserDir(".tsuru", "token-v2.json")
	c.Assert(rfs.HasAction("create "+tokenPath1), check.Equals, true)
	tokenPath2 := JoinWithUserDir(".tsuru", "token-v2.d", "test.json")
	c.Assert(rfs.HasAction("create "+tokenPath2), check.Equals, true)

	fil, err := Filesystem().Open(tokenPath1)
	c.Assert(err, check.IsNil)
	t := TokenV2{}
	err = json.NewDecoder(fil).Decode(&t)
	c.Assert(err, check.IsNil)
	c.Assert(t.Scheme, check.Equals, "oidc")
	c.Assert(t.OAuth2Token.AccessToken, check.Equals, "123")
	c.Assert(t.OAuth2Token.Expiry, check.Not(check.IsNil))
	c.Assert(t.OAuth2Token.RefreshToken, check.Equals, "321")

	fil, err = Filesystem().Open(tokenPath2)
	c.Assert(err, check.IsNil)
	t = TokenV2{}
	err = json.NewDecoder(fil).Decode(&t)
	c.Assert(err, check.IsNil)
	c.Assert(t.Scheme, check.Equals, "oidc")
	c.Assert(t.OAuth2Token.AccessToken, check.Equals, "123")
	c.Assert(t.OAuth2Token.Expiry, check.Not(check.IsNil))
	c.Assert(t.OAuth2Token.RefreshToken, check.Equals, "321")
}

func (s *S) TestReadTokenV2(c *check.C) {
	rfs := &fstest.RecordingFs{}
	SetFileSystem(rfs)
	defer func() {
		ResetFileSystem()
	}()
	initTestTarget()

	f, err := Filesystem().Create(JoinWithUserDir(".tsuru", "token-v2.d", "test.json"))
	c.Assert(err, check.IsNil)
	f.WriteString(`{
		"scheme": "oidc",
		"oauth2_token": {
			"access_token": "321",
			"refresh_token": "123"
		}
	}`)

	token, err := ReadTokenV2()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.DeepEquals, &TokenV2{
		Scheme: "oidc",
		OAuth2Token: &oauth2.Token{
			AccessToken:  "321",
			RefreshToken: "123",
		},
	})
	tokenPath := JoinWithUserDir(".tsuru", "token-v2.d", "test.json")
	c.Assert(rfs.HasAction("open "+tokenPath), check.Equals, true)
	tokenPath = JoinWithUserDir(".tsuru", "token-v2.json")
	c.Assert(rfs.HasAction("open "+tokenPath), check.Equals, false)
}

func (s *S) TestReadTokenV2Fallback(c *check.C) {
	rfs := &fstest.RecordingFs{}
	SetFileSystem(rfs)
	defer func() {
		ResetFileSystem()
	}()

	initTestTarget()
	f, err := Filesystem().Create(JoinWithUserDir(".tsuru", "token-v2.json"))
	c.Assert(err, check.IsNil)
	f.WriteString(`{
		"scheme": "oidc",
		"oauth2_token": {
			"access_token": "321",
			"refresh_token": "123"
		}
	}`)
	token, err := ReadTokenV2()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.DeepEquals, &TokenV2{
		Scheme: "oidc",
		OAuth2Token: &oauth2.Token{
			AccessToken:  "321",
			RefreshToken: "123",
		},
	})
	tokenPath := JoinWithUserDir(".tsuru", "token-v2.d", "test.json")
	c.Assert(rfs.HasAction("open "+tokenPath), check.Equals, true)
	tokenPath = JoinWithUserDir(".tsuru", "token-v2.json")
	c.Assert(rfs.HasAction("open "+tokenPath), check.Equals, true)
}
