// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/tsuru/config"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/admin"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru-client/tsuru/installer/defaultconfig"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	"gopkg.in/yaml.v2"
)

type Install struct {
	fs      *gnuflag.FlagSet
	config  string
	compose string
}

func (c *Install) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "install-create",
		Usage: "install-create [--config/-c config_file] [--compose/-e compose_file]",
		Desc: `Installs Tsuru and It's components as containers on hosts provisioned
with docker machine drivers.

The [[--config]] parameter is the path to a .yml file containing the installation
configuration. If not provided, Tsuru will be installed into a VirtualBox VM for
experimentation.
`,
		MinArgs: 0,
	}
}

func (c *Install) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("install", gnuflag.ExitOnError)
		c.fs.StringVar(&c.config, "c", "", "Configuration file")
		c.fs.StringVar(&c.config, "config", "", "Configuration file")
		c.fs.StringVar(&c.compose, "e", "", "Components docker-compose file")
		c.fs.StringVar(&c.compose, "compose", "", "Components docker-compose file")
	}
	return c.fs
}

func (c *Install) Run(context *cmd.Context, cli *cmd.Client) error {
	context.RawOutput()
	installConfig, err := parseConfigFile(c.config)
	if err != nil {
		return err
	}
	installConfig.ComposeFile = c.compose
	dockerMachine, err := dm.NewDockerMachine(installConfig.DockerMachineConfig, installConfig.Name)
	if err != nil {
		return err
	}
	defer dockerMachine.Close()
	newSwarmServiceCluster := func(machines []*dockermachine.Machine) (ServiceCluster, error) {
		swarm, errNew := NewSwarmCluster(machines)
		return swarm, errNew
	}
	installer := &Installer{
		outWriter:          context.Stdout,
		errWriter:          context.Stderr,
		machineProvisioner: dockerMachine,
		bootstraper:        &TsuruBoostraper{},
		clusterCreator:     newSwarmServiceCluster,
	}
	installation, err := installer.Install(installConfig)
	if err != nil {
		return err
	}
	err = addInstallHosts(installation.InstallMachines, cli)
	if err != nil {
		return fmt.Errorf("failed to register hosts: %s", err)
	}
	fmt.Fprint(context.Stdout, installation.Summary())
	fmt.Fprintf(context.Stdout, "Configured default user:\nUsername: %s\nPassword: %s\n", installConfig.ComponentsConfig.RootUserEmail, installConfig.ComponentsConfig.RootUserPassword)
	fmt.Fprintln(context.Stdout, "Apps Hosts:")
	nodeList := &admin.ListNodesCmd{}
	nodeList.Run(context, cli)
	fmt.Fprintln(context.Stdout, "Apps:")
	appList := &client.AppList{}
	appList.Run(context, cli)
	return nil
}

func parseConfigFile(file string) (*InstallOpts, error) {
	installConfig := DefaultInstallOpts()
	if file == "" {
		return installConfig, nil
	}
	err := config.ReadConfigFile(file)
	if err != nil {
		return nil, err
	}
	driverName, err := config.GetString("driver:name")
	if err == nil {
		defaultConfig := dm.DefaultDriverConfig(driverName)
		for k, v := range defaultConfig {
			if _, ok := config.Get(k); ok != nil {
				config.Set(k, v)
			}
		}
	}
	data, err := config.Bytes()
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, installConfig)
	if err != nil {
		return nil, err
	}
	installConfig.ComponentsConfig.TargetName = installConfig.Name
	defaultIaas := iaasConfig{
		Dockermachine: iaasConfigInternal{
			CaPath:              "/certs",
			InsecureRegistry:    "$REGISTRY_ADDR:$REGISTRY_PORT",
			DockerInstallURL:    installConfig.DockerInstallURL,
			DockerStorageDriver: installConfig.DockerStorageDriver,
			DockerFlags:         strings.Join(installConfig.DockerFlags, ","),
		},
	}
	if dm.IaaSCompatibleDriver(installConfig.DriverOpts.Name) {
		defaultIaas.Dockermachine.Driver = iaasConfigDriver{
			Name:    installConfig.DriverOpts.Name,
			Options: installConfig.DriverOpts.Options,
		}
	}
	conf := installConfig.ComponentsConfig.Tsuru.Config
	if _, ok := conf["iaas"]; ok {
		customIaas, err := yaml.Marshal(conf["iaas"])
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(customIaas, &defaultIaas)
		if err != nil {
			return nil, err
		}
	}
	conf["iaas"] = defaultIaas
	return installConfig, nil
}

