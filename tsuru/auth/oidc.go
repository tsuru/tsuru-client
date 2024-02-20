package auth

import (
	"context"
	stdContext "context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/tsuru/tsuru/cmd"
	"golang.org/x/oauth2"
)

func oidcLogin(ctx *cmd.Context, loginInfo *loginScheme) error {
	pkceVerifier := oauth2.GenerateVerifier()

	fmt.Fprintln(ctx.Stdout, "Starting OIDC login")

	l, err := net.Listen("tcp", port(loginInfo))
	if err != nil {
		return err
	}
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return err
	}

	config := oauth2.Config{
		ClientID:    loginInfo.Data.ClientID,
		Scopes:      loginInfo.Data.Scopes,
		RedirectURL: fmt.Sprintf("http://localhost:%s", port),
		Endpoint: oauth2.Endpoint{
			AuthURL:  loginInfo.Data.AuthURL,
			TokenURL: loginInfo.Data.TokenURL,
		},
	}

	authURL := config.AuthCodeURL("", oauth2.S256ChallengeOption(pkceVerifier))

	finish := make(chan bool)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			finish <- true
		}()

		t, err := config.Exchange(context.Background(), r.URL.Query().Get("code"), oauth2.VerifierOption(pkceVerifier))

		w.Header().Add("Content-Type", "text/html")

		if err != nil {
			msg := fmt.Sprintf(errorMarkup, err.Error())
			fmt.Fprintf(w, callbackPage, msg)
			return
		}

		fmt.Fprintln(ctx.Stdout, "Successfully logged in!")
		fmt.Fprintf(ctx.Stdout, "The token will expiry in %s\n", time.Since(t.Expiry)*-1)

		json.NewEncoder(ctx.Stdout).Encode(t)

		fmt.Println("TODO write token: ", t.AccessToken)
		fmt.Println("TODO write refresh token: ", t.RefreshToken)

		fmt.Printf("%#v\n", t)

		fmt.Fprintf(w, callbackPage, successMarkup)

	})
	server := &http.Server{}
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
