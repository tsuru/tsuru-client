// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsuru/tsuru-client/v2/internal/config"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

func TestNativeLogin(t *testing.T) {
	result := `{"token": "sometoken", "is_admin": true}`
	expected := "Email: Password: \nSuccessfully logged in!\n"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/users/foo@foo.com/tokens"))
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Equal(t, "chico", r.FormValue("password"))
		fmt.Fprint(w, result)
	}))

	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	tsuruCtx.SetTargetURL(mockServer.URL)

	tsuruCtx.SetToken("")
	tsuruCtx.AuthScheme = "native"
	tsuruCtx.Stdin = &tsuructx.FakeStdin{Reader: strings.NewReader("foo@foo.com\nchico\n")}

	cmd := NewLoginCmd(tsuruCtx)
	err := loginCmdRun(tsuruCtx, cmd, []string{})
	assert.NoError(t, err)
	assert.Equal(t, expected, tsuruCtx.Stdout.(*strings.Builder).String())
}

func TestNativeLoginWithoutEmailFromArg(t *testing.T) {
	result := `{"token": "sometoken", "is_admin": true}`
	expected := "Password: \nSuccessfully logged in!\n"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/users/foo@foo.com/tokens"))
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Equal(t, "chico", r.FormValue("password"))
		fmt.Fprint(w, result)
	}))

	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	tsuruCtx.SetTargetURL(mockServer.URL)

	tsuruCtx.SetToken("")
	tsuruCtx.AuthScheme = "native"
	tsuruCtx.Stdin = &tsuructx.FakeStdin{Reader: strings.NewReader("chico\n")}

	cmd := NewLoginCmd(tsuruCtx)
	err := loginCmdRun(tsuruCtx, cmd, []string{"foo@foo.com"})
	assert.NoError(t, err)
	assert.Equal(t, expected, tsuruCtx.Stdout.(*strings.Builder).String())
}

func TestNativeLoginShouldNotDependOnTsuruTokenFile(t *testing.T) {
	result := `{"token": "sometoken", "is_admin": true}`
	expected := "Password: \nSuccessfully logged in!\n"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/users/foo@foo.com/tokens"))
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Equal(t, "chico", r.FormValue("password"))
		fmt.Fprint(w, result)
	}))

	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	tsuruCtx.SetTargetURL(mockServer.URL)

	tsuruCtx.SetToken("")
	tsuruCtx.AuthScheme = "native"
	tsuruCtx.Stdin = &tsuructx.FakeStdin{Reader: strings.NewReader("chico\n")}

	f, err := tsuruCtx.Fs.Create(filepath.Join(config.ConfigPath, "target"))
	assert.NoError(t, err)
	_, err = f.WriteString("http://localhost")
	assert.NoError(t, err)
	f.Close()

	cmd := NewLoginCmd(tsuruCtx)
	err = loginCmdRun(tsuruCtx, cmd, []string{"foo@foo.com"})
	assert.NoError(t, err)
	assert.Equal(t, expected, tsuruCtx.Stdout.(*strings.Builder).String())
}

func TestNativeLoginNoPasswordError(t *testing.T) {
	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	tsuruCtx.SetToken("")
	tsuruCtx.AuthScheme = "native"

	cmd := NewLoginCmd(tsuruCtx)
	err := loginCmdRun(tsuruCtx, cmd, []string{"foo@foo.com"})
	assert.Equal(t, fmt.Errorf("empty password. You must provide the password"), err)
}
