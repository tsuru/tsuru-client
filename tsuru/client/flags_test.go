// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"github.com/spf13/pflag"
	check "gopkg.in/check.v1"
)

func (s *S) TestMergeFlagSet(c *check.C) {
	var x, y bool
	fs1 := pflag.NewFlagSet("x", pflag.ExitOnError)
	fs1.BoolVarP(&x, "x", "x", false, "Something")
	fs2 := pflag.NewFlagSet("y", pflag.ExitOnError)
	fs2.BoolVarP(&y, "y", "y", false, "Something")
	ret := mergeFlagSet(fs1, fs2)
	c.Assert(ret, check.Equals, fs1)
	fs1.Parse([]string{"-x", "-y"})
	c.Assert(x, check.Equals, true)
	c.Assert(y, check.Equals, true)
}
