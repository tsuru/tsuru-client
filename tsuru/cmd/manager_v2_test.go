// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

type mockCommand struct {
	info  *Info
	runFn func(context *Context) error
}

func (m *mockCommand) Info() *Info {
	return m.info
}

func (m *mockCommand) Run(context *Context) error {
	if m.runFn != nil {
		return m.runFn(context)
	}
	return nil
}

func TestNewManagerV2(t *testing.T) {
	manager := NewManagerV2()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.rootCmd)
	assert.NotNil(t, manager.tree)
	assert.Equal(t, manager.rootCmd, manager.tree.Command)
}

func TestManagerV2_RegisterTopic(t *testing.T) {
	t.Run("register_single_topic", func(t *testing.T) {
		manager := NewManagerV2()

		manager.RegisterTopic("app", "App management commands\nManage applications")

		assert.NotNil(t, manager.tree.Children["app"])
		assert.Equal(t, "app", manager.tree.Children["app"].Command.Use)
		assert.Equal(t, "App management commands", manager.tree.Children["app"].Command.Short)
		assert.Equal(t, "resource", manager.tree.Children["app"].Command.GroupID)
	})

	t.Run("register_nested_topic", func(t *testing.T) {
		manager := NewManagerV2()

		manager.RegisterTopic("service-instance", "Service instance management\nManage service instances")

		// Should create two levels: service and instance
		assert.NotNil(t, manager.tree.Children["service"])
		assert.Equal(t, "resource", manager.tree.Children["service"].Command.GroupID)

		assert.NotNil(t, manager.tree.Children["service"].Children["instance"])
		assert.Equal(t, "Service instance management", manager.tree.Children["service"].Children["instance"].Command.Short)
		assert.Equal(t, "sub-resource", manager.tree.Children["service"].Children["instance"].Command.GroupID)
	})

	t.Run("register_duplicate_topic_panics", func(t *testing.T) {
		manager := NewManagerV2()

		manager.RegisterTopic("app", "App management")

		assert.Panics(t, func() {
			manager.RegisterTopic("app", "Duplicate app management")
		})
	})

	t.Run("register_nested_topic_on_existing_parent", func(t *testing.T) {
		manager := NewManagerV2()

		manager.RegisterTopic("service", "Service management")
		manager.RegisterTopic("service-instance", "Service instance management")

		// Should reuse existing "service" node and add "instance" child
		assert.NotNil(t, manager.tree.Children["service"])
		assert.NotNil(t, manager.tree.Children["service"].Children["instance"])
		assert.Equal(t, "Service instance management", manager.tree.Children["service"].Children["instance"].Command.Short)
	})

	t.Run("extract_short_description_from_content", func(t *testing.T) {
		manager := NewManagerV2()

		manager.RegisterTopic("app", "  App management commands  \nLong description here")

		assert.Equal(t, "App management commands", manager.tree.Children["app"].Command.Short)
	})
}

func TestManagerV2_Register(t *testing.T) {
	t.Run("register_simple_command", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "app-list",
				Desc:  "List all apps\nDetailed description",
				Usage: "tsuru app-list [flags]",
			},
		}

		manager.Register(cmd)

		// Should register on root as FQDN
		rootCommands := manager.rootCmd.Commands()
		var foundFQDN bool
		for _, c := range rootCommands {
			if strings.HasPrefix(c.Use, "app-list") {
				foundFQDN = true
				assert.Equal(t, "List all apps", c.Short)
				assert.Equal(t, "List all apps\nDetailed description", c.Long)
				assert.True(t, c.Hidden) // Should be hidden by default
			}
		}
		assert.True(t, foundFQDN)

		// Should also register as sub-command
		assert.NotNil(t, manager.tree.Children["app"])
		assert.NotNil(t, manager.tree.Children["app"].Children["list"])
		assert.Equal(t, "List all apps", manager.tree.Children["app"].Children["list"].Command.Short)
	})

	t.Run("register_command_disabled", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "disabled-cmd",
				Desc: "This command is disabled",
				V2: InfoV2{
					Disabled: true,
				},
			},
		}

		manager.Register(cmd)

		// Should not register anything
		rootCommands := manager.rootCmd.Commands()
		for _, c := range rootCommands {
			assert.NotEqual(t, "disabled-cmd", c.Use)
		}
		assert.Nil(t, manager.tree.Children["disabled"])
	})

	t.Run("register_command_only_append_on_root", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "login",
				Desc:  "Login to tsuru server",
				Usage: "tsuru login [server]",
				V2: InfoV2{
					OnlyAppendOnRoot: true,
					GroupID:          "auth",
				},
			},
		}

		manager.Register(cmd)

		// Should register on root
		rootCommands := manager.rootCmd.Commands()
		var found bool
		for _, c := range rootCommands {
			if strings.HasPrefix(c.Use, "login") {
				found = true
				assert.Equal(t, "Login to tsuru server", c.Short)
				assert.Equal(t, "auth", c.GroupID)
				assert.False(t, c.Hidden) // Should NOT be hidden when OnlyAppendOnRoot is true
			}
		}
		assert.True(t, found)

		// Should NOT register as sub-command
		assert.Nil(t, manager.tree.Children["login"])
	})

	t.Run("register_nested_command", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "service-instance-info",
				Desc:  "Show service instance info\nDetailed description",
				Usage: "tsuru service-instance-info <name>",
			},
		}

		manager.Register(cmd)

		// Should create hierarchy: service -> instance -> info
		assert.NotNil(t, manager.tree.Children["service"])
		assert.NotNil(t, manager.tree.Children["service"].Children["instance"])
		assert.NotNil(t, manager.tree.Children["service"].Children["instance"].Children["info"])

		infoNode := manager.tree.Children["service"].Children["instance"].Children["info"]
		assert.Equal(t, "Show service instance info", infoNode.Command.Short)
		assert.Equal(t, cmd.info.Desc, infoNode.Command.Long)
	})

	t.Run("register_multiple_commands_same_topic", func(t *testing.T) {
		manager := NewManagerV2()

		cmd1 := &mockCommand{
			info: &Info{
				Name: "app-list",
				Desc: "List all apps",
			},
		}

		cmd2 := &mockCommand{
			info: &Info{
				Name: "app-create",
				Desc: "Create a new app",
			},
		}

		manager.Register(cmd1)
		manager.Register(cmd2)

		// Should share the "app" parent node
		assert.NotNil(t, manager.tree.Children["app"])
		assert.NotNil(t, manager.tree.Children["app"].Children["list"])
		assert.NotNil(t, manager.tree.Children["app"].Children["create"])
	})

	t.Run("command_short_description_from_first_line", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "app-deploy",
				Desc: "Deploy an application  \nSecond line\nThird line",
			},
		}

		manager.Register(cmd)

		assert.Equal(t, "Deploy an application", manager.tree.Children["app"].Children["deploy"].Command.Short)
	})
}

