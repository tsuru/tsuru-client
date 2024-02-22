// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestClusterAddInfo(c *check.C) {
	c.Assert((&ClusterAdd{}).Info(), check.NotNil)
}
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
			data, err := io.ReadAll(req.Body)
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
			})
			return true
		},
	}
	s.setupFakeTransport(trans)
	myCmd := ClusterAdd{}
	dir, err := os.MkdirTemp("", "tsuru")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(dir)
	err = os.WriteFile(filepath.Join(dir, "ca"), []byte("cadata"), 0600)
	c.Assert(err, check.IsNil)
	err = os.WriteFile(filepath.Join(dir, "cert"), []byte("certdata"), 0600)
	c.Assert(err, check.IsNil)
	err = os.WriteFile(filepath.Join(dir, "key"), []byte("keydata"), 0600)
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
	err = myCmd.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Cluster successfully added.\n")
}

func (s *S) TestClusterUpdateInfo(c *check.C) {
	c.Assert((&ClusterUpdate{}).Info(), check.NotNil)
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
					data, err = io.ReadAll(req.Body)
					c.Assert(err, check.IsNil)
					err = json.Unmarshal(data, &clus)
					c.Assert(err, check.IsNil)
					c.Assert(clus, check.DeepEquals, tsuru.Cluster{
						Name:        "c1",
						Clientcert:  []byte("clientcert"),
						Clientkey:   []byte("keydata"),
						CustomData:  map[string]string{"a": "b", "e": "f"},
						Addresses:   []string{"addr1", "addr2"},
						Pools:       []string{"p1", "p3"},
						Default:     false,
						Provisioner: "myprov",
					})
					return true
				},
			},
		},
	}
	s.setupFakeTransport(&trans)
	myCmd := ClusterUpdate{}
	dir, err := os.MkdirTemp("", "tsuru")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(dir)
	err = os.WriteFile(filepath.Join(dir, "cert"), []byte("clientcert"), 0600)
	c.Assert(err, check.IsNil)
	err = myCmd.Flags().Parse(true, []string{
		"--remove-pool", "p2",
		"--add-pool", "p3",
		"--remove-custom", "c",
		"--add-custom", "e=f",
		"--remove-cacert",
		"--clientcert", filepath.Join(dir, "cert"),
	})
	c.Assert(err, check.IsNil)
	err = myCmd.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Cluster successfully updated.\n")
}

