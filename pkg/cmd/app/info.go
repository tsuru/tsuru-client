// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
	"github.com/tsuru/tsuru-client/v2/internal/types"
	"github.com/tsuru/tsuru-client/v2/pkg/printer"
)

func newAppInfoCmd(tsuruCtx *tsuructx.TsuruContext) *cobra.Command {
	appInfoCmd := &cobra.Command{
		Use:   "info [APP]",
		Short: "shows information about a specific app",
		Long: `shows information about a specific app.
Its name, platform, state (and its units), address, etc.
You need to be a member of a team that has access to the app to be able to see information about it.
`,
		Example: `$ tsuru app info myapp
$ tsuru app info -a myapp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return printAppInfo(tsuruCtx, cmd, args)
		},
		Args: cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeAppNames(tsuruCtx, cmd, args, toComplete)
		},
	}

	appInfoCmd.Flags().StringP("app", "a", "", "The name of the app (may be passed as argument)")
	appInfoCmd.Flags().BoolP("simplified", "s", false, "Show simplified view of app")
	return appInfoCmd
}

func printAppInfo(tsuruCtx *tsuructx.TsuruContext, cmd *cobra.Command, args []string) error {
	if len(args) == 0 && cmd.Flag("app").Value.String() == "" {
		return fmt.Errorf("no app was provided. Please provide an app name or use the --app flag")
	}
	if len(args) > 0 && cmd.Flag("app").Value.String() != "" {
		return fmt.Errorf("either pass an app name as an argument or use the --app flag, not both")
	}
	cmd.SilenceUsage = true

	appName := cmd.Flag("app").Value.String()
	if appName == "" {
		appName = args[0]
	}

	a, _, err := tsuruCtx.Client().AppApi.AppGet(context.Background(), appName)
	if err != nil {
		return err
	}

	if tsuruCtx.OutputAPIData() {
		return printer.Print(tsuruCtx.Stdout, a, tsuruCtx.OutputFormat())
	}
	return printer.Print(tsuruCtx.Stdout, types.AppInfoSimple(a), tsuruCtx.OutputFormat())
}
