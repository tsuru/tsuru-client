// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"context"
	"net/http"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
	"github.com/tsuru/tsuru-client/v2/internal/types"
	"github.com/tsuru/tsuru-client/v2/pkg/printer"
)

func newAppListCmd(tsuruCtx *tsuructx.TsuruContext) *cobra.Command {
	appListCmd := &cobra.Command{
		Use:   "list",
		Short: "list apps",
		Long: `Lists all apps that you have access to. App access is controlled by teams.
If your team has access to an app, then you have access to it.
Flags can be used to filter the list of applications.`,
		Example: `$ tsuru app list
$ tsuru app list -n my
$ tsuru app list --status error`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return appListCmdRun(tsuruCtx, cmd, args)
		},
		Args: cobra.ExactArgs(0),
	}

	appListCmd.Flags().StringP("name", "n", "", "filter applications by name")
	appListCmd.Flags().StringP("pool", "o", "", "filter applications by pool")
	appListCmd.Flags().StringP("status", "s", "", "filter applications by unit status. Accepts multiple values separated by commas. Possible values can be: building, created, starting, error, started, stopped, asleep")
	appListCmd.Flags().StringP("platform", "p", "", "filter applications by platform")
	appListCmd.Flags().StringP("team", "t", "", "filter applications by team owner")
	appListCmd.Flags().StringP("user", "u", "", "filter applications by owner")
	appListCmd.Flags().BoolP("locked", "l", false, "filter applications by lock status")
	appListCmd.Flags().BoolP("simplified", "q", false, "display only applications name")
	// appListCmd.Flags().Bool("json", false, "display applications in JSON format")
	appListCmd.Flags().StringSliceP("tag", "g", []string{}, "filter applications by tag. Can be used multiple times")

	return appListCmd
}

func appListCmdRun(tsuruCtx *tsuructx.TsuruContext, cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	apps, httpRes, err := tsuruCtx.Client().AppApi.AppList(context.Background(), &tsuru.AppListOpts{
		Name: optional.NewString(cmd.Flag("name").Value.String()),
		// Pool:       optional.NewString(cmd.Flag("pool").Value.String()),
		// Locked:     optional.NewBool(cmd.Flag("locked").Value.String() == "true"),
		// Owner:      optional.NewString(cmd.Flag("user").Value.String()),
		// Platform:   optional.NewString(cmd.Flag("platform").Value.String()),
		// Status:     optional.NewString(cmd.Flag("status").Value.String()),
		// Tag:        optional.NewInterface(cmd.Flag("tag").Value.String()),
		// TeamOwner:  optional.NewString(cmd.Flag("team").Value.String()),
		Simplified: optional.NewBool(true),
	})
	if err != nil {
		if httpRes != nil && httpRes.StatusCode == http.StatusNoContent {
			return printer.Print(tsuruCtx.Stdout, []tsuru.MiniApp{}, tsuruCtx.OutputFormat())
		}
		return err
	}

	if tsuruCtx.OutputAPIData() {
		return printer.Print(tsuruCtx.Stdout, apps, tsuruCtx.OutputFormat())
	}
	return printer.Print(tsuruCtx.Stdout, types.AppList(apps), tsuruCtx.OutputFormat())
}
