// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var ConfigPath string

func init() {
	// Find home directory.
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	ConfigPath = filepath.Join(home, ".tsuru")
}
