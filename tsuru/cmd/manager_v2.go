package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	v2 "github.com/tsuru/tsuru-client/tsuru/cmd/v2"
)

// ManagerV2 is responsible for managing the commands using cobra for tsuru-client.
// this intends to replace the old Manager struct in the future.
type ManagerV2 struct {
	Enabled bool
	rootCmd *cobra.Command
	tree    *v2.CmdNode
}

type ManagerV2Opts struct {
	AfterFlagParseHook func()
}

func NewManagerV2(opts ...*ManagerV2Opts) *ManagerV2 {
	rootCmd := v2.NewRootCmd()

	m := &ManagerV2{
		Enabled: v2.Enabled(),
		rootCmd: rootCmd,
		tree:    v2.NewCmdNode(rootCmd),
	}

	if len(opts) == 1 && opts[0].AfterFlagParseHook != nil {
		originalPersistentPreRun := rootCmd.PersistentPreRun
		rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			originalPersistentPreRun(cmd, args)
			opts[0].AfterFlagParseHook()
		}
	}

	return m
}

func (m *ManagerV2) RegisterTopic(name, content string) {
	curr := m.tree

	parts := strings.Split(name, "-")

	for i, part := range parts {
		if curr.Children[part] != nil {
			if i == len(parts)-1 {
				panic(fmt.Sprintf("topic already registered: %s", part))

			}
			curr = curr.Children[part]
			continue
		}

		groupID := "resource"
		if i > 0 {
			groupID = "sub-resource"
		}

		newCmd := &cobra.Command{
			Use:                part,
			Short:              strings.TrimSpace(strings.Split(content, "\n")[0]),
			GroupID:            groupID,
			DisableFlagParsing: true,
		}

		curr.AddChild(newCmd)
		curr = curr.Children[part]
	}
}

func (m *ManagerV2) Register(command Command) {
	info := command.Info()

	if info.V2.Disabled {
		return
	}

	// 1. Legacy way to interact on tsuru-client
	// ex: tsuru app-deploy tsuru app-list
	m.registerV2FQDNOnRoot(command)
	if info.V2.OnlyAppendOnRoot {
		return
	}

	// 2. New way to interact on tsuru-client
	// ex: tsuru app deploy, tsuru app list, tsuru service instance info
	m.registerV2SubCommand(command)
}

func (m *ManagerV2) Run() {
	err := m.rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func (m *ManagerV2) registerV2SubCommand(command Command) {
	info := command.Info()
	fqdn := info.Name
	parts := strings.Split(fqdn, "-")
	curr := m.tree
	for i, part := range parts {
		found := false
		if _, found = curr.Children[part]; !found {
			newCmd := &cobra.Command{
				Use:                part,
				Short:              "Manage " + strings.Join(parts[:i+1], " ") + "s",
				DisableFlagParsing: true,
			}
			curr.AddChild(newCmd)
		}
		curr = curr.Children[part]

		if i == len(parts)-1 && !found {
			curr.Command.Use = part + stripUsage(fqdn, info.Usage)
			curr.Command.Short = strings.TrimSpace(strings.Split(info.Desc, "\n")[0])
			curr.Command.Long = info.Desc
			curr.Command.SilenceUsage = true
			curr.Command.Hidden = info.V2.Hidden
			curr.Command.Args = cobra.MinimumNArgs(0)
			curr.Command.RunE = func(cobraCommand *cobra.Command, args []string) error {
				return m.runCommand(command, cobraCommand, args)
			}

			_, isFlaggedCommand := command.(FlaggedCommand)

			if !isFlaggedCommand {
				curr.Command.DisableFlagParsing = false
				curr.Command.SilenceUsage = false
				curr.Command.Args = cobra.RangeArgs(info.MinArgs, info.MaxArgs)

			}
		}
	}
}

func stripUsage(fqdn, usage string) string {
	spacedFQDN := strings.ReplaceAll(fqdn, "-", " ")
	usage = strings.Replace(usage, fqdn, "", 1)
	return strings.Replace(usage, spacedFQDN, "", 1)
}

func (m *ManagerV2) runCommand(command Command, cobraCommand *cobra.Command, args []string) error {
	flaggedCommand, ok := command.(FlaggedCommand)
	if ok {
		fmt.Println("TODO: run command with flags", command.Info().Name)
		fmt.Println("Flags:", flaggedCommand.Flags())
		return nil
	}

	return command.Run(&Context{
		Args:   args,
		Stdout: cobraCommand.OutOrStdout(),
		Stderr: cobraCommand.OutOrStderr(),
		Stdin:  cobraCommand.InOrStdin(),
	})
}

func (m *ManagerV2) registerV2FQDNOnRoot(command Command) {
	info := command.Info()

	if info.V2.Disabled {
		return
	}

	fqdn := info.Name

	newCmd := &cobra.Command{
		Use:                fqdn,
		Short:              strings.TrimSpace(strings.Split(info.Desc, "\n")[0]),
		Long:               info.Usage,
		GroupID:            info.V2.GroupID,
		SilenceUsage:       true,
		Args:               cobra.MinimumNArgs(0),
		DisableFlagParsing: true,
		Hidden:             !info.V2.OnlyAppendOnRoot,
		RunE: func(cobraCommand *cobra.Command, args []string) error {
			return m.runCommand(command, cobraCommand, args)
		},
	}

	m.rootCmd.AddCommand(newCmd)
}

type InfoV2 struct {
	Disabled         bool
	Hidden           bool
	OnlyAppendOnRoot bool
	GroupID          string
}