func addInstallHosts(machines []*dockermachine.Machine, client *cmd.Client) error {
	path, err := cmd.GetURLVersion("1.2", "/install/hosts")
	if err != nil {
		return err
	}
	for _, m := range machines {
		rawDriver, err := json.Marshal(m.Host.Driver)
		if err != nil {
			return err
		}
		privateKey := []byte("")
		if m.Host.Driver.GetSSHKeyPath() != "" {
			privateKey, err = ioutil.ReadFile(m.Host.Driver.GetSSHKeyPath())
			if err != nil {
				fmt.Printf("failed to read private ssh key file: %s", err)
			}
		}
		caPrivateKey := []byte("")
		if m.Host.AuthOptions() != nil {
			caPrivateKey, err = ioutil.ReadFile(m.Host.AuthOptions().CaPrivateKeyPath)
			if err != nil {
				fmt.Printf("failed to read ca private key file: %s", err)
			}
		}
		v := url.Values{}
		v.Set("driver", string(rawDriver))
		v.Set("name", m.Host.Name)
		v.Set("driverName", m.Host.DriverName)
		v.Set("sshPrivateKey", string(privateKey))
		v.Set("caCert", string(m.Base.CaCert))
		v.Set("caPrivateKey", string(caPrivateKey))
		body := strings.NewReader(v.Encode())
		request, err := http.NewRequest("POST", path, body)
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, err = client.Do(request)
		if err != nil {
			return err
		}
	}
	return nil
}

type Uninstall struct {
	cmd.ConfirmationCommand
	fs     *gnuflag.FlagSet
	config string
}

func (c *Uninstall) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "install-remove",
		Usage:   "install-remove [name] [-y/--assume-yes]",
		Desc:    "Uninstalls Tsuru and It's components.",
		MinArgs: 0,
	}
}

func (c *Uninstall) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.ConfirmationCommand.Flags()
		c.fs.StringVar(&c.config, "c", "", "Configuration file")
		c.fs.StringVar(&c.config, "config", "", "Configuration file")
	}
	return c.fs
}

