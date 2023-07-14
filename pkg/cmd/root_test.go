// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
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
	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	rootCmd := NewRootCmd(viper.New(), tsuruCtx)

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

			rootCmd.SetArgs(cmdPath)
			rootCmd.Execute()
		})
	}
}

func TestPersistentFlagsGetPassedToSubCommand(t *testing.T) {
	tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
	rootCmd := NewRootCmd(tsuruCtx.Viper, tsuruCtx)

	called := false
	newCmd := &cobra.Command{
		Use: "newtestcommand",
		Run: func(cmd *cobra.Command, args []string) {
			called = true
			if cmd.Flags().Lookup("target") == nil {
				assert.FailNow(t, "flag target not found from subcommand")
			}
			assert.Equal(t, "myNewTarget", cmd.Flag("target").Value.String(), "target from cmd.Flag")
			assert.Equal(t, "http://myNewTarget", tsuruCtx.Viper.GetString("target"), "target from tsuruCtx.Viper")
			assert.Equal(t, "http://myNewTarget", tsuruCtx.TargetURL(), "target from tsuruCtx.TargetURL()")
		},
	}
	rootCmd.AddCommand(newCmd)

	rootCmd.SetArgs([]string{"--target", "myNewTarget", "newtestcommand"})
	rootCmd.ParseFlags([]string{})
	rootCmd.Execute()
	assert.True(t, called)
}

func TestProductionOptsNonZeroValues(t *testing.T) {
	vip := viper.New()
	vip.Set("insecure-skip-verify", true)
	vip.Set("auth-scheme", true)

	opts := productionOpts(afero.NewMemMapFs(), vip)
	value := reflect.ValueOf(opts).Elem()
	errCount := 0
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.IsZero() {
			errCount++
			t.Errorf("field %s has %q", value.Type().Field(i).Name, field.Interface())
		}
	}
	if errCount > 0 {
		t.Log("productionOpts() must declare ALL fields of TsuruContextOpts")
	}
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

func TestRunRootCmd(t *testing.T) {
	t.Run("with_no_args", func(t *testing.T) {
		tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
		cmd := NewRootCmd(viper.New(), tsuruCtx)
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Contains(t, tsuruCtx.Stderr.(*strings.Builder).String(), "A command-line interface for interacting with tsuru")
	})

	t.Run("not_found_command", func(t *testing.T) {
		tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
		cmd := NewRootCmd(viper.New(), tsuruCtx)
		cmd.SetArgs([]string{"myplugin", "arg2"})
		err := cmd.Execute()
		assert.ErrorContains(t, err, "unknown command")
		assert.Equal(t, "", tsuruCtx.Stdout.(*strings.Builder).String())
	})

	t.Run("help_flag", func(t *testing.T) {
		tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
		cmd := NewRootCmd(viper.New(), tsuruCtx)
		cmd.SetArgs([]string{"--help", "arg2"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Contains(t, tsuruCtx.Stderr.(*strings.Builder).String(), "A command-line interface for interacting with tsuru")
	})

	t.Run("version_flag", func(t *testing.T) {
		tsuruCtx := tsuructx.TsuruContextWithConfig(nil)
		cmd := NewRootCmd(viper.New(), tsuruCtx)
		cmd.SetArgs([]string{"--version", "arg2"})
		err := cmd.Execute()
		assert.NoError(t, err)
		assert.Equal(t, tsuruCtx.Stderr.(*strings.Builder).String(), "tsuru-client version: dev\n")
	})
}
