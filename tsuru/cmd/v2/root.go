package v2

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tsuru/go-tsuruclient/pkg/config"
)

type TsuruContext struct {
	// Viper is an instance of the viper.Viper configuration
	Viper *viper.Viper
}

var tsuruCtx = &TsuruContext{
	Viper: preSetupViper(nil),
}

func (tc *TsuruContext) SetOutputFormat(value string) {
	tc.Viper.Set("format", value)
}

func Enabled() bool {
	return tsuruCtx.Viper.GetString("version") == "v2"
}

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "tsuru",
		Short: "A command-line interface for interacting with tsuru",

		PersistentPreRun: rootPersistentPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRootCmd(cmd, args)
		},
		Args: cobra.MinimumNArgs(0),

		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		DisableFlagParsing: true,
	}

	rootCmd.SetVersionTemplate(`{{printf "tsuru-client version: %s" .Version}}` + "\n")

	setupPFlagsAndCommands(rootCmd)

	return rootCmd
}
func setupPFlagsAndCommands(rootCmd *cobra.Command) {
	// Persistent Flags.
	// !!! Double bind them inside PersistentPreRun() !!!
	rootCmd.PersistentFlags().String("target", "", "Tsuru server endpoint")
	tsuruCtx.Viper.BindPFlag("target", rootCmd.PersistentFlags().Lookup("target"))

	rootCmd.PersistentFlags().IntP("verbosity", "v", 0, "Verbosity level: 1 => print HTTP requests; 2 => print HTTP requests/responses")
	tsuruCtx.Viper.BindPFlag("verbosity", rootCmd.PersistentFlags().Lookup("verbosity"))

	rootCmd.PersistentFlags().Bool("json", false, "Output format as json")
	rootCmd.PersistentFlags().MarkHidden("json")

	tsuruCtx.Viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))

	tsuruCtx.Viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
	rootCmd.PersistentFlags().Bool("api-data", false, "Output API response data instead of a parsed data (more useful with --format=json)")
	tsuruCtx.Viper.BindPFlag("api-data", rootCmd.PersistentFlags().Lookup("api-data"))

	// Search config in home directory with name ".tsuru-client" (without extension).
	tsuruCtx.Viper.AddConfigPath(config.JoinWithUserDir(".tsuru"))
	tsuruCtx.Viper.SetConfigType("yaml")
	tsuruCtx.Viper.SetConfigName("client")

	// If a config file is found, read it in.
	if err := tsuruCtx.Viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", tsuruCtx.Viper.ConfigFileUsed()) // TODO: handle this better
	}
}

func rootPersistentPreRun(cmd *cobra.Command, args []string) {
	if l := cmd.Flags().Lookup("target"); l != nil && l.Value.String() != "" {
		target := l.Value.String()
		os.Setenv("TSURU_TARGET", target)
	}

	if v, err := cmd.Flags().GetInt("verbosity"); err != nil {
		os.Setenv("TSURU_VERBOSITY", strconv.Itoa(v))
	}

	if v, err := cmd.Flags().GetBool("json"); err == nil && v {
		tsuruCtx.SetOutputFormat("json")
	} else {
		tsuruCtx.SetOutputFormat("string")
	}

}

func runRootCmd(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	args = parseFirstFlagsOnly(cmd, args)

	versionVal, _ := cmd.Flags().GetBool("version")
	helpVal, _ := cmd.Flags().GetBool("help")
	if len(args) == 0 || versionVal || helpVal {
		cmd.RunE = nil
		cmd.Run = nil
		return cmd.Execute()
	}

	return runTsuruPlugin(args)
}

// parseFirstFlagsOnly handles only the first flags with cmd.ParseFlags()
// before a non-flag element
func parseFirstFlagsOnly(cmd *cobra.Command, args []string) []string {
	if cmd == nil {
		return args
	}
	cmd.DisableFlagParsing = false
	for len(args) > 0 {
		s := args[0]
		if len(s) == 0 || s[0] != '-' || len(s) == 1 {
			return args // any non-flag means we're done
		}
		args = args[1:]

		flagName := s[1:]
		if s[1] == '-' {
			if len(s) == 2 { // "--" terminates the flags
				return args
			}
			flagName = s[2:]
		}

		if strings.Contains(flagName, "=") {
			flagArgPair := strings.SplitN(flagName, "=", 2)
			flagName = flagArgPair[0]
			args = append([]string{flagArgPair[1]}, args...)
		}

		flag := cmd.Flags().Lookup(flagName)
		if flag == nil && len(flagName) == 1 {
			flag = cmd.Flags().ShorthandLookup(flagName)
		}

		if flag != nil && flag.Value.Type() == "bool" {
			cmd.ParseFlags([]string{s})
		} else {
			if len(args) == 0 {
				return args
			}
			cmd.ParseFlags([]string{s, args[0]})
			args = args[1:]
		}
	}
	return args
}

// preSetupViper prepares viper for being used by NewProductionTsuruContext()
func preSetupViper(vip *viper.Viper) *viper.Viper {
	if vip == nil {
		vip = viper.New()
	}
	vip.SetEnvPrefix("tsuru")
	vip.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	vip.AutomaticEnv() // read in environment variables that match
	return vip
}

func runTsuruPlugin(args []string) error {
	return errors.New("TODO: not implemented yet")
}