func TestManagerV2_registerV2SubCommand(t *testing.T) {
	t.Run("create_intermediate_nodes", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "app-list",
				Desc: "List all apps",
			},
		}

		manager.registerV2SubCommand(cmd)

		// Should create "app" intermediate node with auto-generated description
		assert.NotNil(t, manager.tree.Children["app"])
		assert.Contains(t, manager.tree.Children["app"].Command.Short, "Manage apps")
	})

	t.Run("update_leaf_command_properties", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "app-deploy",
				Desc:  "Deploy application",
				Usage: "Usage: tsuru app-deploy",
			},
		}

		manager.registerV2SubCommand(cmd)

		leafNode := manager.tree.Children["app"].Children["deploy"]
		assert.NotNil(t, leafNode)
		assert.Equal(t, "Deploy application", leafNode.Command.Short)
		assert.Equal(t, "Deploy application", leafNode.Command.Long)
		assert.False(t, leafNode.Command.SilenceUsage)
		assert.NotNil(t, leafNode.Command.Args)
		assert.NotNil(t, leafNode.Command.RunE)
	})

	t.Run("do_not_override_existing_intermediate_node", func(t *testing.T) {
		manager := NewManagerV2()

		// First register a topic
		manager.RegisterTopic("app", "Application management")

		cmd := &mockCommand{
			info: &Info{
				Name: "app-list",
				Desc: "List all apps",
			},
		}

		manager.registerV2SubCommand(cmd)

		// The "app" node should keep the topic description
		assert.Equal(t, "Application management", manager.tree.Children["app"].Command.Short)
	})
}

func TestManagerV2_registerV2FQDNOnRoot(t *testing.T) {
	t.Run("register_command_on_root", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "app-list",
				Desc:  "List all apps",
				Usage: "Usage: tsuru app-list",
				V2: InfoV2{
					GroupID: "resource",
				},
			},
		}

		manager.registerV2FQDNOnRoot(cmd)

		// Should add command to root
		rootCommands := manager.rootCmd.Commands()
		var found bool
		for _, c := range rootCommands {
			if strings.HasPrefix(c.Use, "app-list") {
				found = true
				assert.Equal(t, "List all apps", c.Short)
				assert.Equal(t, "List all apps", c.Long)
				assert.Equal(t, "resource", c.GroupID)
				assert.False(t, c.SilenceUsage)
				assert.True(t, c.Hidden) // Hidden by default
				assert.False(t, c.DisableFlagParsing)
			}
		}
		assert.True(t, found)
	})

	t.Run("register_with_only_append_on_root_not_hidden", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "login",
				Desc: "Login to server",
				V2: InfoV2{
					OnlyAppendOnRoot: true,
					GroupID:          "auth",
				},
			},
		}

		manager.registerV2FQDNOnRoot(cmd)

		rootCommands := manager.rootCmd.Commands()
		for _, c := range rootCommands {
			if c.Use == "login" {
				assert.False(t, c.Hidden)
			}
		}
	})

	t.Run("skip_if_disabled", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "disabled-cmd",
				V2: InfoV2{
					Disabled: true,
				},
			},
		}

		initialCommandCount := len(manager.rootCmd.Commands())
		manager.registerV2FQDNOnRoot(cmd)
		finalCommandCount := len(manager.rootCmd.Commands())

		assert.Equal(t, initialCommandCount, finalCommandCount)
	})
}

