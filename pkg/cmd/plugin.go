// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tsuru/tsuru-client/v2/internal/config"
	"github.com/tsuru/tsuru-client/v2/internal/exec"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

func runTsuruPlugin(tsuruCtx *tsuructx.TsuruContext, args []string) error {
	pluginName := args[0]
	if tsuruCtx.Viper.GetString("plugin-name") == pluginName {
		return fmt.Errorf("failing trying to run recursive plugin")
	}

	pluginPath := findExecutablePlugin(tsuruCtx, pluginName)
	if pluginPath == "" {
		return fmt.Errorf("unknown command %q", pluginName)
	}

	envs := os.Environ()
	tsuruEnvs := []string{
		"TSURU_TARGET=" + tsuruCtx.TargetURL(),
		"TSURU_TOKEN=" + tsuruCtx.Token(),
		"TSURU_VERBOSITY=" + fmt.Sprintf("%d", tsuruCtx.Verbosity()),
		"TSURU_PLUGIN_NAME=" + pluginName,
	}
	envs = append(envs, tsuruEnvs...)

	opts := exec.ExecuteOptions{
		Cmd:    pluginPath,
		Args:   args[1:],
		Stdout: tsuruCtx.Stdout,
		Stderr: tsuruCtx.Stderr,
		Stdin:  tsuruCtx.Stdin,
		Envs:   envs,
	}
	return tsuruCtx.Executor.Command(opts)
}

func findExecutablePlugin(tsuruCtx *tsuructx.TsuruContext, pluginName string) (execPath string) {
	basePath := filepath.Join(config.ConfigPath, "plugins")
	testPathGlobs := []string{
		filepath.Join(basePath, pluginName),
		filepath.Join(basePath, pluginName, pluginName),
		filepath.Join(basePath, pluginName, pluginName+".*"),
		filepath.Join(basePath, pluginName+".*"),
	}
	for _, pathGlob := range testPathGlobs {
		var fStat fs.FileInfo
		var err error
		execPath = pathGlob
		if fStat, err = tsuruCtx.Fs.Stat(pathGlob); err != nil {
			files, _ := filepath.Glob(pathGlob)
			if len(files) != 1 {
				continue
			}
			execPath = files[0]
			fStat, err = tsuruCtx.Fs.Stat(execPath)
		}
		if err != nil || fStat.IsDir() || !fStat.Mode().IsRegular() {
			continue
		}
		return execPath
	}
	return ""
}
