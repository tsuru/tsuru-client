// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/fsouza/go-dockerclient"
	"github.com/tsuru/config"
	"github.com/tsuru/tsuru-client/tsuru/admin"
	tclient "github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/provision"
)

var (
	TsuruComponents = []TsuruComponent{
		&MongoDB{},
		&Redis{},
		&PlanB{},
		&Registry{},
		&TsuruAPI{},
	}

	defaultTsuruAPIPort = 8080
)

type InstallConfig struct {
	DockerHubMirror string
	TsuruAPIConfig
}

func NewInstallConfig(targetName string) *InstallConfig {
	hub, err := config.GetString("docker-hub-mirror")
	if err != nil {
		hub = ""
	}
	return &InstallConfig{
		DockerHubMirror: hub,
		TsuruAPIConfig: TsuruAPIConfig{
			TargetName:       targetName,
			RootUserEmail:    "admin@example.com",
			RootUserPassword: "admin123",
		},
	}
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
		Cmd:   []string{"--listen", ":8080", "--read-redis-host", "redis", "--write-redis-host", "redis"},
	}
	hostConfig := &docker.HostConfig{
		PortBindings: map[docker.Port][]docker.PortBinding{
			"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "80"}},
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

type TsuruAPIConfig struct {
	TargetName       string
	RootUserEmail    string
	RootUserPassword string
}

func (c *TsuruAPI) Name() string {
	return "Tsuru API"
}

func (c *TsuruAPI) Install(machine *Machine, i *InstallConfig) error {
	env := []string{fmt.Sprintf("MONGODB_ADDR=%s", "mongo"),
		"MONGODB_PORT=27017",
		fmt.Sprintf("REDIS_ADDR=%s", "redis"),
		"REDIS_PORT=6379",
		fmt.Sprintf("HIPACHE_DOMAIN=%s.nip.io", machine.IP),
		fmt.Sprintf("REGISTRY_ADDR=%s", machine.IP),
		"REGISTRY_PORT=5000",
		fmt.Sprintf("TSURU_ADDR=http://%s", machine.IP),
		fmt.Sprintf("TSURU_PORT=%d", defaultTsuruAPIPort),
	}
	config := &docker.Config{
		Image: i.fullImageName("tsuru/api:latest"),
		Env:   env,
	}
	hostConfig := &docker.HostConfig{
		Binds: []string{"/etc/docker/certs.d:/certs:ro"},
	}
	err := createContainer(machine, "tsuru", config, hostConfig)
	if err != nil {
		return err
	}
	fmt.Println("Waiting for Tsuru API to become responsive...")
	tsuruURL := fmt.Sprintf("http://%s:%d", machine.IP, defaultTsuruAPIPort)
	err = mcnutils.WaitFor(func() bool {
		_, errReq := http.Get(tsuruURL)
		return errReq == nil
	})
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %s", tsuruURL, err)
	}
	err = c.setupRootUser(machine, i.RootUserEmail, i.RootUserPassword)
	if err != nil {
		return err
	}
	return c.bootstrapEnv(i.RootUserEmail, i.RootUserPassword, tsuruURL, i.TargetName, machine.Address)
}

func (c *TsuruAPI) Status(machine *Machine) (*ComponentStatus, error) {
	return containerStatus("tsuru", machine)
}

func (c *TsuruAPI) setupRootUser(machine *Machine, email, password string) error {
	cmd := []string{"tsurud", "root-user-create", email}
	passwordConfirmation := strings.NewReader(fmt.Sprintf("%s\n%s\n", password, password))
	client, err := machine.dockerClient()
	if err != nil {
		return err
	}
	filterOpts := docker.ListContainersOptions{
		All: true,
	}
	containers, err := client.ListContainers(filterOpts)
	if err != nil {
		return err
	}
	var tsuruContainer string
	for _, c := range containers {
		if strings.Contains(strings.Join(c.Names, ","), "tsuru") {
			tsuruContainer = c.ID
		}
	}
	exec, err := client.CreateExec(docker.CreateExecOptions{
		Cmd:          cmd,
		Container:    tsuruContainer,
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

func (c *TsuruAPI) bootstrapEnv(login, password, target, targetName, node string) error {
	manager := cmd.BuildBaseManager("setup-client", "0.0.0", "", nil)
	provisioners := provision.Registry()
	for _, p := range provisioners {
		if c, ok := p.(cmd.AdminCommandable); ok {
			commands := c.AdminCommands()
			for _, comm := range commands {
				manager.Register(comm)
			}
		}
	}
	fmt.Fprintln(os.Stdout, "adding target")
	client := cmd.NewClient(&http.Client{}, nil, manager)
	context := cmd.Context{
		Args:   []string{targetName, target},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	context.RawOutput()
	targetadd := manager.Commands["target-add"]
	t, _ := targetadd.(cmd.FlaggedCommand)
	err := t.Flags().Parse(true, []string{"-s"})
	if err != nil {
		return err
	}
	err = t.Run(&context, client)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, "log in with default user: admin@example.com")
	logincmd := manager.Commands["login"]
	context.Args = []string{login}
	context.Stdin = strings.NewReader(fmt.Sprintf("%s\n", password))
	err = logincmd.Run(&context, client)
	if err != nil {
		return err
	}
	context.Args = []string{"theonepool"}
	context.Stdin = nil
	fmt.Fprintln(os.Stdout, "adding pool")
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
	fmt.Fprintln(os.Stdout, "adding node")
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
	context.Args = []string{"python"}
	fmt.Fprintln(os.Stdout, "adding platform")
	err = mcnutils.WaitFor(func() bool {
		platformAdd := admin.PlatformAdd{}
		return platformAdd.Run(&context, client) == nil
	})
	if err != nil {
		return err
	}
	context.Args = []string{"admin"}
	fmt.Fprintln(os.Stdout, "adding team")
	teamCreate := tclient.TeamCreate{}
	err = teamCreate.Run(&context, client)
	if err != nil {
		return err
	}
	context.Args = []string{"tsuru-dashboard", "python"}
	fmt.Fprintln(os.Stdout, "adding dashboard")
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
	fmt.Fprintln(os.Stdout, "deploying dashboard")
	deployDashboard := tclient.AppDeploy{}
	err = deployDashboard.Flags().Parse(true, []string{"-a", "tsuru-dashboard", "-i", "tsuru/dashboard"})
	if err != nil {
		return err
	}
	return deployDashboard.Run(&context, client)
}

func (t *TsuruAPI) Uninstall(installation string) error {
	manager := cmd.BuildBaseManager("uninstall-client", "0.0.0", "", nil)
	provisioners := provision.Registry()
	for _, p := range provisioners {
		if c, ok := p.(cmd.AdminCommandable); ok {
			commands := c.AdminCommands()
			for _, cmd := range commands {
				manager.Register(cmd)
			}
		}
	}
	fmt.Fprint(os.Stdout, "removing target\n")
	client := cmd.NewClient(&http.Client{}, nil, manager)
	context := cmd.Context{
		Args:   []string{installation},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	targetrm := manager.Commands["target-remove"]
	return targetrm.Run(&context, client)
}
