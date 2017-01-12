// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
)

type CertificateSet struct {
	cmd.GuessingCommand
	cname string
	fs    *gnuflag.FlagSet
}

func (c *CertificateSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "certificate-set",
		Usage:   "certificate-set [-a/--app appname] [-c/--cname CNAME] [certificate] [key]",
		Desc:    `Creates or update a TLS certificate into the specific app.`,
		MinArgs: 2,
	}
}

func (c *CertificateSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		cname := "App CNAME"
		c.fs.StringVar(&c.cname, "cname", "", cname)
		c.fs.StringVar(&c.cname, "c", "", cname)
	}
	return c.fs
}

func (c *CertificateSet) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	if c.cname == "" {
		return errors.New("You must set cname.")
	}
	cert, err := ioutil.ReadFile(context.Args[0])
	if err != nil {
		return err
	}
	key, err := ioutil.ReadFile(context.Args[1])
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("cname", c.cname)
	v.Set("certificate", string(cert))
	v.Set("key", string(key))
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/certificate", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Fprintln(context.Stdout, "Succesfully created the certificated.")
	return nil
}
