// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsuru/tsuru-client/v2/internal/config"
	"github.com/tsuru/tsuru-client/v2/internal/exec"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

func TestPort(t *testing.T) {
	assert.Equal(t, ":0", port(map[string]string{}))
	assert.Equal(t, ":4242", port(map[string]string{"port": "4242"}))
}

func TestCallbackHandler(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"token": "xpto"}`)
	}))
	defer mockServer.Close()

	redirectURL := "someurl"
	finish := make(chan bool, 1)
	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	tsuruCtx.SetTargetURL(mockServer.URL)

	callbackHandler := callback(tsuruCtx, redirectURL, finish)
	request, err := http.NewRequest("GET", "/", strings.NewReader(`{"code":"xpto"}`))
	assert.NoError(t, err)
	recorder := httptest.NewRecorder()
	callbackHandler(recorder, request)

	assert.Equal(t, true, <-finish)
	assert.Equal(t, fmt.Sprintf(callbackPage, successMarkup), recorder.Body.String())
	file, err := tsuruCtx.Fs.Open(filepath.Join(config.ConfigPath, "token"))
	assert.NoError(t, err)
	data, err := io.ReadAll(file)
	assert.NoError(t, err)
	assert.Equal(t, "xpto", string(data))
}

func TestOauthLogin(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get(r.URL.Query().Get("redirect_uri") + "/")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		fmt.Fprintln(w, `{"code": "aRandomCode"}`)
	}))

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/auth/login"))
		fmt.Fprintln(w, `{"token": "mytoken"}`)
	}))

	ls := loginScheme{Name: "oauth", Data: map[string]string{
		"authorizeUrl": authServer.URL + "/authorize?redirect_uri=__redirect_url__",
	}}
	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	tsuruCtx.SetTargetURL(mockServer.URL)
	tsuruCtx.Executor = &mockExec{url: authServer.URL}

	err := oauthLogin(tsuruCtx, &ls)
	assert.NoError(t, err)

	f1, err := tsuruCtx.Fs.Open(filepath.Join(config.ConfigPath, "token"))
	assert.NoError(t, err)
	readToken, err := io.ReadAll(f1)
	assert.NoError(t, err)
	assert.Equal(t, "mytoken", string(readToken))
}

func TestOauthLoginSaveAlias(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get(r.URL.Query().Get("redirect_uri") + "/")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		fmt.Fprintln(w, `{"code": "aRandomCode"}`)
	}))

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/auth/login"))
		fmt.Fprintln(w, `{"token": "mytoken"}`)
	}))

	ls := loginScheme{Name: "oauth", Data: map[string]string{
		"authorizeUrl": authServer.URL + "/authorize?redirect_uri=__redirect_url__",
	}}
	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	tsuruCtx.SetTargetURL(mockServer.URL)
	tsuruCtx.Executor = &mockExec{url: authServer.URL}

	// setup current fs state //////////////////////////////////////////////////
	f, err := tsuruCtx.Fs.Create(filepath.Join(config.ConfigPath, "target"))
	assert.NoError(t, err)
	f.Write([]byte("http://localhost:8080"))
	f.Close()
	f, err = tsuruCtx.Fs.Create(filepath.Join(config.ConfigPath, "targets"))
	assert.NoError(t, err)
	f.Write([]byte("default http://localhost:8080"))
	f.Close()
	////////////////////////////////////////////////////////////////////////////

	err = oauthLogin(tsuruCtx, &ls)
	assert.NoError(t, err)

	f, err = tsuruCtx.Fs.Open(filepath.Join(config.ConfigPath, "token"))
	assert.NoError(t, err)
	readToken, err := io.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, "mytoken", string(readToken))
	f, err = tsuruCtx.Fs.Open(filepath.Join(config.ConfigPath, "token.d", "default"))
	assert.NoError(t, err)
	readToken, err = io.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, "mytoken", string(readToken))
}

type mockExec struct {
	url string
}

func (m *mockExec) Command(opts exec.ExecuteOptions) error {
	url := opts.Args[0]
	if runtime.GOOS == "windows" {
		url = opts.Args[3]
	}
	http.Get(url)
	return nil
}
