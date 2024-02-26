// Copyright 2023 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/config"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	authTypes "github.com/tsuru/tsuru/types/auth"
)

var errTsuruTokenDefined = errors.New("this command can't run with $TSURU_TOKEN environment variable set. Did you forget to unset?")

type Login struct {
	fs *gnuflag.FlagSet

	scheme string
}

func (c *Login) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "login",
		Usage: "login [email]",
		Desc: `Initiates a new tsuru session for a user. If using tsuru native authentication
		scheme, it will ask for the email and the password and check if the user is
		successfully authenticated. If using OAuth, it will open a web browser for the
		user to complete the login.
		
		After that, the token generated by the tsuru server will be stored in
		[[${HOME}/.tsuru/token]].
		
		All tsuru actions require the user to be authenticated (except [[tsuru login]]
		and [[tsuru version]]).`,
		MinArgs: 0,
	}
}

func (c *Login) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("login", gnuflag.ExitOnError)
		desc := `Login with specific auth scheme`
		c.fs.StringVar(&c.scheme, "scheme", "", desc)
		c.fs.StringVar(&c.scheme, "s", "", desc)
	}
	return c.fs
}

func (c *Login) Run(ctx *cmd.Context) error {
	if os.Getenv("TSURU_TOKEN") != "" {
		return errTsuruTokenDefined
	}

	scheme, err := getScheme(c.scheme)
	if err != nil {
		return err
	}

	if scheme.Name == "oidc" {
		return oidcLogin(ctx, scheme)
	} else if scheme.Name == "oauth" {
		return oauthLogin(ctx, scheme)
	} else if scheme.Name == "native" {
		return nativeLogin(ctx)
	}

	return fmt.Errorf("scheme %q is not implemented", scheme.Name)
}

func port(loginInfo *authTypes.SchemeInfo) string {
	if loginInfo.Data.Port != "" {
		return fmt.Sprintf(":%s", loginInfo.Data.Port)
	}
	return ":0"
}

func getScheme(schemeName string) (*authTypes.SchemeInfo, error) {
	schemes, err := schemesInfo()
	if err != nil {
		return nil, err
	}

	foundSchemes := []string{}

	if schemeName == "" {
		for _, scheme := range schemes {
			if scheme.Default {
				return &scheme, nil
			}
		}
	}

	for _, scheme := range schemes {
		foundSchemes = append(foundSchemes, scheme.Name)
		if scheme.Name == schemeName {
			return &scheme, nil
		}
	}

	if len(foundSchemes) == 0 {
		return nil, fmt.Errorf("scheme %q is not found", schemeName)
	}

	return nil, fmt.Errorf("scheme %q is not found, valid schemes are: %s", schemeName, strings.Join(foundSchemes, ", "))
}

func schemesInfo() ([]authTypes.SchemeInfo, error) {
	url, err := config.GetURLVersion("1.18", "/auth/schemes")
	if err != nil {
		return nil, err
	}
	resp, err := tsuruHTTP.UnauthenticatedClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not call %q, status code: %d", url, resp.StatusCode)
	}

	schemes := []authTypes.SchemeInfo{}
	err = json.NewDecoder(resp.Body).Decode(&schemes)
	if err != nil {
		return nil, err
	}
	return schemes, nil
}

const callbackPage = `<!DOCTYPE html>
<html>
<head>
	<style>
	body {
		text-align: center;
	}
	</style>
</head>
<body>
	%s
</body>
</html>
`

const successMarkup = `
	<script>window.close();</script>
	<h1>Login Successful!</h1>
	<p>You can close this window now.</p>
`

const errorMarkup = `
	<h1>Login Failed!</h1>
	<p>%s</p>
`

func writeHTMLError(w io.Writer, err error) {
	msg := fmt.Sprintf(errorMarkup, err.Error())
	fmt.Fprintf(w, callbackPage, msg)
}
