// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/machine/libmachine/mcnutils"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/sethvargo/go-password/password"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/installer/defaultconfig"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	yaml "gopkg.in/yaml.v2"
)

var (
	// rootUserPassswordGenerator generates a password to root user. That's
	// very useful for testing purposes.
	rootUserPassswordGenerator func() string
)

func init() {
	rootUserPassswordGenerator = generateStrongPassword
}

func DefaultInstallOpts() *InstallOpts {
	return &InstallOpts{
		Name: "tsuru",
		DockerMachineConfig: dm.DockerMachineConfig{
			DriverOpts:  &dm.DriverOpts{Name: "virtualbox"},
			DockerFlags: []string{"experimental"},
		},
		ComponentsConfig: &ComponentsConfig{
			InstallDashboard: true,
			TargetName:       "tsuru",
			RootUserEmail:    "admin@example.com",
			RootUserPassword: rootUserPassswordGenerator(),
			TsuruAPIImage:    "tsuru/api:v1",
			Tsuru:            tsuruComponent{Config: defaultconfig.DefaultTsuruConfig()},
		},
		Hosts: hostGroups{
			Apps: hostGroupConfig{Size: 1},
			Core: hostGroupConfig{Size: 1},
		},
	}
}

type InstallOpts struct {
	dm.DockerMachineConfig `yaml:",inline"`
	*ComponentsConfig      `yaml:"components"`
	Name                   string
	Hosts                  hostGroups `yaml:"hosts,omitempty"`
	ComposeFile            string     `yaml:"-"`
}

type hostGroups struct {
	Apps hostGroupConfig `yaml:"apps,omitempty"`
	Core hostGroupConfig `yaml:"core,omitempty"`
}

type hostGroupConfig struct {
	Size      int                `yaml:"size,omitempty"`
	Dedicated bool               `yaml:"dedicated,omitempty"`
	Driver    multiOptionsDriver `yaml:"driver,omitempty"`
}

type multiOptionsDriver struct {
	Options map[string][]interface{} `yaml:"options,omitempty"`
}

type Installer struct {
	outWriter          io.Writer
	errWriter          io.Writer
	machineProvisioner dm.MachineProvisioner
	bootstraper        Bootstraper
	clusterCreator     func([]*dockermachine.Machine) (ServiceCluster, error)
}

func (i *Installer) Install(opts *InstallOpts) (*Installation, error) {
	fmt.Fprintf(i.outWriter, "Running pre-install checks...\n")
	if errChecks := preInstallChecks(opts); errChecks != nil {
		return nil, fmt.Errorf("pre-install checks failed: %s", errChecks)
	}
	setCoreDriverDefaultOpts(opts)
	deployFunc, err := deployTsuruConfig(opts)
	if err != nil {
		return nil, err
	}
	coreMachines, err := i.ProvisionMachines(opts.Hosts.Core.Size, opts.Hosts.Core.Driver.Options, deployFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to provision components machines: %s", err)
	}
	cluster, err := i.clusterCreator(coreMachines)
	if err != nil {
		return nil, fmt.Errorf("failed to setup swarm cluster: %s", err)
	}
	err = composeDeploy(cluster, opts)
	if err != nil {
		return nil, err
	}
	err = waitTsuru(cluster, opts.ComponentsConfig)
	if err != nil {
		return nil, err
	}
	err = waitRegistry(cluster.GetManager().Host, dm.GetPrivateIP(cluster.GetManager()), opts.ComponentsConfig)
	if err != nil {
		return nil, err
	}
	target := fmt.Sprintf("http://%s:%d", cluster.GetManager().Base.Address, defaultTsuruAPIPort)
	installMachines, err := i.BootstrapTsuru(opts, target, coreMachines)
	if err != nil {
		return nil, err
	}
	i.applyIPtablesRules(coreMachines)
	return &Installation{
		CoreCluster:     cluster,
		InstallMachines: installMachines,
	}, nil
}

func deployTsuruConfig(installOpts *InstallOpts) (func(*dockermachine.Machine) error, error) {
	conf, err := yaml.Marshal(installOpts.Tsuru.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tsuru api config: %s", err)
	}
	return func(m *dockermachine.Machine) error {
		fmt.Println("Deploying tsuru config...")
		_, err := m.Host.RunSSHCommand("sudo mkdir -p /etc/tsuru/")
		if err != nil {
			return fmt.Errorf("failed to create tsuru config directory: %s", err)
		}
		err = dm.WriterRemoteData(m.Host, "/etc/tsuru/tsuru.conf", conf)
		if err != nil {
			return fmt.Errorf("failed to write remote file: %s", err)
		}
		return nil
	}, nil
}

type sshTarget interface {
	RunSSHCommand(string) (string, error)
}

