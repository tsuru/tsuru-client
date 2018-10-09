// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestClusterAddRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"c1", "myprov"},
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.URL.Path, check.Equals, "/1.3/provisioner/clusters")
			c.Assert(req.Method, check.Equals, http.MethodPost)
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/json")

			var clus tsuru.Cluster
			data, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &clus)
			c.Assert(err, check.IsNil)
			c.Assert(clus, check.DeepEquals, tsuru.Cluster{
				Name:        "c1",
				Cacert:      "cadata",
				Clientcert:  "certdata",
				Clientkey:   "keydata",
				CustomData:  map[string]string{"a": "b", "c": "d"},
				Addresses:   []string{"addr1", "addr2"},
				Pools:       []string{"p1", "p2"},
				Default_:    true,
				Provisioner: "myprov",
				CreateData:  map[string]string{"iaas": "dockermachine"},
			})
			return true
		},
	}
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	myCmd := ClusterAdd{}
	dir, err := ioutil.TempDir("", "tsuru")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(dir)
	err = ioutil.WriteFile(filepath.Join(dir, "ca"), []byte("cadata"), 0600)
	c.Assert(err, check.IsNil)
	err = ioutil.WriteFile(filepath.Join(dir, "cert"), []byte("certdata"), 0600)
	c.Assert(err, check.IsNil)
	err = ioutil.WriteFile(filepath.Join(dir, "key"), []byte("keydata"), 0600)
	c.Assert(err, check.IsNil)
	err = myCmd.Flags().Parse(true, []string{
		"--cacert", filepath.Join(dir, "ca"),
		"--clientcert", filepath.Join(dir, "cert"),
		"--clientkey", filepath.Join(dir, "key"),
		"--addr", "addr1",
		"--addr", "addr2",
		"--pool", "p1",
		"--pool", "p2",
		"--custom", "a=b",
		"--custom", "c=d",
		"--create-data", "iaas=dockermachine",
		"--default",
	})
	c.Assert(err, check.IsNil)
	err = myCmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Cluster successfully added.\n")
}

func (s *S) TestClusterUpdateRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"c1", "myprov"},
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.URL.Path, check.Equals, "/1.4/provisioner/clusters/c1")
			c.Assert(req.Method, check.Equals, http.MethodPost)
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/json")

			var clus tsuru.Cluster
			data, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &clus)
			c.Assert(err, check.IsNil)
			c.Assert(clus, check.DeepEquals, tsuru.Cluster{
				Name:        "c1",
				Cacert:      "cadata",
				Clientcert:  "certdata",
				Clientkey:   "keydata",
				CustomData:  map[string]string{"a": "b", "c": "d"},
				Addresses:   []string{"addr1", "addr2"},
				Pools:       []string{"p1", "p2"},
				Default_:    true,
				Provisioner: "myprov",
			})
			return true
		},
	}
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	myCmd := ClusterUpdate{}
	dir, err := ioutil.TempDir("", "tsuru")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(dir)
	err = ioutil.WriteFile(filepath.Join(dir, "ca"), []byte("cadata"), 0600)
	c.Assert(err, check.IsNil)
	err = ioutil.WriteFile(filepath.Join(dir, "cert"), []byte("certdata"), 0600)
	c.Assert(err, check.IsNil)
	err = ioutil.WriteFile(filepath.Join(dir, "key"), []byte("keydata"), 0600)
	c.Assert(err, check.IsNil)
	err = myCmd.Flags().Parse(true, []string{
		"--cacert", filepath.Join(dir, "ca"),
		"--clientcert", filepath.Join(dir, "cert"),
		"--clientkey", filepath.Join(dir, "key"),
		"--addr", "addr1",
		"--addr", "addr2",
		"--pool", "p1",
		"--pool", "p2",
		"--custom", "a=b",
		"--custom", "c=d",
		"--default",
	})
	c.Assert(err, check.IsNil)
	err = myCmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Cluster successfully updated.\n")
}

func (s *S) TestClusterListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	clusters := []tsuru.Cluster{{
		Name:        "c1",
		Addresses:   []string{"addr1", "addr2"},
		Cacert:      "cacert",
		Clientcert:  "clientcert",
		Clientkey:   "clientkey",
		CustomData:  map[string]string{"namespace": "ns1"},
		Default_:    true,
		Provisioner: "prov1",
	}, {
		Name:        "c2",
		Addresses:   []string{"addr3"},
		Default_:    false,
		Pools:       []string{"p1", "p2"},
		Provisioner: "prov2",
	}}
	data, err := json.Marshal(clusters)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.URL.Path, check.Equals, "/1.3/provisioner/clusters")
			c.Assert(req.Method, check.Equals, http.MethodGet)
			return true
		},
	}
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	myCmd := ClusterList{}
	err = myCmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `+------+-------------+-----------+---------------+---------+-------+
| Name | Provisioner | Addresses | Custom Data   | Default | Pools |
+------+-------------+-----------+---------------+---------+-------+
| c1   | prov1       | addr1     | namespace=ns1 | true    |       |
|      |             | addr2     |               |         |       |
+------+-------------+-----------+---------------+---------+-------+
| c2   | prov2       | addr3     |               | false   | p1    |
|      |             |           |               |         | p2    |
+------+-------------+-----------+---------------+---------+-------+
`)
}

func (s *S) TestClusterRemoveRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Are you sure you want to remove cluster "c1"? (y/n) `
	context := cmd.Context{
		Args:   []string{"c1"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.3/provisioner/clusters/c1" && req.Method == "DELETE"
		},
	}
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := ClusterRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expectedOut := "Cluster successfully removed.\n"
	c.Assert(stdout.String(), check.Equals, expected+expectedOut)
}
