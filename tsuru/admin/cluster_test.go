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
				Cacert:      []byte("cadata"),
				Clientcert:  []byte("certdata"),
				Clientkey:   []byte("keydata"),
				CustomData:  map[string]string{"a": "b", "c": "d"},
				Addresses:   []string{"addr1", "addr2"},
				Pools:       []string{"p1", "p2"},
				Default:     true,
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
	cluster := tsuru.Cluster{
		Name:        "c1",
		Addresses:   []string{"addr1", "addr2"},
		Cacert:      []byte("cadata"),
		Clientcert:  []byte("certdata"),
		Clientkey:   []byte("keydata"),
		CustomData:  map[string]string{"a": "b", "c": "d"},
		Pools:       []string{"p1", "p2"},
		Default:     false,
		Provisioner: "myprov",
	}
	data, err := json.Marshal(cluster)
	c.Assert(err, check.IsNil)
	trans := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					c.Assert(req.Method, check.Equals, http.MethodGet)
					c.Assert(req.URL.Path, check.Equals, "/1.8/provisioner/clusters/c1")
					return true
				},
			},
			{
				Transport: cmdtest.Transport{Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					c.Assert(req.Method, check.Equals, http.MethodPost)
					c.Assert(req.URL.Path, check.Equals, "/1.4/provisioner/clusters/c1")
					c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/json")

					var clus tsuru.Cluster
					data, err := ioutil.ReadAll(req.Body)
					c.Assert(err, check.IsNil)
					err = json.Unmarshal(data, &clus)
					c.Assert(err, check.IsNil)
					c.Assert(clus, check.DeepEquals, tsuru.Cluster{
						Name:        "c1",
						Clientcert:  []byte("clientcert"),
						Clientkey:   []byte("keydata"),
						CustomData:  map[string]string{"a": "b", "e": "f"},
						Addresses:   []string{"addr1", "addr2"},
						Pools:       []string{"p1"},
						Default:     false,
						Provisioner: "myprov",
					})
					return true
				},
			},
		},
	}
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	myCmd := ClusterUpdate{}
	dir, err := ioutil.TempDir("", "tsuru")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(dir)
	err = ioutil.WriteFile(filepath.Join(dir, "cert"), []byte("clientcert"), 0600)
	c.Assert(err, check.IsNil)
	err = myCmd.Flags().Parse(true, []string{
		"--remove-pool", "p2",
		"--add-pool", "p1",
		"--remove-custom", "c=d",
		"--add-custom", "e=f",
		"--remove-cacert",
		"--clientcert", filepath.Join(dir, "cert"),
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
		Cacert:      []byte("cacert"),
		Clientcert:  []byte("clientcert"),
		Clientkey:   []byte("clientkey"),
		CustomData:  map[string]string{"namespace": "ns1"},
		Default:     true,
		Provisioner: "prov1",
	}, {
		Name:        "c2",
		Addresses:   []string{"addr3"},
		Default:     false,
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
			c.Assert(req.URL.Path, check.Equals, "/1.3/provisioner/clusters/c1")
			c.Assert(req.Method, check.Equals, http.MethodDelete)
			return true
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

func (s *S) TestProvisionerListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	data := `[{
		"name": "p1",
		"cluster_help": {
			"provisioner_help": "help",
			"custom_data_help": {
				"key1": "value1"
			},
			"create_data_help": {
				"create key1": "create value1"
			}
		}
	},{
		"name": "p2",
		"cluster_help": {
			"provisioner_help": "help2",
			"custom_data_help": {
				"key2": "value2"
			},
			"create_data_help": {
				"create key2": "create value2"
			}
		}
	}]`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: data, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.URL.Path, check.Equals, "/1.7/provisioner")
			c.Assert(req.Method, check.Equals, http.MethodGet)
			return true
		},
	}
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	myCmd := ProvisionerList{}
	err := myCmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `+------+---------------+
| Name | Cluster Usage |
+------+---------------+
| p1   | help          |
| p2   | help2         |
+------+---------------+
`)
}

func (s *S) TestProvisionerInfoRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"p2"},
	}
	data := `[{
		"name": "p1",
		"cluster_help": {
			"provisioner_help": "help",
			"custom_data_help": {
				"key1": "value1"
			},
			"create_data_help": {
				"create key1": "create value1"
			}
		}
	},{
		"name": "p2",
		"cluster_help": {
			"provisioner_help": "help2",
			"custom_data_help": {
				"key2": "value2"
			},
			"create_data_help": {
				"create key2": "create value2"
			}
		}
	}]`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: data, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.URL.Path, check.Equals, "/1.7/provisioner")
			c.Assert(req.Method, check.Equals, http.MethodGet)
			return true
		},
	}
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	myCmd := ProvisionerInfo{}
	err := myCmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `Name: p2
Cluster usage: help2

Custom Data:
+------+--------+
| Name | Usage  |
+------+--------+
| key2 | value2 |
+------+--------+

Create Data:
+-------------+---------------+
| Name        | Usage         |
+-------------+---------------+
| create key2 | create value2 |
+-------------+---------------+
`)
}
