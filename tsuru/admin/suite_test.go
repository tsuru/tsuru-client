// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cezarsa/form"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	check "gopkg.in/check.v1"
)

type S struct {
	//manager         *cmd.Manager
	defaultLocation time.Location
}

func (s *S) SetUpSuite(c *check.C) {
	//var stdout, stderr bytes.Buffer
	//s.manager = cmd.NewManagerPanicExiter("glb", "1.0.0", "Supported-Tsuru-Version", &stdout, &stderr, os.Stdin, nil)
	os.Setenv("TSURU_TARGET", "http://localhost")
	form.DefaultEncoder = form.DefaultEncoder.UseJSONTags(false)
	form.DefaultDecoder = form.DefaultDecoder.UseJSONTags(false)
}

func (s *S) TearDownSuite(c *check.C) {
	os.Unsetenv("TSURU_TARGET")
}

func (s *S) SetUpTest(c *check.C) {
	cmd.DisableColors = true
	s.defaultLocation = *formatter.LocalTZ
	location, err := time.LoadLocation("US/Central")
	if err == nil {
		formatter.LocalTZ = location
	}
}

func (s *S) TearDownTest(c *check.C) {
	formatter.LocalTZ = &s.defaultLocation
	tsuruHTTP.AuthenticatedClient = &http.Client{}
}

func (s *S) setupFakeTransport(rt http.RoundTripper) {
	tsuruHTTP.AuthenticatedClient = tsuruHTTP.NewTerminalClient(tsuruHTTP.TerminalClientOptions{
		RoundTripper:  rt,
		ClientName:    "test",
		ClientVersion: "0.1.0",
	})
}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }
