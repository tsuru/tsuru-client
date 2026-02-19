// Copyright 2026 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v2

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
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

	if !IsModernTerminal {
		return true
	}

	return vip.GetBool("disable-colors")
}

var IsModernTerminal = isModernTerminal()

func isModernTerminal() bool {
	// On Windows WT_SESSION is set by the modern terminal component.
	// Older terminals have poor support for UTF-8, VT escape codes, etc.
	if runtime.GOOS == "windows" {
		return os.Getenv("WT_SESSION") != ""
	}

	// https://en.wikipedia.org/wiki/Computer_terminal#Dumb_terminals
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	return term.IsTerminal(int(os.Stdout.Fd()))
}

func Pager() (pager string, found bool) {
	key := "pager"
	if !defaultViper.IsSet(key) {
		return "", false
	}

	value := defaultViper.Get(key)

	// assumes the default behavior of using the default pager when the value is "true"
	if value == true || value == "true" {
		return "", false
	}

	if value == false || value == "false" {
		return "", true
	}

	if str, ok := value.(string); ok {
		return str, true
	}
	return "", false
}

func ColorStream() bool {
	if ColorDisabled() {
		return false
	}

	def := IsModernTerminal

	key := "color-stream"
	if defaultViper.IsSet(key) {
		return defaultViper.GetBool(key)
	}
	return def
}

var TsuruConfigDir = config.JoinWithUserDir(".tsuru")

// preSetupViper prepares viper for being used by NewProductionTsuruContext()
func preSetupViper(vip *viper.Viper) *viper.Viper {
	if vip == nil {
		vip = viper.New()
	}
	vip.SetEnvPrefix("tsuru")
	vip.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	vip.AutomaticEnv() // read in environment variables that match

	vip.AddConfigPath(TsuruConfigDir)
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
	tablecli.TableConfig.UseUTF8Borders = tableUTF8(vip)

	if !colorDisabled(vip) {
		tblColorString := tableColor(vip)

		if tblColorString != "" {
			tablecli.SetBorderColorByString(tblColorString)
		}
	}

	// setup colors
	color.NoColor = colorDisabled(vip)

	// padding
	key := "tab-writer-padding"
	if vip.IsSet(key) {
		standards.SubTableWriterPadding = vip.GetInt(key)
	}

	return vip
}

func TableUTF8() bool {
	return tableUTF8(defaultViper)
}

func TableColor() string {
	return tableColor(defaultViper)
}

func tableColor(vip *viper.Viper) string {
	color := ""

	if isDarkBackground() && tableUTF8(vip) {
		color = "hi-black"
	}

	key := "table-color"
	if vip.IsSet(key) {
		color = strings.ToLower(vip.GetString(key))
	}

	return color
}

func tableUTF8(vip *viper.Viper) bool {
	key := "table-utf8"
	if vip.IsSet(key) {
		return vip.GetBool(key)
	}

	return isModernTerminal()
}

func isDarkBackground() bool {
	colorfgbg := os.Getenv("COLORFGBG")
	if colorfgbg == "" {
		return true
	}
	parts := strings.Split(colorfgbg, ";")
	bg, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return true
	}
	return bg < 8
}
