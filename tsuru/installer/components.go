// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	redis "gopkg.in/redis.v3"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/fsouza/go-dockerclient"
	"github.com/tsuru/config"
	"github.com/tsuru/tsuru-client/tsuru/admin"
	tclient "github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
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

type ComponentsConfig struct {
	ComponentAddress map[string]string
	TsuruAPIConfig
}

func NewInstallConfig(targetName string) *ComponentsConfig {
	mongo, _ := config.GetString("components:mongo")
	redis, _ := config.GetString("components:redis")
	registry, _ := config.GetString("components:registry")
	planb, _ := config.GetString("components:planb")
	tsuruVersion, err := config.GetString("components:tsuru:version")
	if err != nil {
		tsuruVersion = "v1"
	}
	return &ComponentsConfig{
		TsuruAPIConfig: TsuruAPIConfig{
			TargetName:       targetName,
			RootUserEmail:    "admin@example.com",
			RootUserPassword: "admin123",
			IaaSConfig:       map[string]interface{}{},
			ImageTag:         tsuruVersion,
		},
		ComponentAddress: map[string]string{
			"mongo":    mongo,
			"redis":    redis,
			"registry": registry,
			"planb":    planb,
		},
	}
}

type TsuruComponent interface {
	Name() string
	Install(ServiceCluster, *ComponentsConfig) error
	Status(ServiceCluster) (*ServiceInfo, error)
	Healthcheck(string) error
}

type MongoDB struct{}

func (c *MongoDB) Name() string {
	return "MongoDB"
}

func (c *MongoDB) Install(cluster ServiceCluster, i *ComponentsConfig) error {
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
					Image: "mongo:latest",
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

func (c *PlanB) Install(cluster ServiceCluster, i *ComponentsConfig) error {
	if i.ComponentAddress["planb"] != "" {
		return c.Healthcheck(i.ComponentAddress["planb"])
	}
	err := cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "planb",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image: "tsuru/planb:latest",
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
	i.ComponentAddress["planb"] = cluster.GetManager().Base.Address
	return nil
}

func (c *PlanB) Status(cluster ServiceCluster) (*ServiceInfo, error) {
	return cluster.ServiceInfo("planb")
}

func (c *PlanB) Healthcheck(addr string) error {
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return err
	}
	req.Host = "__ping__"
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Planb healthcheck error: Want status code 200. Got %s", resp.Status)
	}
	return nil
}

type Redis struct{}

func (c *Redis) Name() string {
	return "Redis"
}

