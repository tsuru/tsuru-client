package auth

import (
	stdContext "context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tsuru/tsuru-client/tsuru/config"
	"github.com/tsuru/tsuru/cmd"
	tsuruNet "github.com/tsuru/tsuru/net"
	authTypes "github.com/tsuru/tsuru/types/auth"
)

func oauthLogin(ctx *cmd.Context, loginInfo *authTypes.SchemeInfo) error {
	finish := make(chan bool)
	l, err := net.Listen("tcp", port(loginInfo))
	if err != nil {
		return err
	}
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return err
	}
	redirectURL := fmt.Sprintf("http://localhost:%s", port)
	authURL := strings.Replace(loginInfo.Data.AuthorizeURL, "__redirect_url__", redirectURL, 1)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			finish <- true
		}()
		var page string
		token, handlerErr := convertOAuthToken(r.URL.Query().Get("code"), redirectURL)
		if handlerErr != nil {
			writeHTMLError(w, handlerErr)
			return
		}
		handlerErr = config.WriteTokenV1(token)
		if handlerErr != nil {
			writeHTMLError(w, handlerErr)
			return
		}
		handlerErr = config.RemoveTokenV2()
		if handlerErr != nil {
			writeHTMLError(w, handlerErr)
			return
		}
		page = fmt.Sprintf(callbackPage, successMarkup)

		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte(page))
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
	fmt.Fprintln(ctx.Stdout, "Successfully logged in!")
	return nil
}

func convertOAuthToken(code, redirectURL string) (string, error) {
	var token string
	v := url.Values{}
	v.Set("code", code)
	v.Set("redirectUrl", redirectURL)
	u, err := config.GetURL("/auth/login")
	if err != nil {
		return token, errors.Wrap(err, "Error in GetURL")
	}
	resp, err := tsuruNet.Dial15Full300Client.Post(u, "application/x-www-form-urlencoded", strings.NewReader(v.Encode()))
	if err != nil {
		return token, errors.Wrap(err, "Error during login post")
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return token, errors.Wrap(err, "Error reading body")
	}
	data := make(map[string]interface{})
	err = json.Unmarshal(result, &data)
	if err != nil {
		return token, errors.Wrapf(err, "Error parsing response: %s", result)
	}
	return data["token"].(string), nil
}
