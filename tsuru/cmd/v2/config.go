// Copyright 2026 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v2

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/viper"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	"golang.org/x/term"
)

var defaultViper = preSetupViper(nil)

func ColorDisabled() bool {
	return colorDisabled(defaultViper)
}

func colorDisabled(vip *viper.Viper) bool {
	// https://no-color.org/
	if _, nocolor := os.LookupEnv("NO_COLOR"); nocolor {
		return true
	}

	// On Windows WT_SESSION is set by the modern terminal component.
	// Older terminals have poor support for UTF-8, VT escape codes, etc.
	if runtime.GOOS == "windows" && os.Getenv("WT_SESSION") == "" {
		return true
	}

	// https://en.wikipedia.org/wiki/Computer_terminal#Dumb_terminals
	if os.Getenv("TERM") == "dumb" {
		return true
	}

	return vip.GetBool("disable-colors")
}

func Pager() (pager string, found bool) {
	key := "pager"
	if !defaultViper.IsSet(key) {
		return "", false
	}
	return defaultViper.GetString(key), true
}

func ColorStream() bool {
	if ColorDisabled() {
		return false
	}

	def := term.IsTerminal(int(os.Stdout.Fd()))

	key := "color-stream"
	if defaultViper.IsSet(key) {
		return defaultViper.GetBool(key)
	}
	return def
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

	// setup table writer
	tablecli.TableConfig.UseTabWriter = vip.GetBool("tab-writer")
	tablecli.TableConfig.BreakOnAny = vip.GetBool("break-any")
	tablecli.TableConfig.ForceWrap = vip.GetBool("force-wrap")
	tablecli.TableConfig.TabWriterTruncate = vip.GetBool("tab-writer-truncate")

	// setup colors
	color.NoColor = colorDisabled(vip)

	// padding
	key := "tab-writer-padding"
	if vip.IsSet(key) {
		standards.SubTableWriterPadding = vip.GetInt(key)
	}

	return vip
}
