package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestAddMissingV1LegacyCommands(t *testing.T) {
	rootCmd := &cobra.Command{}
	v1CmdManager := newV1LegacyCmdManager()

	assert.Equal(t, 0, len(rootCmd.Commands()))
	addMissingV1LegacyCommands(rootCmd, v1CmdManager)
	assert.GreaterOrEqual(t, len(rootCmd.Commands()), 1)
}
