// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"strconv"

	"github.com/docker/engine-api/types/swarm"
	"github.com/fsouza/go-dockerclient"
)

var swarmPort = 2377

type SwarmCluster struct {
	Manager *Machine
	Workers []*Machine
	network *docker.Network
}

func (c *SwarmCluster) dockerClient() (*docker.Client, error) {
	return c.Manager.dockerClient()
}

// NewSwarmCluster creates a Swarm Cluster using the first machine as a manager
// and the rest as workers and also creates an overlay network between the nodes.
func NewSwarmCluster(machines []*Machine) (*SwarmCluster, error) {
	swarmOpts := docker.InitSwarmOptions{
		InitRequest: swarm.InitRequest{
			ListenAddr:    fmt.Sprintf("0.0.0.0:%d", swarmPort),
			AdvertiseAddr: fmt.Sprintf("%s:%d", machines[0].IP, swarmPort),
		},
	}
	dockerClient, err := machines[0].dockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve machine %s docker client: %s", machines[0].Name, err)
	}
	_, err = dockerClient.InitSwarm(swarmOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to init swarm: %s", err)
	}
	swarmInspect, err := dockerClient.InspectSwarm(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect swarm: %s", err)
	}
	createNetworkOpts := docker.CreateNetworkOptions{
		Name:           "tsuru",
		Driver:         "overlay",
		CheckDuplicate: true,
		IPAM: docker.IPAMOptions{
			Driver: "default",
			Config: []docker.IPAMConfig{
				{
					Subnet: "10.0.9.0/24",
				},
			},
		},
	}
	network, err := dockerClient.CreateNetwork(createNetworkOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create overlay network: %s", err)
	}
	for i, m := range machines {
		if i == 0 {
			continue
		}
		dockerClient, err = m.dockerClient()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve machine %s docker client: %s", m.Name, err)
		}
		opts := docker.JoinSwarmOptions{
			JoinRequest: swarm.JoinRequest{
				ListenAddr:    fmt.Sprintf("0.0.0.0:%d", swarmPort),
				AdvertiseAddr: fmt.Sprintf("%s:%d", m.IP, swarmPort),
				JoinToken:     swarmInspect.JoinTokens.Worker,
				RemoteAddrs:   []string{fmt.Sprintf("%s:%d", machines[0].IP, swarmPort)},
			},
		}
		err = dockerClient.JoinSwarm(opts)
		if err != nil {
			return nil, fmt.Errorf("machine %s failed to join swarm: %s", m.Name, err)
		}
	}
	return &SwarmCluster{
		Manager: machines[0],
		Workers: machines,
		network: network,
	}, nil
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
	var machine *Machine
	for _, m := range c.Workers {
		if m.Name == nodeName {
			machine = m
			break
		}
	}
	if machine == nil {
		return fmt.Errorf("machine %s not found", nodeName)
	}
	client, err := machine.dockerClient()
	if err != nil {
		return fmt.Errorf("failed to retrieve task node %s docker client: %s", machine.Name, err)
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
	opts.Networks = []swarm.NetworkAttachmentConfig{
		{Target: c.network.Name},
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
			ip = m.IP
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
func (c *SwarmCluster) GetMachine(name string) (*Machine, error) {
	for _, m := range c.Workers {
		if m.Name == name {
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

func (c *SwarmCluster) ServiceInfo(name string) (*ServiceInfo, error) {
	client, err := c.dockerClient()
	if err != nil {
		return nil, err
	}
	service, err := client.InspectService(name)
	if err != nil {
		return nil, err
	}
	var ports []string
	for _, p := range service.Endpoint.Ports {
		ports = append(ports, strconv.Itoa(int(p.PublishedPort)))
	}
	return &ServiceInfo{
		Name:     name,
		Replicas: int(*service.Spec.Mode.Replicated.Replicas),
		Ports:    ports,
	}, nil
}
