// Copyright © 2026 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestUsageError(t *testing.T) {
	t.Run("error_returns_wrapped_error_message", func(t *testing.T) {
		wrappedErr := fmt.Errorf("invalid flag value")
		usageErr := &UsageError{Err: wrappedErr}

		assert.Equal(t, "invalid flag value", usageErr.Error())
	})

	t.Run("unwrap_returns_wrapped_error", func(t *testing.T) {
		wrappedErr := fmt.Errorf("original error")
		usageErr := &UsageError{Err: wrappedErr}

		assert.Equal(t, wrappedErr, usageErr.Unwrap())
	})

	t.Run("errors_is_works_with_wrapped_error", func(t *testing.T) {
		wrappedErr := fmt.Errorf("specific error")
		usageErr := &UsageError{Err: wrappedErr}

		assert.True(t, errors.Is(usageErr, wrappedErr))
	})
}

func TestIsUsageError(t *testing.T) {
	t.Run("returns_true_for_usage_error", func(t *testing.T) {
		err := &UsageError{Err: fmt.Errorf("test error")}
		assert.True(t, isUsageError(err))
	})

	t.Run("returns_false_for_regular_error", func(t *testing.T) {
		err := fmt.Errorf("regular error")
		assert.False(t, isUsageError(err))
	})

	t.Run("returns_true_for_wrapped_usage_error", func(t *testing.T) {
		usageErr := &UsageError{Err: fmt.Errorf("inner error")}
		wrappedErr := fmt.Errorf("outer: %w", usageErr)
		assert.True(t, isUsageError(wrappedErr))
	})

	t.Run("returns_false_for_nil_error", func(t *testing.T) {
		assert.False(t, isUsageError(nil))
	})
}

func TestCatchUsageError(t *testing.T) {
	t.Run("wraps_error_in_usage_error", func(t *testing.T) {
		originalArgs := func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not enough arguments")
		}

		wrapped := catchUsageError(originalArgs)
		err := wrapped(nil, []string{})

		assert.Error(t, err)
		assert.True(t, isUsageError(err))
	})

	t.Run("returns_nil_on_success", func(t *testing.T) {
		originalArgs := func(cmd *cobra.Command, args []string) error {
			return nil
		}

		wrapped := catchUsageError(originalArgs)
		err := wrapped(nil, []string{"arg1"})

		assert.NoError(t, err)
	})

	t.Run("preserves_original_error_message", func(t *testing.T) {
		originalArgs := func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("expected 2 args, got 1")
		}

		wrapped := catchUsageError(originalArgs)
		err := wrapped(nil, []string{"arg1"})

		assert.Equal(t, "expected 2 args, got 1", err.Error())
	})
}

func TestManagerV2_FlagErrorFunc(t *testing.T) {
	t.Run("flag_errors_are_wrapped_as_usage_error", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "test-cmd",
				Desc: "Test command",
			},
		}

		manager.Register(cmd)

		// Use an invalid flag to trigger a flag error
		manager.rootCmd.SetArgs([]string{"test-cmd", "--invalid-flag"})
		err := manager.rootCmd.Execute()

		assert.Error(t, err)
		assert.True(t, isUsageError(err))
	})
}

func TestManagerV2_Run_UsagePrintingBehavior(t *testing.T) {
	t.Run("prints_usage_on_usage_error", func(t *testing.T) {
		manager := NewManagerV2()
		var output strings.Builder

		cmd := &mockCommand{
			info: &Info{
				Name:    "test-cmd",
				Desc:    "Test command",
				MinArgs: 2, // Requires 2 arguments
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetOut(&output)
		manager.rootCmd.SetArgs([]string{"test-cmd", "only-one-arg"}) // Only 1 arg provided

		err := manager.Run()

		assert.Error(t, err)
		// Verify usage was printed (output should contain usage info)
		assert.Contains(t, output.String(), "Usage:")
	})

	t.Run("does_not_print_usage_on_regular_error", func(t *testing.T) {
		manager := NewManagerV2()
		var output strings.Builder

		cmd := &mockCommand{
			info: &Info{
				Name: "test-cmd",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				return fmt.Errorf("internal error")
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetOut(&output)
		manager.rootCmd.SetArgs([]string{"test-cmd"})

		err := manager.Run()

		assert.Error(t, err)
		// Verify usage was NOT printed
		assert.NotContains(t, output.String(), "Usage:")
	})

	t.Run("prints_usage_on_flag_error", func(t *testing.T) {
		manager := NewManagerV2()
		var output strings.Builder

		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String("required-flag", "", "A required flag")

		cmd := &mockFlaggedCommand{
			info: &Info{
				Name: "flagged-cmd",
				Desc: "A command with flags",
			},
			flags: flags,
		}

		manager.Register(cmd)
		manager.rootCmd.SetOut(&output)
		manager.rootCmd.SetArgs([]string{"flagged-cmd", "--unknown-flag"})

		err := manager.Run()

		assert.Error(t, err)
		assert.True(t, isUsageError(err))
		assert.Contains(t, output.String(), "Usage:")
	})
}

func TestManagerV2_ArgsValidation_WrapsErrorInUsageError(t *testing.T) {
	t.Run("min_args_violation_is_usage_error", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:    "app-deploy",
				Desc:    "Deploy an app",
				MinArgs: 1,
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetArgs([]string{"app-deploy"}) // No args provided

		err := manager.rootCmd.Execute()

		assert.Error(t, err)
		assert.True(t, isUsageError(err))
	})

	t.Run("range_args_violation_is_usage_error", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:    "app-create",
				Desc:    "Create an app",
				MinArgs: 1,
				MaxArgs: 2,
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetArgs([]string{"app-create", "arg1", "arg2", "arg3"}) // Too many args

		err := manager.rootCmd.Execute()

		assert.Error(t, err)
		assert.True(t, isUsageError(err))
	})

	t.Run("valid_args_do_not_produce_error", func(t *testing.T) {
		manager := NewManagerV2()
		executed := false

		cmd := &mockCommand{
			info: &Info{
				Name:    "app-info",
				Desc:    "Show app info",
				MinArgs: 1,
				MaxArgs: 1,
			},
			runFn: func(ctx *Context) error {
				executed = true
				return nil
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetArgs([]string{"app-info", "myapp"})

		err := manager.rootCmd.Execute()

		assert.NoError(t, err)
		assert.True(t, executed)
	})
}
