// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
)

type Install struct {
	fs         *gnuflag.FlagSet
	driverName string
}

func (c *Install) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "install",
		Usage:   "install",
		Desc:    "",
		MinArgs: 0,
	}
}

func (c *Install) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("install", gnuflag.ExitOnError)
		c.fs.StringVar(&c.driverName, "driver", "virtualbox", "IaaS driver")
		c.fs.StringVar(&c.driverName, "d", "virtualbox", "IaaS driver")
	}
	return c.fs
}

func (c *Install) Run(context *cmd.Context, client *cmd.Client) error {
	fmt.Fprintln(context.Stdout, "Creating machine")
	opts := parseKeyValue(context.Args)
	i, err := NewDockerMachine(c.driverName, opts)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to create machine: %s\n", err)
		return err
	}
	m, err := i.CreateMachine(opts)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Error creating machine: %s\n", err)
		return err
	}
	fmt.Fprintf(context.Stdout, "Machine %s successfully created!\n", m.IP)
	for _, component := range TsuruComponents {
		fmt.Fprintf(context.Stdout, "Installing %s\n", component.Name())
		err := component.Install(m)
		if err != nil {
			fmt.Fprintf(context.Stderr, "Error Installing %s: %s\n", component.Name(), err)
			return err
		}
		fmt.Fprintf(context.Stdout, "%s successfully installed!\n", component.Name())
	}
	return nil
}

type Uninstall struct {
	fs         *gnuflag.FlagSet
	driverName string
}

func (c *Uninstall) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "uninstall",
		Usage:   "uninstall",
		Desc:    "",
		MinArgs: 0,
	}
}

func (c *Uninstall) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("uninstall", gnuflag.ExitOnError)
		c.fs.StringVar(&c.driverName, "driver", "virtualbox", "IaaS driver")
		c.fs.StringVar(&c.driverName, "d", "virtualbox", "IaaS driver")
	}
	return c.fs
}

func (c *Uninstall) Run(context *cmd.Context, client *cmd.Client) error {
	d, err := NewDockerMachine(c.driverName, parseKeyValue(context.Args))
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machine: %s\n", err)
		return err
	}
	err = d.DeleteMachine(&Machine{})
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machine: %s\n", err)
		return err
	}
	fmt.Fprintln(context.Stdout, "Machine successfully removed!")
	return nil
}

func parseKeyValue(args []string) map[string]interface{} {
	opts := make(map[string]interface{})
	for _, arg := range args {
		if strings.Contains(arg, "=") {
			keyValue := strings.SplitN(arg, "=", 2)
			opts[keyValue[0]] = keyValue[1]
		}
	}
	return opts
}
