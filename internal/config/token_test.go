// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

func createFileWithContent(t *testing.T, fsys afero.Fs, path, content string) {
	f, err := fsys.Create(path)
	assert.NoError(t, err)
	fmt.Fprint(f, content)
}

func readFile(t *testing.T, fsys afero.Fs, path string) string {
	f, err := fsys.Open(path)
	assert.NoError(t, err)
	b, err := io.ReadAll(f)
	assert.NoError(t, err)
	return string(b)
}

func TestGetTokenFromFs(t *testing.T) {
	t.Parallel()
	t.Run("current_token", func(t *testing.T) {
		tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tsuruCtx.Fs, filepath.Join(ConfigPath, "token"), "mycurrenttoken")

		got, err := GetTokenFromFs(tsuruCtx.Fs, "")
		assert.NoError(t, err)
		assert.Equal(t, "mycurrenttoken", got)
	})

	t.Run("not_current_token", func(t *testing.T) {
		tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tsuruCtx.Fs, filepath.Join(ConfigPath, "token"), "mycurrenttoken")
		createFileWithContent(t, tsuruCtx.Fs, filepath.Join(ConfigPath, "token.d", "default"), "mydefaulttoken")
		createFileWithContent(t, tsuruCtx.Fs, filepath.Join(ConfigPath, "targets"), "default\thttp://xxx.xxx")

		got, err := GetTokenFromFs(tsuruCtx.Fs, "default")
		assert.NoError(t, err)
		assert.Equal(t, "mydefaulttoken", got)
	})

	t.Run("not_found_fallback_current_token", func(t *testing.T) {
		tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tsuruCtx.Fs, filepath.Join(ConfigPath, "token"), "mycurrenttoken")
		createFileWithContent(t, tsuruCtx.Fs, filepath.Join(ConfigPath, "token.d", "default"), "mydefaulttoken")
		createFileWithContent(t, tsuruCtx.Fs, filepath.Join(ConfigPath, "targets"), "default\thttp://xxx.xxx")

		got, err := GetTokenFromFs(tsuruCtx.Fs, "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, "mycurrenttoken", got)
	})

	t.Run("noErr_not_found", func(t *testing.T) {
		tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tsuruCtx.Fs, filepath.Join(ConfigPath, "targets"), "default\thttp://xxx.xxx")

		got, err := GetTokenFromFs(tsuruCtx.Fs, "default")
		assert.NoError(t, err)
		assert.Equal(t, "", got)
	})
}

func TestSaveTokenToFs(t *testing.T) {
	t.Parallel()
	t.Run("created_token.d", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		SaveTokenToFs(tc.Fs, "", "")
		_, err := tc.Fs.Stat(filepath.Join(ConfigPath, "token.d"))
		assert.NoError(t, err)
	})

	t.Run("first_token", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		SaveTokenToFs(tc.Fs, "mytarget", "mytoken")

		got := readFile(t, tc.Fs, filepath.Join(ConfigPath, "token"))
		assert.Equal(t, "mytoken", got)
		got = readFile(t, tc.Fs, filepath.Join(ConfigPath, "token.d", "mytarget"))
		assert.Equal(t, "mytoken", got)
	})

	t.Run("token_for_current_alias", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "target"), "https://mytarget.xxx")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "targets"), "tar1 https://mytarget.xxx")
		SaveTokenToFs(tc.Fs, "tar1", "mytoken")

		got := readFile(t, tc.Fs, filepath.Join(ConfigPath, "token"))
		assert.Equal(t, "mytoken", got)
		got = readFile(t, tc.Fs, filepath.Join(ConfigPath, "token.d", "tar1"))
		assert.Equal(t, "mytoken", got)
	})

	t.Run("token_for_current_url", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "target"), "https://mytarget.xxx")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "targets"), "tar1 https://mytarget.xxx")
		SaveTokenToFs(tc.Fs, "https://mytarget.xxx", "mytoken")

		got := readFile(t, tc.Fs, filepath.Join(ConfigPath, "token"))
		assert.Equal(t, "mytoken", got)
		got = readFile(t, tc.Fs, filepath.Join(ConfigPath, "token.d", "tar1"))
		assert.Equal(t, "mytoken", got)
	})
}

func TestRemoveTokensFromFs(t *testing.T) {
	t.Parallel()
	t.Run("with_label", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "targets"), "tar1\thttps://mytarget1.xxx\ntar2\thttp://mytar2.yyy\n")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "token.d", "tar1"), "mytoken1")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "token.d", "tar2"), "mytoken2")

		RemoveTokensFromFs(tc.Fs, "tar1")

		_, err := tc.Fs.Open(filepath.Join(ConfigPath, "token.d", "tar1"))
		assert.True(t, os.IsNotExist(err), "file should not exist")
		_, err = tc.Fs.Open(filepath.Join(ConfigPath, "token.d", "tar2"))
		assert.NoError(t, err)
	})

	t.Run("with_url", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "targets"), "tar1\thttps://mytarget1.xxx\ntar2\thttp://mytar2.yyy\n")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "token.d", "tar1"), "mytoken1")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "token.d", "tar2"), "mytoken2")

		RemoveTokensFromFs(tc.Fs, "https://mytarget1.xxx")

		_, err := tc.Fs.Open(filepath.Join(ConfigPath, "token.d", "tar1"))
		assert.True(t, os.IsNotExist(err), "file should not exist")
		_, err = tc.Fs.Open(filepath.Join(ConfigPath, "token.d", "tar2"))
		assert.NoError(t, err)
	})

	t.Run("current", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "target"), "https://mytarget1.xxx")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "targets"), "tar1\thttps://mytarget1.xxx\ntar2\thttp://mytar2.yyy\n")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "token"), "mytoken1")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "token.d", "tar1"), "mytoken1")
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "token.d", "tar2"), "mytoken2")

		RemoveTokensFromFs(tc.Fs, "https://mytarget1.xxx")

		_, err := tc.Fs.Open(filepath.Join(ConfigPath, "token.d", "tar1"))
		assert.True(t, os.IsNotExist(err), "file tar1 should not exist")
		_, err = tc.Fs.Open(filepath.Join(ConfigPath, "token"))
		assert.True(t, os.IsNotExist(err), "file token should not exist")
		_, err = tc.Fs.Open(filepath.Join(ConfigPath, "token.d", "tar2"))
		assert.NoError(t, err)
	})
}

func TestHostFromURL(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		in       string
		expected string
	}{
		{"abc", "abc"},
		{"host.xxx", "host.xxx"},
		{"host.xxx.yyy", "host.xxx.yyy"},
		{"host:1234", "host"},
		{"http://host:1234/abc", "host"},
		{"http://host:1234:abc", "host"},
		{"https://host:1234", "host"},
	} {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.expected, hostFromURL(tt.in))

		})
	}
}
