// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/docker/engine-api/types/swarm"
	"github.com/docker/machine/libmachine/host"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	"gopkg.in/check.v1"
)

type testEndpoint struct {
	endpoint string
	certPath string
}

func (t testEndpoint) dockerClient() (*docker.Client, error) {
	return docker.NewTLSClient(
		t.endpoint,
		filepath.Join(t.certPath, "cert.pem"),
		filepath.Join(t.certPath, "key.pem"),
		filepath.Join(t.certPath, "ca.pem"),
	)
}

func (t testEndpoint) GetNetwork() *docker.Network {
	return &docker.Network{}
}

func (s *S) TestCreateContainer(c *check.C) {
	var requests []*http.Request
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	server, err := testing.NewTLSServer("127.0.0.1:0", nil, func(r *http.Request) {
		if r.URL.Path != "/version" {
			requests = append(requests, r)
		}
	}, tlsConfig)
	c.Assert(err, check.IsNil)
	defer server.Stop()
	endpoint := testEndpoint{endpoint: server.URL(), certPath: s.TLSCertsPath.RootDir}
	config := &docker.Config{Image: "tsuru/api:v1"}
	err = createContainer(endpoint, "contName", config, nil)
	c.Assert(err, check.IsNil)
	c.Assert(requests, check.HasLen, 3)
	c.Assert(requests[0].URL.Path, check.Equals, "/images/create")
	c.Assert(requests[1].URL.Path, check.Equals, "/images/tsuru/api:v1/json")
	c.Assert(requests[2].URL.Path, check.Equals, "/services/create")
}

func (s *S) TestCreateContainerWithExposedPorts(c *check.C) {
	containerChan := make(chan *docker.Container, 2)
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	server, err := testing.NewTLSServer("127.0.0.1:0", containerChan, nil, tlsConfig)
	c.Assert(err, check.IsNil)
	defer server.Stop()
	server.CustomHandler("/images/.*/json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		image := docker.Image{
			ID: "tsuru/api",
			Config: &docker.Config{
				ExposedPorts: map[docker.Port]struct{}{
					docker.Port("90/tcp"): {},
				},
			},
		}
		buf, errMarshal := json.Marshal(image)
		c.Assert(errMarshal, check.IsNil)
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.CustomHandler("/images/.*/json", server.DefaultHandler())
	expected := map[docker.Port][]docker.PortBinding{
		"90/tcp": {
			{HostIP: "0.0.0.0", HostPort: "90"},
		},
	}
	endpoint := testEndpoint{endpoint: server.URL(), certPath: s.TLSCertsPath.RootDir}
	config := &docker.Config{Image: "tsuru/api:v1"}
	err = createContainer(endpoint, "contName", config, nil)
	c.Assert(err, check.IsNil)
	cont := <-containerChan
	c.Assert(cont, check.NotNil)
	c.Assert(expected, check.DeepEquals, cont.HostConfig.PortBindings)
}

func (s *S) TestCreateContainerExposedPortsNotOverridesHostConfig(c *check.C) {
	containerChan := make(chan *docker.Container, 2)
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	server, err := testing.NewTLSServer("127.0.0.1:0", containerChan, nil, tlsConfig)
	c.Assert(err, check.IsNil)
	defer server.Stop()
	server.CustomHandler("/images/.*/json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		image := docker.Image{
			ID: "tsuru/api",
			Config: &docker.Config{
				ExposedPorts: map[docker.Port]struct{}{
					docker.Port("90/tcp"): {},
				},
			},
		}
		buf, errMarshal := json.Marshal(image)
		c.Assert(errMarshal, check.IsNil)
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.CustomHandler("/images/.*/json", server.DefaultHandler())
	expected := map[docker.Port][]docker.PortBinding{
		"100/tcp": {
			{HostIP: "0.0.0.0", HostPort: "100"},
		},
	}
	endpoint := testEndpoint{endpoint: server.URL(), certPath: s.TLSCertsPath.RootDir}
	config := &docker.Config{Image: "tsuru/api:v1"}
	hostConfig := &docker.HostConfig{
		PortBindings: map[docker.Port][]docker.PortBinding{
			"100/tcp": {
				{HostIP: "0.0.0.0", HostPort: "100"},
			},
		},
	}
	err = createContainer(endpoint, "contName", config, hostConfig)
	c.Assert(err, check.IsNil)
	cont := <-containerChan
	c.Assert(cont, check.NotNil)
	c.Assert(expected, check.DeepEquals, cont.HostConfig.PortBindings)
}

