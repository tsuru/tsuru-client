// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/installer/defaultconfig"
	"gopkg.in/check.v1"
)

func (s *S) TestResolverConfig(c *check.C) {
	tt := []struct {
		Description string
		Base        string
		Config      map[string]string
		Result      string
		ErrMsg      string
	}{
		{"invalid path", "invalid", nil, "", ".*(cannot find the file|no such file).*"},
		{"default configuration", "", nil, defaultconfig.Compose, ""},
		{"custom parameter", "", map[string]string{"TSURU_API_IMAGE": "INJECT_TEST"}, "INJECT_TEST", ""},
	}

	for _, tc := range tt {
		result, err := resolveConfig(tc.Base, tc.Config)
		if len(tc.ErrMsg) > 0 {
			c.Assert(err, check.ErrorMatches, tc.ErrMsg)
		} else {
			c.Assert(err, check.IsNil)
		}
		c.Assert(strings.Contains(result, tc.Result), check.Equals, true)
	}
}
