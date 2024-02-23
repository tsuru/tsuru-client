// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"io"
	"net/http"
	"net/url"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru-client/tsuru/config"
	"github.com/tsuru/tsuru/cmd"
)

var AuthenticatedClient = &http.Client{}

func NewTerminalClient(rt http.RoundTripper, context *cmd.Context, clientName, clientVersion string, verbosity int) *http.Client {
	stdout := io.Discard
	stderr := io.Discard

	if context != nil {
		if context.Stdout != nil {
			stdout = context.Stdout
		}

		if context.Stderr != nil {
			stderr = context.Stderr
		}
	}
	transport := &TerminalRoundTripper{
		RoundTripper:   rt,
		Stdout:         stdout,
		Stderr:         stderr,
		Verbosity:      &verbosity,
		Progname:       clientName,
		CurrentVersion: clientVersion,
	}
	return &http.Client{Transport: transport}
}

func TsuruClientFromEnvironment() (*tsuru.APIClient, error) {
	cfg := &tsuru.Configuration{
		HTTPClient:    AuthenticatedClient,
		DefaultHeader: map[string]string{},
	}

	var err error
	cfg.BasePath, err = config.GetTarget()
	if err != nil {
		return nil, err
	}

	if _, authSet := cfg.DefaultHeader["Authorization"]; !authSet {
		if token, tokenErr := config.ReadToken(); tokenErr == nil && token != "" {
			cfg.DefaultHeader["Authorization"] = "bearer " + token
		}
	}
	cli := tsuru.NewAPIClient(cfg)
	return cli, nil
}

func UnwrapErr(err error) error {
	if err == nil {
		return nil
	}
	if urlErr, ok := err.(*url.Error); ok {
		return urlErr.Err
	}
	return err
}
