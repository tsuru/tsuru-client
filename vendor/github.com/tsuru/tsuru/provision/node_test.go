// Copyright 2016 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provision_test

import (
	"github.com/pkg/errors"
	"github.com/tsuru/tsuru/provision"
	"github.com/tsuru/tsuru/provision/provisiontest"
	"gopkg.in/check.v1"
)

type S struct{}

var _ = check.Suite(&S{})

func (s *S) TestFindNodeByAddrs(c *check.C) {
	p := provisiontest.NewFakeProvisioner()
	err := p.AddNode(provision.AddNodeOptions{
		Address: "http://addr1",
	})
	c.Assert(err, check.IsNil)
	node, err := provision.FindNodeByAddrs(p, []string{"addr1", "notfound"})
	c.Assert(err, check.IsNil)
	c.Assert(node.Address(), check.Equals, "http://addr1")
	_, err = provision.FindNodeByAddrs(p, []string{"addr2"})
	c.Assert(err, check.Equals, provision.ErrNodeNotFound)
}

func (s *S) TestFindNodeByAddrsAmbiguous(c *check.C) {
	p := provisiontest.NewFakeProvisioner()
	err := p.AddNode(provision.AddNodeOptions{
		Address: "http://addr1",
	})
	c.Assert(err, check.IsNil)
	err = p.AddNode(provision.AddNodeOptions{
		Address: "http://addr2",
	})
	c.Assert(err, check.IsNil)
	_, err = provision.FindNodeByAddrs(p, []string{"addr1", "addr2"})
	c.Assert(err, check.ErrorMatches, `addrs match multiple nodes: \[addr1 addr2\]`)
}

func (s *S) TestFindNode(c *check.C) {
	p1 := provisiontest.NewFakeProvisioner()
	p2 := provisiontest.NewFakeProvisioner()
	provision.Register("fake1", func() (provision.Provisioner, error) {
		return p1, nil
	})
	provision.Register("fake2", func() (provision.Provisioner, error) {
		return p2, nil
	})
	defer provision.Unregister("fake1")
	defer provision.Unregister("fake2")
	err := p1.AddNode(provision.AddNodeOptions{
		Address: "http://addr1",
	})
	c.Assert(err, check.IsNil)
	err = p2.AddNode(provision.AddNodeOptions{
		Address: "http://addr2",
	})
	c.Assert(err, check.IsNil)
	prov, n, err := provision.FindNode("http://addr1")
	c.Assert(err, check.IsNil)
	c.Assert(n.Address(), check.Equals, "http://addr1")
	c.Assert(prov, check.Equals, p1)
	prov, n, err = provision.FindNode("http://addr2")
	c.Assert(err, check.IsNil)
	c.Assert(n.Address(), check.Equals, "http://addr2")
	c.Assert(prov, check.Equals, p2)
	_, _, err = provision.FindNode("http://addr3")
	c.Assert(err, check.Equals, provision.ErrNodeNotFound)
}

func (s *S) TestFindNodeSkipProvisioner(c *check.C) {
	p1 := provisiontest.NewFakeProvisioner()
	p1.Name = "fake1"
	p2 := provisiontest.NewFakeProvisioner()
	p2.Name = "fake2"
	provision.Register("fake1", func() (provision.Provisioner, error) {
		return p1, nil
	})
	provision.Register("fake2", func() (provision.Provisioner, error) {
		return p2, nil
	})
	defer provision.Unregister("fake1")
	defer provision.Unregister("fake2")
	err := p1.AddNode(provision.AddNodeOptions{
		Address: "http://addr1",
	})
	c.Assert(err, check.IsNil)
	err = p2.AddNode(provision.AddNodeOptions{
		Address: "http://addr2",
	})
	c.Assert(err, check.IsNil)
	prov, n, err := provision.FindNodeSkipProvisioner("http://addr1", "fake1")
	c.Assert(err, check.Equals, provision.ErrNodeNotFound)
	c.Assert(n, check.IsNil)
	c.Assert(prov, check.IsNil)
	prov, n, err = provision.FindNodeSkipProvisioner("http://addr1", "fake2")
	c.Assert(err, check.IsNil)
	c.Assert(n.Address(), check.Equals, "http://addr1")
	c.Assert(prov, check.Equals, p1)
}

func (s *S) TestFindNodeIgnoreErrorIfFound(c *check.C) {
	p1 := provisiontest.NewFakeProvisioner()
	p2 := provisiontest.NewFakeProvisioner()
	provision.Register("fake1", func() (provision.Provisioner, error) {
		return p1, nil
	})
	provision.Register("fake2", func() (provision.Provisioner, error) {
		return p2, nil
	})
	defer provision.Unregister("fake1")
	defer provision.Unregister("fake2")
	err := p1.AddNode(provision.AddNodeOptions{
		Address: "http://addr1",
	})
	c.Assert(err, check.IsNil)
	err = p2.AddNode(provision.AddNodeOptions{
		Address: "http://addr2",
	})
	c.Assert(err, check.IsNil)
	prov, n, err := provision.FindNode("http://addr1")
	c.Assert(err, check.IsNil)
	c.Assert(n.Address(), check.Equals, "http://addr1")
	c.Assert(prov, check.Equals, p1)
	p2.PrepareFailure("GetNode", errors.New("get node error"))
	_, _, err = provision.FindNode("http://addr2")
	c.Assert(err, check.ErrorMatches, `(?s)get node error.*`)
}

