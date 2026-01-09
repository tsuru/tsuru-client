// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	goVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sajari/fuzzy"
	"github.com/spf13/pflag"
	"github.com/tsuru/tsuru/fs"
)

var (
	ErrAbortCommand = errors.New("")

	// ErrLookup is the error that should be returned by lookup functions when it
	// cannot find a matching command for the given parameters.
	ErrLookup = errors.New("lookup error - command not found")
)

type exiter interface {
	Exit(int)
}

type osExiter struct{}

func (e osExiter) Exit(code int) {
	os.Exit(code)
}

type PanicExitError struct {
	Code int
}

func (e *PanicExitError) Error() string {
	return fmt.Sprintf("Exiting with code: %d", e.Code)
}

type PanicExiter struct{}

func (e PanicExiter) Exit(code int) {
	err := &PanicExitError{Code: code}
	panic(err)
}

type Lookup func(context *Context) error

type Manager struct {
	Commands map[string]Command

	topics        map[string]string
	topicCommands map[string][]Command
	name          string
	stdout        io.Writer
	stderr        io.Writer
	stdin         io.Reader
	e             exiter
	original      string
	wrong         bool
	lookup        Lookup
	contexts      []*Context

	AfterFlagParseHook func()
	RetryHook          func(err error) (retry bool)

	// V2 fields using cobra commander
	v2 *ManagerV2
}

// This is discouraged: use NewManagerPanicExiter instead. Handle panic(*PanicExitError) accordingly
func NewManager(name string, stdout, stderr io.Writer, stdin io.Reader, lookup Lookup) *Manager {
	var manager *Manager
	manager = &Manager{
		name:          name,
		stdout:        stdout,
		stderr:        stderr,
		stdin:         stdin,
		lookup:        lookup,
		topics:        map[string]string{},
		topicCommands: map[string][]Command{},

		// v2 will be a replacement for this manager in the future
		v2: NewManagerV2(&ManagerV2Opts{
			AfterFlagParseHook: func() {
				if manager.AfterFlagParseHook != nil {
					manager.AfterFlagParseHook()
				}
			},
			RetryHook: func(err error) bool {
				if manager.RetryHook != nil {
					return manager.RetryHook(err)
				}
				return false
			},
		}),
	}

	manager.Register(&help{manager})
	return manager
}

// When using this, you should handle panic(*PanicExitError) accordingly
func NewManagerPanicExiter(name string, stdout, stderr io.Writer, stdin io.Reader, lookup Lookup) *Manager {
	manager := NewManager(name, stdout, stderr, stdin, lookup)
	manager.e = PanicExiter{}
	return manager
}

// RegisterPlugin registers a plugin command in the manager
// only for v2 engine
func (m *Manager) RegisterPlugin(command Command) {
	if m.v2.Enabled {
		m.v2.Register(command)
	}
}

func (m *Manager) Register(command Command) {
	if m.v2.Enabled {
		m.v2.Register(command)
	}
	if m.Commands == nil {
		m.Commands = make(map[string]Command)
	}
	name := command.Info().Name
	_, found := m.Commands[name]
	if found {
		panic(fmt.Sprintf("command already registered: %s", name))
	}
	m.Commands[name] = command

	parts := strings.Split(name, "-")

	for i := 1; i < len(parts); i++ {
		topic := strings.Join(parts[0:i], " ")
		if _, ok := m.topics[topic]; !ok {
			m.topics[topic] = ""
		}

		m.topicCommands[topic] = append(m.topicCommands[topic], command)
	}
}

func (m *Manager) RegisterDeprecated(command Command, oldName string) {
	deprecatedCmd := &DeprecatedCommand{Command: command, oldName: oldName}

	if m.v2.Enabled {
		m.v2.Register(command)
		m.v2.Register(deprecatedCmd)
	}

	if m.Commands == nil {
		m.Commands = make(map[string]Command)
	}
	name := command.Info().Name
	_, found := m.Commands[name]
	if found {
		panic(fmt.Sprintf("command already registered: %s", name))
	}
	m.Commands[name] = command
	m.Commands[oldName] = deprecatedCmd
}

func (m *Manager) RegisterShorthand(command Command, shorthand string) {
	if m.v2.Enabled {
		m.v2.Register(&ShorthandCommand{Command: command, shorthand: shorthand})
	}
}

type RemovedCommand struct {
	Name string
	Help string
}

func (c *RemovedCommand) Info() *Info {
	return &Info{
		Name:  c.Name,
		Usage: c.Name,
		Desc:  fmt.Sprintf("This command was removed. %s", c.Help),
		fail:  true,
	}
}

