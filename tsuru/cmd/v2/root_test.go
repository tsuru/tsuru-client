// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v2

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/tsuru/go-tsuruclient/pkg/config"
)

func iterateCmdTreeAndRemoveRun(t *testing.T, cmd *cobra.Command, cmdPath []string, cmdPathChan chan []string) {
	if len(cmd.Commands()) == 0 {
		cmdPathChan <- cmdPath
	}
	for _, c := range cmd.Commands() {
		c.RunE = nil
		c.Run = nil
		newCmdPath := make([]string, len(cmdPath))
		copy(newCmdPath, cmdPath)
		newCmdPath = append(newCmdPath, c.Name())
		iterateCmdTreeAndRemoveRun(t, c, newCmdPath, cmdPathChan)
	}
}

func TestNoFlagRedeclarationOnSubCommands(t *testing.T) {
	rootCmd := NewRootCmd()

	cmdPathChan := make(chan []string)
	go func() {
		iterateCmdTreeAndRemoveRun(t, rootCmd, []string{}, cmdPathChan)
		close(cmdPathChan)
	}()

	for cmdPath := range cmdPathChan {
		t.Run(strings.Join(cmdPath, "_"), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic: %v", r)
				}
			}()

			innerRootCmd := NewRootCmd()
			innerRootCmd.SetArgs(cmdPath)
			innerRootCmd.Execute()
		})
	}
}

func TestPersistentFlagsGetPassedToSubCommand(t *testing.T) {
	rootCmd := NewRootCmd()

	called := false
	newCmd := &cobra.Command{
		Use: "newtestcommand",
		Run: func(cmd *cobra.Command, args []string) {
			called = true
			if cmd.Flags().Lookup("target") == nil {
				assert.FailNow(t, "flag target not found from subcommand")
			}
			assert.Equal(t, "myNewTarget", cmd.Flag("target").Value.String(), "target from cmd.Flag")
			assert.Equal(t, "myNewTarget", defaultViper.GetString("target"), "target from tsuruCtx.Viper")
			target, err := config.GetTarget()
			assert.NoError(t, err)
			assert.Equal(t, "http://myNewTarget", target, "target from tsuruCtx.TargetURL()")
		},
	}
	rootCmd.AddCommand(newCmd)

	rootCmd.SetArgs([]string{"--target", "myNewTarget", "newtestcommand"})
	rootCmd.ParseFlags([]string{})
	rootCmd.Execute()
	assert.True(t, called)
}

func TestParseEnvVariables(t *testing.T) {
	vip := preSetupViper(viper.GetViper()) // use global viper here

	t.Run("string_envs", func(t *testing.T) {
		for _, test := range []struct {
			viperEnvName string
			envName      string
		}{
			{"token", "TSURU_TOKEN"},
			{"target", "TSURU_TARGET"},
			{"auth-schema", "TSURU_AUTH_SCHEMA"},
		} {
			func() {
				if oldEnv, ok := os.LookupEnv(test.envName); ok {
					defer os.Setenv(test.envName, oldEnv)
				}
				os.Setenv(test.envName, "ABCDEFGH")
				assert.Equal(t, "ABCDEFGH", vip.GetString(test.viperEnvName))
				os.Unsetenv(test.envName)
			}()
		}
	})

	t.Run("Int_envs", func(t *testing.T) {
		for _, test := range []struct {
			viperEnvName string
			envName      string
		}{
			{"verbosity", "TSURU_VERBOSITY"},
		} {
			func() {
				if oldEnv, ok := os.LookupEnv(test.envName); ok {
					defer os.Setenv(test.envName, oldEnv)
				}
				os.Setenv(test.envName, "123")
				assert.Equal(t, 123, vip.GetInt(test.viperEnvName))
				os.Unsetenv(test.envName)
			}()
		}
	})

	t.Run("Bool_envs", func(t *testing.T) {
		for _, test := range []struct {
			viperEnvName string
			envName      string
		}{
			{"insecure-skip-verify", "TSURU_INSECURE_SKIP_VERIFY"},
		} {
			func() {
				if oldEnv, ok := os.LookupEnv(test.envName); ok {
					defer os.Setenv(test.envName, oldEnv)
				}
				os.Setenv(test.envName, "t")
				assert.Equal(t, true, vip.GetBool(test.viperEnvName))
				os.Unsetenv(test.envName)
			}()
		}
	})
}

