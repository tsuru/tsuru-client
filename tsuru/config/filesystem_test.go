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
	fsystem = &fstest.RecordingFs{}
	c.Assert(filesystem(), check.DeepEquals, fsystem)
	fsystem = nil
	c.Assert(filesystem(), check.DeepEquals, fs.OsFs{})
}
