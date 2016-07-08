// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"fmt"

	"github.com/docker/machine/drivers/amazonec2"
	"github.com/docker/machine/drivers/azure"
	"github.com/docker/machine/drivers/digitalocean"
	"github.com/docker/machine/drivers/exoscale"
	"github.com/docker/machine/drivers/generic"
	"github.com/docker/machine/drivers/google"
	"github.com/docker/machine/drivers/hyperv"
	"github.com/docker/machine/drivers/none"
	"github.com/docker/machine/drivers/openstack"
	"github.com/docker/machine/drivers/rackspace"
	"github.com/docker/machine/drivers/softlayer"
	"github.com/docker/machine/drivers/virtualbox"
	"github.com/docker/machine/drivers/vmwarefusion"
	"github.com/docker/machine/drivers/vmwarevcloudair"
	"github.com/docker/machine/drivers/vmwarevsphere"
	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/docker/machine/libmachine/drivers/rpc"
)

type DockerMachine struct {
	driverOpts rpcdriver.RPCFlags
	rawDriver  []byte
	driverName string
	storePath  string
	certsPath  string
}

type Machine struct {
	Address string
	IP      string
	Config  map[string]string
}

func RunDriver(driverName string) error {
	switch driverName {
	case "amazonec2":
		plugin.RegisterDriver(amazonec2.NewDriver("", ""))
	case "azure":
		plugin.RegisterDriver(azure.NewDriver("", ""))
	case "digitalocean":
		plugin.RegisterDriver(digitalocean.NewDriver("", ""))
	case "exoscale":
		plugin.RegisterDriver(exoscale.NewDriver("", ""))
	case "generic":
		plugin.RegisterDriver(generic.NewDriver("", ""))
	case "google":
		plugin.RegisterDriver(google.NewDriver("", ""))
	case "hyperv":
		plugin.RegisterDriver(hyperv.NewDriver("", ""))
	case "none":
		plugin.RegisterDriver(none.NewDriver("", ""))
	case "openstack":
		plugin.RegisterDriver(openstack.NewDriver("", ""))
	case "rackspace":
		plugin.RegisterDriver(rackspace.NewDriver("", ""))
	case "softlayer":
		plugin.RegisterDriver(softlayer.NewDriver("", ""))
	case "virtualbox":
		plugin.RegisterDriver(virtualbox.NewDriver("", ""))
	case "vmwarefusion":
		plugin.RegisterDriver(vmwarefusion.NewDriver("", ""))
	case "vmwarevcloudair":
		plugin.RegisterDriver(vmwarevcloudair.NewDriver("", ""))
	case "vmwarevsphere":
		plugin.RegisterDriver(vmwarevsphere.NewDriver("", ""))
	default:
		return fmt.Errorf("Unsupported driver: %s\n", driverName)
	}
	return nil
}

func NewDockerMachine(driverName string, opts map[string]interface{}) (*DockerMachine, error) {
	storePath := "/tmp/automatic"
	certsPath := "/tmp/automatic/certs"
	rawDriver, err := json.Marshal(&drivers.BaseDriver{
		MachineName: "tsuru",
		StorePath:   storePath,
	})
	if err != nil {
		return nil, fmt.Errorf("Error creating docker-machine driver: %s", err)
	}
	dm := &DockerMachine{
		driverOpts: rpcdriver.RPCFlags{Values: opts},
		rawDriver:  rawDriver,
		driverName: driverName,
		storePath:  storePath,
		certsPath:  certsPath,
	}
	return dm, nil
}

func configureDriver(driver drivers.Driver, driverOpts rpcdriver.RPCFlags) error {
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
		return fmt.Errorf("Error setting driver configurations: %s", err)
	}
	return nil
}

func (d *DockerMachine) CreateMachine(params map[string]interface{}) (*Machine, error) {
	client := libmachine.NewClient(d.storePath, d.certsPath)
	defer client.Close()
	host, err := client.NewHost(d.driverName, d.rawDriver)
	if err != nil {
		return nil, err
	}
	host.HostOptions.EngineOptions.Env = []string{"DOCKER_TLS=no"}
	host.HostOptions.EngineOptions.ArbitraryFlags = []string{"host=tcp://0.0.0.0:2375"}
	configureDriver(host.Driver, d.driverOpts)
	err = client.Create(host)
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
	client := libmachine.NewClient(d.storePath, d.certsPath)
	defer client.Close()
	h, err := client.Load("tsuru")
	if err != nil {
		return err
	}
	err = h.Driver.Remove()
	if err != nil {
		return err
	}
	return client.Remove("tsuru")
}
