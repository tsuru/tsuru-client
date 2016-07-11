// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"os"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

var TsuruComponents = []TsuruComponent{
	&MongoDB{},
	&Redis{},
	&PlanB{},
	&Registry{},
	&TsuruAPI{},
}

type TsuruComponent interface {
	Name() string
	Install(*Machine) error
}

type MongoDB struct{}

func (c *MongoDB) Name() string {
	return "MongoDB"
}

func (c *MongoDB) Install(machine *Machine) error {
	return createContainer(machine, "mongo", &docker.Config{Image: "mongo:latest"}, nil)
}

type PlanB struct{}

func (c *PlanB) Name() string {
	return "PlanB"
}

func (c *PlanB) Install(machine *Machine) error {
	config := &docker.Config{
		Image: "tsuru/planb:latest",
		Cmd:   []string{"--listen", ":80", "--read-redis-host", machine.IP, "--write-redis-host", machine.IP},
	}
	return createContainer(machine, "planb", config, nil)
}

type Redis struct{}

func (c *Redis) Name() string {
	return "Redis"
}

func (c *Redis) Install(machine *Machine) error {
	return createContainer(machine, "redis", &docker.Config{Image: "redis:latest"}, nil)
}

type Registry struct{}

func (c *Registry) Name() string {
	return "Docker Registry"
}

func (c *Registry) Install(machine *Machine) error {
	config := &docker.Config{
		Image: "registry:2",
		Env:   []string{"REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=/var/lib/registry"},
	}
	hostConfig := &docker.HostConfig{
		Binds: []string{"/var/lib/registry:/var/lib/registry"},
	}
	return createContainer(machine, "registry", config, hostConfig)
}

type TsuruAPI struct{}

func (c *TsuruAPI) Name() string {
	return "Tsuru API"
}

func (c *TsuruAPI) Install(machine *Machine) error {
	env := []string{fmt.Sprintf("MONGODB_ADDR=%s", machine.IP),
		"MONGODB_PORT=27017",
		fmt.Sprintf("REDIS_ADDR=%s", machine.IP),
		"REDIS_PORT=6379",
		fmt.Sprintf("HIPACHE_DOMAIN=%s.nip.io", machine.IP),
	}
	config := &docker.Config{
		Image: "tsuru/api:latest",
		Env:   env,
	}
	err := createContainer(machine, "tsuru", config, nil)
	if err != nil {
		return err
	}
	return c.setupRootUser(machine)
}

func (c *TsuruAPI) setupRootUser(machine *Machine) error {
	cmd := []string{"tsurud", "root-user-create", "admin@example.com"}
	passwordConfirmation := strings.NewReader("admin123\nadmin123\n")
	client, err := machine.dockerClient()
	if err != nil {
		return err
	}
	exec, err := client.CreateExec(docker.CreateExecOptions{
		Cmd:          cmd,
		Container:    "tsuru",
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
	})
	if err != nil {
		return err
	}
	return client.StartExec(exec.ID, docker.StartExecOptions{
		InputStream:  passwordConfirmation,
		Detach:       false,
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		RawTerminal:  true,
	})
}