func (s *S) TestNewSwarmCluster(c *check.C) {
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	var managerReqs, workerReqs []*http.Request
	managerServer, err := testing.NewTLSServer("127.0.0.1:0", nil, func(r *http.Request) {
		managerReqs = append(managerReqs, r)
		if r.URL.Path == "/swarm/init" {
			var initReq swarm.InitRequest
			errDec := json.NewDecoder(r.Body).Decode(&initReq)
			c.Assert(errDec, check.IsNil)
			c.Assert(initReq.AdvertiseAddr, check.Equals, "127.0.0.1:2377")
			c.Assert(initReq.ListenAddr, check.Equals, "0.0.0.0:2377")
		}
	}, tlsConfig)
	c.Assert(err, check.IsNil)
	defer managerServer.Stop()
	managerMachine := &Machine{
		Host:    &host.Host{Name: "manager"},
		IP:      "127.0.0.1",
		Address: managerServer.URL(),
		CAPath:  s.TLSCertsPath.RootDir,
		network: &docker.Network{Name: "tsuru"},
	}
	workerServer, err := testing.NewTLSServer("127.0.0.1:0", nil, func(r *http.Request) {
		workerReqs = append(workerReqs, r)
		if r.URL.Path == "/swarm/join" {
			var joinReq swarm.JoinRequest
			errDec := json.NewDecoder(workerReqs[0].Body).Decode(&joinReq)
			c.Assert(errDec, check.IsNil)
			c.Assert(joinReq.RemoteAddrs, check.DeepEquals, []string{"127.0.0.1:2377"})
			c.Assert(joinReq.AdvertiseAddr, check.Equals, "127.0.0.2:2377")
			c.Assert(joinReq.ListenAddr, check.Equals, "0.0.0.0:2377")
		}
	}, tlsConfig)
	c.Assert(err, check.IsNil)
	defer workerServer.Stop()
	workerMachine := &Machine{
		Host:    &host.Host{Name: "worker"},
		IP:      "127.0.0.2",
		Address: workerServer.URL(),
		CAPath:  s.TLSCertsPath.RootDir,
		network: &docker.Network{Name: "tsuru"},
	}
	cluster, err := NewSwarmCluster([]*Machine{managerMachine, workerMachine})
	c.Assert(err, check.IsNil)
	c.Assert(cluster, check.NotNil)
	c.Assert(managerReqs[0].URL.Path, check.Equals, "/swarm/init")
	c.Assert(managerReqs[1].URL.Path, check.Equals, "/swarm")
	c.Assert(managerReqs[2].URL.Path, check.Equals, "/networks/create")
	c.Assert(workerReqs[0].URL.Path, check.Equals, "/swarm/join")
}

func (s *S) TestServiceExec(c *check.C) {
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	execStarted := false
	managerServer, err := testing.NewTLSServer("127.0.0.1:0", nil, nil, tlsConfig)
	c.Assert(err, check.IsNil)
	defer managerServer.Stop()
	workerServer, err := testing.NewTLSServer("127.0.0.1:0", nil, func(r *http.Request) {
		if r.URL.Path == "/exec/exec-ID/start" {
			execStarted = true
		}
	}, tlsConfig)
	c.Assert(err, check.IsNil)
	defer workerServer.Stop()
	managerMachine := &Machine{
		Host:    &host.Host{Name: "manager"},
		IP:      "127.0.0.1",
		Address: managerServer.URL(),
		CAPath:  s.TLSCertsPath.RootDir,
		network: &docker.Network{Name: "tsuru"},
	}
	workerMachine := &Machine{
		Host:    &host.Host{Name: "worker"},
		IP:      "127.0.0.1",
		Address: workerServer.URL(),
		CAPath:  s.TLSCertsPath.RootDir,
		network: &docker.Network{Name: "tsuru"},
	}
	cluster := &SwarmCluster{
		Manager: managerMachine,
		Workers: []*Machine{managerMachine, workerMachine},
	}
	err = createContainer(cluster, "tsuru", &docker.Config{Image: "tsuru/tsuru"}, &docker.HostConfig{})
	c.Assert(err, check.IsNil)
	client, err := cluster.Workers[0].dockerClient()
	c.Assert(err, check.IsNil)
	cont, err := client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{Image: "tsuru/tsuru"},
	})
	c.Assert(err, check.IsNil)
	managerServer.CustomHandler("/tasks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var filters map[string][]string
		errJSON := json.Unmarshal([]byte(r.URL.Query().Get("filters")), &filters)
		c.Assert(errJSON, check.IsNil)
		c.Assert(filters["service"], check.DeepEquals, []string{"tsuru"})
		c.Assert(filters["desired-state"], check.DeepEquals, []string{"running"})
		tasks := []swarm.Task{
			{
				ID:     "123",
				NodeID: "node-123",
				Status: swarm.TaskStatus{
					ContainerStatus: swarm.ContainerStatus{
						ContainerID: cont.ID},
				},
			},
		}
		buf, errMarshal := json.Marshal(tasks)
		c.Assert(errMarshal, check.IsNil)
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf)
		w.WriteHeader(http.StatusOK)
	}))
	managerServer.CustomHandler("/nodes/node-123", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		node := swarm.Node{
			Description: swarm.NodeDescription{
				Hostname: "worker",
			},
		}
		buf, errMarshal := json.Marshal(node)
		c.Assert(errMarshal, check.IsNil)
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf)
		w.WriteHeader(http.StatusOK)
	}))
	workerServer.CustomHandler("/containers/"+cont.ID+"/exec", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var opts docker.CreateExecOptions
		errJSON := json.NewDecoder(r.Body).Decode(&opts)
		c.Assert(errJSON, check.IsNil)
		c.Assert(opts.Cmd, check.DeepEquals, []string{"exit"})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"Id": "exec-ID"})
	}))
	err = cluster.ServiceExec("tsuru", []string{"exit"}, docker.StartExecOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(execStarted, check.Equals, true)
}
