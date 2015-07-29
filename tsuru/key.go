// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/fs"
	"launchpad.net/gnuflag"
)

const keyTruncate = 60

type keyReader struct {
	fsystem fs.Fs
}

func (r *keyReader) fs() fs.Fs {
	if r.fsystem == nil {
		r.fsystem = fs.OsFs{}
	}
	return r.fsystem
}

func (r *keyReader) readKey(context *cmd.Context) (string, error) {
	keyPath := context.Args[1]
	var input io.Reader
	if keyPath == "-" {
		input = context.Stdin
	} else {
		f, err := r.fs().Open(keyPath)
		if err != nil {
			return "", err
		}
		defer f.Close()
		input = f
	}
	output, err := ioutil.ReadAll(input)
	return string(output), err
}

type keyAdd struct {
	fs    *gnuflag.FlagSet
	force bool
	keyReader
}

func (c *keyAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "key-add",
		Usage:   "key-add <key-name> <path/to/key/file.pub> [-f/--force]",
		Desc:    `Sends your public key to the git server used by tsuru.`,
		MinArgs: 2,
	}
}

func (c *keyAdd) Run(context *cmd.Context, client *cmd.Client) error {
	keyName := context.Args[0]
	keyPath := context.Args[1]
	key, err := c.readKey(context)
	if os.IsNotExist(err) {
		return fmt.Errorf("file %q doesn't exist", keyPath)
	} else if err != nil {
		return err
	}
	body := strings.Replace(key, "\n", "", -1)
	err = c.sendRequest(client, keyName, body, c.force)
	if err != nil {
		if e, ok := err.(*errors.HTTP); ok && e.Code == http.StatusConflict && !c.force {
			var answer string
			fmt.Fprintf(context.Stdout, "WARNING: key %q already exists.\nDo you want to replace it? (y/n) ", keyName)
			fmt.Fscan(context.Stdin, &answer)
			if answer == "y" || answer == "yes" {
				if err = c.sendRequest(client, keyName, body, true); err == nil {
					fmt.Fprintf(context.Stdout, "Key %q successfully replaced!\n", keyName)
					return nil
				}
			}
		}
		return err
	}
	fmt.Fprintf(context.Stdout, "Key %q successfully added!\n", keyName)
	return nil
}

func (c *keyAdd) sendRequest(client *cmd.Client, keyName, keyBody string, force bool) error {
	jsonBody := fmt.Sprintf(`{"key":%q,"name":%q,"force":%v}`, keyBody, keyName, force)
	url, err := cmd.GetURL("/users/keys")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, bytes.NewBufferString(jsonBody))
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	return err
}

func (c *keyAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("key-add", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.force, "force", false, "Force overriding the key if it already exists")
		c.fs.BoolVar(&c.force, "f", false, "Force overriding the key if it already exists")
	}
	return c.fs
}

type keyRemove struct {
	cmd.ConfirmationCommand
}

func (c *keyRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "key-remove",
		Usage: "key-remove <key-name> [-y/--assume-yes]",
		Desc: `Removes your public key from the git server used by tsuru. The key will be
removed from the current logged in user.`,
		MinArgs: 1,
	}
}

func (c *keyRemove) Run(context *cmd.Context, client *cmd.Client) error {
	name := context.Args[0]
	if !c.Confirm(context, fmt.Sprintf("Are you sure you want to remove key %q?", name)) {
		return nil
	}
	b := bytes.NewBufferString(fmt.Sprintf(`{"name":%q}`, name))
	url, err := cmd.GetURL("/users/keys")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, b)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Key %q successfully removed!\n", context.Args[0])
	return nil
}

type keyList struct {
	notrunc bool
	fs      *gnuflag.FlagSet
}

func (c *keyList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "key-list",
		Usage: "key-list [-n/--no-truncate]",
		Desc:  `Lists the public keys registered in the current user account.`,
	}
}

func (c *keyList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/users/keys")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var keys map[string]string
	err = json.NewDecoder(resp.Body).Decode(&keys)
	if err != nil {
		return err
	}
	var table cmd.Table
	table.Headers = cmd.Row{"Name", "Content"}
	table.LineSeparator = c.notrunc
	for name, content := range keys {
		row := []string{name, content}
		if !c.notrunc && len(row[1]) > keyTruncate {
			row[1] = row[1][:keyTruncate] + "..."
		}
		table.AddRow(cmd.Row(row))
	}
	table.SortByColumn(0)
	context.Stdout.Write(table.Bytes())
	return nil
}

func (c *keyList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("key-list", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.notrunc, "n", false, "disable truncation of key content")
		c.fs.BoolVar(&c.notrunc, "no-truncate", false, "disable truncation of key content")
	}
	return c.fs
}
