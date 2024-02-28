package auth

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/config"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/fs/fstest"
	"github.com/tsuru/tsuru/types/auth"
	"gopkg.in/check.v1"
)

func targetInit() {
	f, _ := config.Filesystem().Create(config.JoinWithUserDir(".tsuru", "target"))
	f.Write([]byte("http://localhost"))
	f.Close()
	config.WriteOnTargetList("test", "http://localhost")
}

func setupNativeScheme(trans http.RoundTripper) {
	setupFakeTransport(&cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				Transport: cmdtest.Transport{
					Message: `[{"name": "native", "default": true}]`,
					Status:  http.StatusOK,
				},
				CondFunc: func(r *http.Request) bool {
					return strings.HasSuffix(r.URL.Path, "/1.18/auth/schemes")
				},
			},
			{
				Transport: trans,
				CondFunc: func(r *http.Request) bool {
					return trans != nil
				},
			},
		},
	})
}

func (s *S) TestNativeLogin(c *check.C) {
	os.Unsetenv("TSURU_TOKEN")
	config.SetFileSystem(&fstest.RecordingFs{FileContent: "old-token"})
	targetInit()
	defer func() {
		config.ResetFileSystem()
	}()
	expected := "Password: \nSuccessfully logged in!\n"
	reader := strings.NewReader("chico\n")
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: bytes.NewBufferString(""),
		Stderr: io.Discard,
		Stdin:  reader,
	}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: `{"token": "sometoken", "is_admin": true}`,
			Status:  http.StatusOK,
		},
		CondFunc: func(r *http.Request) bool {
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			password := r.FormValue("password") == "chico"
			url := r.URL.Path == "/1.0/users/foo@foo.com/tokens"
			return contentType && password && url
		},
	}

	command := Login{}
	setupNativeScheme(&transport)
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(context.Stdout.(*bytes.Buffer).String(), check.Equals, expected)
	token, err := config.ReadTokenV1()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.Equals, "sometoken")
}

func (s *S) TestNativeLoginWithoutEmailFromArg(c *check.C) {
	os.Unsetenv("TSURU_TOKEN")
	config.SetFileSystem(&fstest.RecordingFs{})
	targetInit()
	defer func() {
		config.ResetFileSystem()
	}()
	expected := "Email: Password: \nSuccessfully logged in!\n"
	reader := strings.NewReader("chico@tsuru.io\nchico\n")
	context := cmd.Context{
		Args:   []string{},
		Stdout: bytes.NewBufferString(""),
		Stderr: io.Discard,
		Stdin:  reader,
	}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: `{"token": "sometoken", "is_admin": true}`,
			Status:  http.StatusOK,
		},
		CondFunc: func(r *http.Request) bool {
			return r.URL.Path == "/1.0/users/chico@tsuru.io/tokens"
		},
	}
	setupNativeScheme(&transport)
	command := Login{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(context.Stdout.(*bytes.Buffer).String(), check.Equals, expected)
	token, err := config.ReadTokenV1()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.Equals, "sometoken")
}

func (s *S) TestNativeLoginShouldNotDependOnTsuruTokenFile(c *check.C) {
	oldToken := os.Getenv("TSURU_TOKEN")
	os.Unsetenv("TSURU_TOKEN")
	defer func() {
		os.Setenv("TSURU_TOKEN", oldToken)
	}()
	config.SetFileSystem(&fstest.RecordingFs{})
	defer func() {
		config.ResetFileSystem()
	}()
	f, _ := config.Filesystem().Create(config.JoinWithUserDir(".tsuru", "target"))
	f.Write([]byte("http://localhost"))
	f.Close()
	expected := "Password: \nSuccessfully logged in!\n"
	reader := strings.NewReader("chico\n")
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: bytes.NewBufferString(""),
		Stderr: io.Discard,
		Stdin:  reader,
	}
	setupNativeScheme(&cmdtest.Transport{Message: `{"token":"anothertoken"}`, Status: http.StatusOK})
	command := Login{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(context.Stdout.(*bytes.Buffer).String(), check.Equals, expected)
}

func (s *S) TestNativeLoginShouldReturnErrorIfThePasswordIsNotGiven(c *check.C) {
	oldToken := os.Getenv("TSURU_TOKEN")
	os.Unsetenv("TSURU_TOKEN")
	config.SetFileSystem(&fstest.RecordingFs{})
	defer func() {
		config.ResetFileSystem()
		os.Setenv("TSURU_TOKEN", oldToken)
	}()
	targetInit()
	setupNativeScheme(nil)
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: io.Discard,
		Stderr: io.Discard,
		Stdin:  strings.NewReader("\n"),
	}
	command := Login{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^You must provide the password!$")
}

func (s *S) TestNativeLoginWithTsuruToken(c *check.C) {
	oldToken := os.Getenv("TSURU_TOKEN")
	os.Setenv("TSURU_TOKEN", "settoken")
	defer func() {
		os.Setenv("TSURU_TOKEN", oldToken)
	}()
	context := cmd.Context{
		Args:   []string{"foo@foo.com"},
		Stdout: io.Discard,
		Stderr: io.Discard,
		Stdin:  strings.NewReader("\n"),
	}
	command := Login{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "this command can't run with $TSURU_TOKEN environment variable set. Did you forget to unset?")
}

func (s *S) TestPort(c *check.C) {
	c.Assert(":0", check.Equals, port(&auth.SchemeInfo{}))
	c.Assert(":4242", check.Equals, port(&auth.SchemeInfo{Data: auth.SchemeData{Port: "4242"}}))
}
