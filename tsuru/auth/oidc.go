// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"context"
	stdContext "context"
	"fmt"
	"net"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru/cmd"
	authTypes "github.com/tsuru/tsuru/types/auth"
	"golang.org/x/oauth2"
)

func oidcLogin(ctx *cmd.Context, loginInfo *authTypes.SchemeInfo) error {
	pkceVerifier := oauth2.GenerateVerifier()

	fmt.Fprintln(ctx.Stderr, "Starting OIDC login")

	l, err := net.Listen("tcp", port(loginInfo))
	if err != nil {
		return err
	}
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return err
	}

	oauth2Config := oauth2.Config{
		ClientID:    loginInfo.Data.ClientID,
		Scopes:      loginInfo.Data.Scopes,
		RedirectURL: fmt.Sprintf("http://localhost:%s", port),
		Endpoint: oauth2.Endpoint{
			AuthURL:  loginInfo.Data.AuthURL,
			TokenURL: loginInfo.Data.TokenURL,
		},
	}

	authURL := oauth2Config.AuthCodeURL("", oauth2.S256ChallengeOption(pkceVerifier))

	finish := make(chan bool)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			finish <- true
		}()

		t, handlerErr := oauth2Config.Exchange(stdContext.Background(), r.URL.Query().Get("code"), oauth2.VerifierOption(pkceVerifier))

		w.Header().Add("Content-Type", "text/html")

		if handlerErr != nil {
			writeHTMLError(w, handlerErr)
			return
		}

		fmt.Fprintln(ctx.Stderr, "Successfully logged in via OIDC!")
		fmt.Fprintf(ctx.Stderr, "The OIDC token will expiry in %s\n", time.Since(t.Expiry)*-1)

		handlerErr = config.WriteTokenV2(config.TokenV2{
			Scheme:       "oidc",
			OAuth2Token:  t,
			OAuth2Config: &oauth2Config,
		})

		if handlerErr != nil {
			writeHTMLError(w, handlerErr)
			return
		}

		// legacy token
		handlerErr = config.WriteTokenV1(t.AccessToken)

		if handlerErr != nil {
			writeHTMLError(w, handlerErr)
			return
		}

		fmt.Fprintf(w, callbackPage, successMarkup)

	})
	server := &http.Server{
		Handler: mux,
	}
	go server.Serve(l)
	err = open(authURL)
	if err != nil {
		fmt.Fprintln(ctx.Stdout, "Failed to start your browser.")
		fmt.Fprintf(ctx.Stdout, "Please open the following URL in your browser: %s\n", authURL)
	}
	<-finish
	timedCtx, cancel := stdContext.WithTimeout(stdContext.Background(), 15*time.Second)
	defer cancel()
	server.Shutdown(timedCtx)
	return nil
}

func NewOIDCTokenSource(tokenV2 *config.TokenV2) oauth2.TokenSource {
	baseTokenSource := tokenV2.OAuth2Config.TokenSource(context.Background(), tokenV2.OAuth2Token)
	return newTokenSourceFSStorage(baseTokenSource, tokenV2)
}

type TokenSourceFSStorage struct {
	BaseTokenSource oauth2.TokenSource
	LastToken       *config.TokenV2
}

var _ oauth2.TokenSource = &TokenSourceFSStorage{}

func (t *TokenSourceFSStorage) Token() (*oauth2.Token, error) {
	newToken, err := t.BaseTokenSource.Token()
	if err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(t.LastToken.OAuth2Token, newToken) {
		fmt.Fprintf(os.Stderr, "The OIDC token was refreshed and expiry in %s\n", time.Since(newToken.Expiry)*-1)

		t.LastToken.OAuth2Token = newToken
		err = config.WriteTokenV2(*t.LastToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not write refreshed token: %s\n", err.Error())
			return nil, err
		}

		err = config.WriteTokenV1(newToken.AccessToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not write legacy refreshed token: %s\n", err.Error())
			return nil, err
		}
	}

	return newToken, nil
}

func newTokenSourceFSStorage(baseTokenSource oauth2.TokenSource, tokenV2 *config.TokenV2) oauth2.TokenSource {
	return &TokenSourceFSStorage{
		BaseTokenSource: baseTokenSource,
		LastToken:       tokenV2,
	}
}

type OIDCTokenProvider struct {
	OAuthTokenSource oauth2.TokenSource
}

func (ts *OIDCTokenProvider) Token() (string, error) {
	t, err := ts.OAuthTokenSource.Token()
	if err != nil {
		return "", err
	}
	return t.AccessToken, nil
}
