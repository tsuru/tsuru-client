// Copyright 2026 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v2

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tablecli"
)

var defaultViper = preSetupViper(nil)

func ColorDisabled() bool {
	return defaultViper.GetBool("disable-colors")
}

func Pager() (pager string, found bool) {
	key := "pager"
	if !defaultViper.IsSet(key) {
		return "", false
	}
	return defaultViper.GetString("pager"), true
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

	return vip
}
