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

var defaultTsuruAPIPort = 8080

type ComponentsConfig struct {
	InstallDashboard bool
	TsuruAPIConfig
}

func NewInstallConfig(targetName string) *ComponentsConfig {
	installDashboard, err := config.GetBool("components:tsuru:install-dashboard")
	if err != nil {
		installDashboard = true
	}
	return &ComponentsConfig{
		TsuruAPIConfig: TsuruAPIConfig{
			TargetName:       targetName,
			RootUserEmail:    "admin@example.com",
			RootUserPassword: "admin123",
		},
		InstallDashboard: installDashboard,
	}
}

type TsuruAPI struct{}

type TsuruAPIConfig struct {
	TargetName       string
	RootUserEmail    string
	RootUserPassword string
	IaaSConfig       iaasConfig
}

type iaasConfig struct {
	Dockermachine iaasConfigInternal `json:"dockermachine,omitempty"`
}

type iaasConfigInternal struct {
	CaPath           string           `json:"ca-path,omitempty"`
	InsecureRegistry string           `json:"insecure-registry,omitempty"`
	Driver           iaasConfigDriver `json:"driver,omitempty"`
}

type iaasConfigDriver struct {
	Name    string                 `json:"name,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
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
	return cluster.ServiceExec("tsuru_tsuru", cmd, startOpts)
}

type Bootstraper interface {
	Bootstrap(opts BoostrapOptions) error
}

type BoostrapOptions struct {
	Login            string
	Password         string
	Target           string
	TargetName       string
	NodesToRegister  []string
	NodesToCreate    int
	NodesParams      map[string][]interface{}
	InstallDashboard bool
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
	err = s.createNodes("theonepool", opts.NodesToCreate, opts.NodesParams)
	if err != nil {
		return err
	}
	if opts.InstallDashboard {
		err = s.addPlatform("python")
		if err != nil {
			return err
		}
		err = s.addTeam("admin")
		if err != nil {
			return err
		}
		err = s.installDashboard()
	}
	return err
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

func (s *TsuruBoostraper) createNodes(pool string, nodes int, nodesParams map[string][]interface{}) error {
	nodeAdd := admin.AddNodeCmd{}
	for i := 0; i < nodes; i++ {
		fmt.Printf("creating node %d/%d...\n", i+1, nodes)
		s.context.Args = []string{"docker", "iaas=dockermachine", fmt.Sprintf("pool=%s", pool)}
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
