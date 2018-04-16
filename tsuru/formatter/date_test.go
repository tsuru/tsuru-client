// Copyright 2018 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package formatter

import (
	"time"

	check "gopkg.in/check.v1"
)

func (s *S) TestFormatDate(c *check.C) {
	parsedTs, err := time.Parse(time.RFC3339, "2018-02-16T11:03:00.000Z")
	c.Assert(err, check.IsNil)

	c.Assert(FormatDate(parsedTs), check.Equals, "16 Feb 18 05:03 CST")
	c.Assert(FormatDate(time.Time{}), check.Equals, "-")
}

func (s *S) TestFormatDuration(c *check.C) {
	duration := 75 * time.Second

	c.Assert(FormatDuration(&duration), check.Equals, "01:15")
	c.Assert(FormatDuration(nil), check.Equals, "…")
}

func (s *S) TestFormatDateAndDuration(c *check.C) {
	parsedTs, err := time.Parse(time.RFC3339, "2018-02-16T11:03:00.000Z")
	c.Assert(err, check.IsNil)
	duration := 123 * time.Second

	c.Assert(FormatDateAndDuration(parsedTs, &duration), check.Equals, "16 Feb 18 05:03 CST (02:03)")
	c.Assert(FormatDateAndDuration(parsedTs, nil), check.Equals, "16 Feb 18 05:03 CST (…)")
}
