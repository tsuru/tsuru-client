// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/tsuru/tsuru-client/tsuru/config"
	"github.com/tsuru/tsuru-client/tsuru/config/selfupdater"
	"github.com/tsuru/tsuru/cmd"
)

var (
	version = "dev" // overridden at build time
)

func recoverCmdPanicExitError() {
	if r := recover(); r != nil {
		if e, ok := r.(*cmd.PanicExitError); ok {
			os.Exit(e.Code)
		}
		panic(r)
	}
}

func main() {
	defer recoverCmdPanicExitError()
	defer config.SaveChangesWithTimeout()

	checkVerResult := selfupdater.CheckLatestVersionBackground(version)
	defer selfupdater.VerifyLatestVersion(checkVerResult)

	name := cmd.ExtractProgramName(os.Args[0])
	m := config.BuildManager(name, version)
	m.Run(os.Args[1:])
}