func (c *RemovedCommand) Run(context *Context) error {
	return ErrAbortCommand
}

func (m *Manager) RegisterRemoved(name string, help string) {
	if m.Commands == nil {
		m.Commands = make(map[string]Command)
	}
	_, found := m.Commands[name]
	if found {
		panic(fmt.Sprintf("command already registered: %s", name))
	}
	m.Commands[name] = &RemovedCommand{Name: name, Help: help}
}

func (m *Manager) RegisterTopic(name, content string) {
	if m.v2 != nil && m.v2.Enabled {
		m.v2.RegisterTopic(name, content)
	}
	name = strings.ReplaceAll(name, "-", " ")
	if m.topics == nil {
		m.topics = make(map[string]string)
	}
	value := m.topics[name]
	if value != "" {
		panic(fmt.Sprintf("topic already registered: %s", name))
	}
	m.topics[name] = content
}

func (m *Manager) Run(args []string) {
	if m.v2.Enabled {
		defer m.finisher()
		m.v2.Run()
		return
	}
	var (
		status         int
		verbosity      int
		displayHelp    bool
		displayVersion bool
		target         string
	)
	if len(args) == 0 {
		args = append(args, "help")
	}
	flagset := pflag.NewFlagSet("tsuru flags", pflag.ContinueOnError)
	flagset.SetOutput(m.stderr)
	flagset.SetInterspersed(false)
	flagset.IntVarP(&verbosity, "verbosity", "v", 0, "Verbosity level: 1 => print HTTP requests; 2 => print HTTP requests/responses")
	flagset.BoolVarP(&displayHelp, "help", "h", false, "Display help and exit")
	flagset.BoolVar(&displayVersion, "version", false, "Print version and exit")
	flagset.StringVarP(&target, "target", "t", "", "Define target for running command")
	parseErr := flagset.Parse(args)
	if parseErr != nil {
		fmt.Fprint(m.stderr, parseErr)
		m.finisher().Exit(2)
		return
	}
	args = flagset.Args()
	args = m.normalizeCommandArgs(args)
	if displayHelp {
		args = append([]string{"help"}, args...)
	} else if displayVersion {
		args = []string{"version"}
	}

	if len(target) > 0 {
		os.Setenv("TSURU_TARGET", target)
	}

	if verbosity > 0 {
		os.Setenv("TSURU_VERBOSITY", strconv.Itoa(verbosity))
	}

	if m.AfterFlagParseHook != nil {
		m.AfterFlagParseHook()
	}

	if m.lookup != nil {
		context := m.newContext(Context{
			Args:   args,
			Stdout: m.stdout,
			Stderr: m.stderr,
			Stdin:  m.stdin,
		})
		err := m.lookup(context)
		if err != nil && err != ErrLookup {
			fmt.Fprint(m.stderr, err)
			m.finisher().Exit(1)
			return
		} else if err == nil {
			return
		}
	}
	name := args[0]
	command, ok := m.Commands[name]
	if !ok {
		if msg, isTopic := m.tryImplicitTopic(args); isTopic {
			fmt.Fprint(m.stdout, msg)
			return
		}

		topicBasedName := m.findTopicBasedCommand(args)

		fmt.Fprintf(m.stderr, "%s: %q is not a %s command. See %q.\n", m.name, topicBasedName, m.name, m.name+" help")
		var keys []string
		for key := range m.Commands {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		possibleCommands := []string{}

		for _, key := range keys {
			topicBasedKey := strings.ReplaceAll(key, "-", " ")

			if len(args) == 1 && strings.Contains(topicBasedKey, args[0]) {
				possibleCommands = append(possibleCommands, topicBasedKey)
				continue
			}

			if score := fuzzy.Levenshtein(&topicBasedKey, &topicBasedName); score < 3 {
				possibleCommands = append(possibleCommands, topicBasedKey)
			}
		}
		if len(possibleCommands) > 0 {
			fmt.Fprintln(m.stderr, "\nDid you mean?")

			for _, cmd := range possibleCommands {
				fmt.Fprintf(m.stderr, "\t%s\n", cmd)
			}
		}
		m.finisher().Exit(1)
		return
	}
	args = args[1:]
	info := command.Info()
	command, args, err := m.handleFlags(command, name, args)
	if err != nil {
		fmt.Fprint(m.stderr, err)
		m.finisher().Exit(1)
		return
	}
	if info.fail {
		command = m.Commands["help"]
		args = []string{name}
		status = 1
	}
	if length := len(args); (length < info.MinArgs || (info.MaxArgs > 0 && length > info.MaxArgs)) &&
		name != "help" {
		m.wrong = true
		m.original = info.Name
		command = m.Commands["help"]
		args = []string{name}
		status = 1
	}
	context := m.newContext(Context{
		Args:   args,
		Stdout: m.stdout,
		Stderr: m.stderr,
		Stdin:  m.stdin,
	})
	sigChan := make(chan os.Signal, 1)
	if cancelable, ok := command.(Cancelable); ok {
		signal.Notify(sigChan, syscall.SIGINT)
		go func(context Context) {
			for range sigChan {
				fmt.Fprintln(m.stdout, "Attempting command cancellation...")
				errCancel := cancelable.Cancel(context)
				if errCancel == nil {
					return
				}
				fmt.Fprintf(m.stderr, "Error canceling command: %v. Proceeding.", errCancel)
			}
		}(*context)
	}
	err = command.Run(context)

	if m.RetryHook != nil && err != nil {
		if retry := m.RetryHook(err); retry {
			err = command.Run(context)
		}
	}

	close(sigChan)

	if err != nil {
		errorMsg := err.Error()
		if verbosity > 0 {
			errorMsg = fmt.Sprintf("%+v", err)
		}
		if bodyErr, ok := err.(interface {
			Body() []byte
		}); ok {
			body := string(bodyErr.Body())
			if body != "" {
				errorMsg = fmt.Sprintf("%s: %s", errorMsg, body)
			}
		}
		if !strings.HasSuffix(errorMsg, "\n") {
			errorMsg += "\n"
		}
		if err != ErrAbortCommand {
			io.WriteString(m.stderr, "Error: "+errorMsg)
		}
		status = 1
	}
	m.finisher().Exit(status)
}

func (m *Manager) findTopicBasedCommand(args []string) string {
	safeCmd := []string{}
	for _, arg := range args {
		if len(arg) > 0 && arg[0] == '-' {
			break
		}

		safeCmd = append(safeCmd, arg)
	}

	return strings.Join(safeCmd, " ")
}

func (m *Manager) newContext(c Context) *Context {
	stdout := newPagerWriter(c.Stdout)
	stdin := newSyncReader(c.Stdin, c.Stdout)
	ctx := &Context{Args: c.Args, Stdout: stdout, Stderr: c.Stderr, Stdin: stdin}
	m.contexts = append(m.contexts, ctx)
	return ctx
}

func (m *Manager) handleFlags(command Command, name string, args []string) (Command, []string, error) {
	var flagset *pflag.FlagSet
	if flagged, ok := command.(FlaggedCommand); ok {
		flagset = flagged.Flags()
	} else {
		flagset = pflag.NewFlagSet(name, pflag.ExitOnError)
	}
	var helpRequested bool
	flagset.SetOutput(m.stderr)

	if flagset.Lookup("help") == nil {
		flagset.BoolVarP(&helpRequested, "help", "h", false, "Display help and exit")
	}

	err := flagset.Parse(args)
	if err != nil {
		return nil, nil, err
	}

	if helpRequested {
		command = m.Commands["help"]
		args = []string{name}
	} else {
		args = flagset.Args()
	}
	return command, args, nil
}

func (m *Manager) finisher() exiter {
	if m.v2 != nil && m.v2.Enabled {
		m.v2.Finish()
	}
	if pagerWriter, ok := m.stdout.(*pagerWriter); ok {
		pagerWriter.close()
	}
	for _, ctx := range m.contexts {
		if pagerWriter, ok := ctx.Stdout.(*pagerWriter); ok {
			pagerWriter.close()
		}
	}
	if m.e == nil {
		m.e = osExiter{}
	}
	return m.e
}

var topicRE = regexp.MustCompile(`(?s)^(.*)\n*$`)

func (m *Manager) tryImplicitTopic(args []string) (string, bool) {
	topicName := strings.Join(args, " ")

	topic, isExplicit := m.topics[topicName]
	commands := m.topicCommands[topicName]

	if len(commands) == 0 && !isExplicit {
		return "", false
	}

	if len(commands) > 0 {
		if len(topic) > 0 {
			topic = topicRE.ReplaceAllString(topic, "$1\n\n")
		}

		topic += fmt.Sprintf("The following commands are available in the %q topic:\n\n", topicName)
		var group []string
		for _, command := range commands {
			group = append(group, command.Info().Name)
		}
		topic += m.dumpCommands(group)
	}

	return topic, true
}

func formatDescriptionLine(label, description string, maxSize int) string {
	description = strings.Split(description, "\n")[0]
	description = strings.Split(description, ".")[0]
	if len(description) > 2 {
		description = strings.ToUpper(description[:1]) + description[1:]
	}
	fmtStr := fmt.Sprintf("  %%-%ds %%s\n", maxSize)
	return fmt.Sprintf(fmtStr, strings.ReplaceAll(label, "-", " "), description)
}

func maxLabelSize(labels []string) int {
	maxLabelSize := 20
	for _, label := range labels {
		if len(label) > maxLabelSize {
			maxLabelSize = len(label)
		}
	}
	return maxLabelSize
}

func (m *Manager) dumpCommands(commands []string) string {
	sort.Strings(commands)
	var output string
	maxCmdSize := maxLabelSize(commands)
	for _, command := range commands {
		output += formatDescriptionLine(command, m.Commands[command].Info().Desc, maxCmdSize)
	}
	output += fmt.Sprintf("\nUse %s help <commandname> to get more information about a command.\n", m.name)
	return output
}

func (m *Manager) dumpTopics() string {
	topics := m.discoverTopics()
	sort.Strings(topics)
	maxTopicSize := maxLabelSize(topics)
	var output string
	for _, topic := range topics {
		output += formatDescriptionLine(topic, m.topics[topic], maxTopicSize)
	}
	output += fmt.Sprintf("\nUse %s help <topicname> to get more information about a topic.\n", m.name)
	return output
}

func (m *Manager) normalizeCommandArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	newArgs := append([]string{}, args...)
	for len(newArgs) > 0 {
		tryCmd := strings.Join(newArgs, "-")
		if _, ok := m.Commands[tryCmd]; ok {
			break
		}
		newArgs = newArgs[:len(newArgs)-1]
	}
	remainder := len(newArgs)
	if remainder > 0 {
		newArgs = []string{strings.Join(newArgs, "-")}
	}
	newArgs = append(newArgs, args[remainder:]...)
	return newArgs
}

