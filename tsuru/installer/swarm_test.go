// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/machine/libmachine/host"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	"gopkg.in/check.v1"
)

type FakeServiceCluster struct {
	Services chan<- docker.CreateServiceOptions
}

func (c *FakeServiceCluster) GetManager() *dockermachine.Machine {
	return &dockermachine.Machine{
		Base: &iaas.Machine{Address: "127.0.0.1", Port: 2376},
		Host: &host.Host{},
	}
}

func (c *FakeServiceCluster) CreateService(opts docker.CreateServiceOptions) error {
	if c.Services != nil {
		c.Services <- opts
	}
	return nil
}

func (c *FakeServiceCluster) ServiceExec(service string, cmd []string, opts docker.StartExecOptions) error {
	return nil
}

func (c *FakeServiceCluster) ServicesInfo() ([]ServiceInfo, error) {
	return []ServiceInfo{{Name: "service", Replicas: 1, Ports: []string{"8080"}}}, nil
}

func (c *FakeServiceCluster) ClusterInfo() ([]NodeInfo, error) {
	return []NodeInfo{{IP: "127.0.0.1", State: "running", Manager: true}}, nil
}

type testCluster struct {
	SwarmCluster  *SwarmCluster
	ManagerServer *testing.DockerServer
	WorkerServer  *testing.DockerServer
	clientCert    []byte
	clientKey     []byte
	caCert        []byte
}

func (c *testCluster) Stop() {
	c.ManagerServer.Stop()
	c.WorkerServer.Stop()
}

func getPort(server *testing.DockerServer) (int, error) {
	mPort := strings.Split(server.URL()[:len(server.URL())-1], ":")
	return strconv.Atoi(mPort[len(mPort)-1])
}

func (s *S) createCluster() (*testCluster, error) {
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	clientCert, err := ioutil.ReadFile(s.TLSCertsPath.ClientCert)
	if err != nil {
		return nil, err
	}
	clientKey, err := ioutil.ReadFile(s.TLSCertsPath.ClientKey)
	if err != nil {
		return nil, err
	}
	caCert, err := ioutil.ReadFile(s.TLSCertsPath.RootCert)
	if err != nil {
		return nil, err
	}
	managerServer, err := testing.NewTLSServer("127.0.0.1:0", nil, nil, tlsConfig)
	if err != nil {
		return nil, err
	}
	workerServer, err := testing.NewTLSServer("127.0.0.1:0", nil, nil, tlsConfig)
	if err != nil {
		return nil, err
	}
	port, err := getPort(managerServer)
	if err != nil {
		return nil, err
	}
	managerMachine := &dockermachine.Machine{
		Base: &iaas.Machine{
			Address:    "127.0.0.1",
			Protocol:   "https",
			Port:       port,
			ClientCert: clientCert,
			ClientKey:  clientKey,
			CaCert:     caCert,
		},
		Host: &host.Host{Name: "manager"},
	}
	port, err = getPort(workerServer)
	if err != nil {
		return nil, err
	}
	workerMachine := &dockermachine.Machine{
		Base: &iaas.Machine{
			Address:    "127.0.0.1",
			Protocol:   "https",
			Port:       port,
			ClientCert: clientCert,
			ClientKey:  clientKey,
			CaCert:     caCert,
		},
		Host: &host.Host{Name: "worker"},
	}
	swarm, err := NewSwarmCluster([]*dockermachine.Machine{managerMachine, workerMachine})
	if err != nil {
		return nil, err
	}
	return &testCluster{
		SwarmCluster:  swarm,
		ManagerServer: managerServer,
		WorkerServer:  workerServer,
		clientCert:    clientCert,
		clientKey:     clientKey,
		caCert:        caCert,
	}, nil
}

func (s *S) TestNewSwarmCluster(c *check.C) {
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	clientCert, err := ioutil.ReadFile(s.TLSCertsPath.ClientCert)
	c.Assert(err, check.IsNil)
	clientKey, err := ioutil.ReadFile(s.TLSCertsPath.ClientKey)
	c.Assert(err, check.IsNil)
	caCert, err := ioutil.ReadFile(s.TLSCertsPath.RootCert)
	c.Assert(err, check.IsNil)
	var managerReqs, workerReqs []*http.Request
	managerServer, err := testing.NewTLSServer("127.0.0.1:0", nil, func(r *http.Request) {
		managerReqs = append(managerReqs, r)
	}, tlsConfig)
	c.Assert(err, check.IsNil)
	defer managerServer.Stop()
	port, err := getPort(managerServer)
	c.Assert(err, check.IsNil)
	m1 := &dockermachine.Machine{
		Base: &iaas.Machine{
			Address:    "127.0.0.1",
			Protocol:   "https",
			Port:       port,
			ClientCert: clientCert,
			ClientKey:  clientKey,
			CaCert:     caCert,
		},
		Host: &host.Host{Name: "manager"},
	}
	workerServer, err := testing.NewTLSServer("127.0.0.1:0", nil, func(r *http.Request) {
		workerReqs = append(workerReqs, r)
	}, tlsConfig)
	c.Assert(err, check.IsNil)
	defer workerServer.Stop()
	port, err = getPort(workerServer)
	c.Assert(err, check.IsNil)
	m2 := &dockermachine.Machine{
		Base: &iaas.Machine{
			Address:    "127.0.0.1",
			Protocol:   "https",
			Port:       port,
			ClientCert: clientCert,
			ClientKey:  clientKey,
			CaCert:     caCert,
		},
		Host: &host.Host{Name: "worker"},
	}
	cluster, err := NewSwarmCluster([]*dockermachine.Machine{m1, m2})
	c.Assert(err, check.IsNil)
	c.Assert(cluster, check.NotNil)
	c.Assert(cluster.Managers, check.DeepEquals, []*dockermachine.Machine{m1, m2})
	c.Assert(cluster.Workers, check.DeepEquals, []*dockermachine.Machine{m1, m2})
	c.Assert(managerReqs[0].URL.Path, check.Equals, "/swarm/init")
	c.Assert(managerReqs[1].URL.Path, check.Equals, "/swarm")
	c.Assert(workerReqs[0].URL.Path, check.Equals, "/swarm/join")
}

