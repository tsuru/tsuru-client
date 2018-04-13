// Copyright 2018 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"fmt"
	"time"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
)

type TokenCreateCmd struct {
	fs      *gnuflag.FlagSet
	args    tsuru.TeamTokenCreateArgs
	expires time.Duration
}

func (c *TokenCreateCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "token-create",
		Usage:   "token-create [--id/-i token-id] [--team/-t team] [--description/-d description] [--expires/-e expiration-in-seconds]",
		Desc:    `Creates a new API token associated to a team.`,
		MinArgs: 0,
	}
}

func (c *TokenCreateCmd) Run(context *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	c.args.ExpiresIn = int64(c.expires / time.Second)
	token, _, err := apiClient.AuthApi.TeamTokenCreate(nil, c.args)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Token %q created: %s", token.TokenId, token.Token)
	return nil
}

func (c *TokenCreateCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)

		description := "A description on how the token will be used."
		c.fs.StringVar(&c.args.Description, "description", "", description)
		c.fs.StringVar(&c.args.Description, "d", "", description)

		tokenID := "A unique identifier for the token being created."
		c.fs.StringVar(&c.args.TokenId, "id", "", tokenID)
		c.fs.StringVar(&c.args.TokenId, "i", "", tokenID)

		team := "The team name responsible for this token."
		c.fs.StringVar(&c.args.Team, "team", "", team)
		c.fs.StringVar(&c.args.Team, "t", "", team)

		expiration := "The expiration for the token being created. 0 or unset means it never expires."
		c.fs.DurationVar(&c.expires, "expires", 0, expiration)
		c.fs.DurationVar(&c.expires, "e", 0, expiration)
	}
	return c.fs
}
