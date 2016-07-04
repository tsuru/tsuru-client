// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"fmt"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/drivers/rpc"
)

type DockerMachine struct {
	rawDriver  []byte
	driverName string
	driverOpts rpcdriver.RPCFlags
	storePath  string
	certsPath  string
}

type Machine struct {
	Address string
	IP      string
	Config  map[string]string
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
	return &DockerMachine{
		rawDriver:  rawDriver,
		driverName: driverName,
		driverOpts: rpcdriver.RPCFlags{Values: opts},
		storePath:  storePath,
		certsPath:  certsPath,
	}, nil
}

func (d *DockerMachine) CreateMachine(params map[string]interface{}) (*Machine, error) {
	client := libmachine.NewClient(d.storePath, d.certsPath)
	defer client.Close()
	host, err := client.NewHost(d.driverName, d.rawDriver)
	if err != nil {
		return nil, err
	}
	host.HostOptions.EngineOptions.ArbitraryFlags = []string{
		"host=tcp://0.0.0.0:2375",
	}
	host.HostOptions.EngineOptions.Env = []string{"DOCKER_TLS=no"}
	host.Driver.SetConfigFromFlags(d.driverOpts)
	err = client.Create(host)
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
	return h.Driver.Remove()
}