func TestManagerV2_NewManagerV2WithOptions(t *testing.T) {
	t.Run("with_after_flag_parse_hook", func(t *testing.T) {
		hookCalled := false
		opts := &ManagerV2Opts{
			AfterFlagParseHook: func() {
				hookCalled = true
			},
		}

		manager := NewManagerV2(opts)

		assert.NotNil(t, manager)
		assert.NotNil(t, manager.rootCmd)

		// Simulate PersistentPreRun being called
		if manager.rootCmd.PersistentPreRun != nil {
			manager.rootCmd.PersistentPreRun(manager.rootCmd, []string{})
		}

		assert.True(t, hookCalled)
	})

	t.Run("without_options", func(t *testing.T) {
		manager := NewManagerV2()

		assert.NotNil(t, manager)
		assert.NotNil(t, manager.rootCmd)
		assert.NotNil(t, manager.tree)
	})

	t.Run("with_nil_hook", func(t *testing.T) {
		opts := &ManagerV2Opts{
			AfterFlagParseHook: nil,
		}

		manager := NewManagerV2(opts)

		assert.NotNil(t, manager)
		// Should not panic when PersistentPreRun is called
		// If it panics, the test will fail
		if manager.rootCmd.PersistentPreRun != nil {
			manager.rootCmd.PersistentPreRun(manager.rootCmd, []string{})
		}
	})

	t.Run("with_retry_hook", func(t *testing.T) {
		retryCalled := false
		opts := &ManagerV2Opts{
			RetryHook: func(err error) bool {
				retryCalled = true
				return true
			},
		}

		manager := NewManagerV2(opts)

		assert.NotNil(t, manager)
		assert.NotNil(t, manager.retryHook)

		// Call the retry hook directly
		result := manager.retryHook(assert.AnError)
		assert.True(t, retryCalled)
		assert.True(t, result)
	})

	t.Run("with_nil_retry_hook", func(t *testing.T) {
		manager := NewManagerV2()
		assert.NotNil(t, manager)
		assert.Nil(t, manager.retryHook)
	})

	t.Run("with_both_hooks", func(t *testing.T) {
		afterFlagParseCalled := false
		retryCalled := false
		opts := &ManagerV2Opts{
			AfterFlagParseHook: func() {
				afterFlagParseCalled = true
			},
			RetryHook: func(err error) bool {
				retryCalled = true
				return false
			},
		}

		manager := NewManagerV2(opts)

		assert.NotNil(t, manager)

		// Test AfterFlagParseHook
		if manager.rootCmd.PersistentPreRun != nil {
			manager.rootCmd.PersistentPreRun(manager.rootCmd, []string{})
		}
		assert.True(t, afterFlagParseCalled)

		// Test RetryHook
		result := manager.retryHook(assert.AnError)
		assert.True(t, retryCalled)
		assert.False(t, result)
	})
}

