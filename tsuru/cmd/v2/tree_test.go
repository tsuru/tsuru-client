// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v2

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdNode(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "root",
	}

	node := NewCmdNode(rootCmd)

	assert.NotNil(t, node)
	assert.Equal(t, rootCmd, node.Command)
	assert.NotNil(t, node.Children)
	assert.NotNil(t, node.Groups)
	assert.Empty(t, node.Children)
	assert.Empty(t, node.Groups)
}

func TestCmdNode_AddChild(t *testing.T) {
	t.Run("add_single_child", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := NewCmdNode(rootCmd)

		childCmd := &cobra.Command{
			Use: "child",
		}

		node.AddChild(childCmd)

		assert.NotNil(t, node.Children["child"])
		assert.Equal(t, childCmd, node.Children["child"].Command)
		assert.Contains(t, rootCmd.Commands(), childCmd)
	})

	t.Run("add_multiple_children", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := NewCmdNode(rootCmd)

		child1 := &cobra.Command{Use: "child1"}
		child2 := &cobra.Command{Use: "child2"}
		child3 := &cobra.Command{Use: "child3"}

		node.AddChild(child1)
		node.AddChild(child2)
		node.AddChild(child3)

		assert.Len(t, node.Children, 3)
		assert.NotNil(t, node.Children["child1"])
		assert.NotNil(t, node.Children["child2"])
		assert.NotNil(t, node.Children["child3"])
	})

	t.Run("add_child_with_group", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := NewCmdNode(rootCmd)

		childCmd := &cobra.Command{
			Use:     "login",
			GroupID: "auth",
		}

		node.AddChild(childCmd)

		assert.True(t, node.Groups["auth"])
		assert.Contains(t, rootCmd.Commands(), childCmd)

		// Verify group was added to root command
		groups := rootCmd.Groups()
		assert.Len(t, groups, 1)
		assert.Equal(t, "auth", groups[0].ID)
		assert.Equal(t, "Auth commands:", groups[0].Title)
	})

	t.Run("add_child_with_nested_subcommands", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := NewCmdNode(rootCmd)

		subCmd := &cobra.Command{Use: "sub"}
		parentCmd := &cobra.Command{Use: "parent"}
		parentCmd.AddCommand(subCmd)

		node.AddChild(parentCmd)

		assert.NotNil(t, node.Children["parent"])
		assert.NotNil(t, node.Children["parent"].Children["sub"])
		assert.Equal(t, subCmd, node.Children["parent"].Children["sub"].Command)
	})

	t.Run("initialize_children_map_if_nil", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := &CmdNode{
			Command:  rootCmd,
			Children: nil,
		}

		childCmd := &cobra.Command{Use: "child"}
		node.AddChild(childCmd)

		assert.NotNil(t, node.Children)
		assert.NotNil(t, node.Children["child"])
	})
}

func TestCmdNode_addGroup(t *testing.T) {
	t.Run("add_auth_group", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := NewCmdNode(rootCmd)

		node.addGroup("auth")

		assert.True(t, node.Groups["auth"])
		groups := rootCmd.Groups()
		assert.Len(t, groups, 1)
		assert.Equal(t, "auth", groups[0].ID)
		assert.Equal(t, "Auth commands:", groups[0].Title)
	})

	t.Run("add_resource_group", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := NewCmdNode(rootCmd)

		node.addGroup("resource")

		assert.True(t, node.Groups["resource"])
		groups := rootCmd.Groups()
		assert.Len(t, groups, 1)
		assert.Equal(t, "resource", groups[0].ID)
		assert.Equal(t, "Manage resources:", groups[0].Title)
	})

	t.Run("add_sub_resource_group", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := NewCmdNode(rootCmd)

		node.addGroup("sub-resource")

		assert.True(t, node.Groups["sub-resource"])
		groups := rootCmd.Groups()
		assert.Len(t, groups, 1)
		assert.Equal(t, "sub-resource", groups[0].ID)
		assert.Equal(t, "Manage sub-resources:", groups[0].Title)
	})

	t.Run("add_multiple_groups", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := NewCmdNode(rootCmd)

		node.addGroup("auth")
		node.addGroup("resource")
		node.addGroup("sub-resource")

		assert.True(t, node.Groups["auth"])
		assert.True(t, node.Groups["resource"])
		assert.True(t, node.Groups["sub-resource"])
		assert.Len(t, rootCmd.Groups(), 3)
	})

	t.Run("add_duplicate_group", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := NewCmdNode(rootCmd)

		node.addGroup("auth")
		node.addGroup("auth")

		assert.True(t, node.Groups["auth"])
		// Should only have one group, not duplicates
		assert.Len(t, rootCmd.Groups(), 1)
	})

	t.Run("initialize_groups_map_if_nil", func(t *testing.T) {
		rootCmd := &cobra.Command{
			Use: "root",
		}
		node := &CmdNode{
			Command: rootCmd,
			Groups:  nil,
		}

		node.addGroup("auth")

		assert.NotNil(t, node.Groups)
		assert.True(t, node.Groups["auth"])
	})
}

func TestCmdNode_ComplexHierarchy(t *testing.T) {
	t.Run("complex_nested_structure", func(t *testing.T) {
		rootCmd := &cobra.Command{Use: "root"}
		node := NewCmdNode(rootCmd)

		// Create a complex hierarchy
		loginCmd := &cobra.Command{Use: "login", GroupID: "auth"}
		logoutCmd := &cobra.Command{Use: "logout", GroupID: "auth"}

		appCmd := &cobra.Command{Use: "app", GroupID: "resource"}
		appCreateCmd := &cobra.Command{Use: "create", GroupID: "sub-resource"}
		appDeleteCmd := &cobra.Command{Use: "delete", GroupID: "sub-resource"}
		appCmd.AddCommand(appCreateCmd)
		appCmd.AddCommand(appDeleteCmd)

		node.AddChild(loginCmd)
		node.AddChild(logoutCmd)
		node.AddChild(appCmd)

		// Verify structure
		assert.Len(t, node.Children, 3)
		assert.NotNil(t, node.Children["login"])
		assert.NotNil(t, node.Children["logout"])
		assert.NotNil(t, node.Children["app"])

		// Verify nested commands
		assert.Len(t, node.Children["app"].Children, 2)
		assert.NotNil(t, node.Children["app"].Children["create"])
		assert.NotNil(t, node.Children["app"].Children["delete"])

		// Verify groups on root node (groups from direct children)
		assert.True(t, node.Groups["auth"])
		assert.True(t, node.Groups["resource"])

		// Verify groups on app node (groups from app's children)
		assert.True(t, node.Children["app"].Groups["sub-resource"])
	})
}
