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

	"github.com/docker/engine-api/types/swarm"
	"github.com/docker/machine/libmachine/host"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	"gopkg.in/check.v1"
)

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
	return &testCluster{
		SwarmCluster: &SwarmCluster{
			Managers: []*dockermachine.Machine{managerMachine},
			Workers:  []*dockermachine.Machine{managerMachine, workerMachine},
			network:  &docker.Network{Name: "tsuru"},
		},
		ManagerServer: managerServer,
		WorkerServer:  workerServer,
		clientCert:    clientCert,
		clientKey:     clientKey,
		caCert:        caCert,
	}, nil
}

func (s *S) TestNewSwarmCluster(c *check.C) {
	swarmPort = 0
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
	c.Assert(managerReqs[1].URL.Path, check.Equals, "/networks/create")
	c.Assert(managerReqs[2].URL.Path, check.Equals, "/swarm")
	c.Assert(workerReqs[0].URL.Path, check.Equals, "/swarm/join")
}

func (s *S) TestNewSwarmClusterMultipleManagers(c *check.C) {
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
}

func (s *S) TestCreateService(c *check.C) {
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
	var created = false
	server, err := testing.NewTLSServer("127.0.0.1:0", nil, func(r *http.Request) {
		if r.URL.Path == "/services/create" {
			created = true
		}
	}, tlsConfig)
	port, err := getPort(server)
	c.Assert(err, check.IsNil)
	m := &dockermachine.Machine{
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
	cluster, err := NewSwarmCluster([]*dockermachine.Machine{m})
	c.Assert(err, check.IsNil)
	err = cluster.CreateService(docker.CreateServiceOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(created, check.Equals, true)
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
					ContainerStatus: swarm.ContainerStatus{
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

func (s *S) TestServiceInfo(c *check.C) {
	testCluster, err := s.createCluster()
	c.Assert(err, check.IsNil)
	testCluster.ManagerServer.CustomHandler("/services/tsuru", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service := swarm.Service{
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: "tsuru",
				},
				Mode: swarm.ServiceMode{
					Replicated: &swarm.ReplicatedService{
						Replicas: &[]uint64{2}[0],
					},
				},
			},
			Endpoint: swarm.Endpoint{
				Ports: []swarm.PortConfig{
					{PublishedPort: uint32(80)},
				},
			},
		}
		json.NewEncoder(w).Encode(service)
	}))
	info, err := testCluster.SwarmCluster.ServiceInfo("tsuru")
	c.Assert(err, check.IsNil)
	c.Assert(info, check.DeepEquals, &ServiceInfo{Name: "tsuru", Replicas: 2, Ports: []string{"80"}})
}

func (s *S) TestClusterInfo(c *check.C) {
	testCluster, err := s.createCluster()
	c.Assert(err, check.IsNil)
	testCluster.ManagerServer.CustomHandler("/nodes", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nodes := []swarm.Node{
			{
				Description: swarm.NodeDescription{
					Hostname: "manager",
				},
				Status: swarm.NodeStatus{
					State: swarm.NodeStateReady,
				},
				ManagerStatus: &swarm.ManagerStatus{},
			},
			{
				Description: swarm.NodeDescription{
					Hostname: "worker",
				},
				Status: swarm.NodeStatus{
					State: swarm.NodeStateDown,
				},
			},
		}
		json.NewEncoder(w).Encode(nodes)
	}))
	expected := []NodeInfo{
		{IP: "127.0.0.1", State: "ready", Manager: true},
		{IP: "127.0.0.1", State: "down", Manager: false},
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
