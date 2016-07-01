// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/installer"
	"github.com/tsuru/tsuru/cmd"
)

type install struct {
	fs         *gnuflag.FlagSet
	driverName string
}

func (c *install) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "install",
		Usage:   "install",
		Desc:    "",
		MinArgs: 0,
	}
}

func (c *install) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("install", gnuflag.ExitOnError)
		c.fs.StringVar(&c.driverName, "driver", "virtualbox", "IaaS driver")
		c.fs.StringVar(&c.driverName, "d", "virtualbox", "IaaS driver")
	}
	return c.fs
}

func (c *install) Run(context *cmd.Context, client *cmd.Client) error {
	fmt.Println("Creating machine")
	i, err := installer.NewDockerMachine(c.driverName)
	if err != nil {
		fmt.Printf("Failed to create machine: %s\n", err)
		return err
	}
	m, err := i.CreateMachine(nil)
	if err != nil {
		fmt.Println("Error creating machine")
		return err
	}
	fmt.Printf("Machine %s successfully created!\n", m.Address)
	for _, component := range installer.TsuruComponents {
		fmt.Printf("Installing %s\n", component.Name())
		err := component.Install(m)
		if err != nil {
			fmt.Printf("Error installing %s\n", component.Name())
			return err
		}
		fmt.Printf("%s successfully installed!\n", component.Name())
	}
	return nil
}

type uninstall struct {
	fs         *gnuflag.FlagSet
	driverName string
}

func (c *uninstall) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "uninstall",
		Usage:   "uninstall",
		Desc:    "",
		MinArgs: 0,
	}
}

func (c *uninstall) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("uninstall", gnuflag.ExitOnError)
		c.fs.StringVar(&c.driverName, "driver", "virtualbox", "IaaS driver")
		c.fs.StringVar(&c.driverName, "d", "virtualbox", "IaaS driver")
	}
	return c.fs
}

func (c *uninstall) Run(context *cmd.Context, client *cmd.Client) error {
	d, err := installer.NewDockerMachine(c.driverName)
	if err != nil {
		fmt.Printf("Failed to delete machine: %s\n", err)
		return err
	}
	err = d.DeleteMachine(&installer.Machine{})
	if err != nil {
		return err
	}
	fmt.Println("Machine successfully removed!")
	return nil
}
