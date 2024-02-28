// Copyright 2023 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/tsuru/tsuru-client/tsuru/config"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec"
	"github.com/tsuru/tsuru/fs/fstest"

	"github.com/tsuru/tsuru/types/auth"
	"gopkg.in/check.v1"
)

type fakeExecutor struct {
	DoExecute func(opts exec.ExecuteOptions) error
}

func (f *fakeExecutor) Execute(opts exec.ExecuteOptions) error {
	return f.DoExecute(opts)
}

func (s *S) TestOAuthLogin(c *check.C) {

	config.SetFileSystem(&fstest.RecordingFs{})

	execut = &fakeExecutor{
		DoExecute: func(opts exec.ExecuteOptions) error {

			go func() {
				time.Sleep(time.Second)
				_, err := http.Get("http://localhost:41000")
				c.Assert(err, check.IsNil)
			}()

			return nil
		},
	}

	defer func() {
		config.ResetFileSystem()
		execut = nil
	}()

	fakeTsuruServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		c.Assert(req.URL.Path, check.Equals, "/1.0/auth/login")
		rw.Write([]byte(`{"token":"mytoken"}`))
	}))
	defer fakeTsuruServer.Close()

	os.Setenv("TSURU_TARGET", fakeTsuruServer.URL)

	context := &cmd.Context{
		Stdout: &bytes.Buffer{},
	}

	err := oauthLogin(context, &auth.SchemeInfo{
		Data: auth.SchemeData{
			Port: "41000",
		},
	})

	c.Assert(err, check.IsNil)
	tokenV1, err := config.ReadTokenV1()
	c.Assert(err, check.IsNil)
	c.Assert(tokenV1, check.Equals, "mytoken")
}
