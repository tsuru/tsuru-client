// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

func TestGetSavedTargets(t *testing.T) {
	t.Parallel()
	t.Run("regular", func(t *testing.T) {
		targetsContent := `tar1	http://target1.xxx
		tar2	https://target2.yyy:123
		target3	https://my.targ.x.99999
		`
		expected := map[string]string{
			"tar1":    "http://target1.xxx",
			"tar2":    "https://target2.yyy:123",
			"target3": "https://my.targ.x.99999",
		}
		tc := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "targets"), targetsContent)
		got, err := getSavedTargets(tc.Fs)
		assert.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("noFile_noError", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		got, err := getSavedTargets(tc.Fs)
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{}, got)
	})
}

func TestGetCurrentTargetFromFs(t *testing.T) {
	t.Parallel()
	t.Run("current", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "target"), "http://thistarget.xx")
		got, err := GetCurrentTargetFromFs(tc.Fs)
		assert.NoError(t, err)
		assert.Equal(t, "http://thistarget.xx", got)
	})

	t.Run("no_file", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		got, err := GetCurrentTargetFromFs(tc.Fs)
		assert.ErrorAs(t, err, &errUndefinedTarget)
		assert.Equal(t, "", got)
	})
}

func TestGetTargetURL(t *testing.T) {
	t.Parallel()
	t.Run("existing", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "targets"), "def1\thttp://mytarget.xxx")
		got, err := GetTargetURL(tc.Fs, "def1")
		assert.NoError(t, err)
		assert.Equal(t, "http://mytarget.xxx", got)
	})

	t.Run("non_existing", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		got, err := GetTargetURL(tc.Fs, "http://non-existing.xyz")
		assert.NoError(t, err)
		assert.Equal(t, "http://non-existing.xyz", got)
	})

	t.Run("non_existing_normalize", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		got, err := GetTargetURL(tc.Fs, "non-existing.xyz")
		assert.NoError(t, err)
		assert.Equal(t, "https://non-existing.xyz", got)
	})
}

func TestIsCurrentTarget(t *testing.T) {
	t.Parallel()
	t.Run("simple", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		assert.Equal(t, false, IsCurrentTarget(tc.Fs, ""))

		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "target"), "https://mytarget.xxx")
		assert.Equal(t, true, IsCurrentTarget(tc.Fs, "https://mytarget.xxx"))
		assert.Equal(t, true, IsCurrentTarget(tc.Fs, "mytarget.xxx"))
		assert.Equal(t, false, IsCurrentTarget(tc.Fs, "http://mytarget.xxx"))
		assert.Equal(t, false, IsCurrentTarget(tc.Fs, "test"))

		createFileWithContent(t, tc.Fs, filepath.Join(ConfigPath, "target"), "https://mytarget2.xxx:1234")
		assert.Equal(t, true, IsCurrentTarget(tc.Fs, "https://mytarget2.xxx:1234"))
		assert.Equal(t, true, IsCurrentTarget(tc.Fs, "mytarget2.xxx:1234"))
		assert.Equal(t, false, IsCurrentTarget(tc.Fs, "http://mytarget.xxx:1234"))
		assert.Equal(t, false, IsCurrentTarget(tc.Fs, "http://mytarget.xxx"))
		assert.Equal(t, false, IsCurrentTarget(tc.Fs, "https://mytarget.xxx"))
		assert.Equal(t, false, IsCurrentTarget(tc.Fs, "test"))
	})
}

func TestSaveTarget(t *testing.T) {
	t.Parallel()
	t.Run("simple", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)
		getTargetsContent := func() string {
			f, err := tc.Fs.Open(filepath.Join(ConfigPath, "targets"))
			if err != nil {
				return ""
			}
			b, _ := io.ReadAll(f)
			return string(b)
		}

		assert.Equal(t, "", getTargetsContent())

		SaveTarget(tc.Fs, "t1", "https://t1.xxx")
		assert.Equal(t, "t1\thttps://t1.xxx\n", getTargetsContent())

		SaveTarget(tc.Fs, "t2", "https://t2.xxx")
		assert.Equal(t, "t1\thttps://t1.xxx\nt2\thttps://t2.xxx\n", getTargetsContent())

		SaveTarget(tc.Fs, "t1", "https://target1.xxx")
		assert.Equal(t, "t1\thttps://target1.xxx\nt2\thttps://t2.xxx\n", getTargetsContent())

		SaveTarget(tc.Fs, "aaa", "https://aaaa.aaa")
		assert.Equal(t, "aaa\thttps://aaaa.aaa\nt1\thttps://target1.xxx\nt2\thttps://t2.xxx\n", getTargetsContent())

		SaveTarget(tc.Fs, "zzz", "https://zzz.zzz")
		assert.Equal(t, "aaa\thttps://aaaa.aaa\nt1\thttps://target1.xxx\nt2\thttps://t2.xxx\nzzz\thttps://zzz.zzz\n", getTargetsContent())

		SaveTarget(tc.Fs, "aaa", "without.proto")
		assert.Equal(t, "aaa\thttps://without.proto\nt1\thttps://target1.xxx\nt2\thttps://t2.xxx\nzzz\thttps://zzz.zzz\n", getTargetsContent())
	})
}

func TestSaveTargetAsCurrent(t *testing.T) {
	t.Parallel()
	t.Run("simple", func(t *testing.T) {
		tc := tsuructx.TsuruContextWithConfig(nil)

		targetPath := filepath.Join(ConfigPath, "target")
		_, err := tc.Fs.Stat(targetPath)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		SaveTargetAsCurrent(tc.Fs, "https://mytarget.xxx")
		assert.Equal(t, "https://mytarget.xxx\n", readFile(t, tc.Fs, targetPath))

		SaveTargetAsCurrent(tc.Fs, "https://mytarget2.xxx")
		assert.Equal(t, "https://mytarget2.xxx\n", readFile(t, tc.Fs, targetPath))

		SaveTargetAsCurrent(tc.Fs, "no.proto")
		assert.Equal(t, "https://no.proto\n", readFile(t, tc.Fs, targetPath))
	})
}
