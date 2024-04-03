package http

import (
	"bytes"
	"net/http"
	"os"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/fs/fstest"
	check "gopkg.in/check.v1"
)

func (s *S) TestVerboseRoundTripperDumpRequest(c *check.C) {
	os.Setenv("TSURU_VERBOSITY", "1")
	defer func() {
		os.Unsetenv("TSURU_VERBOSITY")
	}()
	out := new(bytes.Buffer)
	r := TerminalRoundTripper{
		Stdout:         out,
		CurrentVersion: "1.0.0",
		RoundTripper: &cmdtest.Transport{
			Message: "Success!",
			Status:  http.StatusOK,
		},
	}
	req, err := http.NewRequest(http.MethodGet, "http://localhost/users", nil)
	c.Assert(err, check.IsNil)
	_, err = r.RoundTrip(req)
	c.Assert(err, check.IsNil)
	c.Assert(out.String(), check.DeepEquals, "*************************** <Request uri=\"/users\"> **********************************\n"+
		"GET /users HTTP/1.1\r\n"+
		"Host: localhost\r\n"+
		"User-Agent: tsuru-client/1.0.0\r\n"+
		"X-Tsuru-Verbosity: 1\r\n"+
		"\r\n"+
		"*************************** </Request uri=\"/users\"> **********************************\n")
}

func (s *S) TestVerboseRoundTripperDumpRequestResponse2(c *check.C) {
	os.Setenv("TSURU_VERBOSITY", "2")
	defer func() {
		os.Unsetenv("TSURU_VERBOSITY")
	}()

	out := new(bytes.Buffer)
	r := TerminalRoundTripper{
		Stdout:         out,
		CurrentVersion: "1.2.0",
		RoundTripper: &cmdtest.Transport{
			Message: "Success!",
			Status:  http.StatusOK,
		},
	}
	req, err := http.NewRequest(http.MethodGet, "http://localhost/users", nil)
	c.Assert(err, check.IsNil)
	_, err = r.RoundTrip(req)
	c.Assert(err, check.IsNil)
	c.Assert(out.String(), check.DeepEquals, "*************************** <Request uri=\"/users\"> **********************************\n"+
		"GET /users HTTP/1.1\r\n"+
		"Host: localhost\r\n"+
		"User-Agent: tsuru-client/1.2.0\r\n"+
		"X-Tsuru-Verbosity: 2\r\n"+
		"\r\n"+
		"*************************** </Request uri=\"/users\"> **********************************\n"+
		"*************************** <Response uri=\"/users\"> **********************************\n"+
		"HTTP/0.0 200 OK\r\n"+
		"\r\n"+
		"Success!\n"+
		"*************************** </Response uri=\"/users\"> **********************************\n")

}

func (s *S) TestTokenV1RoundTripperShouldIncludeTheHeaderAuthorizationWhenTsuruTokenFileExists(c *check.C) {
	os.Unsetenv("TSURU_TOKEN")
	config.SetFileSystem(&fstest.RecordingFs{FileContent: "mytoken"})
	targetInit()
	defer func() {
		config.ResetFileSystem()
	}()
	request, err := http.NewRequest("GET", "/", nil)
	c.Assert(err, check.IsNil)
	trans := cmdtest.Transport{Message: "", Status: http.StatusOK}
	rt := &TokenV1RoundTripper{RoundTripper: &trans}
	_, err = rt.RoundTrip(request)
	c.Assert(err, check.IsNil)
	c.Assert(request.Header.Get("Authorization"), check.Equals, "bearer mytoken")
}
