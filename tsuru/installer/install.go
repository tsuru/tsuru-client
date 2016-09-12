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
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru/cmd"
)

var (
	defaultTsuruInstallConfig = &TsuruInstallConfig{
		DockerMachineConfig: defaultDockerMachineConfig,
		CoreHosts:           1,
		AppsHosts:           1,
		DedicatedAppsHosts:  false,
		CoreDriversOpts:     make(map[string][]interface{}),
	}
)

type TsuruInstallConfig struct {
	*DockerMachineConfig
	CoreHosts          int
	CoreDriversOpts    map[string][]interface{}
	AppsHosts          int
	DedicatedAppsHosts bool
	AppsDriversOpts    map[string][]interface{}
}

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
	config, err := parseConfigFile(c.config)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, "Running pre-install checks...\n")
	err = c.PreInstallChecks(config)
	if err != nil {
		return fmt.Errorf("pre-install checks failed: %s", err)
	}
	dm, err := NewDockerMachine(config.DockerMachineConfig)
	if err != nil {
		return fmt.Errorf("failed to create docker machine: %s", err)
	}
	defer dm.Close()
	config.CoreDriversOpts[config.DriverName+"-open-port"] = []interface{}{strconv.Itoa(defaultTsuruAPIPort)}
	coreMachines, err := ProvisionMachines(dm, config.CoreHosts, config.CoreDriversOpts)
	if err != nil {
		return fmt.Errorf("failed to provision components machines: %s", err)
	}
	cluster, err := NewSwarmCluster(coreMachines)
	if err != nil {
		return fmt.Errorf("failed to setup swarm cluster: %s", err)
	}
	installConfig := NewInstallConfig(config.Name)
	for _, component := range TsuruComponents {
		fmt.Fprintf(context.Stdout, "Installing %s\n", component.Name())
		errInstall := component.Install(cluster, installConfig)
		if errInstall != nil {
			return fmt.Errorf("error installing %s: %s", component.Name(), errInstall)
		}
		fmt.Fprintf(context.Stdout, "%s successfully installed!\n", component.Name())
	}
	appsMachines, err := ProvisionPool(dm, config, coreMachines)
	if err != nil {
		return err
	}
	var nodesAddr []string
	for _, m := range appsMachines {
		nodesAddr = append(nodesAddr, m.GetPrivateAddress())
	}
	fmt.Fprintf(context.Stdout, "Bootstrapping Tsuru API...")
	opts := TsuruSetupOptions{
		Login:           installConfig.RootUserEmail,
		Password:        installConfig.RootUserPassword,
		Target:          fmt.Sprintf("http://%s:%d", cluster.GetManager().IP, defaultTsuruAPIPort),
		TargetName:      installConfig.TargetName,
		NodesAddr:       nodesAddr,
		DockerHubMirror: installConfig.DockerHubMirror,
	}
	err = SetupTsuru(opts)
	if err != nil {
		return fmt.Errorf("Error bootstrapping tsuru: %s", err)
	}
	fmt.Fprintf(context.Stdout, "Applying iptables workaround for docker 1.12...\n")
	for _, m := range coreMachines {
		_, err = m.RunSSHCommand("PATH=$PATH:/usr/sbin/:/usr/local/sbin; sudo iptables -D DOCKER-ISOLATION -i docker_gwbridge -o docker0 -j DROP")
		if err != nil {
			fmt.Fprintf(context.Stderr, "Failed to apply iptables rule: %s. Maybe it is not needed anymore?\n", err)
		}
		_, err = m.RunSSHCommand("PATH=$PATH:/usr/sbin/:/usr/local/sbin; sudo iptables -D DOCKER-ISOLATION -i docker0 -o docker_gwbridge -j DROP")
		if err != nil {
			fmt.Fprintf(context.Stderr, "Failed to apply iptables rule: %s. Maybe it is not needed anymore?\n", err)
		}
	}
	fmt.Fprint(context.Stdout, "--- Installation Overview ---\n")
	fmt.Fprint(context.Stdout, "Swarm Cluster: \n"+buildClusterTable(cluster).String())
	fmt.Fprint(context.Stdout, "Components: \n"+buildComponentsTable(TsuruComponents, cluster).String())
	appList := &client.AppList{}
	fmt.Fprintln(context.Stdout, "Apps:")
	appList.Run(context, cli)
	return nil
}

func ProvisionPool(p MachineProvisioner, config *TsuruInstallConfig, hosts []*Machine) ([]*Machine, error) {
	if config.DedicatedAppsHosts {
		return ProvisionMachines(p, config.AppsHosts, config.AppsDriversOpts)
	}
	if config.AppsHosts > len(hosts) {
		poolMachines, err := ProvisionMachines(p, config.AppsHosts-len(hosts), config.AppsDriversOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to provision pool hosts: %s", err)
		}
		return append(poolMachines, hosts...), nil
	}
	return hosts[:config.AppsHosts], nil
}

func ProvisionMachines(p MachineProvisioner, numMachines int, configs map[string][]interface{}) ([]*Machine, error) {
	var machines []*Machine
	for i := 0; i < numMachines; i++ {
		opts := make(DriverOpts)
		for k, v := range configs {
			idx := i % len(v)
			opts[k] = v[idx]
		}
		m, err := p.ProvisionMachine(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to provision machines: %s", err)
		}
		machines = append(machines, m)
	}
	return machines, nil
}

func (c *Install) PreInstallChecks(config *TsuruInstallConfig) error {
	exists, err := cmd.CheckIfTargetLabelExists(config.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tsuru target \"%s\" already exists", config.Name)
	}
	return nil
}

func buildClusterTable(cluster ServiceCluster) *cmd.Table {
	t := cmd.NewTable()
	t.Headers = cmd.Row{"IP", "State", "Manager"}
	t.LineSeparator = true
	nodes, err := cluster.ClusterInfo()
	if err != nil {
		t.AddRow(cmd.Row{fmt.Sprintf("failed to retrieve cluster info: %s", err)})
	}
	for _, n := range nodes {
		t.AddRow(cmd.Row{n.IP, n.State, strconv.FormatBool(n.Manager)})
	}
	return t
}

func buildComponentsTable(components []TsuruComponent, cluster ServiceCluster) *cmd.Table {
	t := cmd.NewTable()
	t.Headers = cmd.Row{"Component", "Ports", "Replicas"}
	t.LineSeparator = true
	for _, component := range components {
		info, err := component.Status(cluster)
		if err != nil {
			t.AddRow(cmd.Row{component.Name(), "?", fmt.Sprintf("%s", err)})
			continue
		}
		row := cmd.Row{component.Name(),
			strings.Join(info.Ports, ","),
			strconv.Itoa(info.Replicas),
		}
		t.AddRow(row)
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
	d, err := NewDockerMachine(config.DockerMachineConfig)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machine: %s\n", err)
		return err
	}
	defer d.Close()
	err = d.DeleteAll()
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machines: %s\n", err)
		return err
	}
	fmt.Fprintln(context.Stdout, "Machines successfully removed!")
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

func parseConfigFile(file string) (*TsuruInstallConfig, error) {
	installConfig := defaultTsuruInstallConfig
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
	driverOpts := make(DriverOpts)
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
	return installConfig, nil
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
