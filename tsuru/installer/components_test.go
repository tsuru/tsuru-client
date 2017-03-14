// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/docker/machine/libmachine/host"
	"github.com/fsouza/go-dockerclient"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	_ "github.com/tsuru/tsuru/provision/docker"
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

func (c *FakeServiceCluster) ServiceInfo(service string) (*ServiceInfo, error) {
	return &ServiceInfo{Name: service, Replicas: 1, Ports: []string{"8080"}}, nil
}

func (c *FakeServiceCluster) ClusterInfo() ([]NodeInfo, error) {
	return []NodeInfo{{IP: "127.0.0.1", State: "running", Manager: true}}, nil
}

func (s *S) TestTsuruAPIBootstrapLocalEnviroment(c *check.C) {
	var paths []string
	expectedPaths := []string{"/1.0/auth/scheme", "/1.0/users/test/tokens",
		"/1.0/pools", "/1.2/node", "/1.0/platforms", "/1.0/teams", "/1.0/apps",
		"/1.0/apps/tsuru-dashboard", "/1.0/apps/tsuru-dashboard/deploy",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		c.Assert(err, check.IsNil)
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/1.0/users/test/tokens" {
			c.Assert(string(b), check.Equals, "password=test")
			token := map[string]string{"token": "test"}
			buf, err := json.Marshal(token)
			c.Assert(err, check.IsNil)
			w.Write(buf)
		}
		if r.URL.Path == "/1.0/pools" {
			c.Assert(string(b), check.Equals, "default=true&force=false&name=theonepool&provisioner=&public=true")
		}
		if r.URL.Path == "/1.2/node" {
			c.Assert(string(b), check.Matches, "Address=&CaCert=&ClientCert=&ClientKey=&Metadata.address=.*&Metadata.pool=theonepool&Register=true&WaitTO=")
		}
		if r.URL.Path == "/1.0/platforms" {
			expected := "FROM tsuru/python"
			c.Assert(strings.Contains(string(b), expected), check.Equals, true)
		}
		if r.URL.Path == "/1.0/teams" {
			c.Assert(string(b), check.Equals, "name=admin")
		}
		if r.URL.Path == "/1.0/apps" {
			c.Assert(string(b), check.Equals, "description=&name=tsuru-dashboard&plan=&platform=python&pool=&router=&routeropts=&teamOwner=admin")
			buf, err := json.Marshal(map[string]string{})
			c.Assert(err, check.IsNil)
			w.Write(buf)
		}
		if r.URL.Path == "/1.0/apps/tsuru-dashboard/deploy" {
			c.Assert(string(b), check.Equals, "image=tsuru%2Fdashboard&origin=image")
			fmt.Fprintln(w, "\nOK")
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
	}))
	defer server.Close()
	bootstraper := TsuruBoostraper{}
	err := bootstraper.Bootstrap(BoostrapOptions{
		Login:            "test",
		Password:         "test",
		Target:           server.URL,
		TargetName:       "test",
		NodesToRegister:  []string{server.URL},
		InstallDashboard: true,
	})
	c.Assert(err, check.IsNil)
	c.Assert(paths, check.DeepEquals, expectedPaths)
	paths = nil
	bootstraper = TsuruBoostraper{}
	err = bootstraper.Bootstrap(BoostrapOptions{
		Login:            "test",
		Password:         "test",
		Target:           server.URL,
		TargetName:       "test2",
		NodesToRegister:  []string{server.URL},
		InstallDashboard: false,
	})
	c.Assert(err, check.IsNil)
	c.Assert(paths, check.DeepEquals, expectedPaths[:4])
}

type FakeRedis struct {
	URL      string
	listener net.Listener
}

func (r *FakeRedis) Listen() {
	l, _ := net.Listen("tcp", r.URL)
	r.URL = l.Addr().String()
	r.listener = l
}

// ListenAndServer starts a fake redis server that listen for a single
// connection and answers to a PING
func (r *FakeRedis) ListenAndServe() {
	conn, _ := r.listener.Accept()
	defer r.listener.Close()
	defer conn.Close()
	buf := make([]byte, 1024)
	conn.Read(buf)
	if strings.Contains(string(buf), "PING") {
		conn.Write([]byte("$2"))
		conn.Write([]byte("\nPONG"))
	}
}
