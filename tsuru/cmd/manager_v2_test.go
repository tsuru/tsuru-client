// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockCommand struct {
	info   *Info
	runFn  func(context *Context) error
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
			if c.Use == "app-list" {
				foundFQDN = true
				assert.Equal(t, "List all apps", c.Short)
				assert.Equal(t, "tsuru app-list [flags]", c.Long)
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
			if c.Use == "login" {
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
			if c.Use == "app-list" {
				found = true
				assert.Equal(t, "List all apps", c.Short)
				assert.Equal(t, "Usage: tsuru app-list", c.Long)
				assert.Equal(t, "resource", c.GroupID)
				assert.True(t, c.SilenceUsage)
				assert.True(t, c.Hidden) // Hidden by default
				assert.True(t, c.DisableFlagParsing)
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
		assert.Equal(t, "bind", leafNode.Command.Use)
		assert.Equal(t, "Bind a service to an app", leafNode.Command.Short)
		assert.Equal(t, cmd.info.Desc, leafNode.Command.Long)
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
			if c.Use == "login" {
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
