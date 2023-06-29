// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "github.com/tsuru/tsuru-client/v2/pkg/cmd"

var (
	version = "dev"
	commit  = ""
	dateStr = ""
)

func main() {
	cmd.Execute(version, commit, dateStr)
}
