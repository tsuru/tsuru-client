// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"io"

	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestWriteToken(c *check.C) {
	rfs := &fstest.RecordingFs{}
	SetFileSystem(rfs)
	defer func() {
		ResetFileSystem()
	}()
	err := WriteToken("abc")
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
	err := WriteToken("abc")
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

func initTestTarget() {
	f, _ := Filesystem().Create(JoinWithUserDir(".tsuru", "target"))
	f.Write([]byte("http://localhost"))
	f.Close()
	WriteOnTargetList("test", "http://localhost")
}
