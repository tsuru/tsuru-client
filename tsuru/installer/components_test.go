// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	"gopkg.in/check.v1"
)

func (s *S) TestInstallComponents(c *check.C) {
	tests := []struct {
		component     TsuruComponent
		containerName string
		image         string
	}{
		{&MongoDB{}, "mongo", "mongo:latest"},
		{&Redis{}, "redis", "redis:latest"},
		{&PlanB{}, "planb", "tsuru/planb:latest"},
		{&Registry{}, "registry", "registry:2"},
		{&TsuruAPI{}, "tsuru", "tsuru/api:latest"},
	}
	c.Assert(len(tests), check.Equals, len(TsuruComponents))
	containerChan := make(chan *docker.Container)
	server, _ := testing.NewServer("127.0.0.1:0", containerChan, nil)
	mockMachine := &Machine{Address: server.URL()}
	for _, tt := range tests {
		go tt.component.Install(mockMachine, &InstallConfig{})

		cont := <-containerChan
		c.Assert(cont.Name, check.Equals, tt.containerName)
		c.Assert(cont.State.Running, check.Equals, false)
		c.Assert(cont.Image, check.Equals, tt.image)

		cont = <-containerChan
		c.Assert(cont.Name, check.Equals, tt.containerName)
		c.Assert(cont.State.Running, check.Equals, true)
		c.Assert(cont.Image, check.Equals, tt.image)
	}
}

func (s *S) TestInstallComponentsCustomRegistry(c *check.C) {
	tests := []struct {
		component TsuruComponent
		image     string
	}{
		{&MongoDB{}, "myregistry.com/mongo:latest"},
		{&Redis{}, "myregistry.com/redis:latest"},
		{&PlanB{}, "myregistry.com/tsuru/planb:latest"},
		{&Registry{}, "myregistry.com/registry:2"},
		{&TsuruAPI{}, "myregistry.com/tsuru/api:latest"},
	}
	c.Assert(len(tests), check.Equals, len(TsuruComponents))
	containerChan := make(chan *docker.Container)
	server, _ := testing.NewServer("127.0.0.1:0", containerChan, nil)
	mockMachine := &Machine{Address: server.URL()}
	for _, tt := range tests {
		config := &InstallConfig{Registry: "myregistry.com"}
		go tt.component.Install(mockMachine, config)

		cont := <-containerChan
		c.Assert(cont.State.Running, check.Equals, false)
		c.Assert(cont.Image, check.Equals, tt.image)

		cont = <-containerChan
		c.Assert(cont.State.Running, check.Equals, true)
		c.Assert(cont.Image, check.Equals, tt.image)
	}
}
