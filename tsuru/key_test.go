// Copyright 2015 tsuru-client authors. All rights reserved.
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
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/fs/fstest"
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
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "success", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			expectedBody := `{"key":"user-key","name":"my-key"}`
			body, err := ioutil.ReadAll(r.Body)
			c.Assert(err, gocheck.IsNil)
			return r.Method == "POST" && r.URL.Path == "/users/keys" && string(body) == expectedBody
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	fs := fstest.RecordingFs{FileContent: "user-key"}
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
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "success", Status: http.StatusOK},
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
	fs := fstest.FileNotFoundFs{RecordingFs: fstest.RecordingFs{}}
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
	fs := fstest.FailureFs{
		RecordingFs: fstest.RecordingFs{},
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
	transport := cmdtest.Transport{
		Message: "something went wrong",
		Status:  http.StatusInternalServerError,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	fs := fstest.RecordingFs{FileContent: "user-key"}
	command := keyAdd{keyReader{fsystem: &fs}}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "something went wrong")
}

func (s *S) TestInfoKeyAdd(c *gocheck.C) {
	c.Assert((&keyAdd{}).Info(), gocheck.NotNil)
}

func (s *S) TestKeyRemove(c *gocheck.C) {
	var stdout, stderr bytes.Buffer
	keyName := "my-key"
	expected := fmt.Sprintf("Key %q successfully removed!\n", keyName)
	context := cmd.Context{Args: []string{keyName}, Stdout: &stdout, Stderr: &stderr}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "success", Status: http.StatusOK},
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
	transport := cmdtest.Transport{
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
	c.Assert((&keyRemove{}).Info(), gocheck.NotNil)
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
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: body, Status: http.StatusOK},
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
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: body, Status: http.StatusOK},
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
	c.Assert((&keyList{}).Info(), gocheck.NotNil)
}
