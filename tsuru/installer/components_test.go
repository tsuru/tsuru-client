// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"

	"github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	"github.com/tsuru/config"
	"github.com/tsuru/tsuru/cmd"
	_ "github.com/tsuru/tsuru/provision/docker"
	"gopkg.in/check.v1"
)

func (s *S) TestInstallComponentsDefaultConfig(c *check.C) {
	tests := []struct {
		component     TsuruComponent
		containerName string
		image         string
		cmd           []string
		env           []string
	}{
		{&MongoDB{}, "mongo", "mongo:latest", []string(nil), []string(nil)},
		{&Redis{}, "redis", "redis:latest", []string(nil), []string(nil)},
		{&PlanB{}, "planb", "tsuru/planb:latest",
			[]string{"--listen", ":8080",
				"--read-redis-host", "redis",
				"--write-redis-host", "redis",
			}, []string(nil)},
		{&Registry{}, "registry", "registry:2", []string(nil),
			[]string{"REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=/var/lib/registry",
				"REGISTRY_HTTP_TLS_KEY=/certs/127.0.0.1:5000/registry-key.pem",
				"REGISTRY_HTTP_TLS_CERTIFICATE=/certs/127.0.0.1:5000/registry-cert.pem"}},
		{&TsuruAPI{}, "tsuru", "tsuru/api:latest", []string(nil),
			[]string{"MONGODB_ADDR=mongo",
				"MONGODB_PORT=27017",
				"REDIS_ADDR=redis",
				"REDIS_PORT=6379",
				"HIPACHE_DOMAIN=127.0.0.1.nip.io",
				"REGISTRY_ADDR=127.0.0.1",
				"REGISTRY_PORT=5000",
				"TSURU_ADDR=http://127.0.0.1",
				"TSURU_PORT=8080",
			}},
	}
	c.Assert(len(tests), check.Equals, len(TsuruComponents))
	containerChan := make(chan *docker.Container)
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	server, err := testing.NewTLSServer("127.0.0.1:0", containerChan, nil, tlsConfig)
	c.Assert(err, check.IsNil)
	defer server.Stop()
	mockCluster := &SwarmCluster{
		Manager: &Machine{
			Address: server.URL(),
			IP:      "127.0.0.1",
			CAPath:  s.TLSCertsPath.RootDir,
			network: &docker.Network{Name: "tsuru"},
		},
	}
	installConfig := NewInstallConfig("test")
	for _, tt := range tests {
		go tt.component.Install(mockCluster, installConfig)

		cont := <-containerChan
		c.Assert(cont.Name, check.Equals, tt.containerName)
		c.Assert(cont.Image, check.Equals, tt.image)
		c.Assert(cont.Config.Cmd, check.DeepEquals, tt.cmd)
		sort.Strings(cont.Config.Env)
		sort.Strings(tt.env)
		c.Assert(cont.Config.Env, check.DeepEquals, tt.env)
	}
}

func (s *S) TestInstallComponentsCustomRegistry(c *check.C) {
	config.Set("docker-hub-mirror", "myregistry.com")
	defer config.Unset("docker-hub-mirror")
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
	containerChan := make(chan *docker.Container, 1)
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	server, err := testing.NewTLSServer("127.0.0.1:0", containerChan, nil, tlsConfig)
	c.Assert(err, check.IsNil)
	defer server.Stop()
	mockCluster := &SwarmCluster{
		Manager: &Machine{
			Address: server.URL(),
			IP:      "127.0.0.1",
			CAPath:  s.TLSCertsPath.RootDir,
			network: &docker.Network{Name: "tsuru"},
		},
	}
	for _, tt := range tests {
		config := NewInstallConfig("test")
		go tt.component.Install(mockCluster, config)

		cont := <-containerChan
		c.Assert(cont.Image, check.Equals, tt.image)
	}
}

