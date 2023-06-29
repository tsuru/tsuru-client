// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tsuructx

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"
)

type TsuruClientHTTPTransport struct {
	transport http.RoundTripper
	tsuruCtx  *TsuruContext
}

func (t *TsuruClientHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range tsuruDefaultHeadersFromContext(t.tsuruCtx) {
		req.Header.Set(k, v)
	}

	if t.tsuruCtx.InsecureSkipVerify {
		t.transport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	req.Close = true

	req.Header.Set("X-Tsuru-Verbosity", "0")
	// Verbosity level=1: log request
	if t.tsuruCtx.Verbosity() >= 1 {
		req.Header.Set("X-Tsuru-Verbosity", strconv.Itoa(t.tsuruCtx.Verbosity()))
		fmt.Fprintf(t.tsuruCtx.Stdout, "*************************** <Request uri=%q> **********************************\n", req.URL.RequestURI())
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			return nil, err
		}
		fmt.Fprint(t.tsuruCtx.Stdout, string(requestDump))
		if requestDump[len(requestDump)-1] != '\n' {
			fmt.Fprintln(t.tsuruCtx.Stdout)
		}
		fmt.Fprintf(t.tsuruCtx.Stdout, "*************************** </Request uri=%q> **********************************\n", req.URL.RequestURI())
	}

	response, err := t.transport.RoundTrip(req)

	// Verbosity level=2: log response
	if t.tsuruCtx.Verbosity() >= 2 && response != nil {
		fmt.Fprintf(t.tsuruCtx.Stdout, "*************************** <Response uri=%q> **********************************\n", req.URL.RequestURI())
		responseDump, errDump := httputil.DumpResponse(response, true)
		if errDump != nil {
			return nil, errDump
		}
		fmt.Fprint(t.tsuruCtx.Stdout, string(responseDump))
		if responseDump[len(responseDump)-1] != '\n' {
			fmt.Fprintln(t.tsuruCtx.Stdout)
		}
		fmt.Fprintf(t.tsuruCtx.Stdout, "*************************** </Response uri=%q> **********************************\n", req.URL.RequestURI())
	}

	return response, err
}

func (c *TsuruContext) httpTransportWrapper(roundTripper http.RoundTripper) *TsuruClientHTTPTransport {
	t := &TsuruClientHTTPTransport{
		transport: roundTripper,
		tsuruCtx:  c,
	}
	if roundTripper == nil {
		t.transport = http.DefaultTransport
	}
	return t
}

func tsuruDefaultHeadersFromContext(tsuruCtx *TsuruContext) map[string]string {
	result := map[string]string{}
	result["User-Agent"] = tsuruCtx.UserAgent
	if result["User-Agent"] == "" {
		result["User-Agent"] = "tsuru-client"
	}
	if result["Authorization"] == "" {
		result["Authorization"] = "bearer " + tsuruCtx.Token()
	}
	if result["Accept"] == "" {
		result["Accept"] = "application/json"
	}
	return result
}

// NewRequest creates a new http.Request with the correct base path.
func (tc *TsuruContext) NewRequest(method string, url string, body io.Reader) (*http.Request, error) {
	if !strings.HasPrefix(url, tc.TargetURL()) {
		if !strings.HasPrefix(url, "/") {
			url = "/" + url
		}
		if !regexp.MustCompile(`^/[0-9]+\.[0-9]+/`).MatchString(url) {
			url = "/1.0" + url
		}
		url = strings.TrimRight(tc.TargetURL(), "/") + url
	}
	return http.NewRequest(method, url, body)
}

func (tc *TsuruContext) DefaultHeaders() http.Header {
	headers := make(http.Header)
	for k, v := range tc.Config().DefaultHeader {
		headers.Add(k, v)
	}
	return headers
}
