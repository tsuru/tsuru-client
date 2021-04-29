// Copyright 2018 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
)

type mapSliceFlagWrapper struct {
	dst *map[string][]string
}

func (f mapSliceFlagWrapper) String() string {
	repr := *f.dst
	if repr == nil {
		repr = map[string][]string{}
	}
	data, _ := json.Marshal(repr)
	return string(data)
}

func (f mapSliceFlagWrapper) Set(val string) error {
	parts := strings.SplitN(val, "=", 2)
	if *f.dst == nil {
		*f.dst = map[string][]string{}
	}
	if len(parts) < 2 {
		return stdErrors.New("must be on the form \"key=value\"")
	}
	(*f.dst)[parts[0]] = append((*f.dst)[parts[0]], parts[1])
	return nil
}

func flagsForWebhook(webhook *tsuru.Webhook) *gnuflag.FlagSet {
	fs := gnuflag.NewFlagSet("", gnuflag.ExitOnError)

	description := "A description on how the webhook will be used."
	fs.StringVar(&webhook.Description, "description", "", description)
	fs.StringVar(&webhook.Description, "d", "", description)

	team := "The team name responsible for this webhook."
	fs.StringVar(&webhook.TeamOwner, "team", "", team)
	fs.StringVar(&webhook.TeamOwner, "t", "", team)

	method := "The HTTP Method used in the request, if unset defaults to POST."
	fs.StringVar(&webhook.Method, "method", "", method)
	fs.StringVar(&webhook.Method, "m", "", method)

	body := "The HTTP body sent in the request if method is either POST, PUT or PATCH, if unset defaults to the Event that triggered the webhook serialized as JSON. The API will try to parse the body as a Go template string with the event available as context."
	fs.StringVar(&webhook.Body, "body", "", body)
	fs.StringVar(&webhook.Body, "b", "", body)

	proxy := "The proxy server URL used in the request. Supported schemes are http(s) and socks5."
	fs.StringVar(&webhook.ProxyUrl, "proxy", "", proxy)

	header := "The HTTP headers sent in the request."
	wrapper := mapSliceFlagWrapper{dst: &webhook.Headers}
	fs.Var(wrapper, "H", header)
	fs.Var(wrapper, "header", header)

	fs.BoolVar(&webhook.Insecure, "insecure", false, "Ignore TLS errors in the webhook request.")

	webhook.EventFilter = tsuru.WebhookEventFilter{}
	fs.Var(cmd.StringSliceFlagWrapper{Dst: &webhook.EventFilter.TargetTypes}, "target-type",
		"Target Type for matching events.")
	fs.Var(cmd.StringSliceFlagWrapper{Dst: &webhook.EventFilter.TargetValues}, "target-value",
		"Target Value for matching events.")
	fs.Var(cmd.StringSliceFlagWrapper{Dst: &webhook.EventFilter.KindTypes}, "kind-type",
		"Kind Type for matching events.")
	fs.Var(cmd.StringSliceFlagWrapper{Dst: &webhook.EventFilter.KindNames}, "kind-name",
		"Kind Name for matching events.")
	fs.BoolVar(&webhook.EventFilter.ErrorOnly, "error-only", false, "Only matches events with error.")
	fs.BoolVar(&webhook.EventFilter.SuccessOnly, "success-only", false, "Only matches events with success.")
	return fs
}

type WebhookCreate struct {
	fs      *gnuflag.FlagSet
	webhook tsuru.Webhook
}

func (c *WebhookCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-webhook-create",
		Usage:   "event webhook create <name> <url> [-d/--description <description>] [-t/--team <team>] [-m/--method <method>] [-b/--body <body>] [--proxy <url>] [-H/--header <name=value>]... [--insecure] [--error-only] [--success-only] [--target-type <type>]... [--target-value <value>]... [--kind-type <type>]... [--kind-name <name>]...",
		Desc:    `Creates a new webhook triggered when an event matches.`,
		MinArgs: 2,
	}
}

