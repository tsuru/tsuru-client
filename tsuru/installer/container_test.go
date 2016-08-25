// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"net/http"
	"path/filepath"

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

func (s *S) TestCreateContainerWithHostConfigAndExposedPorts(c *check.C) {
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