func TestParseFirstFlagsOnly(t *testing.T) {
	t.Run("nil_command", func(t *testing.T) {
		args := []string{"--flag", "value"}
		result := ParseFirstFlagsOnly(nil, args)
		assert.Equal(t, args, result)
	})

	t.Run("empty_args", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		result := ParseFirstFlagsOnly(cmd, []string{})
		assert.Equal(t, []string{}, result)
	})

	t.Run("no_flags_in_args", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		args := []string{"arg1", "arg2"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{"arg1", "arg2"}, result)
	})

	t.Run("boolean_flag", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().Bool("verbose", false, "verbose output")
		args := []string{"--verbose", "arg1", "arg2"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{"arg1", "arg2"}, result)
		verbose, _ := cmd.Flags().GetBool("verbose")
		assert.True(t, verbose)
	})

	t.Run("flag_with_value", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("target", "", "target server")
		args := []string{"--target", "myserver", "arg1"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{"arg1"}, result)
		target, _ := cmd.Flags().GetString("target")
		assert.Equal(t, "myserver", target)
	})

	t.Run("flag_with_equals_syntax", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("target", "", "target server")
		args := []string{"--target=myserver", "arg1"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{"arg1"}, result)
		target, _ := cmd.Flags().GetString("target")
		assert.Equal(t, "myserver", target)
	})

	t.Run("shorthand_flag", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().StringP("target", "t", "", "target server")
		args := []string{"-t", "myserver", "arg1"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{"arg1"}, result)
		target, _ := cmd.Flags().GetString("target")
		assert.Equal(t, "myserver", target)
	})

	t.Run("double_dash_terminator", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("target", "", "target server")
		args := []string{"--", "--target", "myserver"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{"--target", "myserver"}, result)
	})

	t.Run("mixed_flags_and_args", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("target", "", "target server")
		cmd.Flags().Bool("verbose", false, "verbose output")
		args := []string{"--target", "myserver", "--verbose", "arg1", "--other"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{"arg1", "--other"}, result)
		target, _ := cmd.Flags().GetString("target")
		assert.Equal(t, "myserver", target)
		verbose, _ := cmd.Flags().GetBool("verbose")
		assert.True(t, verbose)
	})

	t.Run("single_dash_arg", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		args := []string{"-", "arg1"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{"-", "arg1"}, result)
	})

	t.Run("unknown_flag_with_value", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		args := []string{"--unknown", "value", "arg1"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{"arg1"}, result)
	})

	t.Run("flag_at_end_without_value", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("target", "", "target server")
		args := []string{"--target"}
		result := ParseFirstFlagsOnly(cmd, args)
		assert.Equal(t, []string{}, result)
	})
}

func TestRunRootCmd(t *testing.T) {
	//cobra stdout/stderr is inconsistent. SetOut()/SetErr() don't work as expected: https://github.com/spf13/cobra/issues/1708

	t.Run("with_no_args", func(t *testing.T) {
		cmd := NewRootCmd()
		out := strings.Builder{}
		cmd.SetOut(&out)
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Contains(t, out.String(), "A command-line interface for interacting with tsuru")
	})

	t.Run("not_found_command", func(t *testing.T) {
		t.Skip("Disabled for while")

		cmd := NewRootCmd()
		stderr := strings.Builder{}
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"myplugin", "arg2"})
		err := cmd.Execute()
		assert.ErrorContains(t, err, "unknown command")
		assert.Contains(t, stderr.String(), `unknown command "myplugin"`)
	})

	t.Run("help_flag", func(t *testing.T) {
		cmd := NewRootCmd()
		out := strings.Builder{}
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--help", "arg2"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Contains(t, out.String(), "A command-line interface for interacting with tsuru")
	})

	t.Run("help_deprecated_flag", func(t *testing.T) {
		cmd := NewRootCmd()
		stderr := strings.Builder{}
		cmd.SetOut(&stderr) // inconsistent cobra stdout/stderr (see above)

		newCmd := &cobra.Command{
			Use: "newtestcommand",
		}
		newCmd.Flags().Bool("deprecatedflag", false, "deprecated flag")
		newCmd.Flags().MarkDeprecated("deprecatedflag", "use --superflag")
		cmd.AddCommand(newCmd)

		cmd.SetArgs([]string{"newtestcommand", "--deprecatedflag"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, "Flag --deprecatedflag has been deprecated, use --superflag\n", stderr.String())
	})

	t.Run("version_flag", func(t *testing.T) {
		t.Skip("Disabled for while")

		cmd := NewRootCmd()
		out := strings.Builder{}
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--version", "arg2"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, "tsuru-client version: dev\n", out.String())
	})

	t.Run("completion_output", func(t *testing.T) {
		cmd := NewRootCmd()
		out := strings.Builder{}
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"completion", "bash"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Contains(t, out.String(), "bash completion")
	})
}
