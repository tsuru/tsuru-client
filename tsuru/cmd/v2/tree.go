// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v2

import "github.com/spf13/cobra"

var groupLabels = map[string]string{
	"auth":         "Auth commands:",
	"resource":     "Manage resources:",
	"sub-resource": "Manage sub-resources:",
	"plugin":       "Plugins:",
	"shorthands":   "Shorthand commands:",
}

type CmdNode struct {
	Command  *cobra.Command
	Children map[string]*CmdNode
	Groups   map[string]bool
}

func (n *CmdNode) AddChild(c *cobra.Command) {
	if n.Children == nil {
		n.Children = make(map[string]*CmdNode)
	}
	n.Children[c.Name()] = &CmdNode{Command: c}

	if c.GroupID != "" {
		n.addGroup(c.GroupID)
	}

	for _, sub := range c.Commands() {
		n.Children[c.Name()].AddChild(sub)
	}
	n.Command.AddCommand(c)
}

func (n *CmdNode) addGroup(groupID string) {
	if n.Groups == nil {
		n.Groups = make(map[string]bool)
	}
	if n.Groups[groupID] {
		return
	}

	n.Command.AddGroup(&cobra.Group{
		ID:    groupID,
		Title: groupLabels[groupID],
	})
	n.Groups[groupID] = true

}

func NewCmdNode(rootCmd *cobra.Command) *CmdNode {
	return &CmdNode{
		Command:  rootCmd,
		Children: make(map[string]*CmdNode),
		Groups:   make(map[string]bool),
	}
}