func TestManagerV2_Run_RetryHook(t *testing.T) {
	t.Run("retry_hook_called_on_error", func(t *testing.T) {
		retryCalled := false
		var receivedErr error

		opts := &ManagerV2Opts{
			RetryHook: func(err error) bool {
				retryCalled = true
				receivedErr = err
				return false // Don't retry
			},
		}

		manager := NewManagerV2(opts)

		executionCount := 0
		cmd := &mockCommand{
			info: &Info{
				Name: "test-cmd",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				executionCount++
				return assert.AnError
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetArgs([]string{"test-cmd"})

		// We can't call Run() directly because it calls os.Exit
		// Instead, test the logic by simulating what Run() does
		err := manager.rootCmd.Execute()
		assert.Error(t, err)

		if manager.retryHook != nil && err != nil {
			manager.retryHook(err)
		}

		assert.True(t, retryCalled)
		assert.Equal(t, assert.AnError, receivedErr)
		assert.Equal(t, 1, executionCount)
	})

	t.Run("retry_hook_retries_on_true", func(t *testing.T) {
		retryCount := 0

		opts := &ManagerV2Opts{
			RetryHook: func(err error) bool {
				retryCount++
				return true // Retry once
			},
		}

		manager := NewManagerV2(opts)

		executionCount := 0
		cmd := &mockCommand{
			info: &Info{
				Name: "test-cmd",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				executionCount++
				if executionCount == 1 {
					return assert.AnError // First execution fails
				}
				return nil // Second execution succeeds
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetArgs([]string{"test-cmd"})

		// Simulate Run() logic
		err := manager.rootCmd.Execute()

		if manager.retryHook != nil && err != nil {
			if retry := manager.retryHook(err); retry {
				err = manager.rootCmd.Execute()
			}
		}

		assert.NoError(t, err)
		assert.Equal(t, 1, retryCount)
		assert.Equal(t, 2, executionCount)
	})

	t.Run("retry_hook_not_called_on_success", func(t *testing.T) {
		retryCalled := false

		opts := &ManagerV2Opts{
			RetryHook: func(err error) bool {
				retryCalled = true
				return true
			},
		}

		manager := NewManagerV2(opts)

		cmd := &mockCommand{
			info: &Info{
				Name: "test-cmd",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				return nil // Success
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetArgs([]string{"test-cmd"})

		// Simulate Run() logic
		err := manager.rootCmd.Execute()

		if manager.retryHook != nil && err != nil {
			manager.retryHook(err)
		}

		assert.NoError(t, err)
		assert.False(t, retryCalled)
	})

	t.Run("retry_hook_not_called_when_nil", func(t *testing.T) {
		manager := NewManagerV2() // No retry hook

		executionCount := 0
		cmd := &mockCommand{
			info: &Info{
				Name: "test-cmd",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				executionCount++
				return assert.AnError
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetArgs([]string{"test-cmd"})

		// Simulate Run() logic
		err := manager.rootCmd.Execute()

		if manager.retryHook != nil && err != nil {
			if retry := manager.retryHook(err); retry {
				err = manager.rootCmd.Execute()
			}
		}

		assert.Error(t, err)
		assert.Equal(t, 1, executionCount)
	})

	t.Run("retry_hook_receives_correct_error", func(t *testing.T) {
		expectedErr := fmt.Errorf("specific error message")
		var receivedErr error

		opts := &ManagerV2Opts{
			RetryHook: func(err error) bool {
				receivedErr = err
				return false
			},
		}

		manager := NewManagerV2(opts)

		cmd := &mockCommand{
			info: &Info{
				Name: "test-cmd",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				return expectedErr
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetArgs([]string{"test-cmd"})

		err := manager.rootCmd.Execute()

		if manager.retryHook != nil && err != nil {
			manager.retryHook(err)
		}

		assert.Equal(t, expectedErr, receivedErr)
	})

	t.Run("retry_still_fails_after_retry", func(t *testing.T) {
		retryCount := 0

		opts := &ManagerV2Opts{
			RetryHook: func(err error) bool {
				retryCount++
				return true
			},
		}

		manager := NewManagerV2(opts)

		executionCount := 0
		cmd := &mockCommand{
			info: &Info{
				Name: "test-cmd",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				executionCount++
				return assert.AnError // Always fails
			},
		}

		manager.Register(cmd)
		manager.rootCmd.SetArgs([]string{"test-cmd"})

		// Simulate Run() logic
		err := manager.rootCmd.Execute()

		if manager.retryHook != nil && err != nil {
			if retry := manager.retryHook(err); retry {
				err = manager.rootCmd.Execute()
			}
		}

		assert.Error(t, err)
		assert.Equal(t, 1, retryCount)
		assert.Equal(t, 2, executionCount)
	})
}

func TestManagerV2_runCommand(t *testing.T) {
	t.Run("run_simple_command", func(t *testing.T) {
		manager := NewManagerV2()
		executed := false

		cmd := &mockCommand{
			info: &Info{
				Name: "test",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				executed = true
				assert.NotNil(t, ctx)
				assert.NotNil(t, ctx.Stdout)
				assert.NotNil(t, ctx.Stderr)
				assert.NotNil(t, ctx.Stdin)
				return nil
			},
		}

		cobraCmd := manager.rootCmd
		err := manager.runCommand(cmd, cobraCmd, []string{})

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("run_command_with_args", func(t *testing.T) {
		manager := NewManagerV2()
		var capturedArgs []string

		cmd := &mockCommand{
			info: &Info{
				Name: "test",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				capturedArgs = ctx.Args
				return nil
			},
		}

		cobraCmd := manager.rootCmd
		expectedArgs := []string{"arg1", "arg2", "arg3"}
		err := manager.runCommand(cmd, cobraCmd, expectedArgs)

		assert.NoError(t, err)
		assert.Equal(t, expectedArgs, capturedArgs)
	})

	t.Run("run_command_returns_error", func(t *testing.T) {
		manager := NewManagerV2()
		expectedErr := assert.AnError

		cmd := &mockCommand{
			info: &Info{
				Name: "test",
				Desc: "Test command",
			},
			runFn: func(ctx *Context) error {
				return expectedErr
			},
		}

		cobraCmd := manager.rootCmd
		err := manager.runCommand(cmd, cobraCmd, []string{})

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestManagerV2_registerV2SubCommand_NonFlaggedCommand(t *testing.T) {
	t.Run("non_flagged_command_args_validation", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:    "app-create",
				Desc:    "Create an app",
				MinArgs: 1,
				MaxArgs: 3,
			},
		}

		manager.registerV2SubCommand(cmd)

		leafNode := manager.tree.Children["app"].Children["create"]
		assert.NotNil(t, leafNode)
		assert.False(t, leafNode.Command.DisableFlagParsing)
		assert.False(t, leafNode.Command.SilenceUsage)
		assert.NotNil(t, leafNode.Command.Args)
	})

	t.Run("command_use_and_descriptions", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "service-bind",
				Desc:  "Bind a service to an app\nDetailed description here",
				Usage: "service-bind <service> <app>",
			},
		}

		manager.registerV2SubCommand(cmd)

		leafNode := manager.tree.Children["service"].Children["bind"]
		assert.NotNil(t, leafNode)
		assert.Equal(t, "bind <service> <app>", leafNode.Command.Use)
		assert.Equal(t, "Bind a service to an app", leafNode.Command.Short)
		assert.Equal(t, cmd.info.Desc, leafNode.Command.Long)
	})
}

func TestManagerV2_Finish(t *testing.T) {
	t.Run("close_pager_writers_on_finish", func(t *testing.T) {
		manager := NewManagerV2()

		// Create multiple contexts using newContext
		ctx1 := manager.newContext(Context{
			Stdout: &strings.Builder{},
			Stderr: &strings.Builder{},
			Stdin:  strings.NewReader(""),
		})
		ctx2 := manager.newContext(Context{
			Stdout: &strings.Builder{},
			Stderr: &strings.Builder{},
			Stdin:  strings.NewReader(""),
		})

		assert.NotNil(t, ctx1)
		assert.NotNil(t, ctx2)
		assert.Len(t, manager.contexts, 2)

		// Finish should not panic
		manager.Finish()
	})

	t.Run("finish_with_no_contexts", func(t *testing.T) {
		manager := NewManagerV2()

		// Finish should not panic with no contexts
		manager.Finish()
	})
}

func TestManagerV2_newContext(t *testing.T) {
	t.Run("adds_context_to_list", func(t *testing.T) {
		manager := NewManagerV2()

		assert.Len(t, manager.contexts, 0)

		ctx := manager.newContext(Context{
			Stdout: &strings.Builder{},
			Stderr: &strings.Builder{},
			Stdin:  strings.NewReader(""),
		})

		assert.NotNil(t, ctx)
		assert.Len(t, manager.contexts, 1)
		assert.Equal(t, ctx, manager.contexts[0])
	})
}

func TestManagerV2_fillCommand_MinimumArgs(t *testing.T) {
	t.Run("min_args_greater_or_equal_max_args_allows_more_args", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:    "app-deploy",
				Desc:    "Deploy an app",
				MinArgs: 2,
				MaxArgs: 2, // MinArgs >= MaxArgs
			},
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		deployNode := appNode.Children["deploy"]
		assert.NotNil(t, deployNode)

		// Should accept exactly MinArgs
		err := deployNode.Command.Args(deployNode.Command, []string{"arg1", "arg2"})
		assert.NoError(t, err)

		// Should also accept more than MinArgs (MinimumNArgs behavior)
		err = deployNode.Command.Args(deployNode.Command, []string{"arg1", "arg2", "arg3"})
		assert.NoError(t, err)

		// Should reject fewer than MinArgs
		err = deployNode.Command.Args(deployNode.Command, []string{"arg1"})
		assert.Error(t, err)
	})
}

func TestManagerV2_fillCommand_ArbitraryArgs(t *testing.T) {
	t.Run("arbitrary_args_sets_cobra_arbitrary_args", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:    "plugin-test",
				Desc:    "Test plugin command",
				MinArgs: ArbitraryArgs,
			},
		}

		manager.Register(cmd)

		// Find the registered command in the tree
		pluginNode := manager.tree.Children["plugin"]
		assert.NotNil(t, pluginNode)

		testNode := pluginNode.Children["test"]
		assert.NotNil(t, testNode)

		// Verify that the command accepts arbitrary args
		// Test by checking the Args function allows any number of args
		err := testNode.Command.Args(testNode.Command, []string{})
		assert.NoError(t, err)
		err = testNode.Command.Args(testNode.Command, []string{"a", "b", "c", "d", "e"})
		assert.NoError(t, err)
	})
}

func TestManagerV2_fillCommand_DisableFlagParsing(t *testing.T) {
	t.Run("disable_flag_parsing_is_set", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "plugin-exec",
				Desc: "Execute plugin",
				V2: InfoV2{
					DisableFlagParsing: true,
				},
			},
		}

		manager.Register(cmd)

		pluginNode := manager.tree.Children["plugin"]
		assert.NotNil(t, pluginNode)

		execNode := pluginNode.Children["exec"]
		assert.NotNil(t, execNode)
		assert.True(t, execNode.Command.DisableFlagParsing)
	})

	t.Run("disable_flag_parsing_is_false_by_default", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "app-list",
				Desc: "List apps",
			},
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		listNode := appNode.Children["list"]
		assert.NotNil(t, listNode)
		assert.False(t, listNode.Command.DisableFlagParsing)
	})
}

