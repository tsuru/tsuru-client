// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/tsuru/tsuru/fs"
)

var (
	fsystem atomic.Pointer[fs.Fs]
)

func Filesystem() fs.Fs {
	f := fsystem.Load()
	if f == nil {
		return &fs.OsFs{}
	}
	return *f
}

func SetFileSystem(f fs.Fs) {
	fsystem.Store(&f)
}

func ResetFileSystem() {
	fsystem.Store(nil)
}

func getHome() string {
	envs := []string{"HOME", "HOMEPATH"}
	var home string
	for i := 0; i < len(envs) && home == ""; i++ {
		home = os.Getenv(envs[i])
	}
	return home
}

func JoinWithUserDir(p ...string) string {
	paths := []string{getHome()}
	paths = append(paths, p...)
	return filepath.Join(paths...)
}