func waitRegistry(target sshTarget, privateIP string, compConf *ComponentsConfig) error {
	registryURL := fmt.Sprintf("https://%s:5000", privateIP)
	if compConf != nil {
		tsuruConf := compConf.Tsuru.Config
		if dockerConf, ok := tsuruConf["docker"].(map[string]interface{}); ok {
			if registry, ok := dockerConf["registry"].(string); ok {
				registryURL = fmt.Sprintf("https://%s", registry)
			}
		}
	}
	fmt.Printf("Waiting for Docker Registry to become responsive at %q...\n", registryURL)
	remoteCurlCommand := fmt.Sprintf("curl -m5 -sSLk %q", registryURL)
	var errCurl error
	err := mcnutils.WaitForSpecific(func() bool {
		_, errCurl = target.RunSSHCommand(remoteCurlCommand)
		return errCurl == nil
	}, 60, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v: %v", registryURL, err, errCurl)
	}
	return nil
}

func waitTsuru(cluster ServiceCluster, compConf *ComponentsConfig) error {
	tsuruURL := fmt.Sprintf("http://%s:%d", cluster.GetManager().Base.Address, defaultTsuruAPIPort)
	fmt.Printf("Waiting for Tsuru API to become responsive at %q...\n", tsuruURL)
	var errReq error
	err := mcnutils.WaitForSpecific(func() bool {
		_, errReq = getWithTimeout(tsuruURL, 5*time.Second)
		return errReq == nil
	}, 60, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v: %v", tsuruURL, err, errReq)
	}
	cmd := []string{"tsurud", "root-user-create", compConf.RootUserEmail}
	passwordConfirmation := strings.NewReader(fmt.Sprintf("%s\n%s\n", compConf.RootUserPassword, compConf.RootUserPassword))
	startOpts := docker.StartExecOptions{
		InputStream:  passwordConfirmation,
		Detach:       false,
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		RawTerminal:  true,
	}
	return cluster.ServiceExec("tsuru_tsuru", cmd, startOpts)
}

func setCoreDriverDefaultOpts(opts *InstallOpts) {
	driverName := opts.DriverOpts.Name
	coreDriverOpts := opts.Hosts.Core.Driver.Options
	if coreDriverOpts == nil {
		coreDriverOpts = make(map[string][]interface{})
	}
	if _, ok := coreDriverOpts[driverName+"-open-port"]; !ok {
		coreDriverOpts[driverName+"-open-port"] = []interface{}{strconv.Itoa(defaultTsuruAPIPort)}
	}
	if (driverName == "google") && (coreDriverOpts["google-scopes"] == nil) {
		coreDriverOpts["google-scopes"] = []interface{}{"https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/compute"}
	}
	opts.Hosts.Core.Driver.Options = coreDriverOpts
}

func (i *Installer) BootstrapTsuru(opts *InstallOpts, target string, coreMachines []*dockermachine.Machine) ([]*dockermachine.Machine, error) {
	fmt.Fprintf(i.outWriter, "Bootstrapping Tsuru API...")
	bootstrapOpts := BoostrapOptions{
		Login:            opts.ComponentsConfig.RootUserEmail,
		Password:         opts.ComponentsConfig.RootUserPassword,
		Target:           target,
		TargetName:       opts.ComponentsConfig.TargetName,
		NodesParams:      opts.Hosts.Apps.Driver.Options,
		InstallDashboard: opts.ComponentsConfig.InstallDashboard,
	}
	var installMachines []*dockermachine.Machine
	if !dm.IaaSCompatibleDriver(opts.DriverOpts.Name) {
		appsMachines, errProv := i.ProvisionPool(opts, coreMachines)
		if errProv != nil {
			return nil, errProv
		}
		machineIndex := make(map[string]*dockermachine.Machine)
		installMachines = append(coreMachines, appsMachines...)
		for _, m := range installMachines {
			machineIndex[m.Host.Name] = m
		}
		var uniqueMachines []*dockermachine.Machine
		for _, v := range machineIndex {
			uniqueMachines = append(uniqueMachines, v)
		}
		installMachines = uniqueMachines
		var nodesAddr []string
		for _, m := range appsMachines {
			nodesAddr = append(nodesAddr, dm.GetPrivateAddress(m))
		}
		bootstrapOpts.NodesToRegister = nodesAddr
	} else {
		installMachines = coreMachines
		if opts.Hosts.Apps.Dedicated {
			bootstrapOpts.NodesToCreate = opts.Hosts.Apps.Size
		} else {
			var nodesAddr []string
			for _, m := range coreMachines {
				nodesAddr = append(nodesAddr, dm.GetPrivateAddress(m))
			}
			if opts.Hosts.Apps.Size > opts.Hosts.Core.Size {
				bootstrapOpts.NodesToCreate = opts.Hosts.Apps.Size - opts.Hosts.Core.Size
				bootstrapOpts.NodesToRegister = nodesAddr
			} else {
				bootstrapOpts.NodesToRegister = nodesAddr[:opts.Hosts.Apps.Size]
			}
		}
	}
	err := i.bootstraper.Bootstrap(bootstrapOpts)
	if err != nil {
		return installMachines, fmt.Errorf("Error bootstrapping tsuru: %s", err)
	}
	return installMachines, nil
}

