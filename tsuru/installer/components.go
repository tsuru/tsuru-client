// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	redis "gopkg.in/redis.v3"

	"github.com/docker/engine-api/types/mount"
	"github.com/docker/engine-api/types/swarm"
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
	DockerHubMirror  string
	ComponentAddress map[string]string
	TsuruAPIConfig
}

func NewInstallConfig(targetName string) *InstallConfig {
	hub, _ := config.GetString("docker-hub-mirror")
	mongo, _ := config.GetString("components:mongo")
	redis, _ := config.GetString("components:redis")
	registry, _ := config.GetString("components:registry")
	planb, _ := config.GetString("components:planb")
	return &InstallConfig{
		DockerHubMirror: hub,
		TsuruAPIConfig: TsuruAPIConfig{
			TargetName:       targetName,
			RootUserEmail:    "admin@example.com",
			RootUserPassword: "admin123",
		},
		ComponentAddress: map[string]string{
			"mongo":    mongo,
			"redis":    redis,
			"registry": registry,
			"planb":    planb,
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
	Install(ServiceCluster, *InstallConfig) error
	Status(ServiceCluster) (*ServiceInfo, error)
	Healthcheck(string) error
}

type MongoDB struct{}

func (c *MongoDB) Name() string {
	return "MongoDB"
}

func (c *MongoDB) Install(cluster ServiceCluster, i *InstallConfig) error {
	if i.ComponentAddress["mongo"] != "" {
		return c.Healthcheck(i.ComponentAddress["mongo"])
	}
	err := cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "mongo",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image: i.fullImageName("mongo:latest"),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	i.ComponentAddress["mongo"] = "mongo"
	return nil
}

func (c *MongoDB) Status(cluster ServiceCluster) (*ServiceInfo, error) {
	return cluster.ServiceInfo("mongo")
}

func (c *MongoDB) Healthcheck(addr string) error {
	s, err := mgo.Dial(addr)
	if err != nil {
		return err
	}
	return s.Ping()
}

type PlanB struct{}

func (c *PlanB) Name() string {
	return "PlanB"
}

func (c *PlanB) Install(cluster ServiceCluster, i *InstallConfig) error {
	if i.ComponentAddress["planb"] != "" {
		return nil
	}
	err := cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "planb",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image: i.fullImageName("tsuru/planb:latest"),
					Args:  []string{"--listen", ":8080", "--read-redis-host", "redis", "--write-redis-host", "redis"},
				},
			},
			EndpointSpec: &swarm.EndpointSpec{
				Ports: []swarm.PortConfig{
					{
						Protocol:      swarm.PortConfigProtocolTCP,
						TargetPort:    uint32(8080),
						PublishedPort: uint32(80),
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	i.ComponentAddress["planb"] = cluster.GetManager().IP
	return nil
}

func (c *PlanB) Status(cluster ServiceCluster) (*ServiceInfo, error) {
	return cluster.ServiceInfo("planb")
}

func (c *PlanB) Healthcheck(addr string) error {
	return nil
}

type Redis struct{}

func (c *Redis) Name() string {
	return "Redis"
}

func (c *Redis) Install(cluster ServiceCluster, i *InstallConfig) error {
	if i.ComponentAddress["redis"] != "" {
		return nil
	}
	err := cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "redis",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image: i.fullImageName("redis:latest"),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	i.ComponentAddress["redis"] = "redis"
	return nil
}

func (c *Redis) Status(cluster ServiceCluster) (*ServiceInfo, error) {
	return cluster.ServiceInfo("redis")
}

func (c *Redis) Healthcheck(addr string) error {
	r := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	_, err := r.Ping().Result()
	return err
}

type Registry struct{}

func (c *Registry) Name() string {
	return "Docker Registry"
}

func (c *Registry) Install(cluster ServiceCluster, i *InstallConfig) error {
	if i.ComponentAddress["registry"] != "" {
		return nil
	}
	err := cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "registry",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image: i.fullImageName("registry:2"),
					Env: []string{
						"REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=/var/lib/registry",
						fmt.Sprintf("REGISTRY_HTTP_TLS_CERTIFICATE=/certs/%s:5000/registry-cert.pem", cluster.GetManager().IP),
						fmt.Sprintf("REGISTRY_HTTP_TLS_KEY=/certs/%s:5000/registry-key.pem", cluster.GetManager().IP),
					},
					Mounts: []mount.Mount{
						{
							Type:     mount.TypeBind,
							Source:   "/var/lib/registry",
							Target:   "/var/lib/registry",
							ReadOnly: false,
						},
						{
							Type:     mount.TypeBind,
							Source:   "/etc/docker/certs.d",
							Target:   "/certs",
							ReadOnly: true,
						},
					},
				},
			},
			EndpointSpec: &swarm.EndpointSpec{
				Ports: []swarm.PortConfig{
					{
						Protocol:      swarm.PortConfigProtocolTCP,
						TargetPort:    uint32(5000),
						PublishedPort: uint32(5000),
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	i.ComponentAddress["registry"] = cluster.GetManager().IP
	return nil
}

func (c *Registry) Status(cluster ServiceCluster) (*ServiceInfo, error) {
	return cluster.ServiceInfo("registry")
}

func (c *Registry) Healthcheck(addr string) error {
	return nil
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

func parseAddress(address, defaultPort string) (addr, port string) {
	parts := strings.Split(address, ":")
	if len(parts) == 1 {
		port = defaultPort
	} else {
		port = parts[1]
	}
	addr = parts[0]
	return addr, port
}

func (c *TsuruAPI) Install(cluster ServiceCluster, i *InstallConfig) error {
	mongo, mongoPort := parseAddress(i.ComponentAddress["mongo"], "27017")
	redis, redisPort := parseAddress(i.ComponentAddress["redis"], "6379")
	registry, registryPort := parseAddress(i.ComponentAddress["registry"], "5000")
	planb, _ := parseAddress(i.ComponentAddress["planb"], "80")
	err := cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "tsuru",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image: i.fullImageName("tsuru/api:latest"),
					Env: []string{fmt.Sprintf("MONGODB_ADDR=%s", mongo),
						fmt.Sprintf("MONGODB_PORT=%s", mongoPort),
						fmt.Sprintf("REDIS_ADDR=%s", redis),
						fmt.Sprintf("REDIS_PORT=%s", redisPort),
						fmt.Sprintf("HIPACHE_DOMAIN=%s.nip.io", planb),
						fmt.Sprintf("REGISTRY_ADDR=%s", registry),
						fmt.Sprintf("REGISTRY_PORT=%s", registryPort),
						fmt.Sprintf("TSURU_ADDR=http://%s", cluster.GetManager().IP),
						fmt.Sprintf("TSURU_PORT=%d", defaultTsuruAPIPort),
					},
					Mounts: []mount.Mount{
						{
							Type:     mount.TypeBind,
							Source:   "/etc/docker/certs.d",
							Target:   "/certs",
							ReadOnly: true,
						},
					},
				},
			},
			EndpointSpec: &swarm.EndpointSpec{
				Ports: []swarm.PortConfig{
					{
						Protocol:      swarm.PortConfigProtocolTCP,
						TargetPort:    uint32(8080),
						PublishedPort: uint32(8080),
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	fmt.Println("Waiting for Tsuru API to become responsive...")
	tsuruURL := fmt.Sprintf("http://%s:%d", cluster.GetManager().IP, defaultTsuruAPIPort)
	err = mcnutils.WaitForSpecific(func() bool {
		_, errReq := http.Get(tsuruURL)
		return errReq == nil
	}, 60, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %s", tsuruURL, err)
	}
	err = c.setupRootUser(cluster, i.RootUserEmail, i.RootUserPassword)
	if err != nil {
		return err
	}
	opts := TsuruSetupOptions{
		Login:           i.RootUserEmail,
		Password:        i.RootUserPassword,
		Target:          tsuruURL,
		TargetName:      i.TargetName,
		NodeAddr:        cluster.GetManager().Address,
		DockerHubMirror: i.DockerHubMirror,
	}
	return c.bootstrapEnv(opts)
}

func (c *TsuruAPI) Status(cluster ServiceCluster) (*ServiceInfo, error) {
	return cluster.ServiceInfo("tsuru")
}

func (c *TsuruAPI) Healthcheck(addr string) error {
	return nil
}

func (c *TsuruAPI) setupRootUser(cluster ServiceCluster, email, password string) error {
	cmd := []string{"tsurud", "root-user-create", email}
	passwordConfirmation := strings.NewReader(fmt.Sprintf("%s\n%s\n", password, password))
	startOpts := docker.StartExecOptions{
		InputStream:  passwordConfirmation,
		Detach:       false,
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		RawTerminal:  true,
	}
	return cluster.ServiceExec("tsuru", cmd, startOpts)
}

type TsuruSetupOptions struct {
	Login           string
	Password        string
	Target          string
	TargetName      string
	NodeAddr        string
	DockerHubMirror string
}

func (c *TsuruAPI) bootstrapEnv(opts TsuruSetupOptions) error {
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
		Args:   []string{opts.TargetName, opts.Target},
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
	context.Args = []string{opts.Login}
	context.Stdin = strings.NewReader(fmt.Sprintf("%s\n", opts.Password))
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
	context.Args = []string{fmt.Sprintf("address=%s", opts.NodeAddr), "pool=theonepool"}
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
		if opts.DockerHubMirror != "" {
			platformAdd.Flags().Parse(true, []string{"-i", fmt.Sprintf("%s/python", opts.DockerHubMirror)})
		}
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

func (c *TsuruAPI) Uninstall(installation string) error {
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
