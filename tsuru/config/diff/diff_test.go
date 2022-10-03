// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diff

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/check.v1"
)

type S struct{}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) TestDiff(c *check.C) {
	current := `# This is a file
this line will be kept
this line will be altered
`
	newer := `# This is a file
this line will be kept
this line was altered
`
	expected := `--- current
+++ newer
@@ -3 +3 @@
-this line will be altered
+this line was altered
`
	result, err := Diff(strings.NewReader(current), strings.NewReader(newer))
	c.Assert(err, check.IsNil)
	c.Assert(string(result), check.Equals, expected)

	result, err = Diff(strings.NewReader("no changes"), strings.NewReader("no changes"))
	c.Assert(err, check.IsNil)
	c.Assert(string(result), check.Equals, "")
}

func (s *S) TestErrOrEmpty(c *check.C) {
	c.Assert(errOrEmpty(nil), check.Equals, "")
	c.Assert(errOrEmpty(fmt.Errorf("This is a new error")), check.Equals, "This is a new error")
}
