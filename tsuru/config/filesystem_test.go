// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"github.com/tsuru/tsuru/fs"
	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestFileSystem(c *check.C) {
	SetFileSystem(&fstest.RecordingFs{})
	c.Assert(Filesystem(), check.DeepEquals, &fstest.RecordingFs{})
	ResetFileSystem()
	c.Assert(Filesystem(), check.DeepEquals, &fs.OsFs{})
}