func (c *WebhookCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = flagsForWebhook(&c.webhook)
	}
	return c.fs
}

func (c *WebhookCreate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	c.webhook.Name, c.webhook.Url = ctx.Args[0], ctx.Args[1]
	_, err = apiClient.EventApi.WebhookCreate(context.TODO(), c.webhook)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Webhook successfully created.")
	return nil
}

type WebhookUpdate struct {
	fs            *gnuflag.FlagSet
	webhook       tsuru.Webhook
	noBody        bool
	noHeader      bool
	noInsecure    bool
	noTargetType  bool
	noTargetValue bool
	noKindType    bool
	noKindName    bool
	noErrorOnly   bool
	noSuccessOnly bool
}

func (c *WebhookUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-webhook-update",
		Usage:   "event webhook update <name> [-u/--url <url>] [-d/--description <description>] [-t/--team <team>] [-m/--method <method>] [-b/--body <body>] [--proxy <url>] [-H/--header <name=value>]... [--insecure] [--error-only] [--success-only] [--target-type <type>]... [--target-value <value>]... [--kind-type <type>]... [--kind-name <name>]... [--no-body] [--no-header] [--no-insecure] [--no-target-type] [--no-target-value] [--no-kind-type] [--no-kind-name] [--no-error-only] [--no-success-only]",
		Desc:    `Updates an existing webhook.`,
		MinArgs: 1,
	}
}

func (c *WebhookUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = flagsForWebhook(&c.webhook)

		url := "The HTTP URL used in the request."
		c.fs.StringVar(&c.webhook.Url, "url", "", url)
		c.fs.StringVar(&c.webhook.Url, "u", "", url)

		c.fs.BoolVar(&c.noBody, "no-body", false, "Unset body value.")
		c.fs.BoolVar(&c.noHeader, "no-header", false, "Unset header value.")
		c.fs.BoolVar(&c.noInsecure, "no-insecure", false, "Unset insecure value.")
		c.fs.BoolVar(&c.noTargetType, "no-target-type", false, "Unset Target Type for matching events.")
		c.fs.BoolVar(&c.noTargetValue, "no-target-value", false, "Unset Target Value for matching events.")
		c.fs.BoolVar(&c.noKindType, "no-kind-type", false, "Unset Kind Type for matching events.")
		c.fs.BoolVar(&c.noKindName, "no-kind-name", false, "Unset Kind Name for matching events.")
		c.fs.BoolVar(&c.noErrorOnly, "no-error-only", false, "Unset only matches events with error.")
		c.fs.BoolVar(&c.noSuccessOnly, "no-success-only", false, "Unset only matches events with success.")
	}
	return c.fs
}

func (c *WebhookUpdate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	name := ctx.Args[0]
	webhook, _, err := apiClient.EventApi.WebhookGet(context.TODO(), name)
	if err != nil {
		return err
	}
	toUpdate := c.mergeWebhooks(webhook)
	_, err = apiClient.EventApi.WebhookUpdate(context.TODO(), name, toUpdate)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Webhook successfully updated.")
	return nil
}

