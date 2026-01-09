// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import check "gopkg.in/check.v1"

func (s *S) TestMapFlag(c *check.C) {
	var f MapFlag
	f.Set("a=1")
	f.Set("b=2")
	f.Set("c=3")
	c.Assert(f, check.DeepEquals, MapFlag{
		"a": "1",
		"b": "2",
		"c": "3",
	})
}

func (s *S) TestMapFlagInvalid(c *check.C) {
	var f MapFlag
	err := f.Set("a")
	c.Assert(err, check.NotNil)
}

func (s *S) TestMapFlagWrapperInvalid(c *check.C) {
	m := make(map[string]string)
	f := MapFlagWrapper{Dst: &m}
	err := f.Set("a")
	c.Assert(err, check.NotNil)
}

func (s *S) TestMapFlagType(c *check.C) {
	var f MapFlag
	c.Assert(f.Type(), check.Equals, "key=value")
}

func (s *S) TestMapFlagStringEmpty(c *check.C) {
	var f MapFlag
	c.Assert(f.String(), check.Equals, "")
}

func (s *S) TestMapFlagStringWithValues(c *check.C) {
	f := MapFlag{"a": "1"}
	c.Assert(f.String(), check.Equals, `{"a":"1"}`)
}

func (s *S) TestStringSliceFlag(c *check.C) {
	var f StringSliceFlag
	f.Set("a")
	f.Set("b")
	f.Set("c")
	c.Assert(f, check.DeepEquals, StringSliceFlag{
		"a", "b", "c",
	})
}

func (s *S) TestStringSliceFlagType(c *check.C) {
	var f StringSliceFlag
	c.Assert(f.Type(), check.Equals, "string")
}

func (s *S) TestStringSliceFlagStringEmpty(c *check.C) {
	var f StringSliceFlag
	c.Assert(f.String(), check.Equals, "")
}

func (s *S) TestStringSliceFlagStringWithValues(c *check.C) {
	f := StringSliceFlag{"a", "b", "c"}
	c.Assert(f.String(), check.Equals, "a,b,c")
}

func (s *S) TestStringSliceFlagWrapperType(c *check.C) {
	var s2 []string
	f := StringSliceFlagWrapper{Dst: &s2}
	c.Assert(f.Type(), check.Equals, "string")
}

func (s *S) TestStringSliceFlagWrapperStringEmpty(c *check.C) {
	var s2 []string
	f := StringSliceFlagWrapper{Dst: &s2}
	c.Assert(f.String(), check.Equals, "")
}

func (s *S) TestStringSliceFlagWrapperSet(c *check.C) {
	var s2 []string
	f := StringSliceFlagWrapper{Dst: &s2}
	f.Set("a")
	f.Set("b")
	c.Assert(s2, check.DeepEquals, []string{"a", "b"})
}