func (m *Manager) discoverTopics() []string {
	freq := map[string]int{}
	for cmdName, cmd := range m.Commands {
		if _, isDeprecated := cmd.(*DeprecatedCommand); isDeprecated {
			continue
		}
		idx := strings.Index(cmdName, "-")
		if idx != -1 {
			freq[cmdName[:idx]] += 1
		}
	}
	for topic := range m.topics {
		freq[topic] = 999
	}
	var result []string
	for topic, count := range freq {
		if count > 1 {
			result = append(result, topic)
		}
	}
	sort.Strings(result)
	return result
}

// Cancelable are implemented by commands that support cancellation
type Cancelable interface {
	// Cancel handles the command cancellation and is required to be thread safe as
	// this method is called by a different goroutine.
	// Cancel should return an error if the operation is not cancelable yet/anymore or there
	// was any error during the cancellation.
	// Cancel may be called multiple times.
	Cancel(context Context) error
}

type Command interface {
	Info() *Info
	Run(context *Context) error
}

type FlaggedCommand interface {
	Command
	Flags() *pflag.FlagSet
}

type DeprecatedCommand struct {
	Command
	oldName string
}

func (c *DeprecatedCommand) Info() *Info {
	info := c.Command.Info()
	info.Usage = strings.Replace(info.Usage, info.Name, c.oldName, 1)
	info.Name = c.oldName
	info.V2.Hidden = true
	return info
}

