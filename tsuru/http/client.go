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
)

var (
	AuthenticatedClient   = &http.Client{}
	UnauthenticatedClient = &http.Client{}
)

var (
	TerminalClientOnlyRequest = 1
	TerminalClientVerbose     = 2
)

type TerminalClientOptions struct {
	RoundTripper  http.RoundTripper
	ClientName    string
	ClientVersion string
	Stdout        io.Writer
	Stderr        io.Writer
	Verbosity     *int
}

func NewTerminalClient(opts TerminalClientOptions) *http.Client {
	stdout := io.Discard
	stderr := io.Discard

	if opts.Stdout != nil {
		stdout = opts.Stdout
	}

	if opts.Stderr != nil {
		stderr = opts.Stderr
	}

	transport := &TerminalRoundTripper{
		RoundTripper:   opts.RoundTripper,
		Stdout:         stdout,
		Stderr:         stderr,
		Verbosity:      opts.Verbosity,
		Progname:       opts.ClientName,
		CurrentVersion: opts.ClientVersion,
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
