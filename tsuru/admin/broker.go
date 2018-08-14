package admin

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru/cmd"
)

type BrokerAdd struct {
	broker tsuru.ServiceBroker
	fs     *gnuflag.FlagSet
}

func (c *BrokerAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-broker-add",
		Usage:   "service-broker-add <name> <url> [-i/--insecure] [-c/--context key=value] [-t/--token token] [-u/--user username] [-p/--password password]",
		Desc:    `Adds a new Service Broker.`,
		MinArgs: 2,
	}
}

func (c *BrokerAdd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = flagsForServiceBroker(&c.broker)
	}
	return c.fs
}

func (c *BrokerAdd) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	c.broker.Name, c.broker.URL = ctx.Args[0], ctx.Args[1]
	_, err = apiClient.ServiceApi.ServiceBrokerCreate(context.TODO(), c.broker)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Service broker successfully added.")
	return nil
}

type BrokerUpdate struct {
	broker tsuru.ServiceBroker
	fs     *gnuflag.FlagSet
}

func (c *BrokerUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-broker-update",
		Usage:   "service-broker-update <name> <url> [-i/--insecure] [-c/--context key=value] [-t/--token token] [-u/--user username] [-p/--password password]",
		Desc:    `Updates a service broker.`,
		MinArgs: 2,
	}
}

func (c *BrokerUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = flagsForServiceBroker(&c.broker)
	}
	return c.fs
}

func (c *BrokerUpdate) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	c.broker.Name, c.broker.URL = ctx.Args[0], ctx.Args[1]
	_, err = apiClient.ServiceApi.ServiceBrokerUpdate(context.TODO(), c.broker.Name, c.broker)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Service broker successfully updated.")
	return nil
}

type BrokerDelete struct{}

func (c *BrokerDelete) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "service-broker-delete",
		Usage:   "service-broker-delete <name>",
		Desc:    `Removes a service broker.`,
		MinArgs: 1,
	}
}

func (c *BrokerDelete) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	_, err = apiClient.ServiceApi.ServiceBrokerDelete(context.TODO(), ctx.Args[0])
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Service broker successfully deleted.")
	return nil
}

type BrokerList struct{}

func (c *BrokerList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-broker-list",
		Usage: "service-broker-list",
		Desc:  `List service brokers.`,
	}
}

func (c *BrokerList) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	brokerList, _, err := apiClient.ServiceApi.ServiceBrokerList(context.TODO())
	if err != nil {
		return err
	}
	tbl := tablecli.Table{
		Headers:       tablecli.Row{"Name", "URL", "Insecure", "Auth", "Context"},
		LineSeparator: true,
	}
	for _, b := range brokerList.Brokers {
		authMethod := "None"
		if b.Config.AuthConfig != nil {
			if b.Config.AuthConfig.BasicAuthConfig != nil {
				authMethod = "Basic\n"
			} else if b.Config.AuthConfig.BearerConfig != nil {
				authMethod = "Bearer\n"
			}
		}
		var contexts []string
		for k, v := range b.Config.Context {
			contexts = append(contexts, fmt.Sprintf("%v: %v", k, v))
		}
		sort.Strings(contexts)
		tbl.AddRow(tablecli.Row{
			b.Name,
			b.URL,
			strconv.FormatBool(b.Config.Insecure),
			authMethod,
			strings.Join(contexts, "\n"),
		})
	}
	fmt.Fprint(ctx.Stdout, tbl.String())
	return nil
}

func flagsForServiceBroker(broker *tsuru.ServiceBroker) *gnuflag.FlagSet {
	fs := gnuflag.NewFlagSet("", gnuflag.ExitOnError)

	if broker.Config == nil {
		broker.Config = &tsuru.ServiceBrokerConfig{
			AuthConfig: &tsuru.ServiceBrokerConfigAuthConfig{
				BasicAuthConfig: &tsuru.ServiceBrokerConfigAuthConfigBasicAuthConfig{},
				BearerConfig:    &tsuru.ServiceBrokerConfigAuthConfigBearerConfig{},
			},
		}
	}

	insecure := "Ignore TLS errors in the broker request."
	fs.BoolVar(&broker.Config.Insecure, "insecure", false, insecure)
	fs.BoolVar(&broker.Config.Insecure, "i", false, insecure)

	context := "Context values to be sent on every broker request."
	fs.Var(cmd.MapFlagWrapper{Dst: &broker.Config.Context}, "context", context)
	fs.Var(cmd.MapFlagWrapper{Dst: &broker.Config.Context}, "c", context)

	pass := "Service broker authentication password."
	fs.StringVar(&broker.Config.AuthConfig.BasicAuthConfig.Password, "password", "", pass)
	fs.StringVar(&broker.Config.AuthConfig.BasicAuthConfig.Password, "p", "", pass)

	user := "Service broker authentication username."
	fs.StringVar(&broker.Config.AuthConfig.BasicAuthConfig.Username, "user", "", user)
	fs.StringVar(&broker.Config.AuthConfig.BasicAuthConfig.Username, "u", "", user)

	token := "Service broker authentication token."
	fs.StringVar(&broker.Config.AuthConfig.BearerConfig.Token, "token", "", token)
	fs.StringVar(&broker.Config.AuthConfig.BearerConfig.Token, "t", "", token)

	return fs
}
