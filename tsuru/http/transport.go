// Copyright 2018 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"

	goVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/tsuru/tsuru-client/tsuru/config"
	tsuruerr "github.com/tsuru/tsuru/errors"
)

var (
	_                   http.RoundTripper = &TerminalRoundTripper{}
	defaultRoundTripper                   = http.DefaultTransport
)

const (
	versionHeader   = "Supported-Tsuru"
	verbosityHeader = "X-Tsuru-Verbosity"

	invalidVersionFormat = `#####################################################################

WARNING: You're using an unsupported version of %s.

You must have at least version %s, your current
version is %s.

Please go to http://docs.tsuru.io/en/latest/using/install-client.html
and download the last version.

#####################################################################

`
)

var errUnauthorized = &tsuruerr.HTTP{Code: http.StatusUnauthorized, Message: "unauthorized"}

// TerminalRoundTripper is a RoundTripper that dumps request and response
// based on the Verbosity.
// Verbosity >= 1 --> Dumps request
// Verbosity >= 2 --> Dumps response
type TerminalRoundTripper struct {
	http.RoundTripper
	Stdout         io.Writer
	Stderr         io.Writer
	CurrentVersion string
	Progname       string
}

func getVerbosity() int {
	v, _ := strconv.Atoi(os.Getenv("TSURU_VERBOSITY"))
	return v
}

func (v *TerminalRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	roundTripper := v.RoundTripper
	if roundTripper == nil {
		roundTripper = defaultRoundTripper
	}
	verbosity := getVerbosity()
	req.Header.Add(verbosityHeader, strconv.Itoa(verbosity))
	req.Header.Set("User-Agent", fmt.Sprintf("tsuru-client/%s", v.CurrentVersion))
	req.Close = true

	if verbosity >= TerminalClientOnlyRequest {
		fmt.Fprintf(v.Stdout, "*************************** <Request uri=%q> **********************************\n", req.URL.RequestURI())
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			return nil, err
		}
		fmt.Fprint(v.Stdout, string(requestDump))
		if requestDump[len(requestDump)-1] != '\n' {
			fmt.Fprintln(v.Stdout)
		}
		fmt.Fprintf(v.Stdout, "*************************** </Request uri=%q> **********************************\n", req.URL.RequestURI())
	}

	response, err := roundTripper.RoundTrip(req)
	if verbosity >= TerminalClientVerbose && response != nil {
		fmt.Fprintf(v.Stdout, "*************************** <Response uri=%q> **********************************\n", req.URL.RequestURI())
		responseDump, errDump := httputil.DumpResponse(response, true)
		if errDump != nil {
			return nil, errDump
		}
		fmt.Fprint(v.Stdout, string(responseDump))
		if responseDump[len(responseDump)-1] != '\n' {
			fmt.Fprintln(v.Stdout)
		}
		fmt.Fprintf(v.Stdout, "*************************** </Response uri=%q> **********************************\n", req.URL.RequestURI())
	}
	err = detectClientError(err)
	if err != nil {
		return nil, err
	}

	supported := response.Header.Get(versionHeader)
	if !validateVersion(supported, v.CurrentVersion) {
		fmt.Fprintf(v.Stderr, invalidVersionFormat, v.Progname, supported, v.CurrentVersion)
	}

	if response.StatusCode > 399 {
		err := &tsuruerr.HTTP{
			Code:    response.StatusCode,
			Message: response.Status,
		}

		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		if len(body) > 0 {
			err.Message = string(body)
		}

		return nil, err
	}

	return response, err
}

func detectClientError(err error) error {
	if err == nil {
		return nil
	}
	detectErr := func(e error) error {
		target, _ := config.ReadTarget()

		switch e := e.(type) {
		case *tsuruerr.HTTP:
			return errors.Wrapf(e, "Error received from tsuru server (%s), %d", target, e.Code)
		case x509.UnknownAuthorityError:
			return errors.Wrapf(e, "Failed to connect to tsuru server (%s)", target)
		}
		return errors.Wrapf(e, "Failed to connect to tsuru server (%s), it's probably down", target)
	}

	return detectErr(UnwrapErr(err))
}

// validateVersion checks whether current version is greater or equal to
// supported version.
func validateVersion(supported, current string) bool {
	if current == "dev" {
		return true
	}
	if supported == "" {
		return true
	}
	vSupported, err := goVersion.NewVersion(supported)
	if err != nil {
		return false
	}
	vCurrent, err := goVersion.NewVersion(current)
	if err != nil {
		return false
	}
	return vCurrent.Compare(vSupported) >= 0
}

type TokenV1RoundTripper struct {
	http.RoundTripper
}

func (v *TokenV1RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	roundTripper := v.RoundTripper
	if roundTripper == nil {
		roundTripper = defaultRoundTripper
	}

	if token, err := config.ReadTokenV1(); err == nil && token != "" {
		req.Header.Set("Authorization", "bearer "+token)
	}

	response, err := roundTripper.RoundTrip(req)

	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusUnauthorized {
		if teamToken := config.ReadTeamToken(); teamToken != "" {
			fmt.Fprintln(os.Stderr, "Invalid session - maybe invalid defined token on TSURU_TOKEN envvar")
		} else {
			fmt.Fprintln(os.Stderr, "Invalid session")
		}

		return nil, errUnauthorized
	}

	return response, nil
}

func NewTokenV1RoundTripper() http.RoundTripper {
	return &TokenV1RoundTripper{
		RoundTripper: defaultRoundTripper,
	}
}
