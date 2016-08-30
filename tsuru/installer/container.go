// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"os"
	"strconv"
	"strings"

	"github.com/docker/engine-api/types/mount"
	"github.com/docker/engine-api/types/swarm"
	"github.com/fsouza/go-dockerclient"
)

type dockerEnpoint interface {
	dockerClient() (*docker.Client, error)
	GetNetwork() *docker.Network
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
