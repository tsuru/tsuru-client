// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tsuru/tsuru-client/v2/internal/config"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

func NewLogoutCmd(tsuruCtx *tsuructx.TsuruContext) *cobra.Command {
	loginCmd := &cobra.Command{
		Use:   "logout",
		Short: "logout will terminate the session with the tsuru server",
		Long: `logout will terminate the session with the tsuru server
and cleanup the token from the local machine.
`,
		Example: `$ tsuru logout`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return logoutCmdRun(tsuruCtx, cmd, args)
		},
		Args: cobra.ExactArgs(0),
	}

	return loginCmd
}

func logoutCmdRun(tsuruCtx *tsuructx.TsuruContext, cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	errs := []error{}
	if tsuruCtx.Token() != "" {
		func() {
			request, err := tsuruCtx.NewRequest("DELETE", "/users/tokens", nil)
			if err != nil {
				errs = append(errs, err)
				return
			}
			httpResponse, err := tsuruCtx.RawHTTPClient().Do(request)
			if err != nil {
				errs = append(errs, err)
				return
			}
			if httpResponse.StatusCode != 200 {
				errs = append(errs, fmt.Errorf("unexpected response from server: %d: %s", httpResponse.StatusCode, httpResponse.Status))
			}
			defer httpResponse.Body.Close()
		}()
	}

	if err := config.RemoveCurrentTokensFromFs(tsuruCtx.Fs); err != nil {
		errs = append(errs, err)
		return errors.Join(errs...)
	}

	if len(errs) == 0 {
		fmt.Fprintln(tsuruCtx.Stdout, "Successfully logged out!")
	} else {
		fmt.Fprintln(tsuruCtx.Stdout, "Logged out, but some errors occurred:")
	}
	return errors.Join(errs...)

}
