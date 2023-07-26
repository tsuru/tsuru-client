// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsuru/tsuru-client/v2/internal/config"
	"github.com/tsuru/tsuru-client/v2/internal/exec"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

func TestRunTsuruPlugin(t *testing.T) {
	t.Parallel()
	t.Run("simple_plugin", func(t *testing.T) {
		tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
		pluginPath := filepath.Join(config.ConfigPath, "plugins", "myplugin")
		tsuruCtx.Fs.Create(pluginPath)
		err := runTsuruPlugin(tsuruCtx, []string{"myplugin", "--arg1", "one", "--flag"})
		assert.NoError(t, err)
		assert.Equal(t, pluginPath, tsuruCtx.Executor.(*exec.FakeExec).CalledOpts.Cmd)
		assert.Equal(t, []string{"--arg1", "one", "--flag"}, tsuruCtx.Executor.(*exec.FakeExec).CalledOpts.Args)
	})
}
