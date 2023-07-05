// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tsuru/tsuru-client/v2/internal/config"
	"github.com/tsuru/tsuru-client/v2/internal/tsuructx"
	"golang.org/x/term"
)

func nativeLogin(tsuruCtx *tsuructx.TsuruContext, cmd *cobra.Command, args []string) error {
	var email string
	if len(args) > 0 {
		email = args[0]
	} else {
		fmt.Fprint(tsuruCtx.Stdout, "Email: ")
		fmt.Fscanf(tsuruCtx.Stdin, "%s\n", &email)
	}
	fmt.Fprint(tsuruCtx.Stdout, "Password: ")
	password, err := PasswordFromReader(tsuruCtx.Stdin)
	if err != nil {
		return err
	}
	fmt.Fprintln(tsuruCtx.Stdout)

	v := url.Values{}
	v.Set("password", password)
	b := strings.NewReader(v.Encode())
	request, err := tsuruCtx.NewRequest("POST", "/users/"+email+"/tokens", b)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResponse, err := tsuruCtx.RawHTTPClient().Do(request)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	result, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}
	out := make(map[string]interface{})
	err = json.Unmarshal(result, &out)
	if err != nil {
		return err
	}
	fmt.Fprintln(tsuruCtx.Stdout, "Successfully logged in!")
	return config.SaveToken(tsuruCtx.Fs, out["token"].(string))
}

func PasswordFromReader(reader io.Reader) (string, error) {
	var (
		password []byte
		err      error
	)
	if desc, ok := reader.(tsuructx.DescriptorReader); ok && term.IsTerminal(int(desc.Fd())) {
		password, err = term.ReadPassword(int(desc.Fd()))
		if err != nil {
			return "", err
		}
	} else {
		fmt.Fscanf(reader, "%s\n", &password)
	}
	if len(password) == 0 {
		return "", fmt.Errorf("empty password. You must provide the password")
	}
	return string(password), err
}
