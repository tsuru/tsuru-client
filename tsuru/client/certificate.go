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
	"sort"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
)

type CertificateSet struct {
	cmd.AppNameMixIn
	cname string
	fs    *gnuflag.FlagSet
}

func (c *CertificateSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "certificate-set",
		Usage:   "certificate set [-a/--app appname] [-c/--cname CNAME] [certificate] [key]",
		Desc:    `Creates or update a TLS certificate into the specific app.`,
		MinArgs: 2,
	}
}

func (c *CertificateSet) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		cname := "App CNAME"
		c.fs.StringVar(&c.cname, "cname", "", cname)
		c.fs.StringVar(&c.cname, "c", "", cname)
	}
	return c.fs
}

func (c *CertificateSet) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
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
	cmd.AppNameMixIn
	cname string
	fs    *gnuflag.FlagSet
}

func (c *CertificateUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "certificate-unset",
		Usage: "certificate unset [-a/--app appname] [-c/--cname CNAME]",
		Desc:  `Unset a TLS certificate from a specific app.`,
	}
}

func (c *CertificateUnset) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		cname := "App CNAME"
		c.fs.StringVar(&c.cname, "cname", "", cname)
		c.fs.StringVar(&c.cname, "c", "", cname)
	}
	return c.fs
}

func (c *CertificateUnset) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
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
	cmd.AppNameMixIn
	fs   *gnuflag.FlagSet
	raw  bool
	json bool
}

func (c *CertificateList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "certificate-list",
		Usage: "certificate list [-a/--app appname] [-r/--raw]",
		Desc:  `List App TLS certificates.`,
	}
}

func (c *CertificateList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVar(&c.raw, "r", false, "Display raw certificates")
		c.fs.BoolVar(&c.raw, "raw", false, "Display raw certificates")
		c.fs.BoolVar(&c.json, "json", false, "Display JSON format")

	}
	return c.fs
}

func (c *CertificateList) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
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
	rawCerts := make(map[string]map[string]string)
	err = json.NewDecoder(response.Body).Decode(&rawCerts)
	if err != nil {
		return err
	}

	if c.json {
		return c.renderJSON(context, rawCerts)
	}

	routerNames := []string{}
	routerMap := make(map[string][]string)
	for k := range rawCerts {
		routerNames = append(routerNames, k)
		for v := range rawCerts[k] {
			routerMap[k] = append(routerMap[k], v)
		}
	}
	sort.Strings(routerNames)
	for k := range routerMap {
		sort.Strings(routerMap[k])
	}

	if c.raw {
		for _, r := range routerNames {
			fmt.Fprintf(context.Stdout, "%s:\n", r)
			for n, rawCert := range rawCerts[r] {
				if rawCert == "" {
					rawCert = "No certificate.\n"
				}
				fmt.Fprintf(context.Stdout, "%s:\n%s", n, rawCert)
			}
		}
		return nil
	}
	tbl := tablecli.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = tablecli.Row{"Router", "CName", "Expires", "Issuer", "Subject"}
	dateFormat := "2006-01-02 15:04:05"
	for r, cnames := range routerMap {
		for _, n := range cnames {
			rawCert := rawCerts[r][n]
			if rawCert == "" {
				tbl.AddRow(tablecli.Row{r, n, "-", "-", "-"})
				continue
			}
			cert, err := parseCert([]byte(rawCert))
			if err != nil {
				tbl.AddRow(tablecli.Row{r, n, err.Error(), "-", "-"})
				continue
			}
			tbl.AddRow(tablecli.Row{r, n, formatter.Local(cert.NotAfter).Format(dateFormat),
				formatName(&cert.Issuer), formatName(&cert.Subject),
			})
		}
	}
	tbl.Sort()
	fmt.Fprint(context.Stdout, tbl.String())
	return nil
}

func (c *CertificateList) renderJSON(context *cmd.Context, rawCerts map[string]map[string]string) error {
	type certificateJSONFriendly struct {
		Router   string     `json:"router"`
		Domain   string     `json:"domain"`
		Raw      string     `json:"raw"`
		Issuer   *pkix.Name `json:"issuer"`
		Subject  *pkix.Name `json:"subject"`
		NotAfter string     `json:"notAfter"`
	}

	data := []certificateJSONFriendly{}

	for router, domainMap := range rawCerts {
	domainLoop:
		for domain, raw := range domainMap {
			if raw == "" {
				continue domainLoop
			}
			item := certificateJSONFriendly{
				Domain: domain,
				Router: router,
				Raw:    raw,
			}

			parsedCert, err := parseCert([]byte(raw))
			if err == nil {
				item.Issuer = &parsedCert.Issuer
				item.Subject = &parsedCert.Subject
				item.NotAfter = formatter.Local(parsedCert.NotAfter).Format("2006-01-02 15:04:05")
			}

			data = append(data, item)
		}
	}

	return formatter.JSON(context.Stdout, data)
}

func parseCert(data []byte) (*x509.Certificate, error) {
	certBlock, _ := pem.Decode(data)
	if certBlock == nil {
		return nil, errors.New("failed to decode data")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse certificate data")
	}

	return cert, nil
}

func formatName(n *pkix.Name) string {
	country := strings.Join(n.Country, ",")
	state := strings.Join(n.Province, ",")
	locality := strings.Join(n.Locality, ",")
	org := strings.Join(n.Organization, ",")
	cname := n.CommonName
	return fmt.Sprintf("C=%s; ST=%s; \nL=%s; O=%s;\nCN=%s", country, state, locality, org, cname)
}
