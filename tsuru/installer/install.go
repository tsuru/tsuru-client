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

	"github.com/tsuru/config"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/admin"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/iaas/dockermachine"
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

- hosts:core:size
Number of machines to be provisioned and used to host tsuru core components.

- hosts:core:driver:options
Driver parameters specific to the core hosts can be set on this namespace. The format is: <driver-param>>: ["value1", "value2"]. Each
host will use one value from the list. Refer to the driver configuration for more information on what parameter are available.

- hosts:apps:size
Number of machines to be provisioned and used to host tsuru applications.

- hosts:apps:dedicated
Boolean to indicated if the installer should not reuse the machines created for
the core components.

- hosts:apps:driver:options
Driver parameters specific to the applications hosts can be set on this namespace. The format is: <driver-param>>: ["value1", "value2"]. Each
host will use one value from the list. Refer to the driver configuration for more information on what parameter are available.

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

func (c *Install) Run(context *cmd.Context, cli *cmd.Client) error {
	context.RawOutput()
	installConfig, err := parseConfigFile(c.config)
	if err != nil {
		return err
	}
	dockerMachine, err := dm.NewDockerMachine(installConfig.DockerMachineConfig)
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
		components:         TsuruComponents,
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
	fmt.Fprintln(context.Stdout, "Apps Hosts:")
	nodeList := &admin.ListNodesCmd{}
	nodeList.Run(context, cli)
	fmt.Fprintln(context.Stdout, "Apps:")
	appList := &client.AppList{}
	appList.Run(context, cli)
	return nil
}

func parseConfigFile(file string) (*InstallOpts, error) {
	installConfig := defaultInstallOpts
	if file == "" {
		return installConfig, nil
	}
	err := config.ReadConfigFile(file)
	if err != nil {
		return nil, err
	}
	driverName, err := config.GetString("driver:name")
	if err == nil {
		installConfig.DriverName = driverName
	}
	name, err := config.GetString("name")
	if err == nil {
		installConfig.Name = name
	}
	hub, err := config.GetString("docker-hub-mirror")
	if err == nil {
		installConfig.DockerHubMirror = hub
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
		installConfig.DriverOpts = driverOpts
	}
	caPath, err := config.GetString("ca-path")
	if err == nil {
		installConfig.CAPath = caPath
	}
	cHosts, err := config.GetInt("hosts:core:size")
	if err == nil {
		installConfig.CoreHosts = cHosts
	}
	pHosts, err := config.GetInt("hosts:apps:size")
	if err == nil {
		installConfig.AppsHosts = pHosts
	}
	dedicated, err := config.GetBool("hosts:apps:dedicated")
	if err == nil {
		installConfig.DedicatedAppsHosts = dedicated
	}
	opts, _ = config.Get("hosts:core:driver:options")
	if opts != nil {
		installConfig.CoreDriversOpts, err = parseDriverOptsSlice(opts)
		if err != nil {
			return nil, err
		}
	}
	opts, _ = config.Get("hosts:apps:driver:options")
	if opts != nil {
		installConfig.AppsDriversOpts, err = parseDriverOptsSlice(opts)
		if err != nil {
			return nil, err
		}
	}
	installConfig.ComponentsConfig = NewInstallConfig(installConfig.Name)
	installConfig.ComponentsConfig.IaaSConfig = map[string]interface{}{
		"dockermachine": map[string]interface{}{
			"ca-path": "/certs",
			"driver": map[string]interface{}{
				"name":    installConfig.DriverName,
				"options": map[string]interface{}(installConfig.DriverOpts),
			},
		},
	}
	return installConfig, nil
}

func addInstallHosts(machines []*dockermachine.Machine, client *cmd.Client) error {
	path, err := cmd.GetURLVersion("1.3", "/install/hosts")
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
		Name:    "uninstall",
		Usage:   "uninstall [name]",
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

func (c *Uninstall) Run(ctx *cmd.Context, cli *cmd.Client) error {
	ctx.RawOutput()
	var installName string
	var dockerMachine dockermachine.DockerMachineAPI
	hosts, errList := listHosts(ctx, cli)
	if errList != nil {
		fmt.Fprintf(ctx.Stderr, "Unable to fetch installed hosts: %s.\n Falling back to configuration file.\n", errList)
		config, err := parseConfigFile(c.config)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "Failed to read configuration file: %s\n", err)
			return err
		}
		d, err := dm.NewDockerMachine(config.DockerMachineConfig)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "Failed to delete machine: %s\n", err)
			return err
		}
		installName = config.Name
		dockerMachine = d.API
	} else {
		d, err := dockermachine.NewDockerMachine(dockermachine.DockerMachineConfig{})
		if err != nil {
			return err
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
				return errReg
			}
		}
		dockerMachine = d
		installName, err = cmd.ReadTarget()
		if err != nil {
			return err
		}
	}
	defer dockerMachine.Close()
	machines, err := dockerMachine.List()
	if err != nil {
		return err
	}
	tbl := cmd.Table{
		LineSeparator: true,
		Headers:       cmd.Row{"Name", "IP", "Data"},
	}
	for _, m := range machines {
		data, errMarshal := json.MarshalIndent(m.Base.CustomData, "", "")
		if errMarshal != nil {
			data = []byte("failed to marshal data.")
		}
		tbl.AddRow(cmd.Row{m.Host.Name, m.Base.Address, string(data)})
	}
	fmt.Fprintf(ctx.Stdout, "The following machines will be destroyed:\n%s", tbl.String())
	if !c.Confirm(ctx, fmt.Sprint("Are you sure you sure you want to uninstall tsuru?")) {
		return nil
	}
	err = dockerMachine.DeleteAll()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "Failed to delete machines: %s\n", err)
		return err
	}
	fmt.Fprintln(ctx.Stdout, "Machines successfully removed!")
	api := TsuruAPI{}
	err = api.Uninstall(installName)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "Failed to uninstall tsuru API: %s\n", err)
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Uninstall finished successfully!\n")
	return nil
}

func parseDriverOptsSlice(opts interface{}) (map[string][]interface{}, error) {
	unparsed, ok := opts.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse opts: %+v", opts)
	}
	parsedOpts := make(map[string][]interface{})
	if opts != nil {
		for k, v := range unparsed {
			switch k := k.(type) {
			case string:
				l, ok := v.([]interface{})
				if ok {
					parsedOpts[k] = l
				} else {
					parsedOpts[k] = []interface{}{v}
				}
			}
		}
	}
	return parsedOpts, nil
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
	url, err := cmd.GetURLVersion("1.3", "/install/hosts")
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
	table := cmd.NewTable()
	table.LineSeparator = true
	table.Headers = cmd.Row([]string{"Name", "Driver Name", "State", "Driver"})
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
		table.AddRow(cmd.Row([]string{h.Name, h.DriverName, stateStr, string(driver)}))
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
	url, err := cmd.GetURLVersion("1.3", "/install/hosts/"+hostName)
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