func (c *WebhookUpdate) mergeWebhooks(existing tsuru.Webhook) tsuru.Webhook {
	new := c.webhook
	if new.Url != "" {
		existing.Url = new.Url
	}
	if new.Description != "" {
		existing.Description = new.Description
	}
	if new.Method != "" {
		existing.Method = new.Method
	}
	if new.TeamOwner != "" {
		existing.TeamOwner = new.TeamOwner
	}
	if new.Body != "" {
		existing.Body = new.Body
	} else if c.noBody {
		existing.Body = ""
	}
	if new.Insecure {
		existing.Insecure = new.Insecure
	} else if c.noInsecure {
		existing.Insecure = false
	}
	if len(new.Headers) != 0 {
		existing.Headers = new.Headers
	} else if c.noHeader {
		existing.Headers = nil
	}
	if len(new.EventFilter.KindNames) != 0 {
		existing.EventFilter.KindNames = new.EventFilter.KindNames
	} else if c.noKindName {
		existing.EventFilter.KindNames = nil
	}
	if len(new.EventFilter.KindTypes) != 0 {
		existing.EventFilter.KindTypes = new.EventFilter.KindTypes
	} else if c.noKindType {
		existing.EventFilter.KindTypes = nil
	}
	if len(new.EventFilter.TargetTypes) != 0 {
		existing.EventFilter.TargetTypes = new.EventFilter.TargetTypes
	} else if c.noTargetType {
		existing.EventFilter.TargetTypes = nil
	}
	if len(new.EventFilter.TargetValues) != 0 {
		existing.EventFilter.TargetValues = new.EventFilter.TargetValues
	} else if c.noTargetValue {
		existing.EventFilter.TargetValues = nil
	}
	if new.EventFilter.ErrorOnly {
		existing.EventFilter.ErrorOnly = new.EventFilter.ErrorOnly
	} else if c.noErrorOnly {
		existing.EventFilter.ErrorOnly = false
	}
	if new.EventFilter.SuccessOnly {
		existing.EventFilter.SuccessOnly = new.EventFilter.SuccessOnly
	} else if c.noSuccessOnly {
		existing.EventFilter.SuccessOnly = false
	}
	return existing
}

type WebhookList struct {
}

func (c *WebhookList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "event-webhook-list",
		Usage: "event webhook list",
		Desc:  `List existing webhooks.`,
	}
}

func (c *WebhookList) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	webhooks, rsp, err := apiClient.EventApi.WebhookList(context.TODO())
	if err != nil {
		if rsp != nil && rsp.StatusCode == http.StatusNoContent {
			return nil
		}
		return err
	}
	tbl := tablecli.Table{
		Headers:       tablecli.Row{"Name", "Description", "Team", "URL", "Headers", "Body", "Insecure", "Filters"},
		LineSeparator: true,
	}
	for _, w := range webhooks {
		url := w.Url
		if w.Method != "" {
			url = fmt.Sprintf("%s %s", w.Method, url)
		}
		var headers []string
		for k, vals := range w.Headers {
			for _, v := range vals {
				headers = append(headers, fmt.Sprintf("%s: %s", k, v))
			}
		}
		sort.Strings(headers)
		body := w.Body
		if body == "" {
			body = "<event>"
		}
		tbl.AddRow(tablecli.Row{
			w.Name,
			w.Description,
			w.TeamOwner,
			url,
			strings.Join(headers, "\n"),
			body,
			strconv.FormatBool(w.Insecure),
			filterToStr(w.EventFilter),
		})
	}
	fmt.Fprint(ctx.Stdout, tbl.String())
	return nil
}

func filterToStr(f tsuru.WebhookEventFilter) string {
	var strs []string
	for _, v := range f.KindTypes {
		strs = append(strs, fmt.Sprintf("kind-type == %s", v))
	}
	for _, v := range f.KindNames {
		strs = append(strs, fmt.Sprintf("kind-name == %s", v))
	}
	for _, v := range f.TargetTypes {
		strs = append(strs, fmt.Sprintf("target-type == %s", v))
	}
	for _, v := range f.TargetValues {
		strs = append(strs, fmt.Sprintf("target-value == %s", v))
	}
	if f.SuccessOnly {
		strs = append(strs, "success-only")
	}
	if f.ErrorOnly {
		strs = append(strs, "error-only")
	}
	return strings.Join(strs, "\n")
}

type WebhookDelete struct{}

func (c *WebhookDelete) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "event-webhook-delete",
		Usage:   "event webhook delete <name>",
		Desc:    `Deletes an existing webhook.`,
		MinArgs: 1,
	}
}

func (c *WebhookDelete) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	name := ctx.Args[0]
	_, err = apiClient.EventApi.WebhookDelete(context.TODO(), name)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Webhook successfully deleted.")
	return nil
}
