// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"time"

	check "gopkg.in/check.v1"
)

func (s *S) TestFormatDate(c *check.C) {
	startTs := "2018-02-16T11:03:00.000Z"
	parsedTs, err := time.Parse(time.RFC3339, startTs)
	c.Assert(err, check.IsNil)
	formattedTs := parsedTs.Local().Format(time.Stamp)

	c.Assert(formatDate(parsedTs), check.Equals, formattedTs)
}

func (s *S) TestFormatDuration(c *check.C) {
	duration := 75 * time.Second

	c.Assert(formatDuration(&duration), check.Equals, "01:15")
	c.Assert(formatDuration(nil), check.Equals, "…")
}

func (s *S) TestFormatDateAndDuration(c *check.C) {
	startTs := "2018-02-16T11:03:00.000Z"
	parsedTs, err := time.Parse(time.RFC3339, startTs)
	c.Assert(err, check.IsNil)
	formattedTs := parsedTs.Local().Format(time.Stamp)
	duration := 123 * time.Second

	c.Assert(formatDateAndDuration(parsedTs, &duration), check.Equals, formattedTs+" (02:03)")
	c.Assert(formatDateAndDuration(parsedTs, nil), check.Equals, formattedTs+" (…)")
}