func (c *Uninstall) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()
	var installName string
	var dockerMachine dockermachine.DockerMachineAPI
	var appMachines []iaas.Machine
	machineList := admin.MachineList{}
	hosts, errList := listHosts(ctx, cli)
	if errList != nil || len(hosts) == 0 {
		fmt.Fprint(ctx.Stderr, "Unable to fetch installed hosts.\nFalling back to configuration file.\n")
		config, err := parseConfigFile(c.config)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "Failed to read configuration file: %s\n", err)
			return err
		}
		d, err := dm.NewDockerMachine(config.DockerMachineConfig, config.Name)
		if err != nil {
			return err
		}
		installName = config.Name
		dockerMachine = d.API
	} else {
		api, err := dockerMachineWithHosts(hosts...)
		if err != nil {
			return err
		}
		dockerMachine = api
		tLabel, tURL, err := getTargetData()
		if err != nil {
			return err
		}
		installName = tLabel
		fmt.Fprintf(ctx.Stdout, "This will uninstall Tsuru installed on your target %s: %s.\n", tLabel, tURL)
		appMachines, err = machineList.List(cli)
		if err != nil {
			return err
		}
		if len(appMachines) > 0 {
			fmt.Fprintf(ctx.Stdout, "The following app machines will be destroyed: \n%s", machineList.Tabulate(appMachines, nil).String())
		}
	}
	defer dockerMachine.Close()
	machines, err := dockerMachine.List()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "No dockerMachine found: %s\n", err)
	}
	tbl := tablecli.Table{LineSeparator: true, Headers: tablecli.Row([]string{"Name", "IP", "Data"})}
	for _, m := range machines {
		data, errMarshal := json.MarshalIndent(m.Base.CustomData, "", "")
		if errMarshal != nil {
			data = []byte("failed to marshal data.")
		}
		tbl.AddRow(tablecli.Row{m.Host.Name, m.Base.Address, string(data)})
	}
	fmt.Fprintf(ctx.Stdout, "The following core machines will be destroyed:\n%s", tbl.String())
	if !c.Confirm(ctx, "Are you sure you sure you want to uninstall tsuru?") {
		return nil
	}
	if !c.Confirm(ctx, "Are you really sure? I wont ask you again.") {
		return nil
	}
	destroyMachineCmd := admin.MachineDestroy{}
	destroyMachineCmd.Flags().Parse(true, []string{"-y"})
	destroyCtx := &cmd.Context{Stdout: ctx.Stdout, Stderr: ctx.Stderr}
	for _, m := range appMachines {
		destroyCtx.Args = []string{m.Id}
		fmt.Fprintf(ctx.Stdout, "Destroying machine %s...\n", m.FormatNodeAddress())
		errDest := destroyMachineCmd.Run(destroyCtx, cli)
		if errDest != nil {
			fmt.Fprintf(ctx.Stderr, "Failed to delete machine: %s\n", errDest)
		}
	}
	err = dockerMachine.DeleteAll()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "Failed to delete core machines: %s\n", err)
	}
	fmt.Fprintln(ctx.Stdout, "Core Machines successfully removed!")
	api := TsuruAPI{}
	err = api.Uninstall(installName)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "Failed to uninstall tsuru API: %s\n", err)
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Uninstall finished successfully!\n")
	return nil
}

func dockerMachineWithHosts(hosts ...installHost) (dockermachine.DockerMachineAPI, error) {
	d, err := dockermachine.NewDockerMachine(dockermachine.DockerMachineConfig{})
	if err != nil {
		return nil, err
	}
	for _, h := range hosts {
		_, errReg := d.RegisterMachine(dockermachine.RegisterMachineOpts{
			Base: &iaas.Machine{
				CustomData: h.Driver,
			},
			DriverName:    h.DriverName,
			SSHPrivateKey: []byte(h.SSHPrivateKey),
		})
		if errReg != nil {
			return nil, errReg
		}
	}
	return d, nil
}

func getTargetData() (string, string, error) {
	targetLabel, err := cmd.GetTargetLabel()
	if err != nil {
		return "", "", err
	}
	targetURL, err := cmd.GetTarget()
	if err != nil {
		return "", "", err
	}
	return targetLabel, targetURL, nil
}

type InstallHostList struct{}

type installHost struct {
	Name          string
	DriverName    string
	Driver        map[string]interface{}
	SSHPrivateKey string
}

func (c *InstallHostList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "install-host-list",
		Usage:   "install-host-list",
		Desc:    "List hosts created and registered by the installer.",
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *InstallHostList) Flags() *gnuflag.FlagSet {
	return gnuflag.NewFlagSet("install-host-list", gnuflag.ExitOnError)
}

func (c *InstallHostList) Run(context *cmd.Context, cli *cmd.Client) error {
	hosts, err := listHosts(context, cli)
	if err != nil {
		return err
	}
	return c.Show(hosts, context)
}

func listHosts(context *cmd.Context, cli *cmd.Client) ([]installHost, error) {
	url, err := cmd.GetURLVersion("1.2", "/install/hosts")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	response, err := cli.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var hosts []installHost
	err = json.NewDecoder(response.Body).Decode(&hosts)
	return hosts, err
}

