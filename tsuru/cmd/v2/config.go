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
	tablecli.TableConfig.UseUTF8Borders = isModernTerminal()

	key := "table-utf8"
	if vip.IsSet(key) {
		tablecli.TableConfig.UseUTF8Borders = vip.GetBool(key)
	}

	if !colorDisabled(vip) {
		var fgColor color.Attribute
		if isDarkBackground() {
			fgColor = color.FgHiBlack
		}
		key := "table-color"
		if vip.IsSet(key) {
			fgColor = colorMap[strings.ToLower(vip.GetString(key))]
		}

		if fgColor != 0 {
			tablecli.TableConfig.BorderColorFunc = func(s string) string {
				return color.New(fgColor).Sprint(s)
			}
		}
	}

	// setup colors
	color.NoColor = colorDisabled(vip)

	// padding
	key = "tab-writer-padding"
	if vip.IsSet(key) {
		standards.SubTableWriterPadding = vip.GetInt(key)
	}

	return vip
}

var colorMap = map[string]color.Attribute{
	"black":      color.FgBlack,
	"red":        color.FgRed,
	"green":      color.FgGreen,
	"yellow":     color.FgYellow,
	"blue":       color.FgBlue,
	"magenta":    color.FgMagenta,
	"cyan":       color.FgCyan,
	"white":      color.FgWhite,
	"hi-black":   color.FgHiBlack,
	"hi-red":     color.FgHiRed,
	"hi-green":   color.FgHiGreen,
	"hi-yellow":  color.FgHiYellow,
	"hi-blue":    color.FgHiBlue,
	"hi-magenta": color.FgHiMagenta,
	"hi-cyan":    color.FgHiCyan,
	"hi-white":   color.FgHiWhite,
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
