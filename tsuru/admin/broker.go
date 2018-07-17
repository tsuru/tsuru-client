package admin

import (
	"context"
	"fmt"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
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