func (c *Redis) Install(cluster ServiceCluster, i *ComponentsConfig) error {
	if i.ComponentAddress["redis"] != "" {
		return c.Healthcheck(i.ComponentAddress["redis"])
	}
	err := cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "redis",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image: "redis:latest",
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

func (c *Registry) Install(cluster ServiceCluster, i *ComponentsConfig) error {
	if i.ComponentAddress["registry"] != "" {
		return c.Healthcheck(i.ComponentAddress["registry"])
	}
	address := dm.GetPrivateIP(cluster.GetManager())
	err := cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "registry",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image: "registry:2",
					Env: []string{
						"REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=/var/lib/registry",
						fmt.Sprintf("REGISTRY_HTTP_TLS_CERTIFICATE=/certs/%s:5000/registry-cert.pem", address),
						fmt.Sprintf("REGISTRY_HTTP_TLS_KEY=/certs/%s:5000/registry-key.pem", address),
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
	i.ComponentAddress["registry"] = address
	return nil
}

func (c *Registry) Status(cluster ServiceCluster) (*ServiceInfo, error) {
	return cluster.ServiceInfo("registry")
}

func (c *Registry) Healthcheck(addr string) error {
	resp, err := http.Get(fmt.Sprintf("%s/v2/", addr))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Registry healthcheck error: Want status code 200. Got %s.", resp.Status)
	}
	return nil
}

type TsuruAPI struct{}

type TsuruAPIConfig struct {
	ImageTag         string
	TargetName       string
	RootUserEmail    string
	RootUserPassword string
	IaaSConfig       map[string]interface{}
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

func (c *TsuruAPI) Install(cluster ServiceCluster, i *ComponentsConfig) error {
	mongo, mongoPort := parseAddress(i.ComponentAddress["mongo"], "27017")
	redis, redisPort := parseAddress(i.ComponentAddress["redis"], "6379")
	registry, registryPort := parseAddress(i.ComponentAddress["registry"], "5000")
	planb, _ := parseAddress(i.ComponentAddress["planb"], "80")
	iaasConfig, err := json.Marshal(i.IaaSConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal iaas config: %s", err)
	}
	err = cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "tsuru",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image: fmt.Sprintf("tsuru/api:%s", i.TsuruAPIConfig.ImageTag),
					Env: []string{fmt.Sprintf("MONGODB_ADDR=%s", mongo),
						fmt.Sprintf("MONGODB_PORT=%s", mongoPort),
						fmt.Sprintf("REDIS_ADDR=%s", redis),
						fmt.Sprintf("REDIS_PORT=%s", redisPort),
						fmt.Sprintf("HIPACHE_DOMAIN=%s.nip.io", planb),
						fmt.Sprintf("REGISTRY_ADDR=%s", registry),
						fmt.Sprintf("REGISTRY_PORT=%s", registryPort),
						fmt.Sprintf("TSURU_ADDR=http://%s", cluster.GetManager().Base.Address),
						fmt.Sprintf("TSURU_PORT=%d", defaultTsuruAPIPort),
						fmt.Sprintf("IAAS_CONF=%s", string(iaasConfig)),
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
	tsuruURL := fmt.Sprintf("http://%s:%d", cluster.GetManager().Base.Address, defaultTsuruAPIPort)
	err = mcnutils.WaitForSpecific(func() bool {
		_, errReq := http.Get(tsuruURL)
		return errReq == nil
	}, 60, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %s", tsuruURL, err)
	}
	return c.setupRootUser(cluster, i.RootUserEmail, i.RootUserPassword)
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

type Bootstraper interface {
	Bootstrap(opts BoostrapOptions) error
}

type BoostrapOptions struct {
	Login           string
	Password        string
	Target          string
	TargetName      string
	NodesToRegister []string
	NodesToCreate   int
	NodesParams     map[string][]interface{}
	RegistryAddr    string
}

type TsuruBoostraper struct {
	manager *cmd.Manager
	client  *cmd.Client
	context cmd.Context
}

func (s *TsuruBoostraper) Bootstrap(opts BoostrapOptions) error {
	manager := cmd.BuildBaseManager("setup-client", "0.0.0", "", nil)
	provisioners, err := provision.Registry()
	if err != nil {
		return err
	}
	for _, p := range provisioners {
		if c, ok := p.(cmd.AdminCommandable); ok {
			commands := c.AdminCommands()
			for _, comm := range commands {
				manager.Register(comm)
			}
		}
	}
	s.manager = manager
	s.client = cmd.NewClient(&http.Client{}, nil, s.manager)
	s.context = cmd.Context{
		Args:   []string{opts.TargetName, opts.Target},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	s.context.RawOutput()
	err = s.addTarget()
	if err != nil {
		return err
	}
	err = s.login(opts.Login, opts.Password)
	if err != nil {
		return err
	}
	err = s.addPool("theonepool")
	if err != nil {
		return err
	}
	err = s.registerNodes("theonepool", opts.NodesToRegister)
	if err != nil {
		return err
	}
	err = s.createNodes("theonepool", opts.NodesToCreate, opts.RegistryAddr, opts.NodesParams)
	if err != nil {
		return err
	}
	err = s.addPlatform("python")
	if err != nil {
		return err
	}
	err = s.addTeam("admin")
	if err != nil {
		return err
	}
	err = s.installDashboard()
	if err != nil {
		return err
	}
	return nil
}

func (s *TsuruBoostraper) addTarget() error {
	fmt.Fprintln(os.Stdout, "adding target")
	targetadd := s.manager.Commands["target-add"]
	t, _ := targetadd.(cmd.FlaggedCommand)
	err := t.Flags().Parse(true, []string{"-s"})
	if err != nil {
		return err
	}
	err = t.Run(&s.context, s.client)
	if err != nil {
		return fmt.Errorf("failed to add tsuru target: %s", err)
	}
	return nil
}

func (s *TsuruBoostraper) login(login, password string) error {
	fmt.Fprintf(os.Stdout, "log in with default user: %s ", login)
	logincmd := s.manager.Commands["login"]
	s.context.Args = []string{login}
	s.context.Stdin = strings.NewReader(fmt.Sprintf("%s\n", password))
	err := logincmd.Run(&s.context, s.client)
	if err != nil {
		return fmt.Errorf("failed to login to tsuru: %s", err)
	}
	return nil
}

func (s *TsuruBoostraper) addPool(pool string) error {
	s.context.Args = []string{pool}
	s.context.Stdin = nil
	fmt.Fprintln(os.Stdout, "adding pool")
	poolAdd := admin.AddPoolToSchedulerCmd{}
	err := poolAdd.Flags().Parse(true, []string{"-d", "-p"})
	if err != nil {
		return err
	}
	err = poolAdd.Run(&s.context, s.client)
	if err != nil {
		return fmt.Errorf("failed to add pool: %s", err)
	}
	return nil
}

func (s *TsuruBoostraper) registerNodes(pool string, nodes []string) error {
	nodeAdd := admin.AddNodeCmd{}
	err := nodeAdd.Flags().Parse(true, []string{"--register"})
	if err != nil {
		return err
	}
	for _, n := range nodes {
		fmt.Printf("adding node %s\n", n)
		s.context.Args = []string{"docker", fmt.Sprintf("address=%s", n), fmt.Sprintf("pool=%s", pool)}
		err = nodeAdd.Run(&s.context, s.client)
		if err != nil {
			return fmt.Errorf("failed to register node: %s", err)
		}
	}
	return nil
}

func (s *TsuruBoostraper) createNodes(pool string, nodes int, insecureRegistry string, nodesParams map[string][]interface{}) error {
	nodeAdd := admin.AddNodeCmd{}
	for i := 0; i < nodes; i++ {
		fmt.Printf("creating node %d/%d...\n", i+1, nodes)
		s.context.Args = []string{"docker", "iaas=dockermachine", fmt.Sprintf("pool=%s", pool),
			fmt.Sprintf("insecure-registry=%s", insecureRegistry)}
		for k, v := range nodesParams {
			idx := i % len(v)
			s.context.Args = append(s.context.Args, fmt.Sprintf("%s=%s", k, v[idx]))
		}
		err := nodeAdd.Run(&s.context, s.client)
		if err != nil {
			return fmt.Errorf("failed to create node: %s", err)
		}
	}
	return nil
}

func (s *TsuruBoostraper) addPlatform(platform string) error {
	fmt.Fprintln(os.Stdout, "adding platform")
	platformAdd := admin.PlatformAdd{}
	s.context.Args = []string{platform}
	err := mcnutils.WaitFor(func() bool {
		return platformAdd.Run(&s.context, s.client) == nil
	})
	if err != nil {
		return fmt.Errorf("failed to add platform: %s", err)
	}
	return nil
}

func (s *TsuruBoostraper) addTeam(team string) error {
	s.context.Args = []string{team}
	fmt.Fprintln(os.Stdout, "adding team")
	teamCreate := tclient.TeamCreate{}
	err := teamCreate.Run(&s.context, s.client)
	if err != nil {
		return fmt.Errorf("failed to create admin team: %s", err)
	}
	return nil
}

func (s *TsuruBoostraper) installDashboard() error {
	fmt.Fprintln(os.Stdout, "adding dashboard")
	s.context.Args = []string{"tsuru-dashboard", "python"}
	createDashboard := tclient.AppCreate{}
	err := createDashboard.Flags().Parse(true, []string{"-t", "admin"})
	if err != nil {
		return err
	}
	err = createDashboard.Run(&s.context, s.client)
	if err != nil {
		return fmt.Errorf("failed to create dashboard app: %s", err)
	}
	s.context.Args = []string{}
	fmt.Fprintln(os.Stdout, "deploying dashboard")
	deployDashboard := tclient.AppDeploy{}
	deployFlags := []string{"-a", "tsuru-dashboard", "-i", "tsuru/dashboard"}
	err = deployDashboard.Flags().Parse(true, deployFlags)
	if err != nil {
		return err
	}
	err = deployDashboard.Run(&s.context, s.client)
	if err != nil {
		return fmt.Errorf("failed to deploy dashboard app: %s", err)
	}
	return nil
}

func (c *TsuruAPI) Uninstall(installation string) error {
	manager := cmd.BuildBaseManager("uninstall-client", "0.0.0", "", nil)
	provisioners, err := provision.Registry()
	if err != nil {
		return err
	}
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