func (s *S) TestInstallPlanbHostPortBindings(c *check.C) {
	containerChan := make(chan *docker.Container)
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	server, err := testing.NewTLSServer("127.0.0.1:0", containerChan, nil, tlsConfig)
	c.Assert(err, check.IsNil)
	defer server.Stop()
	mockCluster := &SwarmCluster{
		Manager: &Machine{
			Address: server.URL(),
			IP:      "127.0.0.1",
			CAPath:  s.TLSCertsPath.RootDir,
			network: &docker.Network{Name: "tsuru"},
		},
	}
	planb := &PlanB{}
	expectedExposed := map[docker.Port]struct{}{
		docker.Port("8080/tcp"): {},
	}
	expectedBinds := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "80"}},
	}
	config.Unset("docker-hub-mirror")
	installConfig := NewInstallConfig("test")
	go planb.Install(mockCluster, installConfig)
	cont := <-containerChan
	c.Assert(cont.HostConfig.PortBindings, check.DeepEquals, expectedBinds)
	c.Assert(cont.Config.ExposedPorts, check.DeepEquals, expectedExposed)
}

func (s *S) TestComponentStatusReport(c *check.C) {
	tlsConfig := testing.TLSConfig{
		CertPath:    s.TLSCertsPath.ServerCert,
		CertKeyPath: s.TLSCertsPath.ServerKey,
		RootCAPath:  s.TLSCertsPath.RootCert,
	}
	containerHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		portBinding := docker.PortBinding{HostIP: "127.0.0.1", HostPort: "8000"}
		cont := docker.Container{HostConfig: &docker.HostConfig{
			PortBindings: map[docker.Port][]docker.PortBinding{
				docker.Port("8000/tcp"): {portBinding},
			},
		}}
		contBuf, err := json.Marshal(cont)
		c.Assert(err, check.IsNil)
		w.Write(contBuf)
	}
	server, err := testing.NewTLSServer("127.0.0.1:0", nil, nil, tlsConfig)
	c.Assert(err, check.IsNil)
	defer server.Stop()
	server.CustomHandler("/containers/.*/json", http.HandlerFunc(containerHandler))
	defer server.CustomHandler("/containers/.*/json", server.DefaultHandler())
	mockMachine := &Machine{Address: server.URL(), IP: "127.0.0.1", CAPath: s.TLSCertsPath.RootDir}
	status, err := containerStatus("mongo", mockMachine)
	c.Assert(err, check.IsNil)
	c.Assert(status.addresses, check.DeepEquals, []string{"tcp://127.0.0.1:8000"})
}

func (s *S) TestTsuruAPIBootstrapLocalEnviroment(c *check.C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		c.Assert(err, check.IsNil)
		if r.URL.Path == "/1.0/users/test/tokens" {
			c.Assert(string(b), check.Equals, "password=test")
			token := map[string]string{"token": "test"}
			buf, err := json.Marshal(token)
			c.Assert(err, check.IsNil)
			w.Write(buf)
		}
		if r.URL.Path == "/1.0/pools" {
			c.Assert(string(b), check.Equals, "default=true&force=false&name=theonepool&public=true")
		}
		if r.URL.Path == "/1.0/docker/node" {
			c.Assert(string(b), check.Matches, "Metadata.address=.*&Metadata.pool=theonepool&Register=true")
		}
		if r.URL.Path == "/1.0/team" {
			c.Assert(string(b), check.Equals, "name=admin")
		}
		if r.URL.Path == "/1.0/apps" {
			c.Assert(string(b), check.Equals, "description=&name=tsuru-dashboard&plan=&platform=python&pool=&routeropts=&teamOwner=admin")
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
	defer func() {
		manager := cmd.BuildBaseManager("uninstall-client", "0.0.0", "", nil)
		c := cmd.NewClient(&http.Client{}, nil, manager)
		cont := cmd.Context{
			Args:   []string{"test"},
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		targetrm := manager.Commands["target-remove"]
		targetrm.Run(&cont, c)
	}()
	t := TsuruAPI{}
	err := t.bootstrapEnv("test", "test", server.URL, "test", server.URL)
	c.Assert(err, check.IsNil)
}
