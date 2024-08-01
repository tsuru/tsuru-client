// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"testing"

	"github.com/tsuru/gnuflag"
	check "gopkg.in/check.v1"
)

var appflag = &gnuflag.Flag{
	Name:     "app",
	Usage:    "The name of the app.",
	Value:    nil,
	DefValue: "",
}

var appshortflag = &gnuflag.Flag{
	Name:     "a",
	Usage:    "The name of the app.",
	Value:    nil,
	DefValue: "",
}

func Test(t *testing.T) { check.TestingT(t) }

type S struct{}

var _ = check.Suite(&S{})

func (s *S) TestAppNameMixInWithFlagDefined(c *check.C) {
	g := AppNameMixIn{}
	g.Flags().Parse(true, []string{"--app", "myapp"})
	name, err := g.AppNameByFlag()
	c.Assert(err, check.IsNil)
	c.Assert(name, check.Equals, "myapp")
}

func (s *S) TestAppNameMixInWithShortFlagDefined(c *check.C) {
	g := AppNameMixIn{}
	g.Flags().Parse(true, []string{"-a", "myapp"})
	name, err := g.AppNameByFlag()
	c.Assert(err, check.IsNil)
	c.Assert(name, check.Equals, "myapp")
}

func (s *S) TestAppNameMixInArgs(c *check.C) {
	g := AppNameMixIn{}
	g.Flags().Parse(true, []string{})
	name, err := g.AppNameByArgsAndFlag([]string{"myapp"})
	c.Assert(err, check.IsNil)
	c.Assert(name, check.Equals, "myapp")
}

func (s *S) TestAppNameMixInArgsConflict(c *check.C) {
	g := AppNameMixIn{}
	g.Flags().Parse(true, []string{"-a", "myapp"})
	_, err := g.AppNameByArgsAndFlag([]string{"myapp2"})
	c.Assert(err, check.Not(check.IsNil))
	c.Assert(err.Error(), check.Equals, "You can't use the app flag and specify the app name as an argument at the same time.")
}

func (s *S) TestAppNameMixInWithoutFlagDefinedFails(c *check.C) {
	g := AppNameMixIn{}
	name, err := g.AppNameByFlag()
	c.Assert(name, check.Equals, "")
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, `The name of the app is required.

Use the --app flag to specify it.

`)
}

func (s *S) TestAppNameMixInFlags(c *check.C) {
	var flags []gnuflag.Flag
	expected := []gnuflag.Flag{*appshortflag, *appflag}
	command := AppNameMixIn{}
	flagset := command.Flags()
	flagset.VisitAll(func(f *gnuflag.Flag) {
		f.Value = nil
		flags = append(flags, *f)
	})
	c.Assert(flags, check.DeepEquals, expected)
}