func (i *Installer) applyIPtablesRules(machines []*dockermachine.Machine) {
	fmt.Fprintf(i.outWriter, "Applying iptables workaround for docker 1.12...\n")
	for _, m := range machines {
		_, err := m.Host.RunSSHCommand("PATH=$PATH:/usr/sbin/:/usr/local/sbin; sudo iptables -D DOCKER-ISOLATION -i docker_gwbridge -o docker0 -j DROP")
		if err != nil {
			fmt.Fprintf(i.errWriter, "Failed to apply iptables rule: %s. Maybe it is not needed anymore?\n", err)
		}
		_, err = m.Host.RunSSHCommand("PATH=$PATH:/usr/sbin/:/usr/local/sbin; sudo iptables -D DOCKER-ISOLATION -i docker0 -o docker_gwbridge -j DROP")
		if err != nil {
			fmt.Fprintf(i.errWriter, "Failed to apply iptables rule: %s. Maybe it is not needed anymore?\n", err)
		}
	}
}

func preInstallChecks(config *InstallOpts) error {
	exists, err := cmd.CheckIfTargetLabelExists(config.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tsuru target \"%s\" already exists", config.Name)
	}
	return nil
}

func (i *Installer) ProvisionPool(config *InstallOpts, hosts []*dockermachine.Machine) ([]*dockermachine.Machine, error) {
	if config.Hosts.Apps.Dedicated {
		return i.ProvisionMachines(config.Hosts.Apps.Size, config.Hosts.Apps.Driver.Options, nil)
	}
	if config.Hosts.Apps.Size > len(hosts) {
		poolMachines, err := i.ProvisionMachines(config.Hosts.Apps.Size-len(hosts), config.Hosts.Apps.Driver.Options, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to provision pool hosts: %s", err)
		}
		return append(poolMachines, hosts...), nil
	}
	return hosts[:config.Hosts.Apps.Size], nil
}

func (i *Installer) ProvisionMachines(numMachines int, configs map[string][]interface{}, initFunc func(*dockermachine.Machine) error) ([]*dockermachine.Machine, error) {
	var machines []*dockermachine.Machine
	for j := 0; j < numMachines; j++ {
		opts := make(map[string]interface{})
		for k, v := range configs {
			idx := j % len(v)
			opts[k] = v[idx]
		}
		m, err := i.machineProvisioner.ProvisionMachine(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to provision machines: %s", err)
		}
		if initFunc != nil {
			if err := initFunc(m); err != nil {
				return nil, fmt.Errorf("failed to initialize host: %s", err)
			}
		}
		machines = append(machines, m)
	}
	return machines, nil
}

type Installation struct {
	CoreCluster     ServiceCluster
	InstallMachines []*dockermachine.Machine
}

func (i *Installation) Summary() string {
	summary := fmt.Sprintf(`--- Installation Overview ---
Core Hosts:
%s
Core Components:
%s`, i.buildClusterTable().String(), i.buildComponentsTable().String())
	return summary
}

func (i *Installation) buildClusterTable() *tablecli.Table {
	t := tablecli.NewTable()
	t.Headers = tablecli.Row{"IP", "State", "Manager"}
	t.LineSeparator = true
	nodes, err := i.CoreCluster.ClusterInfo()
	if err != nil {
		t.AddRow(tablecli.Row{fmt.Sprintf("failed to retrieve cluster info: %s", err)})
	}
	for _, n := range nodes {
		t.AddRow(tablecli.Row{n.IP, n.State, strconv.FormatBool(n.Manager)})
	}
	return t
}

func (i *Installation) buildComponentsTable() *tablecli.Table {
	t := tablecli.NewTable()
	t.Headers = tablecli.Row{"Component", "Ports", "Replicas"}
	t.LineSeparator = true
	services, err := i.CoreCluster.ServicesInfo()
	if err != nil {
		t.AddRow(tablecli.Row{"Failed to fetch services.", "", ""})
		return nil
	}
	for _, s := range services {
		t.AddRow(tablecli.Row{s.Name,
			strings.Join(s.Ports, ","),
			strconv.Itoa(s.Replicas),
		})
	}
	return t
}

func getWithTimeout(url string, timeout time.Duration) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)
	return http.DefaultClient.Do(req)
}

// generateStrongPassword produces a pseudorandom string which contains 50 chars
// with 10 numbers, 10 symbols, and 30 letters, including uppercase,
// distinct between itself.
func generateStrongPassword() string {
	randomPassword, _ := password.Generate(50, 10, 10, false, false)
	return randomPassword
}