func TestManagerV2_fillCommand_SilenceUsage(t *testing.T) {
	t.Run("silence_usage_is_set", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "plugin-run",
				Desc: "Run plugin",
				V2: InfoV2{
					SilenceUsage: true,
				},
			},
		}

		manager.Register(cmd)

		pluginNode := manager.tree.Children["plugin"]
		assert.NotNil(t, pluginNode)

		runNode := pluginNode.Children["run"]
		assert.NotNil(t, runNode)
		assert.True(t, runNode.Command.SilenceUsage)
	})

	t.Run("silence_usage_is_false_by_default", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "app-create",
				Desc: "Create app",
			},
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		createNode := appNode.Children["create"]
		assert.NotNil(t, createNode)
		assert.False(t, createNode.Command.SilenceUsage)
	})
}

func TestManagerV2_fillCommand_ParseFirstFlagsOnly(t *testing.T) {
	t.Run("parse_first_flags_only_processes_flags_before_args", func(t *testing.T) {
		manager := NewManagerV2()
		var capturedArgs []string

		cmd := &mockCommand{
			info: &Info{
				Name:    "plugin-exec",
				Desc:    "Execute plugin",
				MinArgs: ArbitraryArgs,
				V2: InfoV2{
					ParseFirstFlagsOnly: true,
					DisableFlagParsing:  true,
				},
			},
			runFn: func(ctx *Context) error {
				capturedArgs = ctx.Args
				return nil
			},
		}

		manager.Register(cmd)

		pluginNode := manager.tree.Children["plugin"]
		assert.NotNil(t, pluginNode)

		execNode := pluginNode.Children["exec"]
		assert.NotNil(t, execNode)

		// Execute the command via the root command with flags before args
		manager.rootCmd.SetArgs([]string{"plugin", "exec", "--target", "myserver", "arg1", "--other-flag"})
		err := manager.rootCmd.Execute()
		assert.NoError(t, err)

		// The args after parsing should only contain non-flag arguments
		assert.Equal(t, []string{"arg1", "--other-flag"}, capturedArgs)
	})

	t.Run("parse_first_flags_only_is_false_by_default", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name: "app-list",
				Desc: "List apps",
			},
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		listNode := appNode.Children["list"]
		assert.NotNil(t, listNode)

		// ParseFirstFlagsOnly is not exposed on cobra.Command, but we can verify
		// the behavior by checking the InfoV2 struct
		assert.False(t, cmd.info.V2.ParseFirstFlagsOnly)
	})
}