func (s *S) TestSplitMetadata(c *check.C) {
	var err error
	makeNode := func(addr string, metadata map[string]string) provision.Node {
		return &provisiontest.FakeNode{Addr: addr, Meta: metadata}
	}
	params := []provision.Node{
		makeNode("n1", map[string]string{"1": "a", "2": "z1", "3": "n1"}),
		makeNode("n2", map[string]string{"1": "a", "2": "z2", "3": "n2"}),
		makeNode("n3", map[string]string{"1": "a", "2": "z3", "3": "n3"}),
		makeNode("n4", map[string]string{"1": "a", "2": "z3", "3": "n3"}),
	}
	exclusive, common, err := provision.NodeList(params).SplitMetadata()
	c.Assert(err, check.IsNil)
	c.Assert(exclusive, check.DeepEquals, provision.MetaWithFrequencyList([]provision.MetaWithFrequency{
		{Metadata: map[string]string{"2": "z1", "3": "n1"}, Nodes: []provision.Node{params[0]}},
		{Metadata: map[string]string{"2": "z2", "3": "n2"}, Nodes: []provision.Node{params[1]}},
		{Metadata: map[string]string{"2": "z3", "3": "n3"}, Nodes: []provision.Node{params[2], params[3]}},
	}))
	c.Assert(common, check.DeepEquals, map[string]string{
		"1": "a",
	})
	params = []provision.Node{
		makeNode("n1", map[string]string{"1": "a", "2": "z1", "3": "n1", "4": "b"}),
		makeNode("n2", map[string]string{"1": "a", "2": "z2", "3": "n2", "4": "b"}),
	}
	exclusive, common, err = provision.NodeList(params).SplitMetadata()
	c.Assert(err, check.IsNil)
	c.Assert(exclusive, check.DeepEquals, provision.MetaWithFrequencyList([]provision.MetaWithFrequency{
		{Metadata: map[string]string{"2": "z1", "3": "n1"}, Nodes: []provision.Node{params[0]}},
		{Metadata: map[string]string{"2": "z2", "3": "n2"}, Nodes: []provision.Node{params[1]}},
	}))
	c.Assert(common, check.DeepEquals, map[string]string{
		"1": "a",
		"4": "b",
	})
	params = []provision.Node{
		makeNode("n1", map[string]string{"1": "a", "2": "b"}),
		makeNode("n2", map[string]string{"1": "a", "2": "b"}),
	}
	exclusive, common, err = provision.NodeList(params).SplitMetadata()
	c.Assert(err, check.IsNil)
	c.Assert(exclusive, check.IsNil)
	c.Assert(common, check.DeepEquals, map[string]string{
		"1": "a",
		"2": "b",
	})
	exclusive, common, err = provision.NodeList([]provision.Node{}).SplitMetadata()
	c.Assert(err, check.IsNil)
	c.Assert(exclusive, check.IsNil)
	c.Assert(common, check.DeepEquals, map[string]string{})
	params = []provision.Node{
		makeNode("n1", map[string]string{"1": "a"}),
		makeNode("n2", map[string]string{}),
	}
	_, _, err = provision.NodeList(params).SplitMetadata()
	c.Assert(err, check.ErrorMatches, "unbalanced metadata for node group:.*")
	params = []provision.Node{
		makeNode("n1", map[string]string{"1": "a", "2": "z1", "3": "n1", "4": "b"}),
		makeNode("n2", map[string]string{"1": "a", "2": "z2", "3": "n2", "4": "b"}),
		makeNode("n3", map[string]string{"1": "a", "2": "z3", "3": "n3", "4": "c"}),
	}
	_, _, err = provision.NodeList(params).SplitMetadata()
	c.Assert(err, check.ErrorMatches, "unbalanced metadata for node group:.*")
	params = []provision.Node{
		makeNode("n1", map[string]string{"1": "a", "2": "z1", "3": "n1", "4": "b"}),
		makeNode("n2", map[string]string{"1": "a", "2": "z2", "3": "n2", "4": "b"}),
		makeNode("n3", map[string]string{"1": "a", "2": "z3", "3": "n1", "4": "b"}),
	}
	_, _, err = provision.NodeList(params).SplitMetadata()
	c.Assert(err, check.ErrorMatches, "unbalanced metadata for node group:.*")
}
