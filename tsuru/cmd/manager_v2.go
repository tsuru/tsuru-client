package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	v2 "github.com/tsuru/tsuru-client/tsuru/cmd/v2"
)

// ManagerV2 is responsible for managing the commands using cobra for tsuru-client.
// this intends to replace the old Manager struct in the future.
type ManagerV2 struct {
	rootCmd     *cobra.Command
	tree        *v2.CmdNode
	contexts    []*Context
	retryHook   func(err error) bool
	completions map[string]CompletionFunc
}

type CompletionFunc func(toComplete string) ([]string, error)

type ManagerV2Opts struct {
	AfterFlagParseHook func()
	RetryHook          func(err error) bool
}

func NewManagerV2(opts ...*ManagerV2Opts) *ManagerV2 {
	rootCmd := v2.NewRootCmd()

	m := &ManagerV2{
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
	if len(opts) == 1 && opts[0].RetryHook != nil {
		m.retryHook = opts[0].RetryHook
	}
	return m
}

func (m *ManagerV2) Cobra() *cobra.Command {
	return m.rootCmd
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

func (m *ManagerV2) RegisterDeprecated(command Command, oldName string) {
	deprecatedCmd := &DeprecatedCommand{Command: command, oldName: oldName}

	m.Register(command)
	m.Register(deprecatedCmd)
}

func (m *ManagerV2) RegisterShorthand(command Command, shorthand string) {
	m.Register(&ShorthandCommand{Command: command, shorthand: shorthand})
}

func (m *ManagerV2) Register(command Command) {
	info := command.Info()

	// 1. Legacy way to interact on tsuru-client
	// ex: tsuru app-deploy tsuru app-list
	m.registerV2FQDNOnRoot(command)
	if info.OnlyAppendOnRoot {
		return
	}

	// 2. New way to interact on tsuru-client
	// ex: tsuru app deploy, tsuru app list, tsuru service instance info
	m.registerV2SubCommand(command)
}

func (m *ManagerV2) SetFlagCompletions(completions map[string]CompletionFunc) {
	m.completions = completions
}

func (m *ManagerV2) Run() error {
	ctx := context.Background()
	cmd, err := m.rootCmd.ExecuteContextC(ctx)

	if m.retryHook != nil && err != nil {
		if retry := m.retryHook(err); retry {
			cmd, err = m.rootCmd.ExecuteContextC(ctx)
		}
	}

	if err != nil {
		cmd.Println(cmd.UsageString())
	}

	return err
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
			curr.Command.Use = part

			if info.Usage != "" {
				curr.Command.Use = part + " " + strings.TrimSpace(info.Usage)
			}
			curr.Command.Aliases = standards.CommonAliases[part]
			m.fillCommand(curr.Command, command)
		}
	}
}

func (m *ManagerV2) fillCommand(cobraCommand *cobra.Command, command Command) {
	info := command.Info()

	cobraCommand.Short = strings.TrimSpace(strings.Split(info.Desc, "\n")[0])
	cobraCommand.Long = info.Desc
	cobraCommand.DisableFlagParsing = info.DisableFlagParsing
	cobraCommand.SilenceUsage = info.SilenceUsage
	cobraCommand.Hidden = info.Hidden
	cobraCommand.Args = cobra.ArbitraryArgs

	if info.MinArgs > 0 && info.MinArgs >= info.MaxArgs {
		cobraCommand.Args = cobra.MinimumNArgs(info.MinArgs)
	} else if info.MaxArgs >= 0 && info.MinArgs >= 0 && info.MaxArgs > info.MinArgs {
		cobraCommand.Args = cobra.RangeArgs(info.MinArgs, info.MaxArgs)
	}

	cobraCommand.RunE = func(cobraCommand *cobra.Command, args []string) error {
		if info.ParseFirstFlagsOnly {
			args = v2.ParseFirstFlagsOnly(cobraCommand, args)

			target, _ := cobraCommand.Flags().GetString("target")
			if target != "" {
				os.Setenv("TSURU_TARGET", target)
			}
		}
		return m.runCommand(command, cobraCommand, args)
	}

	flaggedCommand, isFlaggedCommand := command.(FlaggedCommand)

	if isFlaggedCommand {
		cobraCommand.Flags().SortFlags = false
		flags := flaggedCommand.Flags()
		cobraCommand.Flags().AddFlagSet(flags)

		m.registerCompletionsOnCommand(cobraCommand)
	}
	autoCompleteCommand, isAutoCompleteCommand := command.(AutoCompleteCommand)
	if isAutoCompleteCommand {
		cobraCommand.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			result, err := autoCompleteCommand.Complete(args, toComplete)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return nil, cobra.ShellCompDirectiveError
			}
			return result, cobra.ShellCompDirectiveNoFileComp
		}
	}
}

func (m *ManagerV2) registerCompletionsOnCommand(cobraCommand *cobra.Command) {
	for name, fn := range m.completions {
		cobraCommand.RegisterFlagCompletionFunc(name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			result, err := fn(toComplete)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return nil, cobra.ShellCompDirectiveError
			}
			return result, cobra.ShellCompDirectiveNoFileComp
		})
	}
}

func (m *ManagerV2) runCommand(command Command, cobraCommand *cobra.Command, args []string) error {
	context := m.newContext(Context{
		Args:   args,
		Stdout: cobraCommand.OutOrStdout(),
		Stderr: cobraCommand.OutOrStderr(),
		Stdin:  cobraCommand.InOrStdin(),
	})

	sigChan := make(chan os.Signal, 1)
	if cancelable, ok := command.(Cancelable); ok {
		signal.Notify(sigChan, syscall.SIGINT)
		go func(context Context) {
			for range sigChan {
				fmt.Fprintln(context.Stdout, "Attempting command cancellation...")
				errCancel := cancelable.Cancel(context)
				if errCancel == nil {
					return
				}
				fmt.Fprintf(context.Stderr, "Error canceling command: %v. Proceeding.", errCancel)
			}
		}(*context)
	}

	return command.Run(context)
}

func (m *ManagerV2) newContext(c Context) *Context {
	stdout := newPagerWriter(c.Stdout)
	stdin := newSyncReader(c.Stdin, c.Stdout)
	ctx := &Context{Args: c.Args, Stdout: stdout, Stderr: c.Stderr, Stdin: stdin}
	m.contexts = append(m.contexts, ctx)

	return ctx
}

func (m *ManagerV2) Finish() {
	for _, ctx := range m.contexts {
		if pagerWriter, ok := ctx.Stdout.(*pagerWriter); ok {
			pagerWriter.close()
		}
	}
}

func (m *ManagerV2) registerV2FQDNOnRoot(command Command) {
	info := command.Info()

	fqdn := info.Name

	newCmd := &cobra.Command{
		Use:     fqdn,
		GroupID: info.GroupID,
	}

	if info.Usage != "" {
		newCmd.Use = fqdn + " " + strings.TrimSpace(info.Usage)
	}

	m.fillCommand(newCmd, command)
	newCmd.Hidden = !info.OnlyAppendOnRoot
	newCmd.Aliases = standards.CommonAliases[fqdn]
	m.tree.AddChild(newCmd)
}
