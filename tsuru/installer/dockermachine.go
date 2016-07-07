// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"path/filepath"

	"github.com/docker/machine/drivers/none"
	"github.com/docker/machine/drivers/virtualbox"
	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/auth"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/drivers/rpc"
	"github.com/docker/machine/libmachine/engine"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/persist"
	"github.com/docker/machine/libmachine/swarm"
	"github.com/docker/machine/libmachine/version"
)

type DockerMachine struct {
	driverOpts rpcdriver.RPCFlags
	driver     drivers.Driver
	fileStore  *persist.Filestore
}

type Machine struct {
	Address string
	IP      string
	Config  map[string]string
}

func NewDockerMachine(driverName string, opts map[string]interface{}) (*DockerMachine, error) {
	storePath := "/tmp/automatic"
	certsPath := "/tmp/automatic/certs"
	driver, err := getDriver(driverName, storePath, opts)
	if err != nil {
		return nil, err
	}
	dm := &DockerMachine{
		driverOpts: rpcdriver.RPCFlags{Values: opts},
		fileStore:  persist.NewFilestore(storePath, certsPath, certsPath),
		driver:     driver,
	}
	if err != nil {
		return nil, err
	}
	return dm, nil
}

func getDriver(driverName, storePath string, opts map[string]interface{}) (drivers.Driver, error) {
	driverOpts := rpcdriver.RPCFlags{Values: opts}
	if driverOpts.Values == nil {
		driverOpts.Values = make(map[string]interface{})
	}
	var driver drivers.Driver
	switch driverName {
	case "virtualbox":
		driver = virtualbox.NewDriver("tsuru", storePath)
	case "none":
		driver = none.NewDriver("tsuru", storePath)
	default:
		return nil, fmt.Errorf("Unsuported driver %s", driverName)
	}
	for _, c := range driver.GetCreateFlags() {
		_, ok := driverOpts.Values[c.String()]
		if ok == false {
			driverOpts.Values[c.String()] = c.Default()
			if c.Default() == nil {
				driverOpts.Values[c.String()] = false
			}
		}
	}
	if err := driver.SetConfigFromFlags(driverOpts); err != nil {
		return nil, fmt.Errorf("Error setting driver configurations: %s", err)
	}
	return driver, nil
}

func (d *DockerMachine) newHost() *host.Host {
	return &host.Host{
		ConfigVersion: version.ConfigVersion,
		Name:          d.driver.GetMachineName(),
		Driver:        d.driver,
		DriverName:    d.driver.DriverName(),
		HostOptions: &host.Options{
			AuthOptions: &auth.Options{
				CertDir:          d.fileStore.CaCertPath,
				CaCertPath:       filepath.Join(d.fileStore.CaCertPath, "ca.pem"),
				CaPrivateKeyPath: filepath.Join(d.fileStore.CaCertPath, "ca-key.pem"),
				ClientCertPath:   filepath.Join(d.fileStore.CaCertPath, "cert.pem"),
				ClientKeyPath:    filepath.Join(d.fileStore.CaCertPath, "key.pem"),
				ServerCertPath:   filepath.Join(d.fileStore.GetMachinesDir(), "server.pem"),
				ServerKeyPath:    filepath.Join(d.fileStore.GetMachinesDir(), "server-key.pem"),
			},
			EngineOptions: &engine.Options{
				InstallURL:     drivers.DefaultEngineInstallURL,
				StorageDriver:  "aufs",
				TLSVerify:      true,
				Env:            []string{"DOCKER_TLS=no"},
				ArbitraryFlags: []string{"host=tcp://0.0.0.0:2375"},
			},
			SwarmOptions: &swarm.Options{
				Host:     "tcp://0.0.0.0:3376",
				Image:    "swarm:latest",
				Strategy: "spread",
			},
		},
	}
}

func (d *DockerMachine) provisionHost() (*host.Host, error) {
	host := d.newHost()
	client := libmachine.NewClient(d.fileStore.Path, d.fileStore.CaCertPath)
	defer client.Close()
	return host, client.Create(host)
}

func (d *DockerMachine) CreateMachine(params map[string]interface{}) (*Machine, error) {
	host, err := d.provisionHost()
	if err != nil {
		fmt.Printf("Ignoring error on machine creation: %s", err)
	}
	ip, err := host.Driver.GetIP()
	if err != nil {
		return nil, err
	}
	config := map[string]string{}
	m := Machine{
		Address: fmt.Sprintf("http://%s:2375", ip),
		IP:      ip,
		Config:  config,
	}
	return &m, nil
}

func (d *DockerMachine) DeleteMachine(m *Machine) error {
	err := d.driver.Remove()
	if err != nil {
		return err
	}
	return d.fileStore.Remove("tsuru")
}
