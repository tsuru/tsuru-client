// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"fmt"
	"net/http"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
)

type Logout struct{}

func (c *Logout) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "logout",
		Usage: "logout",
		Desc:  "Logout will terminate the session with the tsuru server.",
	}
}

func (c *Logout) Run(context *cmd.Context) error {
	if url, err := config.GetURL("/users/tokens"); err == nil {
		request, _ := http.NewRequest("DELETE", url, nil)
		tsuruHTTP.AuthenticatedClient.Do(request)
	}

	err := config.RemoveTokenV1()
	if err != nil {
		return err
	}
	err = config.RemoveTokenV2()
	if err != nil {
		return err
	}

	fmt.Fprintln(context.Stdout, "Successfully logged out!")
	return nil
}