func (s *S) TestCreateService(c *check.C) {
	testCluster, err := s.createCluster()
	c.Assert(err, check.IsNil)
	cluster := testCluster.SwarmCluster
	err = cluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{},
			},
		},
	})
	c.Assert(err, check.IsNil)
	client, err := cluster.dockerClient()
	c.Assert(err, check.IsNil)
	services, err := client.ListServices(docker.ListServicesOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(len(services), check.Equals, 1)
}

func (s *S) TestServiceExec(c *check.C) {
	testCluster, err := s.createCluster()
	c.Assert(err, check.IsNil)
	defer testCluster.Stop()
	execStarted := false
	testCluster.ManagerServer.CustomHandler("/tasks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var filters map[string][]string
		errJSON := json.Unmarshal([]byte(r.URL.Query().Get("filters")), &filters)
		c.Assert(errJSON, check.IsNil)
		c.Assert(filters["service"], check.DeepEquals, []string{"tsuru"})
		c.Assert(filters["desired-state"], check.DeepEquals, []string{"running"})
		tasks := []swarm.Task{
			{
				ID:     "123",
				NodeID: "node-id",
				Status: swarm.TaskStatus{
					ContainerStatus: &swarm.ContainerStatus{
						ContainerID: "container-id"},
				},
			},
		}
		buf, errMarshal := json.Marshal(tasks)
		c.Assert(errMarshal, check.IsNil)
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf)
		w.WriteHeader(http.StatusOK)
	}))
	testCluster.ManagerServer.CustomHandler("/nodes/node-id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	testCluster.WorkerServer.CustomHandler("/containers/container-id/exec", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var opts docker.CreateExecOptions
		errJSON := json.NewDecoder(r.Body).Decode(&opts)
		c.Assert(errJSON, check.IsNil)
		c.Assert(opts.Cmd, check.DeepEquals, []string{"exit"})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"Id": "exec-ID"})
	}))
	testCluster.WorkerServer.CustomHandler("/exec/exec-ID/start", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		execStarted = true
		w.WriteHeader(http.StatusOK)
	}))
	err = testCluster.SwarmCluster.ServiceExec("tsuru", []string{"exit"}, docker.StartExecOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(execStarted, check.Equals, true)
}

func (s *S) TestServicesInfo(c *check.C) {
	testCluster, err := s.createCluster()
	c.Assert(err, check.IsNil)
	replicas := uint64(2)
	testCluster.SwarmCluster.CreateService(docker.CreateServiceOptions{
		ServiceSpec: swarm.ServiceSpec{
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{},
			},
			Annotations:  swarm.Annotations{Name: "tsuru"},
			Mode:         swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
			EndpointSpec: &swarm.EndpointSpec{Ports: []swarm.PortConfig{{PublishedPort: 80}}},
		},
	})
	info, err := testCluster.SwarmCluster.ServicesInfo()
	c.Assert(err, check.IsNil)
	c.Assert(info, check.DeepEquals, []ServiceInfo{{Name: "tsuru", Replicas: 2, Ports: []string{"80"}}})
}

func (s *S) TestClusterInfo(c *check.C) {
	testCluster, err := s.createCluster()
	c.Assert(err, check.IsNil)
	expected := []NodeInfo{
		{IP: "???", State: "ready", Manager: true},
		{IP: "???", State: "ready", Manager: true},
	}
	info, err := testCluster.SwarmCluster.ClusterInfo()
	c.Assert(err, check.IsNil)
	c.Assert(info, check.DeepEquals, expected)
}

func (s *S) TestGetMachine(c *check.C) {
	testCluster, err := s.createCluster()
	c.Assert(err, check.IsNil)
	m, err := testCluster.SwarmCluster.GetMachine("manager")
	c.Assert(err, check.IsNil)
	c.Assert(m.Base.Address, check.DeepEquals, "127.0.0.1")
}

func (s *S) TestGetMachineNotFound(c *check.C) {
	testCluster, err := s.createCluster()
	c.Assert(err, check.IsNil)
	m, err := testCluster.SwarmCluster.GetMachine("not-found")
	c.Assert(err, check.NotNil)
	c.Assert(m, check.IsNil)
}
