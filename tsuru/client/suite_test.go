// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/ajg/form"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
	"gopkg.in/check.v1"
)

type S struct {
	defaultLocation time.Location
}

func (s *S) SetUpSuite(c *check.C) {
	os.Setenv("TSURU_TARGET", "http://localhost:8080")
	os.Setenv("TSURU_TOKEN", "sometoken")
	form.DefaultEncoder = form.DefaultEncoder.UseJSONTags(false)
	form.DefaultDecoder = form.DefaultDecoder.UseJSONTags(false)
}

func (s *S) TearDownSuite(c *check.C) {
	os.Unsetenv("TSURU_TARGET")
	os.Unsetenv("TSURU_TOKEN")
}

func (s *S) SetUpTest(c *check.C) {
	var stdout, stderr bytes.Buffer
	manager = cmd.NewManager("glb", "1.0.0", "Supported-Tsuru", &stdout, &stderr, os.Stdin, nil)

	s.defaultLocation = *formatter.LocalTZ
	location, err := time.LoadLocation("US/Central")
	if err == nil {
		formatter.LocalTZ = location
	}
}

func (s *S) TearDownTest(c *check.C) {
	formatter.LocalTZ = &s.defaultLocation
}

var _ = check.Suite(&S{})
var manager *cmd.Manager

func Test(t *testing.T) { check.TestingT(t) }
