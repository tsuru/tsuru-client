// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dockermachine

import (
	"encoding/json"
	"fmt"

	"github.com/docker/machine/drivers/virtualbox"
	"github.com/docker/machine/libmachine"
	"github.com/tsuru/tsuru-client/tsuru/installer/iaas"
)

func init() {
	iaas.Register("docker-machine", &dmIaas{})
}

type dmIaas struct{}

func (i *dmIaas) CreateMachine(params map[string]string) (*iaas.Machine, error) {
	client := libmachine.NewClient("/tmp/automatic", "/tmp/automatic/certs")
	defer client.Close()
	driver := virtualbox.NewDriver("tsuru", "/tmp/automatic")
	data, err := json.Marshal(driver)
	if err != nil {
		return nil, err
	}
	host, err := client.NewHost("virtualbox", data)
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
	m := iaas.Machine{
		Address: fmt.Sprintf("http://%s:2375", ip),
		Iaas:    "docker-machine",
		IP:      ip,
		Config:  config,
	}
	return &m, nil
}

func (i *dmIaas) DeleteMachine(m *iaas.Machine) error {
	client := libmachine.NewClient("/tmp/automatic", "/tmp/automatic/certs")
	defer client.Close()
	h, err := client.Load("tsuru")
	if err != nil {
		return err
	}
	return h.Driver.Remove()
}
