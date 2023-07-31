// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
	"github.com/tsuru/tsuru-client/v2/internal/wrappers"
)

func NewAppCmd(tsuruCtx *tsuructx.TsuruContext) *cobra.Command {
	appCmd := &cobra.Command{
		Use:   "app",
		Short: "app is a runnable application running on Tsuru",
	}
	appCmd.AddCommand(newAppInfoCmd(tsuruCtx))
	return appCmd
}

func completeAppNames(tsuruCtx *tsuructx.TsuruContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_ = wrappers.ForceCallPreRun(cmd, args)
	target := tsuruCtx.TargetURL()
	token := tsuruCtx.Token()
	tokenFs := tsuruCtx.TokenSetFromFS
	_, _, _ = target, token, tokenFs
	apps, _, err := tsuruCtx.Client().AppApi.AppList(cmd.Context(), &tsuru.AppListOpts{
		Simplified: optional.NewBool(true),
		Name:       optional.NewString(toComplete),
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, app := range apps {
		names = append(names, app.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
