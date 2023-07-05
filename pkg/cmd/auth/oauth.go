// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tsuru/tsuru-client/v2/internal/config"
	"github.com/tsuru/tsuru-client/v2/internal/exec"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
)

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

func port(schemeData map[string]string) string {
	p := schemeData["port"]
	if p != "" {
		return fmt.Sprintf(":%s", p)
	}
	return ":0"
}

func getToken(tsuruCtx *tsuructx.TsuruContext, code, redirectURL string) (token string, err error) {
	v := url.Values{}
	v.Set("code", code)
	v.Set("redirectUrl", redirectURL)
	b := strings.NewReader(v.Encode())
	request, err := tsuruCtx.NewRequest("POST", "/auth/login", b)
	if err != nil {
		return
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResponse, err := tsuruCtx.RawHTTPClient().Do(request)
	if err != nil {
		return token, errors.Wrap(err, "error during login post")
	}
	defer httpResponse.Body.Close()

	result, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return token, errors.Wrap(err, "error reading body")
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(result, &data)
	if err != nil {
		return token, errors.Wrapf(err, "error parsing response: %s", result)
	}
	return data["token"].(string), nil
}

func callback(tsuruCtx *tsuructx.TsuruContext, redirectURL string, finish chan bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			finish <- true
		}()
		var page string
		token, err := getToken(tsuruCtx, r.URL.Query().Get("code"), redirectURL)
		if err == nil {
			config.SaveToken(tsuruCtx.Fs, token)
			page = fmt.Sprintf(callbackPage, successMarkup)
		} else {
			msg := fmt.Sprintf(errorMarkup, err.Error())
			page = fmt.Sprintf(callbackPage, msg)
		}
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte(page))
	}
}

func oauthLogin(tsuruCtx *tsuructx.TsuruContext, scheme *loginScheme) error {
	if _, ok := scheme.Data["authorizeUrl"]; !ok {
		return fmt.Errorf("missing authorizeUrl in scheme data")
	}

	l, err := net.Listen("tcp", port(scheme.Data)) // use low level net.Listen for random port with :0
	if err != nil {
		return err
	}
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return err
	}
	redirectURL := fmt.Sprintf("http://localhost:%s", port)
	authURL := strings.Replace(scheme.Data["authorizeUrl"], "__redirect_url__", redirectURL, 1)
	finish := make(chan bool, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", callback(tsuruCtx, redirectURL, finish))
	server := &http.Server{}
	server.Handler = mux
	go server.Serve(l)

	err = exec.Open(tsuruCtx.Executor, authURL)
	if err != nil {
		fmt.Fprintln(tsuruCtx.Stdout, "Failed to start your browser.")
		fmt.Fprintf(tsuruCtx.Stdout, "Please open the following URL in your browser: %s\n", authURL)
	}
	<-finish

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	fmt.Fprintln(tsuruCtx.Stdout, "Successfully logged in!")
	return nil
}
