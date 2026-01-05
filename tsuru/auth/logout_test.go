// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package auth

import (
	"bytes"
	"net/http"
	"os"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestLogout(c *check.C) {
	var called bool
	rfs := &fstest.RecordingFs{}
	config.SetFileSystem(rfs)
	defer func() {
		config.ResetFileSystem()
	}()
	config.WriteTokenV1("mytoken")
	os.Setenv("TSURU_TARGET", "localhost:8080")
	expected := "Successfully logged out!\n"
	context := cmd.Context{
		Args:   []string{},
		Stdout: bytes.NewBufferString(""),
	}
	command := Logout{}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: "",
			Status:  http.StatusOK,
		},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && req.URL.Path == "/users/tokens" &&
				req.Header.Get("Authorization") == "bearer mytoken"
		},
	}
	setupFakeTransport(&transport)
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(context.Stdout.(*bytes.Buffer).String(), check.Equals, expected)
	c.Assert(rfs.HasAction("remove "+config.JoinWithUserDir(".tsuru", "token")), check.Equals, true)
	c.Assert(called, check.Equals, true)
}

func (s *S) TestLogoutNoTarget(c *check.C) {
	rfs := &fstest.RecordingFs{}
	config.SetFileSystem(rfs)
	defer func() {
		config.ResetFileSystem()
	}()
	config.WriteTokenV1("mytoken")
	expected := "Successfully logged out!\n"
	context := cmd.Context{
		Args:   []string{},
		Stdout: bytes.NewBufferString(""),
	}
	command := Logout{}
	transport := cmdtest.Transport{Message: "", Status: http.StatusOK}
	setupFakeTransport(transport)
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(context.Stdout.(*bytes.Buffer).String(), check.Equals, expected)
	c.Assert(rfs.HasAction("remove "+config.JoinWithUserDir(".tsuru", "token")), check.Equals, true)
}
