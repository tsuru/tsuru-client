// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	stdContext "context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
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

		w.Header().Add("Content-Type", "text/html")

		// RFC 6749 section 4.1.2.1: the authorization server may redirect
		// back with an error instead of a code.
		query := r.URL.Query()
		if errCode := query.Get("error"); errCode != "" {
			handlerErr := fmt.Errorf("authorization server returned an error: %s: %s", errCode, query.Get("error_description"))
			fmt.Fprintf(ctx.Stderr, "Login failed: %v\n", handlerErr)
			writeHTMLError(w, handlerErr)
			return
		}
		code := query.Get("code")
		if code == "" {
			handlerErr := errors.New("authorization response missing 'code' parameter")
			fmt.Fprintf(ctx.Stderr, "Login failed: %v\n", handlerErr)
			writeHTMLError(w, handlerErr)
			return
		}

		t, handlerErr := oauth2Config.Exchange(stdContext.Background(), code, oauth2.VerifierOption(pkceVerifier))

		if handlerErr != nil {
			writeHTMLError(w, handlerErr)
			return
		}

		fmt.Fprintln(ctx.Stderr, "Successfully logged in via OIDC!")
		tokenExpiry := time.Since(t.Expiry) * -1
		fmt.Fprintf(ctx.Stderr, "The OIDC token will expiry in %s\n", tokenExpiry.Round(time.Second))

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
