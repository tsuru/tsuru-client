// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/docker/engine-api/types/mount"
	"github.com/docker/engine-api/types/swarm"
	"github.com/fsouza/go-dockerclient"
)

var swarmPort = 2377

type dockerEnpoint interface {
	dockerClient() (*docker.Client, error)
	GetNetwork() *docker.Network
}

type SwarmCluster struct {
	Manager *Machine
	Workers []*Machine
}

func (c *SwarmCluster) dockerClient() (*docker.Client, error) {
	return c.Manager.dockerClient()
}

func (c *SwarmCluster) GetNetwork() *docker.Network {
	return c.Manager.GetNetwork()
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
		m.network = network
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
	}, nil
}

// ServiceTaskExec finds a container running a service task and runs exec on it
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

func createContainer(d dockerEnpoint, name string, config *docker.Config, hostConfig *docker.HostConfig) error {
	client, err := d.dockerClient()
	if err != nil {
		return err
	}
	pullOpts := docker.PullImageOptions{
		Repository:   config.Image,
		OutputStream: os.Stdout,
	}
	err = client.PullImage(pullOpts, docker.AuthConfiguration{})
	if err != nil {
		return err
	}
	imageInspect, err := client.InspectImage(config.Image)
	if err != nil {
		return err
	}
	if hostConfig == nil {
		hostConfig = &docker.HostConfig{}
	}
	hostConfig.RestartPolicy = docker.AlwaysRestart()
	if len(imageInspect.Config.ExposedPorts) > 0 && hostConfig.PortBindings == nil {
		hostConfig.PortBindings = make(map[docker.Port][]docker.PortBinding)
		for k := range imageInspect.Config.ExposedPorts {
			hostConfig.PortBindings[k] = []docker.PortBinding{{HostIP: "0.0.0.0", HostPort: k.Port()}}
		}
	}
	ports := []swarm.PortConfig{}
	for k, p := range hostConfig.PortBindings {
		targetPort, terr := strconv.Atoi(k.Port())
		if terr != nil {
			return terr
		}
		publishedPort, terr := strconv.Atoi(p[0].HostPort)
		if terr != nil {
			return terr
		}
		port := swarm.PortConfig{
			Protocol:      swarm.PortConfigProtocolTCP,
			TargetPort:    uint32(targetPort),
			PublishedPort: uint32(publishedPort),
		}
		ports = append(ports, port)
	}
	mounts := []mount.Mount{}
	for _, bind := range hostConfig.Binds {
		bindParts := strings.Split(bind, ":")
		var ro bool
		if len(bindParts) > 2 {
			ro = true
		}
		mount := mount.Mount{
			Type:     mount.TypeBind,
			Source:   bindParts[0],
			Target:   bindParts[1],
			ReadOnly: ro,
		}
		mounts = append(mounts, mount)
	}
	serviceCreateOpts := docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: name,
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: swarm.ContainerSpec{
					Image:  config.Image,
					Args:   config.Cmd,
					Env:    config.Env,
					Labels: config.Labels,
					Mounts: mounts,
					User:   config.User,
					Dir:    config.WorkingDir,
					TTY:    config.Tty,
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{
				{Target: d.GetNetwork().Name},
			},
			EndpointSpec: &swarm.EndpointSpec{Ports: ports},
		},
	}
	_, err = client.CreateService(serviceCreateOpts)
	return err
}

func (c *SwarmCluster) ListNodes() ([]swarm.Node, error) {
	client, err := c.dockerClient()
	if err != nil {
		return nil, err
	}
	return client.ListNodes(docker.ListNodesOptions{})
}

func (c *SwarmCluster) ListServices() ([]swarm.Service, error) {
	client, err := c.dockerClient()
	if err != nil {
		return nil, err
	}
	return client.ListServices(docker.ListServicesOptions{})
}