func (c *InstallHostList) Show(hosts []installHost, context *cmd.Context) error {
	dockerMachine, err := dockermachine.NewDockerMachine(dockermachine.DockerMachineConfig{})
	if err != nil {
		return err
	}
	defer dockerMachine.Close()
	table := tablecli.NewTable()
	table.LineSeparator = true
	table.Headers = tablecli.Row([]string{"Name", "Driver Name", "State", "Driver"})
	for _, h := range hosts {
		driver, err := json.MarshalIndent(h.Driver, "", " ")
		if err != nil {
			return err
		}
		m, err := dockerMachine.RegisterMachine(dockermachine.RegisterMachineOpts{
			Base: &iaas.Machine{
				CustomData: h.Driver,
			},
			DriverName:    h.DriverName,
			SSHPrivateKey: []byte(h.SSHPrivateKey),
		})
		if err != nil {
			return err
		}
		state, err := m.Host.Driver.GetState()
		var stateStr string
		if err != nil {
			stateStr = err.Error()
		} else {
			stateStr = state.String()
		}
		table.AddRow(tablecli.Row([]string{h.Name, h.DriverName, stateStr, string(driver)}))
	}
	context.Stdout.Write(table.Bytes())
	return nil
}

type InstallSSH struct{}

func (c *InstallSSH) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "install-ssh",
		Usage:   "install-ssh <hostname> [arg...]",
		Desc:    "Log into or run a command on a host with SSH.",
		MinArgs: 1,
	}
}

func (c *InstallSSH) Flags() *gnuflag.FlagSet {
	return gnuflag.NewFlagSet("install-ssh", gnuflag.ExitOnError)
}

func (c *InstallSSH) Run(context *cmd.Context, cli *cmd.Client) error {
	hostName := context.Args[0]
	url, err := cmd.GetURLVersion("1.2", "/install/hosts/"+hostName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := cli.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var ih *installHost
	err = json.NewDecoder(response.Body).Decode(&ih)
	if err != nil {
		return err
	}
	dockerMachine, err := dockermachine.NewDockerMachine(dockermachine.DockerMachineConfig{})
	if err != nil {
		return err
	}
	defer dockerMachine.Close()
	m, err := dockerMachine.RegisterMachine(dockermachine.RegisterMachineOpts{
		Base: &iaas.Machine{
			CustomData: ih.Driver,
		},
		DriverName:    ih.DriverName,
		SSHPrivateKey: []byte(ih.SSHPrivateKey),
	})
	if err != nil {
		return err
	}
	sshClient, err := m.Host.CreateSSHClient()
	if err != nil {
		return fmt.Errorf("failed to create ssh client: %s", err)
	}
	sshArgs := []string{}
	if len(context.Args) > 1 {
		sshArgs = context.Args[1:]
	}
	return sshClient.Shell(sshArgs...)
}

type InstallConfigInit struct{}

func (c *InstallConfigInit) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "install-config-init",
		Usage: "install-config-init <config_file> <compose_file>",
		Desc:  "Generate install configuration files.",
	}
}

func (c *InstallConfigInit) Run(context *cmd.Context, cli *cmd.Client) error {
	configFile := "install-config.yml"
	composeFile := "install-compose.yml"
	if len(context.Args) > 0 {
		configFile = context.Args[0]
	}
	if len(context.Args) > 1 {
		composeFile = context.Args[1]
	}
	err := ioutil.WriteFile(composeFile, []byte(defaultconfig.Compose), 0644)
	if err != nil {
		return errors.Errorf("failed to write compose file: %s", err)
	}
	out, err := yaml.Marshal(DefaultInstallOpts())
	if err != nil {
		return errors.Errorf("failed to generate config file: %s", err)
	}
	err = ioutil.WriteFile(configFile, out, 0644)
	if err != nil {
		return errors.Errorf("failed to write config file: %s", err)
	}
	return nil
}
