// Copyright 2018 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/tsuru/tablecli"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
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
		Usage:   "token-create [--id/-i token-id] [--team/-t team] [--description/-d description] [--expires/-e expiration]",
		Desc:    `Creates a new API token associated to a team.`,
		MinArgs: 0,
	}
}

func (c *TokenCreateCmd) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	c.args.ExpiresIn = int64(c.expires / time.Second)
	token, _, err := apiClient.AuthApi.TeamTokenCreate(context.TODO(), c.args)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Token %q created: %s\n", token.TokenId, token.Token)
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

		expiration := "The expiration for the token being created. A duration suffix is mandatory (s for seconds, m for minutes, h for hours, ...). 0 or unset means it never expires."
		c.fs.DurationVar(&c.expires, "expires", 0, expiration)
		c.fs.DurationVar(&c.expires, "e", 0, expiration)
	}
	return c.fs
}

type TokenUpdateCmd struct {
	fs      *gnuflag.FlagSet
	args    tsuru.TeamTokenUpdateArgs
	expires time.Duration
}

func (c *TokenUpdateCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "token-update",
		Usage:   "token-update <token-id> [--regenerate] [--description/-d description] [--expires/-e expiration]",
		Desc:    `Creates a new API token associated to a team.`,
		MinArgs: 1,
	}
}

func (c *TokenUpdateCmd) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	c.args.ExpiresIn = int64(c.expires / time.Second)
	token, _, err := apiClient.AuthApi.TeamTokenUpdate(context.TODO(), ctx.Args[0], c.args)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Token %q updated: %s\n", token.TokenId, token.Token)
	return nil
}

func (c *TokenUpdateCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)

		description := "A description on how the token will be used."
		c.fs.StringVar(&c.args.Description, "description", "", description)
		c.fs.StringVar(&c.args.Description, "d", "", description)

		regenerate := "Setting regenerate will change de value of the token, invalidating the previous value."
		c.fs.BoolVar(&c.args.Regenerate, "regenerate", false, regenerate)

		expiration := "The expiration for the token being updated. A duration suffix is mandatory ('s' for seconds, 'm' for minutes, 'h' for hours, ...). Setting to 0 or unset means the previous value will be used. Setting to a negative value will remove any existing expiration."
		c.fs.DurationVar(&c.expires, "expires", 0, expiration)
		c.fs.DurationVar(&c.expires, "e", 0, expiration)
	}
	return c.fs
}

type TokenListCmd struct {
}

func (c *TokenListCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "token-list",
		Usage:   "token-list",
		Desc:    `List existing API tokens associated with a team.`,
		MinArgs: 0,
	}
}

func (c *TokenListCmd) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	tokens, rsp, err := apiClient.AuthApi.TeamTokensList(context.TODO())
	if err != nil {
		if rsp != nil && rsp.StatusCode == http.StatusNoContent {
			return nil
		}
		return err
	}
	table := tablecli.Table{
		Headers:       tablecli.Row{"Token ID", "Team", "Timestamps", "Roles"},
		LineSeparator: true,
	}
	for _, t := range tokens {
		if t.Token == "" {
			t.Token = "Not authorized"
		}
		table.AddRow(tablecli.Row{
			t.TokenId,
			t.Team,
			fmt.Sprintf(" Created At: %s\n Expires At: %s\nLast Access: %s",
				formatter.FormatDate(t.CreatedAt),
				formatter.FormatDate(t.ExpiresAt),
				formatter.FormatDate(t.LastAccess),
			),
			formatRoles(t.Roles),
		})
	}
	table.Sort()
	fmt.Fprint(ctx.Stdout, table.String())
	return nil
}

func formatRoles(roles []tsuru.RoleInstance) string {
	rolesStr := make([]string, len(roles))
	for i, r := range roles {
		rolesStr[i] = fmt.Sprintf("%s(%s)", r.Name, r.Contextvalue)
	}
	sort.Strings(rolesStr)
	return strings.Join(rolesStr, "\n")
}

type TokenDeleteCmd struct {
}

func (c *TokenDeleteCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "token-delete",
		Usage:   "token-delete <token id>",
		Desc:    `Delete an existing token.`,
		MinArgs: 1,
	}
}

func (c *TokenDeleteCmd) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	_, err = apiClient.AuthApi.TeamTokenDelete(context.TODO(), ctx.Args[0])
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Token successfully deleted.")
	return nil
}

type TokenInfoCmd struct {
}

func (c *TokenInfoCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "token-info",
		Usage:   "token-info <token id>",
		Desc:    `Shows information about a specific token.`,
		MinArgs: 1,
	}
}

func (c *TokenInfoCmd) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}

	tokenID := ctx.Args[0]
	token, rsp, err := apiClient.AuthApi.TeamTokenInfo(context.TODO(), tokenID)
	if err != nil {
		if rsp != nil && rsp.StatusCode == http.StatusNoContent {
			return nil
		}
		return err
	}

	if tokenID != "" {
		fmt.Fprintf(ctx.Stdout, "Token: %s\nToken Id: %s\nDescription: %s\nCreated at: %s\nExpires at: %s\nLast Acess: %s\nCreator: %s\nTeam: %s\nRoles: %s\n",
			token.Token,
			token.TokenId,
			token.Description,
			formatter.FormatDate(token.CreatedAt),
			formatter.FormatDate(token.ExpiresAt),
			formatter.FormatDate(token.LastAccess),
			token.CreatorEmail,
			token.Team,
			formatRoles(token.Roles),
		)
		return nil
	}
	return fmt.Errorf("Token not found")
}
