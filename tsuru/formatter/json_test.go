// Copyright 2026 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package formatter

import (
	"bytes"

	"gopkg.in/check.v1"
)

func (s *S) TestEmptyJSON(c *check.C) {
	var buf bytes.Buffer
	err := EmptyJSON(&buf)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "[]\n")
}
