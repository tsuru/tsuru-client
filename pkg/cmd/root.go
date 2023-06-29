// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tsuru/tsuru-client/v2/internal/config"
	"github.com/tsuru/tsuru-client/v2/internal/exec"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

var (
	version  cmdVersion
	commands = []func(*tsuructx.TsuruContext) *cobra.Command{}
)

type cmdVersion struct {
	Version string
	Commit  string
	Date    string
}

func (v *cmdVersion) String() string {
	if v.Version == "" {
		v.Version = "dev"
	}
	if v.Commit == "" && v.Date == "" {
		return v.Version
	}
	return fmt.Sprintf("%s (%s - %s)", v.Version, v.Commit, v.Date)
}

// Execute will create the cli with all subcommands and run it
func Execute(_version, _commit, _dateStr string) {
	version = cmdVersion{_version, _commit, _dateStr}
	rootCmd := NewRootCmd(viper.GetViper(), nil)
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func NewRootCmd(vip *viper.Viper, tsuruCtx *tsuructx.TsuruContext) *cobra.Command {
	vip = preSetupViper(vip)
	if tsuruCtx == nil {
		tsuruCtx = NewProductionTsuruContext(vip, afero.NewOsFs())
	}
	rootCmd := newBareRootCmd(tsuruCtx)
	setupPFlagsAndCommands(rootCmd, tsuruCtx)
	return rootCmd
}

func newBareRootCmd(tsuruCtx *tsuructx.TsuruContext) *cobra.Command {
	rootCmd := &cobra.Command{
		Version: version.String(),
		Use:     "tsuru",
		Short:   "A command-line interface for interacting with tsuru",

		PersistentPreRun: rootPersistentPreRun(tsuruCtx),
	}

	rootCmd.SetVersionTemplate(`{{printf "tsuru-client version: %s" .Version}}` + "\n")
	rootCmd.SetIn(tsuruCtx.Stdin)
	rootCmd.SetOut(tsuruCtx.Stdout)
	rootCmd.SetErr(tsuruCtx.Stderr)

	return rootCmd
}

func rootPersistentPreRun(tsuruCtx *tsuructx.TsuruContext) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if l := cmd.Flags().Lookup("target"); l != nil && l.Value.String() != "" {
			tsuruCtx.SetTargetURL(l.Value.String())
		}
		if v, err := cmd.Flags().GetInt("verbosity"); err != nil {
			tsuruCtx.SetVerbosity(v)
		}
	}
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

// setupPFlagsAndCommands reads in config file and ENV variables if set.
func setupPFlagsAndCommands(rootCmd *cobra.Command, tsuruCtx *tsuructx.TsuruContext) {
	// Persistent Flags.
	// !!! Double bind them inside PersistentPreRun() !!!
	rootCmd.PersistentFlags().String("target", "", "Tsuru server endpoint")
	tsuruCtx.Viper.BindPFlag("target", rootCmd.PersistentFlags().Lookup("target"))
	rootCmd.PersistentFlags().IntP("verbosity", "v", 0, "Verbosity level: 1 => print HTTP requests; 2 => print HTTP requests/responses")
	tsuruCtx.Viper.BindPFlag("verbosity", rootCmd.PersistentFlags().Lookup("verbosity"))

	// Search config in home directory with name ".tsuru-client" (without extension).
	tsuruCtx.Viper.AddConfigPath(config.ConfigPath)
	tsuruCtx.Viper.SetConfigType("yaml")
	tsuruCtx.Viper.SetConfigName(".tsuru-client")

	// If a config file is found, read it in.
	if err := tsuruCtx.Viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", tsuruCtx.Viper.ConfigFileUsed()) // TODO: handle this better
	}

	// Add subcommands
	for _, cmd := range commands {
		rootCmd.AddCommand(cmd(tsuruCtx))
	}

	v1LegacyCmdManager := newV1LegacyCmdManager()
	addMissingV1LegacyCommands(rootCmd, v1LegacyCmdManager)
	rootCmd.AddCommand(newLegacyCommand(v1LegacyCmdManager))
}

func NewProductionTsuruContext(vip *viper.Viper, fs afero.Fs) *tsuructx.TsuruContext {
	var err error
	var tokenSetFromFS bool

	// Get target
	target := vip.GetString("target")
	if target == "" {
		target, err = config.GetCurrentTargetFromFs(fs)
		cobra.CheckErr(err)
	}
	target, err = config.GetTargetURL(fs, target)
	cobra.CheckErr(err)
	vip.Set("target", target)

	// Get token
	token := vip.GetString("token")
	if token == "" {
		token, err = config.GetTokenFromFs(fs)
		cobra.CheckErr(err)
		tokenSetFromFS = true
		vip.Set("token", token)
	}

	tsuruCtx := tsuructx.TsuruContextWithConfig(productionOpts(fs, vip))
	tsuruCtx.TokenSetFromFS = tokenSetFromFS
	return tsuruCtx
}

func productionOpts(fs afero.Fs, vip *viper.Viper) *tsuructx.TsuruContextOpts {
	return &tsuructx.TsuruContextOpts{
		InsecureSkipVerify: vip.GetBool("insecure-skip-verify"),
		LocalTZ:            time.Local,
		AuthScheme:         vip.GetString("auth-scheme"),
		Executor:           &exec.OsExec{},
		Fs:                 fs,
		Viper:              vip,

		UserAgent: "tsuru-client:" + version.Version,

		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}
}