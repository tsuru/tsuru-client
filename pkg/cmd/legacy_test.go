// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestAddMissingV1LegacyCommands(t *testing.T) {
	t.Parallel()
	rootCmd := &cobra.Command{}
	v1CmdManager := newV1LegacyCmdManager()

	assert.Equal(t, 0, len(rootCmd.Commands()))
	addMissingV1LegacyCommands(rootCmd, v1CmdManager)
	assert.GreaterOrEqual(t, len(rootCmd.Commands()), 1)
}