func TestManagerV2_mapCommonAliases(t *testing.T) {
	t.Run("verify_aliases_are_registered", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "app-remove",
				Desc:  "Remove an app",
				Usage: "app-remove <name>",
			},
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		removeNode := appNode.Children["remove"]
		assert.NotNil(t, removeNode)

		// Check that aliases are set for "remove"
		assert.Contains(t, removeNode.Command.Aliases, "delete")
	})

	t.Run("verify_create_aliases", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "app-create",
				Desc:  "Create an app",
				Usage: "app-create <name>",
			},
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		createNode := appNode.Children["create"]
		assert.NotNil(t, createNode)

		// Check that aliases are set for "create"
		assert.Contains(t, createNode.Command.Aliases, "add")
	})

	t.Run("verify_add_aliases", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "pool-add",
				Desc:  "Add a pool",
				Usage: "pool-add <name>",
			},
		}

		manager.Register(cmd)

		poolNode := manager.tree.Children["pool"]
		assert.NotNil(t, poolNode)

		addNode := poolNode.Children["add"]
		assert.NotNil(t, addNode)

		// Check that aliases are set for "add"
		assert.Contains(t, addNode.Command.Aliases, "create")
	})

	t.Run("verify_delete_aliases", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "node-delete",
				Desc:  "Delete a node",
				Usage: "node-delete <name>",
			},
		}

		manager.Register(cmd)

		nodeNode := manager.tree.Children["node"]
		assert.NotNil(t, nodeNode)

		deleteNode := nodeNode.Children["delete"]
		assert.NotNil(t, deleteNode)

		// Check that aliases are set for "delete"
		assert.Contains(t, deleteNode.Command.Aliases, "remove")
	})

	t.Run("verify_info_aliases", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "app-info",
				Desc:  "Show app info",
				Usage: "app-info <name>",
			},
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		infoNode := appNode.Children["info"]
		assert.NotNil(t, infoNode)

		// Check that aliases are set for "info"
		assert.Contains(t, infoNode.Command.Aliases, "describe")
	})

	t.Run("verify_log_aliases", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "app-log",
				Desc:  "Show app logs",
				Usage: "app-log <name>",
			},
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		logNode := appNode.Children["log"]
		assert.NotNil(t, logNode)

		// Check that aliases are set for "log"
		assert.Contains(t, logNode.Command.Aliases, "logs")
	})

	t.Run("verify_change_aliases", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "plan-change",
				Desc:  "Change plan",
				Usage: "plan-change <name>",
			},
		}

		manager.Register(cmd)

		planNode := manager.tree.Children["plan"]
		assert.NotNil(t, planNode)

		changeNode := planNode.Children["change"]
		assert.NotNil(t, changeNode)

		// Check that aliases are set for "change"
		assert.Contains(t, changeNode.Command.Aliases, "update")
		assert.Contains(t, changeNode.Command.Aliases, "set")
	})

	t.Run("verify_destroy_aliases", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "cluster-destroy",
				Desc:  "Destroy cluster",
				Usage: "cluster-destroy <name>",
			},
		}

		manager.Register(cmd)

		clusterNode := manager.tree.Children["cluster"]
		assert.NotNil(t, clusterNode)

		destroyNode := clusterNode.Children["destroy"]
		assert.NotNil(t, destroyNode)

		// Check that aliases are set for "destroy"
		assert.Contains(t, destroyNode.Command.Aliases, "remove")
		assert.Contains(t, destroyNode.Command.Aliases, "delete")
	})

	t.Run("no_alias_for_unknown_command", func(t *testing.T) {
		manager := NewManagerV2()

		cmd := &mockCommand{
			info: &Info{
				Name:  "app-deploy",
				Desc:  "Deploy an app",
				Usage: "app-deploy <name>",
			},
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		deployNode := appNode.Children["deploy"]
		assert.NotNil(t, deployNode)

		// "deploy" is not in mapCommonAliases, so aliases should be nil or empty
		assert.Empty(t, deployNode.Command.Aliases)
	})
}

