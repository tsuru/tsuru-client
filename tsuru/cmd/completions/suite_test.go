// Copyright 2024 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package completions

import (
	"net/http"
	"os"
	"testing"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
)

func setupTest(t *testing.T) {
	t.Helper()
	os.Setenv("TSURU_TARGET", "http://localhost:8080")
	os.Setenv("TSURU_TOKEN", "sometoken")
	config.ResetFileSystem()

	t.Cleanup(func() {
		os.Unsetenv("TSURU_TARGET")
		os.Unsetenv("TSURU_TOKEN")
	})
}

func setupFakeTransport(rt http.RoundTripper) {
	tsuruHTTP.AuthenticatedClient = tsuruHTTP.NewTerminalClient(tsuruHTTP.TerminalClientOptions{
		RoundTripper:  rt,
		ClientName:    "test",
		ClientVersion: "0.1.0",
	})
}
