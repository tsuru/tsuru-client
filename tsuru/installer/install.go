// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tsuru/config"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
)

type Install struct {
	fs     *gnuflag.FlagSet
	config string
}

func (c *Install) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "install",
		Usage: "install [--config/-c config_file]",
		Desc: `Installs Tsuru and It's components as containers on hosts provisioned
with docker machine drivers.

The [[--config]] parameter is the path to a .yml file containing the installation
configuration. If not provided, Tsuru will be installed into a VirtualBox VM for
experimentation.

The following is an example of installation configuration to install Tsuru on
Amazon EC2:

==========
name: tsuru-ec2
driver:
    name: amazonec2
    options:
        amazonec2-access-key: myAmazonAccessKey
        amazonec2-secret-key: myAmazonSecretKey
        amazonec2-vpc-id: vpc-abc1234
        amazonec2-subnet-id: subnet-abc1234
==========

Available configuration parameters:

- name
Name of the installation.

- docker-hub-mirror
Url of a docker hub mirror used to fetch the components docker images.

- ca-path
A path to a directory containing a ca.pem and ca-key.pem files that are going to be used to sign certificates used by docker and docker registry.
If not set, a CA will be created, copied to every host provisioned and used to sign the certificates.

- driver
Under this namespace lies all the docker machine driver configuration.

- driver:name
Name of the driver to be used by the installer. This can be any core or 3rd party driver supported by docker machine. If a 3rd party driver name is used, it's binary must be available on the user path.

- driver:options
Under this namespace every driver parameters can be set. Refer to the driver configuration for more information on what parameter are available.
`,
		MinArgs: 0,
	}
}

func (c *Install) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("install", gnuflag.ExitOnError)
		c.fs.StringVar(&c.config, "c", "", "Configuration file")
		c.fs.StringVar(&c.config, "config", "", "Configuration file")
	}
	return c.fs
}

func (c *Install) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	config, err := parseConfigFile(c.config)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Running pre install checks...\n")
	err = c.PreInstallChecks(config)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Pre Install checks failed: %s\n", err)
		return err
	}
	i, err := NewDockerMachine(config)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to create machine: %s\n", err)
		return err
	}
	defer i.Close()
	m, err := i.CreateMachine([]string{strconv.Itoa(defaultTsuruAPIPort)})
	if err != nil {
		fmt.Fprintf(context.Stderr, "Error creating machine: %s\n", err)
		return err
	}
	err = i.uploadRegistryCertificate(m)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Machine %s successfully created!\n", m.IP)
	installConfig := NewInstallConfig(config.Name)
	for _, component := range TsuruComponents {
		fmt.Fprintf(context.Stdout, "Installing %s\n", component.Name())
		err := component.Install(m, installConfig)
		if err != nil {
			fmt.Fprintf(context.Stderr, "Error Installing %s: %s\n", component.Name(), err)
			return err
		}
		fmt.Fprintf(context.Stdout, "%s successfully installed!\n", component.Name())
	}
	fmt.Fprint(context.Stdout, c.buildStatusTable(TsuruComponents, m).String())
	return nil
}

func (c *Install) PreInstallChecks(config *DockerMachineConfig) error {
	exists, err := cmd.CheckIfTargetLabelExists(config.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tsuru target \"%s\" already exists", config.Name)
	}
	return nil
}

func (c *Install) buildStatusTable(components []TsuruComponent, m *Machine) *cmd.Table {
	t := cmd.NewTable()
	t.Headers = cmd.Row{"Component", "Address", "State"}
	t.LineSeparator = true
	for _, component := range components {
		status, err := component.Status(m)
		if err != nil {
			t.AddRow(cmd.Row{component.Name(), "", fmt.Sprintf("%s", err)})
			continue
		}
		addresses := strings.Join(status.addresses, "\n")
		t.AddRow(cmd.Row{component.Name(), addresses, status.containerState.StateString()})
	}
	return t
}

type Uninstall struct {
	fs     *gnuflag.FlagSet
	config string
}

func (c *Uninstall) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "uninstall",
		Usage:   "uninstall [--config/-c config_file]",
		Desc:    "Uninstalls Tsuru and It's components.",
		MinArgs: 0,
	}
}

func (c *Uninstall) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("uninstall", gnuflag.ExitOnError)
		c.fs.StringVar(&c.config, "c", "", "Configuration file")
		c.fs.StringVar(&c.config, "config", "", "Configuration file")
	}
	return c.fs
}

func (c *Uninstall) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	config, err := parseConfigFile(c.config)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to read configuration file: %s\n", err)
		return err
	}
	d, err := NewDockerMachine(config)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machine: %s\n", err)
		return err
	}
	defer d.Close()
	err = d.DeleteMachine(d.Name)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machine: %s\n", err)
		return err
	}
	fmt.Fprintln(context.Stdout, "Machine successfully removed!")
	api := TsuruAPI{}
	err = api.Uninstall(config.Name)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to uninstall tsuru API: %s\n", err)
		return err
	}
	err = os.RemoveAll(d.storePath)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete installation directory: %s\n", err)
		return err
	}
	fmt.Fprintf(context.Stdout, "Uninstall finished successfully!\n")
	return nil
}

func parseConfigFile(file string) (*DockerMachineConfig, error) {
	dmConfig := defaultDockerMachineConfig
	if file == "" {
		return dmConfig, nil
	}
	err := config.ReadConfigFile(file)
	if err != nil {
		return nil, err
	}
	driverName, err := config.GetString("driver:name")
	if err == nil {
		dmConfig.DriverName = driverName
	}
	name, err := config.GetString("name")
	if err == nil {
		dmConfig.Name = name
	}
	driverOpts := make(map[string]interface{})
	opts, _ := config.Get("driver:options")
	if opts != nil {
		for k, v := range opts.(map[interface{}]interface{}) {
			switch k := k.(type) {
			case string:
				driverOpts[k] = v
			}
		}
		dmConfig.DriverOpts = driverOpts
	}
	caPath, err := config.GetString("ca-path")
	if err == nil {
		dmConfig.CAPath = caPath
	}
	return dmConfig, nil
}