func (s *S) TestClusterUpdateMergeCluster(c *check.C) {
	baseDir := c.MkDir()
	caCertPath := fmt.Sprintf("%s/cacert.crt", baseDir)
	clientCertPath := fmt.Sprintf("%s/client.crt", baseDir)
	clientKeyPath := fmt.Sprintf("%s/client.key", baseDir)

	writeStringToFile := func(path, content string) {
		file, err := os.Create(path)
		if err != nil {
			c.Fatal(err)
		}
		_, err = file.WriteString(content)
		if err != nil {
			c.Fatal(err)
		}
		err = file.Close()
		if err != nil {
			c.Fatal(err)
		}
	}

	writeStringToFile(caCertPath, "ANOTHER CA CERTIFICATE")
	writeStringToFile(clientCertPath, "ANOTHER CLIENT CERTIFICATE")
	writeStringToFile(clientKeyPath, "ANOTHER CLIENT KEY")

	getCluster := func() tsuru.Cluster {
		return tsuru.Cluster{
			Name:        "c1",
			Addresses:   []string{"https://c1.test:443"},
			Provisioner: "kubernetes",
			Cacert:      []byte("ca certificate"),
			Clientcert:  []byte("client certificate"),
			Clientkey:   []byte("client key"),
			Pools:       []string{"pool1", "pool2"},
			CustomData:  map[string]string{"key": "value"},
			Default:     false,
		}
	}

	tests := []struct {
		command     ClusterUpdate
		cluster     tsuru.Cluster
		want        tsuru.Cluster
		errorString string
	}{
		{
			command: ClusterUpdate{
				cacert:       "/path/to/my/ca.crt",
				removeCacert: true,
			},
			cluster:     getCluster(),
			errorString: "cannot both remove and replace the CA certificate",
		},
		{
			command: ClusterUpdate{
				clientcert:       "/path/to/my/ca.crt",
				removeClientcert: true,
			},
			cluster:     getCluster(),
			errorString: "cannot both remove and replace the client certificate",
		},
		{
			command: ClusterUpdate{
				clientkey:       "/path/to/my/ca.crt",
				removeClientkey: true,
			},
			cluster:     getCluster(),
			errorString: "cannot both remove and replace the client key",
		},
		{
			command: ClusterUpdate{
				removeCustomData: cmd.StringSliceFlag{"some-not-found-key"},
			},
			cluster:     getCluster(),
			errorString: "cannot unset custom data entry: key \"some-not-found-key\" not found",
		},
		{
			command: ClusterUpdate{
				isDefault: "true",
				addPool:   cmd.StringSliceFlag{"new-pool"},
			},
			cluster:     getCluster(),
			errorString: "cannot add or remove pools in a default cluster",
		},
		{
			command: ClusterUpdate{
				addPool: cmd.StringSliceFlag{"pool1"},
			},
			cluster:     getCluster(),
			errorString: "pool \"pool1\" already defined",
		},
		{
			command: ClusterUpdate{
				removePool: cmd.StringSliceFlag{"pool-not-found"},
			},
			cluster:     getCluster(),
			errorString: "pool \"pool-not-found\" not found",
		},
		{
			cluster: getCluster(),
			want: tsuru.Cluster{
				Name:        "c1",
				Addresses:   []string{"https://c1.test:443"},
				Provisioner: "kubernetes",
				Cacert:      []byte("ca certificate"),
				Clientcert:  []byte("client certificate"),
				Clientkey:   []byte("client key"),
				Pools:       []string{"pool1", "pool2"},
				CustomData:  map[string]string{"key": "value"},
				Default:     false,
			},
		},
		{
			command: ClusterUpdate{
				isDefault: "true",
			},
			cluster: getCluster(),
			want: tsuru.Cluster{
				Name:        "c1",
				Addresses:   []string{"https://c1.test:443"},
				Provisioner: "kubernetes",
				Cacert:      []byte("ca certificate"),
				Clientcert:  []byte("client certificate"),
				Clientkey:   []byte("client key"),
				Pools:       []string{},
				CustomData:  map[string]string{"key": "value"},
				Default:     true,
			},
		},
		{
			command: ClusterUpdate{
				removeCacert:     true,
				removeClientcert: true,
				removeClientkey:  true,
			},
			cluster: getCluster(),
			want: tsuru.Cluster{
				Name:        "c1",
				Addresses:   []string{"https://c1.test:443"},
				Provisioner: "kubernetes",
				Cacert:      nil,
				Clientcert:  nil,
				Clientkey:   nil,
				Pools:       []string{"pool1", "pool2"},
				CustomData:  map[string]string{"key": "value"},
				Default:     false,
			},
		},
		{
			command: ClusterUpdate{
				removeCustomData: cmd.StringSliceFlag{"key"},
			},
			cluster: getCluster(),
			want: tsuru.Cluster{
				Name:        "c1",
				Addresses:   []string{"https://c1.test:443"},
				Provisioner: "kubernetes",
				Cacert:      []byte("ca certificate"),
				Clientcert:  []byte("client certificate"),
				Clientkey:   []byte("client key"),
				Pools:       []string{"pool1", "pool2"},
				CustomData:  map[string]string{},
				Default:     false,
			},
		},
		{
			command: ClusterUpdate{
				cacert:     caCertPath,
				clientcert: clientCertPath,
				clientkey:  clientKeyPath,
			},
			cluster: getCluster(),
			want: tsuru.Cluster{
				Name:        "c1",
				Addresses:   []string{"https://c1.test:443"},
				Provisioner: "kubernetes",
				Cacert:      []byte("ANOTHER CA CERTIFICATE"),
				Clientcert:  []byte("ANOTHER CLIENT CERTIFICATE"),
				Clientkey:   []byte("ANOTHER CLIENT KEY"),
				Pools:       []string{"pool1", "pool2"},
				CustomData:  map[string]string{"key": "value"},
				Default:     false,
			},
		},
	}

	for index, tt := range tests {
		fmt.Printf("Executing test case %v\n", index)
		err := tt.command.mergeCluster(&tt.cluster)
		if tt.errorString != "" {
			c.Assert(err.Error(), check.Equals, tt.errorString)
			continue
		}
		c.Assert(err, check.IsNil)
		c.Assert(tt.cluster, check.DeepEquals, tt.want)
	}
}

