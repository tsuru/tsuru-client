// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/fsouza/go-dockerclient"
	"github.com/tsuru/config"
	"github.com/tsuru/tsuru-client/tsuru/admin"
	tclient "github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/provision"
)

var TsuruComponents = []TsuruComponent{
	&MongoDB{},
	&Redis{},
	&PlanB{},
	&Registry{},
	&TsuruAPI{},
}

type InstallConfig struct {
	DockerHubMirror string
}

func NewInstallConfig() *InstallConfig {
	hub, err := config.GetString("docker-hub-mirror")
	if err != nil {
		hub = ""
	}
	return &InstallConfig{DockerHubMirror: hub}
}

func (i *InstallConfig) fullImageName(name string) string {
	if i.DockerHubMirror != "" {
		return fmt.Sprintf("%s/%s", i.DockerHubMirror, name)
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
		ExposedPorts: map[docker.Port]struct{}{
			docker.Port("80/tcp"): {},
		},
	}
	hostConfig := &docker.HostConfig{
		PortBindings: map[docker.Port][]docker.PortBinding{
			"80/tcp": {{HostIP: "0.0.0.0", HostPort: "80"}},
		},
	}
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
		Env: []string{
			"REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=/var/lib/registry",
			fmt.Sprintf("REGISTRY_HTTP_TLS_CERTIFICATE=/certs/%s:5000/registry-cert.pem", machine.IP),
			fmt.Sprintf("REGISTRY_HTTP_TLS_KEY=/certs/%s:5000/registry-key.pem", machine.IP),
		},
	}
	hostConfig := &docker.HostConfig{
		Binds: []string{"/var/lib/registry:/var/lib/registry", "/etc/docker/certs.d:/certs:ro"},
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
		fmt.Sprintf("REGISTRY_ADDR=%s", machine.IP),
		"REGISTRY_PORT=5000",
		fmt.Sprintf("TSURU_ADDR=http://%s", machine.IP),
		fmt.Sprintf("TSURU_PORT=8080"),
	}
	config := &docker.Config{
		Image: i.fullImageName("tsuru/api:latest"),
		Env:   env,
	}
	err := createContainer(machine, "tsuru", config, nil)
	if err != nil {
		return err
	}
	fmt.Println("Wating API container to be running...")
	err = mcnutils.WaitFor(func() bool {
		status, errSt := c.Status(machine)
		if errSt != nil {
			return false
		}
		return status.containerState.Running
	})
	if err != nil {
		return err
	}
	err = c.setupRootUser(machine)
	if err != nil {
		return err
	}
	return c.bootstrapEnv("admin@example.com", "admin123", machine.IP, machine.Address)
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

func (c *TsuruAPI) bootstrapEnv(login, password, target, node string) error {
	var stdout, stderr bytes.Buffer
	manager := cmd.BuildBaseManager("setup-client", "0.0.0", "", nil)
	provisioners := provision.Registry()
	for _, p := range provisioners {
		if c, ok := p.(cmd.AdminCommandable); ok {
			commands := c.AdminCommands()
			for _, cmd := range commands {
				manager.Register(cmd)
			}
		}
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	context := cmd.Context{
		Args:   []string{"test", fmt.Sprintf("%s:8080", target)},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	context.RawOutput()
	println("adding target")
	targetset := manager.Commands["target-add"]
	t, _ := targetset.(cmd.FlaggedCommand)
	err := t.Flags().Parse(true, []string{"-s"})
	if err != nil {
		return err
	}
	err = t.Run(&context, client)
	if err != nil {
		return err
	}
	println("logging")
	logincmd := manager.Commands["login"]
	context.Args = []string{login}
	context.Stdin = strings.NewReader(fmt.Sprintf("%s\n", password))
	err = logincmd.Run(&context, client)
	if err != nil {
		return err
	}
	context.Args = []string{"theonepool"}
	context.Stdin = nil
	println("adding pool")
	poolAdd := admin.AddPoolToSchedulerCmd{}
	err = poolAdd.Flags().Parse(true, []string{"-d", "-p"})
	if err != nil {
		return err
	}
	err = poolAdd.Run(&context, client)
	if err != nil {
		return err
	}
	context.Args = []string{fmt.Sprintf("address=%s", node), "pool=theonepool"}
	println("adding node")
	nodeAdd := manager.Commands["docker-node-add"]
	n, _ := nodeAdd.(cmd.FlaggedCommand)
	err = n.Flags().Parse(true, []string{"--register"})
	if err != nil {
		return err
	}
	err = n.Run(&context, client)
	if err != nil {
		return err
	}
	time.Sleep(60 * time.Second)
	context.Args = []string{"python"}
	println("adding platform")
	platformAdd := admin.PlatformAdd{}
	err = platformAdd.Run(&context, client)
	if err != nil {
		return err
	}
	context.Args = []string{"admin"}
	println("adding team")
	teamCreate := tclient.TeamCreate{}
	err = teamCreate.Run(&context, client)
	if err != nil {
		return err
	}
	context.Args = []string{"tsuru-dashboard", "python"}
	println("adding dashboard")
	createDashboard := tclient.AppCreate{}
	err = createDashboard.Flags().Parse(true, []string{"-t", "admin"})
	if err != nil {
		return err
	}
	err = createDashboard.Run(&context, client)
	if err != nil {
		return err
	}
	context.Args = []string{}
	println("deploying dashboard")
	deployDashboard := tclient.AppDeploy{}
	err = deployDashboard.Flags().Parse(true, []string{"-a", "tsuru-dashboard", "-i", "tsuru/dashboard"})
	if err != nil {
		return err
	}
	return deployDashboard.Run(&context, client)
}
