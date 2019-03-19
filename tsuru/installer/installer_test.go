// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"reflect"

	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	"github.com/tsuru/tsuru/iaas"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	check "gopkg.in/check.v1"
)

type FakeMachineProvisioner struct {
	hostsProvisioned int
}

func (p *FakeMachineProvisioner) ProvisionMachine(opts map[string]interface{}) (*dockermachine.Machine, error) {
	p.hostsProvisioned = p.hostsProvisioned + 1
	return &dockermachine.Machine{Base: &iaas.Machine{CustomData: opts}}, nil
}

func (s *S) TestBuildClusterTable(c *check.C) {
	i := &Installation{CoreCluster: &FakeServiceCluster{}}
	table := i.buildClusterTable()
	expected := `+-----------+---------+---------+
| IP        | State   | Manager |
+-----------+---------+---------+
| 127.0.0.1 | running | true    |
+-----------+---------+---------+
`
	c.Assert(table.String(), check.Equals, expected)
}

func (s *S) TestBuildComponentsTable(c *check.C) {
	i := &Installation{CoreCluster: &FakeServiceCluster{}}
	table := i.buildComponentsTable()
	expected := `+-----------+-------+----------+
| Component | Ports | Replicas |
+-----------+-------+----------+
| service   | 8080  | 1        |
+-----------+-------+----------+
`
	c.Assert(table.String(), check.Equals, expected)
}

func (s *S) TestProvisionPool(c *check.C) {
	opt1 := map[string]interface{}{"variable-opt": "opt1"}
	opt2 := map[string]interface{}{"variable-opt": "opt2"}
	tt := []struct {
		poolHosts           int
		dedicatedPool       bool
		machines            []*dockermachine.Machine
		expectedProvisioned int
		expectedDriverOpts  []map[string]interface{}
	}{
		{1, false, []*dockermachine.Machine{{}}, 0, []map[string]interface{}{}},
		{2, false, []*dockermachine.Machine{{}}, 1, []map[string]interface{}{opt1, {}}},
		{1, true, []*dockermachine.Machine{{}}, 1, []map[string]interface{}{opt1}},
		{2, true, []*dockermachine.Machine{{}, {}}, 2, []map[string]interface{}{opt1, opt2}},
		{3, true, []*dockermachine.Machine{{}}, 3, []map[string]interface{}{opt1, opt2, opt1}},
	}
	for ti, t := range tt {
		p := &FakeMachineProvisioner{}
		installer := &Installer{machineProvisioner: p}
		config := &InstallOpts{
			Hosts: hostGroups{
				Apps: hostGroupConfig{
					Size:      t.poolHosts,
					Dedicated: t.dedicatedPool,
					Driver: multiOptionsDriver{Options: map[string][]interface{}{
						"variable-opt": {"opt1", "opt2"},
					}},
				},
			},
		}
		machines, err := installer.ProvisionPool(config, t.machines)
		c.Assert(err, check.IsNil)
		c.Assert(p.hostsProvisioned, check.Equals, t.expectedProvisioned)
		for i := 0; i < t.expectedProvisioned; i++ {
			if !reflect.DeepEqual(machines[i].Base.CustomData, t.expectedDriverOpts[i]) {
				c.Errorf("Test case %d/%d failed. Expected %+v. Got %+v", ti, i, t.expectedDriverOpts[i], machines[i].Base.CustomData)
			}
		}
	}
}

func (s *S) TestsetCoreDriverDefaultOpts(c *check.C) {
	tt := []struct {
		test     *InstallOpts
		expected map[string][]interface{}
	}{
		{
			test: &InstallOpts{
				DockerMachineConfig: dm.DockerMachineConfig{
					DriverOpts: &dm.DriverOpts{Name: "google"},
				},
				Hosts: hostGroups{
					Core: hostGroupConfig{Driver: multiOptionsDriver{Options: map[string][]interface{}{"google-open-port": {"8081"}}}},
				},
			},
			expected: map[string][]interface{}{
				"google-open-port": {"8081"},
				"google-scopes":    {"https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/compute"},
			},
		},
		{
			test: &InstallOpts{
				DockerMachineConfig: dm.DockerMachineConfig{
					DriverOpts: &dm.DriverOpts{Name: "generic"},
				},
			},
			expected: map[string][]interface{}{
				"generic-open-port": {"8080"},
			},
		},
	}
	for _, v := range tt {
		setCoreDriverDefaultOpts(v.test)
		c.Check(v.test.Hosts.Core.Driver.Options, check.DeepEquals, v.expected)
	}
}

func (s *S) TestDefaulInstalltOptsRootUserPasswordIsRandom(c *check.C) {
	rootPassword1 := DefaultInstallOpts().RootUserPassword
	rootPassword2 := DefaultInstallOpts().RootUserPassword
	c.Check(rootPassword1, check.Not(check.Equals), rootPassword2)
}

type fakeSSH struct {
	cmd string
}

func (f *fakeSSH) RunSSHCommand(command string) (string, error) {
	f.cmd = command
	return "", nil
}

func (s *S) TestWaitRegistry(c *check.C) {
	fssh := &fakeSSH{}
	tests := []struct {
		ip         string
		compConfig *ComponentsConfig
		expected   string
	}{
		{
			ip:       "10.0.0.1",
			expected: "curl -m5 -sSLk \"https://10.0.0.1:5000\"",
		},
		{
			ip: "10.0.0.1",
			compConfig: &ComponentsConfig{
				Tsuru: tsuruComponent{
					Config: map[string]interface{}{
						"docker": map[string]interface{}{
							"registry": "192.168.1.1:5001",
						},
					},
				},
			},
			expected: "curl -m5 -sSLk \"https://192.168.1.1:5001\"",
		},
		{
			ip: "10.0.0.1",
			compConfig: &ComponentsConfig{
				Tsuru: tsuruComponent{
					Config: map[string]interface{}{
						"docker": map[string]interface{}{},
					},
				},
			},
			expected: "curl -m5 -sSLk \"https://10.0.0.1:5000\"",
		},
		{
			ip: "10.0.0.1",
			compConfig: &ComponentsConfig{
				Tsuru: tsuruComponent{
					Config: map[string]interface{}{
						"docker": nil,
					},
				},
			},
			expected: "curl -m5 -sSLk \"https://10.0.0.1:5000\"",
		},
		{
			ip: "10.0.0.1",
			compConfig: &ComponentsConfig{
				Tsuru: tsuruComponent{
					Config: map[string]interface{}{
						"docker": map[string]interface{}{
							"registry": nil,
						},
					},
				},
			},
			expected: "curl -m5 -sSLk \"https://10.0.0.1:5000\"",
		},
	}
	for _, tt := range tests {
		err := waitRegistry(fssh, tt.ip, tt.compConfig)
		c.Assert(err, check.IsNil)
		c.Assert(fssh.cmd, check.Equals, tt.expected)
	}
}
