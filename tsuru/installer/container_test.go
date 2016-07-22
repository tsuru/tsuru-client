// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"net/http"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"
	"gopkg.in/check.v1"
)

type testEndpoint struct {
	endpoint string
}

func (t testEndpoint) dockerClient() (*docker.Client, error) {
	return docker.NewClient(t.endpoint)
}

func (s *S) TestCreateContainer(c *check.C) {
	var requests []*http.Request
	server, err := testing.NewServer("127.0.0.1:0", nil, func(r *http.Request) {
		if r.URL.Path != "/version" {
			requests = append(requests, r)
		}
	})
	c.Assert(err, check.IsNil)
	defer server.Stop()
	endpoint := testEndpoint{endpoint: server.URL()}
	config := &docker.Config{Image: "tsuru/api:v1"}
	err = createContainer(endpoint, "contName", config, nil)
	c.Assert(err, check.IsNil)
	c.Assert(requests, check.HasLen, 4)
	c.Assert(requests[0].URL.Path, check.Equals, "/images/create")
	c.Assert(requests[1].URL.Path, check.Equals, "/images/tsuru/api:v1/json")
	c.Assert(requests[2].URL.Path, check.Equals, "/containers/create")
	c.Assert(requests[3].URL.Path, check.Equals, "/containers/contName/start")
}

func (s *S) TestInspectContainer(c *check.C) {
	var requests []*http.Request
	server, err := testing.NewServer("127.0.0.1:0", nil, func(r *http.Request) {
		requests = append(requests, r)
	})
	c.Assert(err, check.IsNil)
	defer server.Stop()
	endpoint := testEndpoint{endpoint: server.URL()}
	inspectContainer(endpoint, "contName")
	c.Assert(requests, check.HasLen, 1)
	c.Assert(requests[0].URL.Path, check.Equals, "/containers/contName/json")
}