func (s *S) TestClusterListInfo(c *check.C) {
	c.Assert((&ClusterList{}).Info(), check.NotNil)
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
	s.setupFakeTransport(trans)
	myCmd := ClusterList{}
	err = myCmd.Run(&context)
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

func (s *S) TestClusterRemoveInfo(c *check.C) {
	c.Assert((&ClusterRemove{}).Info(), check.NotNil)
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
	s.setupFakeTransport(trans)
	command := ClusterRemove{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	expectedOut := "Cluster successfully removed.\n"
	c.Assert(stdout.String(), check.Equals, expected+expectedOut)
}

func (s *S) TestProvisionerListInfo(c *check.C) {
	c.Assert((&ProvisionerList{}).Info(), check.NotNil)
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
			}
		}
	},{
		"name": "p2",
		"cluster_help": {
			"provisioner_help": "help2",
			"custom_data_help": {
				"key2": "value2"
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
	s.setupFakeTransport(trans)
	myCmd := ProvisionerList{}
	err := myCmd.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `+------+---------------+
| Name | Cluster Usage |
+------+---------------+
| p1   | help          |
| p2   | help2         |
+------+---------------+
`)
}

func (s *S) TestProvisionerInfoInfo(c *check.C) {
	c.Assert((&ProvisionerInfo{}).Info(), check.NotNil)
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
			}
		}
	},{
		"name": "p2",
		"cluster_help": {
			"provisioner_help": "help2",
			"custom_data_help": {
				"key2": "value2"
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
	s.setupFakeTransport(trans)
	myCmd := ProvisionerInfo{}
	err := myCmd.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, `Name: p2
Cluster usage: help2

Custom Data:
+------+--------+
| Name | Usage  |
+------+--------+
| key2 | value2 |
+------+--------+
`)
}

func (s *S) TestClusterClientSideFilter(c *check.C) {
	clusters := []tsuru.Cluster{
		{
			Name: "gcp-cluster-01",
			Pools: []string{
				"gcp-pool-01",
				"gcp-pool-02",
			},
		},

		{
			Name: "gcp-cluster-02",
			Pools: []string{
				"gcp-pool-03",
				"gcp-pool-04",
			},
		},

		{
			Name: "aws-cluster-01",
			Pools: []string{
				"aws-pool-01",
			},
		},
	}

	filters := []clusterFilter{
		{
			name: "gcp",
		},
		{
			name: "aws",
		},
		{
			pool: "aws-pool-01",
		},

		{
			name: "gcp",
			pool: "gcp-pool-03",
		},
	}

	expectedResults := [][]string{
		{"gcp-cluster-01", "gcp-cluster-02"},
		{"aws-cluster-01"},
		{"aws-cluster-01"},
		{"gcp-cluster-02"},
	}

	for i := range filters {
		cl := ClusterList{
			filter: filters[i],
		}

		filteredClusters := cl.clientSideFilter(clusters)
		clustersNames := []string{}
		for _, cluster := range filteredClusters {
			clustersNames = append(clustersNames, cluster.Name)
		}

		c.Assert(clustersNames, check.DeepEquals, expectedResults[i])
	}
}
