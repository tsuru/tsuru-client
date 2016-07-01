// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"fmt"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/drivers"
)

type DockerMachine struct {
	rawDriver  []byte
	driverName string
}

type Machine struct {
	Address string
	IP      string
	Config  map[string]string
}

func NewDockerMachine(driverName string) (*DockerMachine, error) {
	rawDriver, err := json.Marshal(&drivers.BaseDriver{
		MachineName: "tsuru",
		StorePath:   "/tmp/automatic",
	})
	if err != nil {
		return nil, fmt.Errorf("Error creating docker-machine driver: %s", err)
	}
	return &DockerMachine{rawDriver: rawDriver, driverName: driverName}, nil
}

func (d *DockerMachine) CreateMachine(params map[string]string) (*Machine, error) {
	client := libmachine.NewClient("/tmp/automatic", "/tmp/automatic/certs")
	defer client.Close()
	host, err := client.NewHost(d.driverName, d.rawDriver)
	if err != nil {
		return nil, err
	}
	host.HostOptions.EngineOptions.ArbitraryFlags = []string{
		"host=tcp://0.0.0.0:2375",
	}
	host.HostOptions.EngineOptions.Env = []string{"DOCKER_TLS=no"}
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
	client := libmachine.NewClient("/tmp/automatic", "/tmp/automatic/certs")
	defer client.Close()
	h, err := client.Load("tsuru")
	if err != nil {
		return err
	}
	return h.Driver.Remove()
}
