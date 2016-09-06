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
	"strings"

	"github.com/docker/engine-api/types/swarm"
	"github.com/fsouza/go-dockerclient"
	"github.com/tsuru/config"
	"github.com/tsuru/tsuru/cmd"
	_ "github.com/tsuru/tsuru/provision/docker"
	"gopkg.in/check.v1"
)

type FakeServiceCluster struct {
	Services chan<- docker.CreateServiceOptions
}

func (c *FakeServiceCluster) GetManager() *Machine {
	return &Machine{IP: "127.0.0.1", Address: "127.0.0.1:2376"}
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
	return nil, nil
}

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
	services := make(chan docker.CreateServiceOptions)
	fakeCluster := &FakeServiceCluster{Services: services}
	installConfig := NewInstallConfig("test")
	for _, tt := range tests {
		go tt.component.Install(fakeCluster, installConfig)
		opts := <-services
		cont := opts.ServiceSpec.TaskTemplate.ContainerSpec
		c.Assert(opts.Annotations.Name, check.Equals, tt.containerName)
		c.Assert(cont.Image, check.Equals, tt.image)
		c.Assert(cont.Args, check.DeepEquals, tt.cmd)
		sort.Strings(cont.Env)
		sort.Strings(tt.env)
		c.Assert(cont.Env, check.DeepEquals, tt.env)
	}
	c.Assert(installConfig.ComponentAddress["mongo"], check.Equals, "mongo")
	c.Assert(installConfig.ComponentAddress["redis"], check.Equals, "redis")
	c.Assert(installConfig.ComponentAddress["registry"], check.Equals, "127.0.0.1")
	c.Assert(installConfig.ComponentAddress["planb"], check.Equals, "127.0.0.1")
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
	services := make(chan docker.CreateServiceOptions)
	fakeCluster := &FakeServiceCluster{Services: services}
	for _, tt := range tests {
		config := NewInstallConfig("test")
		go tt.component.Install(fakeCluster, config)
		s := <-services
		img := s.TaskTemplate.ContainerSpec.Image
		c.Assert(img, check.Equals, tt.image)
	}
}

func (s *S) TestInstallPlanbHostPortBindings(c *check.C) {
	services := make(chan docker.CreateServiceOptions, 1)
	fakeCluster := &FakeServiceCluster{Services: services}
	planb := &PlanB{}
	expectedConfigs := []swarm.PortConfig{
		{
			Protocol:      swarm.PortConfigProtocolTCP,
			TargetPort:    uint32(8080),
			PublishedPort: uint32(80),
		},
	}
	installConfig := NewInstallConfig("test")
	planb.Install(fakeCluster, installConfig)
	config := <-services
	c.Assert(config.EndpointSpec.Ports, check.DeepEquals, expectedConfigs)
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
		if r.URL.Path == "/1.0/platforms" {
			expected := `--.*
Content-Disposition: form-data; name="dockerfile_content"; filename="Dockerfile"
Content-Type: application/octet-stream

FROM tsuru/python
--.*
Content-Disposition: form-data; name="name"

python
--.*--
`
			expected = strings.Replace(expected, "\n", "\r\n", -1)
			c.Assert(string(b), check.Matches, expected)
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
	opts := TsuruSetupOptions{
		Login:      "test",
		Password:   "test",
		Target:     server.URL,
		TargetName: "test",
		NodeAddr:   server.URL,
	}
	err := t.bootstrapEnv(opts)
	c.Assert(err, check.IsNil)
}

func (s *S) TestTsuruAPIBootstrapCustomDockerRegistry(c *check.C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		c.Assert(err, check.IsNil)
		if r.URL.Path == "/1.0/users/test/tokens" {
			token := map[string]string{"token": "test"}
			buf, err := json.Marshal(token)
			c.Assert(err, check.IsNil)
			w.Write(buf)
		}
		if r.URL.Path == "/1.0/platforms" {
			expected := `--.*
Content-Disposition: form-data; name="dockerfile_content"; filename="Dockerfile"
Content-Type: application/octet-stream

FROM test.com/python
--.*
Content-Disposition: form-data; name="name"

python
--.*--
`
			expected = strings.Replace(expected, "\n", "\r\n", -1)
			c.Assert(string(b), check.Matches, expected)
		}
		if r.URL.Path == "/1.0/apps" {
			buf, err := json.Marshal(map[string]string{})
			c.Assert(err, check.IsNil)
			w.Write(buf)
		}
		if r.URL.Path == "/1.0/apps/tsuru-dashboard/deploy" {
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
	opts := TsuruSetupOptions{
		Login:           "test",
		Password:        "test",
		Target:          server.URL,
		TargetName:      "test",
		NodeAddr:        server.URL,
		DockerHubMirror: "test.com",
	}
	err := t.bootstrapEnv(opts)
	c.Assert(err, check.IsNil)
}

func (s *S) TestPreInstalledComponents(c *check.C) {
	err := config.ReadConfigFile("./testdata/components-conf.yml")
	c.Assert(err, check.IsNil)
	conf := NewInstallConfig("testing")
	cluster := &FakeServiceCluster{}
	m := &MongoDB{}
	err = m.Install(cluster, conf)
	c.Assert(err, check.IsNil)
	c.Assert(conf.ComponentAddress["mongo"], check.Equals, "192.168.0.100:27017")
	r := &Redis{}
	err = r.Install(cluster, conf)
	c.Assert(err, check.IsNil)
	c.Assert(conf.ComponentAddress["redis"], check.Equals, "192.168.0.100:6379")
	registry := &Registry{}
	err = registry.Install(cluster, conf)
	c.Assert(err, check.IsNil)
	c.Assert(conf.ComponentAddress["registry"], check.Equals, "192.168.0.100:5000")
	planb := &PlanB{}
	err = planb.Install(cluster, conf)
	c.Assert(err, check.IsNil)
	c.Assert(conf.ComponentAddress["planb"], check.Equals, "192.168.0.100")
}

func (s *S) TestInstallTsuruApiWithCustomComponentsAddress(c *check.C) {
	err := config.ReadConfigFile("./testdata/components-conf.yml")
	c.Assert(err, check.IsNil)
	conf := NewInstallConfig("testing")
	services := make(chan docker.CreateServiceOptions, 1)
	cluster := &FakeServiceCluster{Services: services}
	api := &TsuruAPI{}
	go api.Install(cluster, conf)
	apiConf := <-services
	expected := []string{
		"MONGODB_ADDR=192.168.0.100",
		"MONGODB_PORT=27017",
		"REDIS_ADDR=192.168.0.100",
		"REDIS_PORT=6379",
		"HIPACHE_DOMAIN=192.168.0.100.nip.io",
		"REGISTRY_ADDR=192.168.0.100",
		"REGISTRY_PORT=5000",
		"TSURU_ADDR=http://127.0.0.1",
		"TSURU_PORT=8080",
	}
	c.Assert(apiConf.TaskTemplate.ContainerSpec.Env, check.DeepEquals, expected)
}
