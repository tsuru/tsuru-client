// Copyright 2026 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v2

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tsuru/go-tsuruclient/pkg/config"
)

var defaultViper = preSetupViper(nil)

func Enabled() bool {
	return defaultViper.GetString("version") == "v2"
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

	setupPFlagsAndCommands(rootCmd)

	return rootCmd
}
func setupPFlagsAndCommands(rootCmd *cobra.Command) {
	// Persistent Flags.
	// !!! Double bind them inside PersistentPreRun() !!!
	rootCmd.PersistentFlags().String("target", "", "Tsuru server endpoint")
	defaultViper.BindPFlag("target", rootCmd.PersistentFlags().Lookup("target"))

	rootCmd.PersistentFlags().Int("verbosity", 0, "Verbosity level: 1 => print HTTP requests; 2 => print HTTP requests/responses")
	defaultViper.BindPFlag("verbosity", rootCmd.PersistentFlags().Lookup("verbosity"))
}

func rootPersistentPreRun(cmd *cobra.Command, args []string) {
	if l := cmd.Flags().Lookup("target"); l != nil && l.Value.String() != "" {
		target := l.Value.String()
		os.Setenv("TSURU_TARGET", target)
	}

	if v, err := cmd.Flags().GetInt("verbosity"); v > 0 && err == nil {
		os.Setenv("TSURU_VERBOSITY", strconv.Itoa(v))
	}
}

func runRootCmd(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	args = ParseFirstFlagsOnly(cmd, args)

	versionVal, _ := cmd.Flags().GetBool("version")
	helpVal, _ := cmd.Flags().GetBool("help")
	if len(args) == 0 || versionVal || helpVal {
		cmd.RunE = nil
		cmd.Run = nil
		return cmd.Execute()
	}

	return nil
}

// parseFirstFlagsOnly handles only the first flags with cmd.ParseFlags()
// before a non-flag element
func ParseFirstFlagsOnly(cmd *cobra.Command, args []string) []string {
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

	vip.AddConfigPath(config.JoinWithUserDir(".tsuru"))
	vip.SetConfigType("yaml")
	vip.SetConfigName("client")

	// If a config file is found, read it in.
	err := vip.ReadInConfig()
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error Using config file:", err)
		}
	}

	return vip
}