func (c *DeprecatedCommand) Run(context *Context) error {
	fmt.Fprintf(context.Stderr, "WARNING: %q has been deprecated, please use %q instead.\n\n", c.oldName, c.Command.Info().Name)
	return c.Command.Run(context)
}

func (c *DeprecatedCommand) Flags() *pflag.FlagSet {
	if cmd, ok := c.Command.(FlaggedCommand); ok {
		return cmd.Flags()
	}
	return pflag.NewFlagSet("", pflag.ContinueOnError)
}

type ShorthandCommand struct {
	Command
	shorthand string
}

func (c *ShorthandCommand) Info() *Info {
	info := c.Command.Info()

	info.Usage = c.shorthand + stripUsage(info.Name, info.Usage)
	strings.Replace(info.Usage, info.Name, c.shorthand, 1)

	info.Name = c.shorthand
	info.V2.GroupID = "shorthands"
	info.V2.OnlyAppendOnRoot = true
	return info
}

func (c *ShorthandCommand) Run(context *Context) error {
	return c.Command.Run(context)
}

func (c *ShorthandCommand) Flags() *pflag.FlagSet {
	if cmd, ok := c.Command.(FlaggedCommand); ok {
		return cmd.Flags()
	}
	return pflag.NewFlagSet("", pflag.ContinueOnError)
}

