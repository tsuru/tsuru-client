// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsuru/tsuru-client/v2/internal/config"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

func TestNewLogoutCmd(t *testing.T) {
	assert.NotNil(t, NewLogoutCmd(tsuructx.TsuruContextWithConfig(nil)))
}

func TestLogoutCmdRun(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/users/tokens"))
		w.WriteHeader(http.StatusOK)
	}))

	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	tsuruCtx.SetTargetURL(mockServer.URL)

	// setup current fs state //////////////////////////////////////////////////
	f, err := tsuruCtx.Fs.Create(filepath.Join(config.ConfigPath, "target"))
	assert.NoError(t, err)
	f.Write([]byte("http://localhost:8080"))
	f.Close()
	f, err = tsuruCtx.Fs.Create(filepath.Join(config.ConfigPath, "targets"))
	assert.NoError(t, err)
	f.Write([]byte("default http://localhost:8080"))
	f.Close()
	f, err = tsuruCtx.Fs.Create(filepath.Join(config.ConfigPath, "token"))
	assert.NoError(t, err)
	f.Write([]byte("sometoken"))
	f.Close()
	f, err = tsuruCtx.Fs.Create(filepath.Join(config.ConfigPath, "token.d", "default"))
	assert.NoError(t, err)
	f.Write([]byte("sometoken"))
	f.Close()
	////////////////////////////////////////////////////////////////////////////

	logoutCmd := NewLogoutCmd(tsuruCtx)
	err = logoutCmdRun(tsuruCtx, logoutCmd, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Successfully logged out!\n", tsuruCtx.Stdout.(*strings.Builder).String())
}

func TestLogoutCmdRunServerError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/users/tokens"))
		w.WriteHeader(http.StatusForbidden)
	}))

	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	tsuruCtx.SetTargetURL(mockServer.URL)

	logoutCmd := NewLogoutCmd(tsuruCtx)
	err := logoutCmdRun(tsuruCtx, logoutCmd, nil)
	assert.ErrorContains(t, err, "unexpected response from server: 403: 403 Forbidden")
	assert.Equal(t, "Logged out, but some errors occurred:\n", tsuruCtx.Stdout.(*strings.Builder).String())
}
