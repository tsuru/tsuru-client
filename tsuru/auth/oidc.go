package auth

import (
	stdContext "context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/tsuru/tsuru-client/tsuru/config"
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

	oautn2Config := oauth2.Config{
		ClientID:    loginInfo.Data.ClientID,
		Scopes:      loginInfo.Data.Scopes,
		RedirectURL: fmt.Sprintf("http://localhost:%s", port),
		Endpoint: oauth2.Endpoint{
			AuthURL:  loginInfo.Data.AuthURL,
			TokenURL: loginInfo.Data.TokenURL,
		},
	}

	authURL := oautn2Config.AuthCodeURL("", oauth2.S256ChallengeOption(pkceVerifier))

	finish := make(chan bool)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			finish <- true
		}()

		t, err := oautn2Config.Exchange(stdContext.Background(), r.URL.Query().Get("code"), oauth2.VerifierOption(pkceVerifier))

		w.Header().Add("Content-Type", "text/html")

		if err != nil {
			writeHTMLError(w, err)
			return
		}

		fmt.Fprintln(ctx.Stderr, "Successfully logged in via OIDC!")
		fmt.Fprintf(ctx.Stderr, "The OIDC token will expiry in %s\n", time.Since(t.Expiry)*-1)

		err = config.WriteTokenV2(config.TokenV2{
			Scheme:      "oidc",
			OAuth2Token: t,
		})

		if err != nil {
			writeHTMLError(w, err)
			return
		}

		// legacy token
		err = config.WriteToken(t.AccessToken)

		if err != nil {
			writeHTMLError(w, err)
			return
		}

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

func writeHTMLError(w io.Writer, err error) {
	msg := fmt.Sprintf(errorMarkup, err.Error())
	fmt.Fprintf(w, callbackPage, msg)
}
