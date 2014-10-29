// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/user"
	"path"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/testing"
	fstesting "github.com/tsuru/tsuru/fs/testing"
	"launchpad.net/gocheck"
)

func (s *S) TestKeyAdd(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	u, err := user.Current()
	c.Assert(err, gocheck.IsNil)
	p := path.Join(u.HomeDir, ".ssh", "id_rsa.pub")
	name := "my-key"
	expected := fmt.Sprintf("Key %q successfully added!\n", name)
	context := cmd.Context{
		Args:   []string{name, p},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := testing.ConditionalTransport{
		Transport: testing.Transport{Message: "success", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			expectedBody := `{"key":"user-key","name":"my-key"}`
			body, err := ioutil.ReadAll(r.Body)
			c.Assert(err, gocheck.IsNil)
			return r.Method == "POST" && r.URL.Path == "/users/keys" && string(body) == expectedBody
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	fs := fstesting.RecordingFs{FileContent: "user-key"}
	command := keyAdd{keyReader{fsystem: &fs}}
	err = command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestKeyAddStdin(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	stdin := bytes.NewBufferString("my powerful key")
	name := "my-key"
	expected := fmt.Sprintf("Key %q successfully added!\n", name)
	context := cmd.Context{
		Args:   []string{name, "-"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  stdin,
	}
	transport := testing.ConditionalTransport{
		Transport: testing.Transport{Message: "success", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			expectedBody := `{"key":"my powerful key","name":"my-key"}`
			body, err := ioutil.ReadAll(r.Body)
			c.Assert(err, gocheck.IsNil)
			return r.Method == "POST" && r.URL.Path == "/users/keys" && string(body) == expectedBody
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := keyAdd{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestKeyAddReturnsProperErrorIfTheGivenKeyFileDoesNotExist(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"my-key", "/unknown/key.pub"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fs := fstesting.FileNotFoundFs{RecordingFs: fstesting.RecordingFs{}}
	command := keyAdd{keyReader{fsystem: &fs}}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, `file "/unknown/key.pub" doesn't exist`)
}

func (s *S) TestKeyAddFileSystemError(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"my-key", "/unknown/key.pub"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fs := fstesting.FailureFs{
		RecordingFs: fstesting.RecordingFs{},
		Err:         errors.New("what happened?"),
	}
	command := keyAdd{keyReader{fsystem: &fs}}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "what happened?")
}

func (s *S) TestKeyAddError(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{Args: []string{"my-key", "/tmp/id_rsa.pub"}, Stdout: &stdout, Stderr: &stderr}
	transport := testing.Transport{
		Message: "something went wrong",
		Status:  http.StatusInternalServerError,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	fs := fstesting.RecordingFs{FileContent: "user-key"}
	command := keyAdd{keyReader{fsystem: &fs}}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "something went wrong")
}

func (s *S) TestInfoKeyAdd(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "key-add",
		Usage:   "key-add <key-name> <path/to/key/file.pub>",
		Desc:    "adds a public key to your account",
		MinArgs: 2,
	}
	c.Assert((&keyAdd{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestKeyRemove(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	keyName := "my-key"
	expected := fmt.Sprintf("Key %q successfully removed!\n", keyName)
	context := cmd.Context{Args: []string{keyName}, Stdout: &stdout, Stderr: &stderr}
	transport := testing.ConditionalTransport{
		Transport: testing.Transport{Message: "success", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			expectedBody := `{"name":"my-key"}`
			body, err := ioutil.ReadAll(r.Body)
			c.Assert(err, gocheck.IsNil)
			return r.Method == "DELETE" && r.URL.Path == "/users/keys" && string(body) == expectedBody
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := keyRemove{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestKeyRemoveError(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	keyName := "my-key"
	context := cmd.Context{Args: []string{keyName}, Stdout: &stdout, Stderr: &stderr}
	transport := testing.Transport{
		Message: "something went wrong",
		Status:  http.StatusInternalServerError,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := keyRemove{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "something went wrong")
}

func (s *S) TestInfoKeyRemove(c *gocheck.C) {
	expected := &cmd.Info{
		Name:    "key-remove",
		Usage:   "key-remove <key-name>",
		Desc:    "removes the given public key from your account",
		MinArgs: 1,
	}
	c.Assert((&keyRemove{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestKeyList(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	expected := `+------+-----------------------------------------------------------------+
| Name | Content                                                         |
+------+-----------------------------------------------------------------+
| key1 | key1 content                                                    |
| key2 | key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 ... |
+------+-----------------------------------------------------------------+` + "\n"
	context := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	key2Content := strings.Repeat("key2 ", 16)
	body := fmt.Sprintf(`{"key1":"key1 content","key2":%q}`, key2Content)
	transport := testing.ConditionalTransport{
		Transport: testing.Transport{Message: body, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			return r.Method == "GET" && r.URL.Path == "/users/keys"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	var command keyList
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestKeyListNoTruncate(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	expected := `+------+----------------------------------------------------------------------------------+
| Name | Content                                                                          |
+------+----------------------------------------------------------------------------------+
| key1 | key1 content                                                                     |
+------+----------------------------------------------------------------------------------+
| key2 | key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2  |
+------+----------------------------------------------------------------------------------+` + "\n"
	context := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	key2Content := strings.Repeat("key2 ", 16)
	body := fmt.Sprintf(`{"key1":"key1 content","key2":%q}`, key2Content)
	transport := testing.ConditionalTransport{
		Transport: testing.Transport{Message: body, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			return r.Method == "GET" && r.URL.Path == "/users/keys"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	var command keyList
	command.Flags().Parse(true, []string{"--no-truncate"})
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(stdout.String(), gocheck.Equals, expected)
}

func (s *S) TestInfoKeyList(c *gocheck.C) {
	expected := &cmd.Info{
		Name:  "key-list",
		Usage: "key-list [-n/--no-truncate]",
		Desc:  "lists public keys registered in your account",
	}
	c.Assert((&keyList{}).Info(), gocheck.DeepEquals, expected)
}
