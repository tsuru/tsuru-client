// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"io"
	"os"

	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestWriteToken(c *check.C) {
	rfs := &fstest.RecordingFs{}
	SetFileSystem(rfs)
	defer func() {
		ResetFileSystem()
	}()
	err := WriteTokenV1("abc")
	c.Assert(err, check.IsNil)
	tokenPath := JoinWithUserDir(".tsuru", "token")
	c.Assert(rfs.HasAction("create "+tokenPath), check.Equals, true)
	fil, _ := Filesystem().Open(tokenPath)
	b, _ := io.ReadAll(fil)
	c.Assert(string(b), check.Equals, "abc")
}

func (s *S) TestWriteTokenWithTarget(c *check.C) {
	rfs := &fstest.RecordingFs{}
	SetFileSystem(rfs)
	initTestTarget()

	defer func() {
		ResetFileSystem()
	}()
	err := WriteTokenV1("abc")
	c.Assert(err, check.IsNil)
	tokenPath1 := JoinWithUserDir(".tsuru", "token")
	c.Assert(rfs.HasAction("create "+tokenPath1), check.Equals, true)
	tokenPath2 := JoinWithUserDir(".tsuru", "token.d", "test")
	c.Assert(rfs.HasAction("create "+tokenPath2), check.Equals, true)
	fil, _ := Filesystem().Open(tokenPath1)
	b, _ := io.ReadAll(fil)
	c.Assert(string(b), check.Equals, "abc")
	fil, _ = Filesystem().Open(tokenPath2)
	b, _ = io.ReadAll(fil)
	c.Assert(string(b), check.Equals, "abc")
}

func (s *S) TestReadToken(c *check.C) {
	os.Unsetenv("TSURU_TOKEN")
	rfs := &fstest.RecordingFs{}
	SetFileSystem(rfs)
	initTestTarget()
	f, err := Filesystem().Create(JoinWithUserDir(".tsuru", "token.d", "test"))
	c.Assert(err, check.IsNil)
	f.WriteString("mytoken")
	defer func() {
		ResetFileSystem()
	}()
	token, err := ReadTokenV1()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.Equals, "mytoken")
	tokenPath := JoinWithUserDir(".tsuru", "token.d", "test")
	c.Assert(rfs.HasAction("open "+tokenPath), check.Equals, true)
	tokenPath = JoinWithUserDir(".tsuru", "token")
	c.Assert(rfs.HasAction("open "+tokenPath), check.Equals, false)
}

func (s *S) TestReadTokenFallback(c *check.C) {
	os.Unsetenv("TSURU_TOKEN")
	rfs := &fstest.RecordingFs{}
	SetFileSystem(rfs)
	defer func() {
		ResetFileSystem()
	}()

	initTestTarget()
	f, err := Filesystem().Create(JoinWithUserDir(".tsuru", "token"))
	c.Assert(err, check.IsNil)
	f.WriteString("mytoken")
	token, err := ReadTokenV1()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.Equals, "mytoken")
	tokenPath := JoinWithUserDir(".tsuru", "token.d", "test")
	c.Assert(rfs.HasAction("open "+tokenPath), check.Equals, true)
	tokenPath = JoinWithUserDir(".tsuru", "token")
	c.Assert(rfs.HasAction("open "+tokenPath), check.Equals, true)
}

func (s *S) TestReadTokenFileNotFound(c *check.C) {
	os.Unsetenv("TSURU_TOKEN")
	errFs := &fstest.FileNotFoundFs{}
	SetFileSystem(errFs)
	defer func() {
		ResetFileSystem()
	}()
	token, err := ReadTokenV1()
	c.Assert(err, check.IsNil)
	tokenPath := JoinWithUserDir(".tsuru", "token")
	c.Assert(err, check.IsNil)
	c.Assert(errFs.HasAction("open "+tokenPath), check.Equals, true)
	c.Assert(token, check.Equals, "")
}

func (s *S) TestReadTokenEnvironmentVariable(c *check.C) {
	os.Setenv("TSURU_TOKEN", "ABCDEFGH")
	defer os.Setenv("TSURU_TOKEN", "")
	token, err := ReadTokenV1()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.Equals, "ABCDEFGH")
}

func initTestTarget() {
	f, _ := Filesystem().Create(JoinWithUserDir(".tsuru", "target"))
	f.Write([]byte("http://localhost"))
	f.Close()
	WriteOnTargetList("test", "http://localhost")
}
