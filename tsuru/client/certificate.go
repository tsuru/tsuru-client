// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tablecli"
	tsuruClientApp "github.com/tsuru/tsuru-client/tsuru/app"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
)

type CertificateSet struct {
	tsuruClientApp.AppNameMixIn
	cname string
	fs    *pflag.FlagSet
}

func (c *CertificateSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "certificate-set",
		Usage:   "certificate set [-a/--app appname] [-c/--cname CNAME] [certificate] [key]",
		Desc:    `Creates or update a TLS certificate into the specific app.`,
		MinArgs: 2,
	}
}

func (c *CertificateSet) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		cname := "App CNAME"
		c.fs.StringVarP(&c.cname, "cname", "c", "", cname)
	}
	return c.fs
}

func (c *CertificateSet) Run(context *cmd.Context) error {
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	if c.cname == "" {
		return errors.New("you must set cname")
	}
	cert, err := os.ReadFile(context.Args[0])
	if err != nil {
		return err
	}
	key, err := os.ReadFile(context.Args[1])
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("cname", c.cname)
	v.Set("certificate", string(cert))
	v.Set("key", string(key))
	u, err := config.GetURLVersion("1.2", fmt.Sprintf("/apps/%s/certificate", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPut, u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Fprintln(context.Stdout, "Successfully created the certificate.")
	return nil
}

type CertificateUnset struct {
	tsuruClientApp.AppNameMixIn
	cname string
	fs    *pflag.FlagSet
}

func (c *CertificateUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "certificate-unset",
		Usage: "certificate unset [-a/--app appname] [-c/--cname CNAME]",
		Desc:  `Unset a TLS certificate from a specific app.`,
	}
}

func (c *CertificateUnset) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		cname := "App CNAME"
		c.fs.StringVarP(&c.cname, "cname", "c", "", cname)
	}
	return c.fs
}

func (c *CertificateUnset) Run(context *cmd.Context) error {
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	if c.cname == "" {
		return errors.New("you must set cname")
	}
	v := url.Values{}
	v.Set("cname", c.cname)
	u, err := config.GetURLVersion("1.2", fmt.Sprintf("/apps/%s/certificate?%s", appName, v.Encode()))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Fprintln(context.Stdout, "Certificate removed.")
	return nil
}

type CertificateList struct {
	tsuruClientApp.AppNameMixIn
	fs   *pflag.FlagSet
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

func (c *CertificateList) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.BoolVarP(&c.raw, "raw", "r", false, "Display raw certificates")
		c.fs.BoolVar(&c.json, standards.FlagJSON, false, "Display JSON format")

	}
	return c.fs
}

type cnameCertificate struct {
	Certificate string `json:"certificate"`
	Issuer      string `json:"issuer"`
}

type routerCertificate struct {
	CNameCertificates map[string]cnameCertificate `json:"cnames"`
}

type appCertificate struct {
	RouterCertificates map[string]routerCertificate `json:"routers"`
}

