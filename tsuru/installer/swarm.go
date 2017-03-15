// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/swarm"
	"github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	"github.com/tsuru/tsuru/iaas/dockermachine"
)

var swarmPort = 2377

type ServiceCluster interface {
	GetManager() *dockermachine.Machine
	ServiceExec(string, []string, docker.StartExecOptions) error
	CreateService(docker.CreateServiceOptions) error
	ServicesInfo() ([]ServiceInfo, error)
	ClusterInfo() ([]NodeInfo, error)
}

type SwarmCluster struct {
	Managers []*dockermachine.Machine
	Workers  []*dockermachine.Machine
}

func (c *SwarmCluster) dockerClient() (*docker.Client, error) {
	return getDockerClient(c.GetManager())
}

func (c *SwarmCluster) GetManager() *dockermachine.Machine {
	return c.Managers[0]
}

// NewSwarmCluster creates a Swarm Cluster using the first machine as a manager
// and the rest as workers and also creates an overlay network between the nodes.
func NewSwarmCluster(machines []*dockermachine.Machine) (*SwarmCluster, error) {
	managerDockerClient, err := getDockerClient(machines[0])
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve machine %s docker client: %s", machines[0].Host.Name, err)
	}
	err = initSwarm(managerDockerClient, fmt.Sprintf("%s:%d", dm.GetPrivateIP(machines[0]), swarmPort))
	if err != nil {
		return nil, fmt.Errorf("failed to init swarm: %s", err)
	}
	var managers []*dockermachine.Machine
	for i, m := range machines {
		managers = append(managers, m)
		if i == 0 {
			continue
		}
		dockerClient, err := getDockerClient(m)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve machine %s docker client: %s", m.Host.Name, err)
		}
		err = joinSwarm(managerDockerClient, dockerClient)
		if err != nil {
			return nil, fmt.Errorf("machine %s failed to join swarm: %s", m.Host.Name, err)
		}
	}
	return &SwarmCluster{
		Managers: managers,
		Workers:  machines,
	}, nil
}

func initSwarm(client *docker.Client, addr string) error {
	_, err := client.InitSwarm(docker.InitSwarmOptions{
		InitRequest: swarm.InitRequest{
			ListenAddr:    fmt.Sprintf("0.0.0.0:%d", swarmPort),
			AdvertiseAddr: addr,
		},
	})
	if err != nil && errors.Cause(err) != docker.ErrNodeAlreadyInSwarm {
		return errors.WithStack(err)
	}
	return nil
}

func joinSwarm(existingClient *docker.Client, newClient *docker.Client) error {
	swarmInfo, err := existingClient.InspectSwarm(nil)
	if err != nil {
		return errors.WithStack(err)
	}
	dockerInfo, err := existingClient.Info()
	if err != nil {
		return errors.WithStack(err)
	}
	if len(dockerInfo.Swarm.RemoteManagers) == 0 {
		return errors.Errorf("no remote managers found in node %#v", dockerInfo)
	}
	addrs := make([]string, len(dockerInfo.Swarm.RemoteManagers))
	for i, peer := range dockerInfo.Swarm.RemoteManagers {
		addrs[i] = peer.Addr
	}
	opts := docker.JoinSwarmOptions{
		JoinRequest: swarm.JoinRequest{
			ListenAddr:  fmt.Sprintf("0.0.0.0:%d", swarmPort),
			JoinToken:   swarmInfo.JoinTokens.Manager,
			RemoteAddrs: addrs,
		},
	}
	err = newClient.JoinSwarm(opts)
	if err != nil {
		if err == docker.ErrNodeAlreadyInSwarm {
			return nil
		}
		return errors.WithStack(err)
	}
	return nil
}

// ServiceExec finds a container running a service task and runs exec on it
func (c *SwarmCluster) ServiceExec(service string, cmd []string, startOpts docker.StartExecOptions) error {
	mClient, err := c.dockerClient()
	if err != nil {
		return fmt.Errorf("failed to retrive swarm docker client: %s", err)
	}
	tasks, err := mClient.ListTasks(docker.ListTasksOptions{
		Filters: map[string][]string{
			"service":       {service},
			"desired-state": {"running"},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks for service %s: %s", service, err)
	}
	if len(tasks) == 0 {
		return fmt.Errorf("no running task found for service %s", service)
	}
	node, err := mClient.InspectNode(tasks[0].NodeID)
	if err != nil {
		return fmt.Errorf("failed to inspect node %s: %s", tasks[0].NodeID, err)
	}
	nodeName := node.Description.Hostname
	var machine *dockermachine.Machine
	for _, m := range c.Workers {
		if m.Host.Name == nodeName {
			machine = m
			break
		}
	}
	if machine == nil {
		return fmt.Errorf("machine %s not found", nodeName)
	}
	client, err := getDockerClient(machine)
	if err != nil {
		return fmt.Errorf("failed to retrieve task node %s docker client: %s", machine.Host.Name, err)
	}
	container := tasks[0].Status.ContainerStatus.ContainerID
	exec, err := client.CreateExec(docker.CreateExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Container:    container,
	})
	if err != nil {
		return fmt.Errorf("failed to exec in task container %s: %s", container, err)
	}
	return client.StartExec(exec.ID, startOpts)
}

// CreateService creates a service on the swarm cluster
func (c *SwarmCluster) CreateService(opts docker.CreateServiceOptions) error {
	client, err := c.dockerClient()
	if err != nil {
		return err
	}
	_, err = client.CreateService(opts)
	return err
}

type NodeInfo struct {
	IP      string
	State   string
	Manager bool
}

func (c *SwarmCluster) ClusterInfo() ([]NodeInfo, error) {
	client, err := c.dockerClient()
	if err != nil {
		return nil, err
	}
	nodes, err := client.ListNodes(docker.ListNodesOptions{})
	if err != nil {
		return nil, err
	}
	var infos []NodeInfo
	for _, n := range nodes {
		var ip string
		m, err := c.GetMachine(n.Description.Hostname)
		if err != nil {
			ip = "???"
		} else {
			ip = m.Base.Address
		}
		infos = append(infos, NodeInfo{
			IP:      ip,
			State:   string(n.Status.State),
			Manager: n.ManagerStatus != nil,
		})
	}
	return infos, nil
}

// GetMachine retrieves a worker machine by its name
func (c *SwarmCluster) GetMachine(name string) (*dockermachine.Machine, error) {
	for _, m := range c.Workers {
		if m.Host.Name == name {
			return m, nil
		}
	}
	return nil, fmt.Errorf("machine %s not found", name)
}

type ServiceInfo struct {
	Name     string
	Replicas int
	Ports    []string
}

func (c *SwarmCluster) ServicesInfo() ([]ServiceInfo, error) {
	client, err := c.dockerClient()
	if err != nil {
		return nil, err
	}
	services, err := client.ListServices(docker.ListServicesOptions{})
	if err != nil {
		return nil, err
	}
	var infos []ServiceInfo
	for _, s := range services {
		var ports []string
		for _, p := range s.Endpoint.Ports {
			ports = append(ports, strconv.Itoa(int(p.PublishedPort)))
		}
		infos = append(infos, ServiceInfo{
			Name:     s.Spec.Name,
			Replicas: int(*s.Spec.Mode.Replicated.Replicas),
			Ports:    ports,
		})
	}
	return infos, nil
}

func getDockerClient(m *dockermachine.Machine) (*docker.Client, error) {
	return docker.NewTLSClientFromBytes(m.Base.FormatNodeAddress(),
		m.Base.ClientCert,
		m.Base.ClientKey,
		m.Base.CaCert,
	)
}
