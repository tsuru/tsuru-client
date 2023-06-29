// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	tsuruV1Config "github.com/tsuru/tsuru-client/tsuru/config"
	tsuruCmd "github.com/tsuru/tsuru/cmd"
)

var ignoredLegacyCommands = map[string]bool{
	"change-password": true,
	"cluster-add":     true,
	"cluster-list":    true,
	"cluster-remove":  true,
	"cluster-update":  true,
	"help":            true,
	"reset-password":  true,
}

func newV1LegacyCmdManager() *tsuruCmd.Manager {
	versionForLegacy := strings.TrimLeft(version.Version, "v") + "-legacy-plugin"
	if version.Version == "dev" {
		versionForLegacy = "dev"
	}
	return tsuruV1Config.BuildManager("tsuru", versionForLegacy)
}

func newLegacyCommand(v1CmdManager *tsuruCmd.Manager) *cobra.Command {
	legacyCmd := &cobra.Command{
		Use:   "legacy",
		Short: "legacy is the previous version of tsuru cli",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLegacyCommand(v1CmdManager, args)
		},
		Args:               cobra.MinimumNArgs(0),
		DisableFlagParsing: true,
	}
	return legacyCmd
}

func runLegacyCommand(v1CmdManager *tsuruCmd.Manager, args []string) error {
	var err error
	defer recoverCmdPanicExitError(&err)

	v1CmdManager.Run(args)
	return err
}

func recoverCmdPanicExitError(err *error) {
	if r := recover(); r != nil {
		if e, ok := r.(*tsuruCmd.PanicExitError); ok {
			if e.Code > 0 {
				*err = e
			}
			return
		}
		panic(r)
	}
}

type cmdNode struct {
	command  *cobra.Command
	children map[string]*cmdNode
}

func (n *cmdNode) addChild(c *cobra.Command) {
	if n.children == nil {
		n.children = make(map[string]*cmdNode)
	}
	n.children[c.Name()] = &cmdNode{command: c}
	for _, sub := range c.Commands() {
		n.children[c.Name()].addChild(sub)
	}
}

func addMissingV1LegacyCommands(rootCmd *cobra.Command, v1CmdManager *tsuruCmd.Manager) {
	// build current commands tree (without legacy commands)
	tree := &cmdNode{command: rootCmd}
	for _, c := range rootCmd.Commands() {
		tree.addChild(c)
	}

	// sort legacy commands by less specific ones first (create "deploy" before "deploy list" )
	v1Commands := make([]v1Command, 0, len(v1CmdManager.Commands))
	for cmdName, v1Cmd := range v1CmdManager.Commands {
		v1Commands = append(v1Commands, v1Command{cmdName, v1Cmd})
	}
	sort.Sort(ByPriority(v1Commands))

	// add missing legacy commands
	for _, v1Cmd := range v1Commands {
		// ignore this legacy commands
		if ignoredLegacyCommands[v1Cmd.name] {
			continue
		}
		addMissingV1LegacyCommand(tree, v1CmdManager, v1Cmd)
	}
}

func addMissingV1LegacyCommand(tree *cmdNode, v1CmdManager *tsuruCmd.Manager, v1Cmd v1Command) {
	curr := tree
	parts := strings.Split(strings.ReplaceAll(v1Cmd.name, "-", " "), " ")
	for i, part := range parts {
		found := false
		if _, found = curr.children[part]; !found {
			newCmd := &cobra.Command{
				Use:                part,
				Short:              "[v1] " + strings.Join(parts[:i+1], " "),
				DisableFlagParsing: true,
			}
			curr.addChild(newCmd)
			curr.command.AddCommand(newCmd)
		}
		curr = curr.children[part]

		if i == len(parts)-1 && !found {
			curr.command.Short = "[v1] " + strings.Split(v1Cmd.cmd.Info().Desc, "\n")[0]
			curr.command.Long = v1Cmd.cmd.Info().Usage
			curr.command.SilenceUsage = true
			curr.command.Args = cobra.MinimumNArgs(0)
			curr.command.RunE = func(cmd *cobra.Command, args []string) error {
				return runLegacyCommand(v1CmdManager, append(parts, args...))
			}
		}
	}
}

type v1Command struct {
	name string
	cmd  tsuruCmd.Command
}

type ByPriority []v1Command

func (a ByPriority) Len() int      { return len(a) }
func (a ByPriority) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByPriority) Less(i, j int) bool {
	Li := len(strings.Split(a[i].name, " "))
	Lj := len(strings.Split(a[j].name, " "))
	if Li == Lj {
		return a[i].name < a[j].name
	}
	return Li < Lj
}