func (c *CertificateList) Run(context *cmd.Context) error {
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	u, err := config.GetURLVersion("1.24", fmt.Sprintf("/apps/%s/certificate", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	appCerts := appCertificate{}
	err = json.NewDecoder(response.Body).Decode(&appCerts)
	if err != nil {
		return err
	}

	if c.json {
		return c.renderJSON(context, appCerts)
	}

	if c.raw {
		for router, routerCerts := range appCerts.RouterCertificates {
			fmt.Fprintf(context.Stdout, "%s:\n", router)
			for cname, cnameCert := range routerCerts.CNameCertificates {
				if cnameCert.Certificate == "" {
					fmt.Fprintf(context.Stdout, "%s:\nNo certificate.", cname)
					continue
				}
				fmt.Fprintf(context.Stdout, "%s:\n%s", cname, cnameCert.Certificate)
			}
		}

		return nil
	}

	tbl := tablecli.NewTable()
	tbl.LineSeparator = true
	tbl.Headers = tablecli.Row{"Router", "CName", "Public Key Info", "Certificate Validity"}

	rows := []tablecli.Row{}

	for router, routerCerts := range appCerts.RouterCertificates {
		for cname, cnameCert := range routerCerts.CNameCertificates {
			var publicKeyInfo string
			var certificateValidity string
			var ready bool

			if cnameCert.Certificate != "" {
				cert, err := parseCert([]byte(cnameCert.Certificate))
				if err != nil {
					rows = append(rows, tablecli.Row{router, cname, err.Error(), "-"})
					continue
				}

				ready = true
				publicKeyInfo = formatPublicKeyInfo(*cert)
				certificateValidity = formatCertificateValidity(*cert)
			}

			rows = append(rows, tablecli.Row{
				router,
				formatCName(cname, cnameCert.Issuer, ready),
				publicKeyInfo,
				certificateValidity,
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i][0] == rows[j][0] {
			return rows[i][1] < rows[j][1]
		}
		return rows[i][0] < rows[j][0]
	})

	for _, row := range rows {
		tbl.AddRow(row)
	}

	tbl.Sort()
	fmt.Fprint(context.Stdout, tbl.String())
	return nil
}

func publicKeySize(publicKey interface{}) int {
	switch pk := publicKey.(type) {
	case *rsa.PublicKey:
		return pk.Size() * 8 // convert bytes to bits
	case *ecdsa.PublicKey:
		return pk.Params().BitSize
	}
	return 0
}

func formatCName(cname string, issuer string, ready bool) string {
	lines := []string{
		cname,
	}

	if issuer != "" {
		lines = append(lines, "  managed by: cert-manager", fmt.Sprintf("  issuer: %s", issuer))

		if !ready {
			lines = append(lines, "  status: not ready")
		}

	}

	return strings.Join(lines, "\n")
}

func formatPublicKeyInfo(cert x509.Certificate) (pkInfo string) {
	publicKey := cert.PublicKeyAlgorithm.String()
	if publicKey != "" {
		pkInfo += fmt.Sprintf("Algorithm\n%s\n\n", publicKey)
	}

	publicKeySize := publicKeySize(cert.PublicKey)
	if publicKeySize > 0 {
		pkInfo += fmt.Sprintf("Key size (in bits)\n%d", publicKeySize)
	}

	return
}

func formatCertificateValidity(cert x509.Certificate) string {
	return fmt.Sprintf(
		"Not before\n%s\n\nNot after\n%s",
		formatTime(cert.NotBefore),
		formatTime(cert.NotAfter),
	)
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func (c *CertificateList) renderJSON(context *cmd.Context, appCerts appCertificate) error {
	type certificateJSONFriendly struct {
		Router   string     `json:"router"`
		Domain   string     `json:"domain"`
		Raw      string     `json:"raw"`
		Issuer   *pkix.Name `json:"issuer"`
		Subject  *pkix.Name `json:"subject"`
		NotAfter string     `json:"notAfter"`
	}

	data := []certificateJSONFriendly{}

	for router, routerCerts := range appCerts.RouterCertificates {
		for cname, cnameCert := range routerCerts.CNameCertificates {
			item := certificateJSONFriendly{
				Domain: cname,
				Router: router,
				Raw:    cnameCert.Certificate,
			}

			parsedCert, err := parseCert([]byte(cnameCert.Certificate))
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

type CertificateIssuerSet struct {
	tsuruClientApp.AppNameMixIn
	cname string
	fs    *pflag.FlagSet
}

func (c *CertificateIssuerSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "certificate-issuer-set",
		Usage:   "certificate issuer set [-a/--app appname] [-c/--cname CNAME] [issuer]",
		Desc:    `Creates or update a certificate issuer into the specific app.`,
		MinArgs: 1,
	}
}

func (c *CertificateIssuerSet) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		cname := "App CNAME"
		c.fs.StringVarP(&c.cname, "cname", "c", "", cname)
	}
	return c.fs
}

func (c *CertificateIssuerSet) Run(context *cmd.Context) error {
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}

	if c.cname == "" {
		return errors.New("you must set cname")
	}

	issuer := context.Args[0]
	if issuer == "" {
		return errors.New("you must set issuer")
	}

	v := url.Values{}
	v.Set("cname", c.cname)
	v.Set("issuer", issuer)
	u, err := config.GetURLVersion("1.24", fmt.Sprintf("/apps/%s/certissuer", appName))
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPut, u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	fmt.Fprintln(context.Stdout, "Successfully created the certificate issuer.")
	return nil
}

type CertificateIssuerUnset struct {
	tsuruClientApp.AppNameMixIn
	cmd.ConfirmationCommand
	fs    *pflag.FlagSet
	cname string
}

func (c *CertificateIssuerUnset) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "certificate-issuer-unset",
		Usage: "certificate issuer unset [-a/--app appname] [-c/--cname CNAME] [-y/--assume-yes]",
		Desc:  `Unset a certificate issuer from a specific app.`,
	}
}

func (c *CertificateIssuerUnset) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = mergeFlagSet(
			c.AppNameMixIn.Flags(),
			c.ConfirmationCommand.Flags(),
		)

		cname := "App CNAME"
		c.fs.StringVarP(&c.cname, "cname", "c", "", cname)
	}
	return c.fs
}

func (c *CertificateIssuerUnset) Run(context *cmd.Context) error {
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}

	if c.cname == "" {
		return errors.New("you must set cname")
	}

	if !c.Confirm(context, fmt.Sprintf(`Are you sure you want to remove certificate issuer for cname: "%s"?`, c.cname)) {
		return nil
	}

	v := url.Values{}
	v.Set("cname", c.cname)
	u, err := config.GetURLVersion("1.24", fmt.Sprintf("/apps/%s/certissuer?%s", appName, v.Encode()))
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	fmt.Fprintln(context.Stdout, "Certificate issuer removed.")
	return nil
}
