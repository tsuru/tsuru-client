// Copyright 2018 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package formatter

import (
	"testing"
	"time"

	"gopkg.in/check.v1"
)

type S struct {
	defaultLocation time.Location
}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) SetUpTest(c *check.C) {
	s.defaultLocation = *time.Local
	location, err := time.LoadLocation("US/Central")
	if err == nil {
		time.Local = location
	}
}

func (s *S) TearDownTest(c *check.C) {
	time.Local = &s.defaultLocation
}
