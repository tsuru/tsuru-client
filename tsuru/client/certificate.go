// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
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
	u, err := cmd.GetURLVersion("1.2", fmt.Sprintf("/apps/%s/certificate", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPut, u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Fprintln(context.Stdout, "Successfully created the certificated.")
	return nil
}

type CertificateUnset struct {
	cmd.GuessingCommand
	cname string
	fs    *gnuflag.FlagSet
}

func (c *CertificateUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "certificate-unset",
		Usage: "certificate-unset [-a/--app appname] [-c/--cname CNAME]",
		Desc:  `Unset a TLS certificate from a specific app.`,
	}
}

func (c *CertificateUnset) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		cname := "App CNAME"
		c.fs.StringVar(&c.cname, "cname", "", cname)
		c.fs.StringVar(&c.cname, "c", "", cname)
	}
	return c.fs
}

func (c *CertificateUnset) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	if c.cname == "" {
		return errors.New("You must set cname.")
	}
	v := url.Values{}
	v.Set("cname", c.cname)
	u, err := cmd.GetURLVersion("1.2", fmt.Sprintf("/apps/%s/certificate?%s", appName, v.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Fprintln(context.Stdout, "Certificate removed.")
	return nil
}

type CertificateList struct {
	cmd.GuessingCommand
	fs  *gnuflag.FlagSet
	raw bool
}

func (c *CertificateList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "certificate-list",
		Usage: "certificate-list [-a/--app appname] [-r/--raw]",
		Desc:  `List App TLS certificates.`,
	}
}

func (c *CertificateList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		c.fs.BoolVar(&c.raw, "r", false, "Display raw certificates")
		c.fs.BoolVar(&c.raw, "raw", false, "Display raw certificates")
	}
	return c.fs
}

func (c *CertificateList) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	u, err := cmd.GetURLVersion("1.2", fmt.Sprintf("/apps/%s/certificate", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	rawCerts := make(map[string]string)
	err = json.NewDecoder(response.Body).Decode(&rawCerts)
	if err != nil {
		return err
	}
	if c.raw {
		for n, rawCert := range rawCerts {
			if rawCert == "" {
				rawCert = "No certificate.\n"
			}
			fmt.Fprintf(context.Stdout, "%s:\n%s", n, rawCert)
		}
		return nil
	}
	tbl := cmd.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = cmd.Row{"CName", "Expires", "Issuer", "Subject"}
	dateFormat := "2006-01-02 15:04:05"
	for n, rawCert := range rawCerts {
		if rawCert == "" {
			tbl.AddRow(cmd.Row{n, "-", "-", "-"})
			continue
		}
		certBlock, _ := pem.Decode([]byte(rawCert))
		if certBlock == nil {
			tbl.AddRow(cmd.Row{n, "failed to decode data", "-", "-"})
			continue
		}
		cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			tbl.AddRow(cmd.Row{n, "failed to parse certificate data", "-", "-"})
			continue
		}
		tbl.AddRow(cmd.Row{n, cert.NotAfter.Local().Format(dateFormat),
			formatName(&cert.Issuer), formatName(&cert.Subject),
		})
	}
	tbl.Sort()
	fmt.Fprint(context.Stdout, tbl.String())
	return nil
}

func formatName(n *pkix.Name) string {
	country := strings.Join(n.Country, ",")
	state := strings.Join(n.Province, ",")
	locality := strings.Join(n.Locality, ",")
	org := strings.Join(n.Organization, ",")
	cname := n.CommonName
	return fmt.Sprintf("C=%s; ST=%s; \nL=%s; O=%s;\nCN=%s", country, state, locality, org, cname)
}