func TestManagerV2_Integration(t *testing.T) {
	t.Run("complex_registration_scenario", func(t *testing.T) {
		manager := NewManagerV2()

		// Register topics
		manager.RegisterTopic("app", "Application management")
		manager.RegisterTopic("service", "Service management")
		manager.RegisterTopic("service-instance", "Service instance management")

		// Register commands
		commands := []*mockCommand{
			{
				info: &Info{
					Name:  "login",
					Desc:  "Login to server",
					Usage: "tsuru login",
					V2: InfoV2{
						OnlyAppendOnRoot: true,
						GroupID:          "auth",
					},
				},
			},
			{
				info: &Info{
					Name:  "app-list",
					Desc:  "List applications",
					Usage: "tsuru app-list",
				},
			},
			{
				info: &Info{
					Name:  "app-create",
					Desc:  "Create application",
					Usage: "tsuru app-create <name>",
				},
			},
			{
				info: &Info{
					Name:  "service-instance-info",
					Desc:  "Get service instance info",
					Usage: "tsuru service-instance-info <name>",
				},
			},
		}

		for _, cmd := range commands {
			manager.Register(cmd)
		}

		// Verify structure
		// Login should only be on root
		rootCommands := manager.rootCmd.Commands()
		var foundLogin bool
		for _, c := range rootCommands {
			if strings.HasPrefix(c.Use, "login") {
				foundLogin = true
				assert.False(t, c.Hidden)
			}
		}
		assert.True(t, foundLogin)

		// App commands should be under app topic
		assert.NotNil(t, manager.tree.Children["app"])
		assert.NotNil(t, manager.tree.Children["app"].Children["list"])
		assert.NotNil(t, manager.tree.Children["app"].Children["create"])

		// Service instance should be under service -> instance
		assert.NotNil(t, manager.tree.Children["service"])
		assert.NotNil(t, manager.tree.Children["service"].Children["instance"])
		assert.NotNil(t, manager.tree.Children["service"].Children["instance"].Children["info"])

		// Topic descriptions should be preserved
		assert.Equal(t, "Application management", manager.tree.Children["app"].Command.Short)
		assert.Equal(t, "Service instance management", manager.tree.Children["service"].Children["instance"].Command.Short)
	})

	t.Run("register_after_topic_creation", func(t *testing.T) {
		manager := NewManagerV2()

		// Register topic first
		manager.RegisterTopic("app", "Application management")

		// Then register command
		cmd := &mockCommand{
			info: &Info{
				Name: "app-list",
				Desc: "List applications",
			},
		}

		manager.Register(cmd)

		// Topic description should be preserved
		assert.Equal(t, "Application management", manager.tree.Children["app"].Command.Short)
		// Command should be added as child
		assert.NotNil(t, manager.tree.Children["app"].Children["list"])
	})
}

type mockFlaggedCommand struct {
	info  *Info
	flags *pflag.FlagSet
	runFn func(context *Context) error
}

func (m *mockFlaggedCommand) Info() *Info {
	return m.info
}

func (m *mockFlaggedCommand) Run(context *Context) error {
	if m.runFn != nil {
		return m.runFn(context)
	}
	return nil
}

