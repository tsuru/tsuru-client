// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package selfupdater

import (
	"testing"
	"time"

	"github.com/tsuru/tsuru/fs"
	"gopkg.in/check.v1"
)

type S struct{}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) SetUpTest(c *check.C) {
	nowUTC = func() time.Time { return time.Now().UTC() } // so we can test time-dependent features
	fsystem = fs.OsFs{}
}
