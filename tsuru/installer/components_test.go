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
	"strings"

	"gopkg.in/check.v1"
)

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
			c.Assert(string(b), check.Matches, "Address=&CaCert=&ClientCert=&ClientKey=&IaaSID=&Metadata.address=.*&Metadata.pool=theonepool&Pool=&Register=true&WaitTO=")
		}
		if r.URL.Path == "/1.0/platforms" {
			expected := "FROM tsuru/python"
			c.Assert(strings.Contains(string(b), expected), check.Equals, true)
		}
		if r.URL.Path == "/1.0/teams" {
			c.Assert(string(b), check.Equals, "{\"name\":\"admin\"}\n")
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
