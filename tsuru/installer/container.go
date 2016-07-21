// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"os"

	"github.com/fsouza/go-dockerclient"
)

type dockerEnpoint interface {
	dockerClient() (*docker.Client, error)
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
	if len(imageInspect.Config.ExposedPorts) > 0 {
		if hostConfig.PortBindings == nil {
			hostConfig.PortBindings = make(map[docker.Port][]docker.PortBinding)
		}
		for k := range imageInspect.Config.ExposedPorts {
			hostConfig.PortBindings[k] = []docker.PortBinding{{HostIP: "0.0.0.0", HostPort: k.Port()}}
		}
	}
	opts := docker.CreateContainerOptions{Config: config, HostConfig: hostConfig, Name: name}
	_, err = client.CreateContainer(opts)
	if err != nil {
		return err
	}
	return client.StartContainer(name, nil)
}

func inspectContainer(d dockerEnpoint, name string) (*docker.Container, error) {
	client, err := d.dockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed create docker client: %s", err)
	}
	container, err := client.InspectContainer(name)
	if err != nil {
		return nil, err
	}
	return container, nil
}
