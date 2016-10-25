// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"reflect"

	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
	check "gopkg.in/check.v1"
)

type FakeMachineProvisioner struct {
	hostsProvisioned int
}

func (p *FakeMachineProvisioner) ProvisionMachine(opts map[string]interface{}) (*dm.Machine, error) {
	p.hostsProvisioned = p.hostsProvisioned + 1
	return &dm.Machine{DriverOpts: dm.DriverOpts(opts)}, nil
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
	i := &Installation{CoreCluster: &FakeServiceCluster{}, Components: []TsuruComponent{&MongoDB{}}}
	table := i.buildComponentsTable()
	expected := `+-----------+-------+----------+
| Component | Ports | Replicas |
+-----------+-------+----------+
| MongoDB   | 8080  | 1        |
+-----------+-------+----------+
`
	c.Assert(table.String(), check.Equals, expected)
}

func (s *S) TestProvisionPool(c *check.C) {
	opt1 := dm.DriverOpts{"variable-opt": "opt1"}
	opt2 := dm.DriverOpts{"variable-opt": "opt2"}
	tt := []struct {
		poolHosts           int
		dedicatedPool       bool
		machines            []*dm.Machine
		expectedProvisioned int
		expectedDriverOpts  []dm.DriverOpts
	}{
		{1, false, []*dm.Machine{{}}, 0, []dm.DriverOpts{}},
		{2, false, []*dm.Machine{{}}, 1, []dm.DriverOpts{opt1, {}}},
		{1, true, []*dm.Machine{{}}, 1, []dm.DriverOpts{opt1}},
		{2, true, []*dm.Machine{{}, {}}, 2, []dm.DriverOpts{opt1, opt2}},
		{3, true, []*dm.Machine{{}}, 3, []dm.DriverOpts{opt1, opt2, opt1}},
	}
	for ti, t := range tt {
		p := &FakeMachineProvisioner{}
		installer := &Installer{machineProvisioner: p}
		config := &InstallOpts{
			AppsHosts:          t.poolHosts,
			DedicatedAppsHosts: t.dedicatedPool,
			AppsDriversOpts: map[string][]interface{}{
				"variable-opt": {"opt1", "opt2"},
			},
		}
		machines, err := installer.ProvisionPool(config, t.machines)
		c.Assert(err, check.IsNil)
		c.Assert(p.hostsProvisioned, check.Equals, t.expectedProvisioned)
		for i := 0; i < t.expectedProvisioned; i++ {
			if !reflect.DeepEqual(machines[i].DriverOpts, t.expectedDriverOpts[i]) {
				c.Errorf("Test case %d/%d failed. Expected %+v. Got %+v", ti, i, t.expectedDriverOpts[i], machines[i].DriverOpts)
			}
		}
	}
}
