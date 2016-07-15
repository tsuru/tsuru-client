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

type InstallConfig struct {
	Registry string
}

func (i *InstallConfig) fullImageName(name string) string {
	if i.Registry != "" {
		return fmt.Sprintf("%s/%s", i.Registry, name)
	}
	return name
}

type TsuruComponent interface {
	Name() string
	Install(*Machine, *InstallConfig) error
	Status(*Machine) (*ComponentStatus, error)
}

type MongoDB struct{}

func (c *MongoDB) Name() string {
	return "MongoDB"
}

func (c *MongoDB) Install(machine *Machine, i *InstallConfig) error {
	image := i.fullImageName("mongo:latest")
	return createContainer(machine, "mongo", &docker.Config{Image: image}, nil)
}

func (c *MongoDB) Status(machine *Machine) (*ComponentStatus, error) {
	return containerStatus("mongo", machine)
}

type PlanB struct{}

func (c *PlanB) Name() string {
	return "PlanB"
}

func (c *PlanB) Install(machine *Machine, i *InstallConfig) error {
	config := &docker.Config{
		Image: i.fullImageName("tsuru/planb:latest"),
		Cmd:   []string{"--listen", ":80", "--read-redis-host", machine.IP, "--write-redis-host", machine.IP},
	}
	hostConfig := &docker.HostConfig{
		PortBindings: make(map[docker.Port][]docker.PortBinding),
	}
	hostConfig.PortBindings["80"] = []docker.PortBinding{{HostIP: "0.0.0.0", HostPort: "80"}}
	return createContainer(machine, "planb", config, hostConfig)
}

func (c *PlanB) Status(machine *Machine) (*ComponentStatus, error) {
	return containerStatus("planb", machine)
}

type Redis struct{}

func (c *Redis) Name() string {
	return "Redis"
}

func (c *Redis) Install(machine *Machine, i *InstallConfig) error {
	image := i.fullImageName("redis:latest")
	return createContainer(machine, "redis", &docker.Config{Image: image}, nil)
}

func (c *Redis) Status(machine *Machine) (*ComponentStatus, error) {
	return containerStatus("redis", machine)
}

type Registry struct{}

func (c *Registry) Name() string {
	return "Docker Registry"
}

func (c *Registry) Install(machine *Machine, i *InstallConfig) error {
	config := &docker.Config{
		Image: i.fullImageName("registry:2"),
		Env:   []string{"REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=/var/lib/registry"},
	}
	hostConfig := &docker.HostConfig{
		Binds: []string{"/var/lib/registry:/var/lib/registry"},
	}
	return createContainer(machine, "registry", config, hostConfig)
}

func (c *Registry) Status(machine *Machine) (*ComponentStatus, error) {
	return containerStatus("registry", machine)
}

type TsuruAPI struct{}

func (c *TsuruAPI) Name() string {
	return "Tsuru API"
}

func (c *TsuruAPI) Install(machine *Machine, i *InstallConfig) error {
	env := []string{fmt.Sprintf("MONGODB_ADDR=%s", machine.IP),
		"MONGODB_PORT=27017",
		fmt.Sprintf("REDIS_ADDR=%s", machine.IP),
		"REDIS_PORT=6379",
		fmt.Sprintf("HIPACHE_DOMAIN=%s.nip.io", machine.IP),
	}
	config := &docker.Config{
		Image: i.fullImageName("tsuru/api:latest"),
		Env:   env,
	}
	err := createContainer(machine, "tsuru", config, nil)
	if err != nil {
		return err
	}
	return c.setupRootUser(machine)
}

func (c *TsuruAPI) Status(machine *Machine) (*ComponentStatus, error) {
	return containerStatus("tsuru", machine)
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

type ComponentStatus struct {
	containerState *docker.State
	addresses      []string
}

func containerStatus(name string, m *Machine) (*ComponentStatus, error) {
	client, err := m.dockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed create docker client: %s", err)
	}
	container, err := client.InspectContainer(name)
	if err != nil {
		return nil, err
	}
	var addresses []string
	for p := range container.HostConfig.PortBindings {
		address := fmt.Sprintf("%s://%s:%s", p.Proto(), m.IP, p.Port())
		addresses = append(addresses, address)
	}
	return &ComponentStatus{
		containerState: &container.State,
		addresses:      addresses,
	}, nil
}
