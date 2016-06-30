// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"os"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

func createContainer(address, name string, config *docker.Config, hostConfig *docker.HostConfig) error {
	client, err := docker.NewClient(address)
	if err != nil {
		return err
	}
	tag := "latest"
	if strings.Contains(config.Image, ":") {
		tag = strings.Split(config.Image, ":")[1]
	}
	pullOpts := docker.PullImageOptions{
		Repository:   config.Image,
		OutputStream: os.Stdout,
		Tag:          tag,
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
		hostConfig.PortBindings = make(map[docker.Port][]docker.PortBinding)
		for k := range imageInspect.Config.ExposedPorts {
			hostConfig.PortBindings[k] = []docker.PortBinding{{HostIP: "0.0.0.0", HostPort: k.Port()}}
		}
	}
	opts := docker.CreateContainerOptions{Config: config, HostConfig: hostConfig, Name: name}
	container, err := client.CreateContainer(opts)
	if err != nil {
		return err
	}
	return client.StartContainer(container.ID, nil)
}