type Context struct {
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

func (c *Context) RawOutput() {
	if pager, ok := c.Stdout.(*pagerWriter); ok {
		c.Stdout = pager.baseWriter
	}
	if sync, ok := c.Stdin.(*syncReader); ok {
		c.Stdin = sync.baseReader
	}
}

var ArbitraryArgs = -1

type Info struct {
	Name    string
	MinArgs int
	MaxArgs int
	Usage   string
	Desc    string
	V2      InfoV2
	fail    bool
}

type help struct {
	manager *Manager
}

func (c *help) Info() *Info {
	return &Info{Name: "help", Usage: "command [args]", V2: InfoV2{Disabled: true}}
}

func (c *help) Run(context *Context) error {
	const deprecatedMsg = "WARNING: %q is deprecated. Showing help for %q instead.\n\n"
	output := ""
	if c.manager.wrong {
		output += "ERROR: wrong number of arguments.\n\n"
	}
	if len(context.Args) > 0 {
		if cmd, ok := c.manager.Commands[context.Args[0]]; ok {
			if deprecated, ok := cmd.(*DeprecatedCommand); ok {
				fmt.Fprintf(context.Stderr, deprecatedMsg, deprecated.oldName, deprecated.Command.Info().Name)
			}
			info := cmd.Info()
			output += fmt.Sprintf("Usage: %s %s\n", c.manager.name, info.Usage)
			output += fmt.Sprintf("\n%s\n", info.Desc)
			flags := c.parseFlags(cmd)
			if flags != "" {
				output += fmt.Sprintf("\n%s", flags)
			}
			if info.MinArgs > 0 {
				output += fmt.Sprintf("\nMinimum # of arguments: %d", info.MinArgs)
			}
			if info.MaxArgs > 0 {
				output += fmt.Sprintf("\nMaximum # of arguments: %d", info.MaxArgs)
			}
			output += "\n"
		} else if msg, ok := c.manager.tryImplicitTopic(context.Args); ok {
			output += msg
		} else {
			return errors.Errorf("command %q does not exist.", context.Args[0])
		}
	} else {
		output += fmt.Sprintf("Usage: %s %s\n\nAvailable commands:\n", c.manager.name, c.Info().Usage)
		var commands []string
		for name, cmd := range c.manager.Commands {
			if _, ok := cmd.(*DeprecatedCommand); !ok {
				commands = append(commands, name)
			}
		}
		output += c.manager.dumpCommands(commands)
		if len(c.manager.topics) > 0 {
			output += fmt.Sprintln("\nAvailable topics:")
			output += c.manager.dumpTopics()
		}
	}
	io.WriteString(context.Stdout, output)
	return nil
}

var flagFormatRegexp = regexp.MustCompile(`(?m)^([^-\s])`)

func (c *help) parseFlags(command Command) string {
	var output string
	if cmd, ok := command.(FlaggedCommand); ok {
		var buf bytes.Buffer
		flagset := cmd.Flags()
		flagset.SetOutput(&buf)
		flagset.PrintDefaults()
		if buf.String() != "" {
			output = flagFormatRegexp.ReplaceAllString(buf.String(), `    $1`)
			output = fmt.Sprintf("Flags:\n\n%s", output)
		}
	}
	return strings.ReplaceAll(output, "\n", "\n  ")
}

func ExtractProgramName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

var (
	fsystem   fs.Fs
	fsystemMu sync.Mutex
)

func filesystem() fs.Fs {
	fsystemMu.Lock()
	defer fsystemMu.Unlock()
	if fsystem == nil {
		fsystem = fs.OsFs{}
	}
	return fsystem
}

// validateVersion checks whether current version is greater or equal to
// supported version.
func validateVersion(supported, current string) bool {
	if current == "dev" {
		return true
	}
	if supported == "" {
		return true
	}
	vSupported, err := goVersion.NewVersion(supported)
	if err != nil {
		return false
	}
	vCurrent, err := goVersion.NewVersion(current)
	if err != nil {
		return false
	}
	return vCurrent.Compare(vSupported) >= 0
}

func (m *Manager) SetExiter(e exiter) {
	m.e = e
}
