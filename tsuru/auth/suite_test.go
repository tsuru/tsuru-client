package auth

import (
	"net/http"
	"testing"

	"github.com/tsuru/tsuru-client/tsuru/config"
	"gopkg.in/check.v1"

	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
)

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

type S struct{}

func (s *S) SetUpTest(c *check.C) {
	config.ResetFileSystem()
}

func setupFakeTransport(rt http.RoundTripper) {
	tsuruHTTP.AuthenticatedClient = tsuruHTTP.NewTerminalClient(tsuruHTTP.TerminalClientOptions{
		RoundTripper:  rt,
		ClientName:    "test",
		ClientVersion: "0.1.0",
	})

	tsuruHTTP.UnauthenticatedClient = &http.Client{
		Transport: rt,
	}
}