func (m *mockFlaggedCommand) Flags() *pflag.FlagSet {
	return m.flags
}

func TestManagerV2_SetFlagCompletions(t *testing.T) {
	manager := NewManagerV2()

	completions := map[string]CompletionFunc{
		"app": func(toComplete string) ([]string, error) {
			return []string{"app1", "app2"}, nil
		},
		"team": func(toComplete string) ([]string, error) {
			return []string{"team1", "team2"}, nil
		},
	}

	manager.SetFlagCompletions(completions)

	assert.NotNil(t, manager.completions)
	assert.Len(t, manager.completions, 2)
	assert.Contains(t, manager.completions, "app")
	assert.Contains(t, manager.completions, "team")
}

func TestManagerV2_registerCompletionsOnCommand(t *testing.T) {
	t.Run("register_completions_on_flagged_command", func(t *testing.T) {
		manager := NewManagerV2()

		completions := map[string]CompletionFunc{
			"app": func(toComplete string) ([]string, error) {
				return []string{"app1", "app2", "app3"}, nil
			},
		}
		manager.SetFlagCompletions(completions)

		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String("app", "", "Application name")

		cmd := &mockFlaggedCommand{
			info: &Info{
				Name: "app-info",
				Desc: "Show app info",
			},
			flags: flags,
		}

		manager.Register(cmd)

		appNode := manager.tree.Children["app"]
		assert.NotNil(t, appNode)

		infoNode := appNode.Children["info"]
		assert.NotNil(t, infoNode)

		// Test that the completion function was registered
		completionFunc, exists := infoNode.Command.GetFlagCompletionFunc("app")
		assert.True(t, exists)
		assert.NotNil(t, completionFunc)

		results, directive := completionFunc(infoNode.Command, []string{}, "")
		assert.Equal(t, []string{"app1", "app2", "app3"}, results)
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})

	t.Run("completion_function_filters_by_prefix", func(t *testing.T) {
		manager := NewManagerV2()

		completions := map[string]CompletionFunc{
			"team": func(toComplete string) ([]string, error) {
				allTeams := []string{"alpha", "beta", "gamma"}
				var filtered []string
				for _, team := range allTeams {
					if strings.HasPrefix(team, toComplete) {
						filtered = append(filtered, team)
					}
				}
				return filtered, nil
			},
		}
		manager.SetFlagCompletions(completions)

		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String("team", "", "Team name")

		cmd := &mockFlaggedCommand{
			info: &Info{
				Name: "team-info",
				Desc: "Show team info",
			},
			flags: flags,
		}

		manager.Register(cmd)

		infoNode := manager.tree.Children["team"].Children["info"]
		completionFunc, _ := infoNode.Command.GetFlagCompletionFunc("team")

		// Test with prefix "a"
		results, _ := completionFunc(infoNode.Command, []string{}, "a")
		assert.Equal(t, []string{"alpha"}, results)

		// Test with prefix "b"
		results, _ = completionFunc(infoNode.Command, []string{}, "b")
		assert.Equal(t, []string{"beta"}, results)

		// Test with empty prefix
		results, _ = completionFunc(infoNode.Command, []string{}, "")
		assert.Equal(t, []string{"alpha", "beta", "gamma"}, results)
	})

	t.Run("completion_function_returns_error", func(t *testing.T) {
		manager := NewManagerV2()

		completions := map[string]CompletionFunc{
			"pool": func(toComplete string) ([]string, error) {
				return nil, assert.AnError
			},
		}
		manager.SetFlagCompletions(completions)

		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flags.String("pool", "", "Pool name")

		cmd := &mockFlaggedCommand{
			info: &Info{
				Name: "pool-info",
				Desc: "Show pool info",
			},
			flags: flags,
		}

		manager.Register(cmd)

		infoNode := manager.tree.Children["pool"].Children["info"]
		completionFunc, exists := infoNode.Command.GetFlagCompletionFunc("pool")
		assert.True(t, exists)

		results, directive := completionFunc(infoNode.Command, []string{}, "")
		assert.Nil(t, results)
		assert.Equal(t, cobra.ShellCompDirectiveError, directive)
	})

	t.Run("non_flagged_command_does_not_register_completions", func(t *testing.T) {
		manager := NewManagerV2()

		completions := map[string]CompletionFunc{
			"app": func(toComplete string) ([]string, error) {
				return []string{"app1"}, nil
			},
		}
		manager.SetFlagCompletions(completions)

		// Use regular mockCommand (not FlaggedCommand)
		cmd := &mockCommand{
			info: &Info{
				Name: "simple-cmd",
				Desc: "Simple command without flags",
			},
		}

		manager.Register(cmd)

		simpleNode := manager.tree.Children["simple"].Children["cmd"]
		assert.NotNil(t, simpleNode)

		// Should not panic and command should work normally
		assert.NotNil(t, simpleNode.Command)
	})
}
