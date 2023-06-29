// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows && !darwin
// +build !windows,!darwin

package exec

import (
	"fmt"
	"strings"

	"golang.org/x/sys/unix"
)

func isWSL() bool {
	var u unix.Utsname
	err := unix.Uname(&u)
	if err != nil {
		fmt.Println(err)
		return false
	}
	release := strings.ToLower(string(u.Release[:]))
	return strings.Contains(release, "microsoft")
}

func Open(ex Executor, url string) error {
	cmd := "xdg-open"
	args := []string{url}

	if isWSL() {
		cmd = "cmd"
		args = []string{"-c", "start", "'" + url + "'"}
	}

	opts := ExecuteOptions{
		Cmd:  cmd,
		Args: args,
	}
	return ex.Command(opts)
}
