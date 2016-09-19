// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/machine/libmachine/host"

	check "gopkg.in/check.v1"
)

type S struct{}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) TestExists(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, check.Equals, "HEAD")
		if r.URL.Path == "/hosts/host1" {
			w.WriteHeader(http.StatusOK)
		}
		if r.URL.Path == "/hosts/host2" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	store := NewRestStore(ts.URL+"/hosts/", nil)
	b, err := store.Exists("host1")
	c.Assert(err, check.IsNil)
	c.Assert(b, check.Equals, true)

	b, err = store.Exists("host2")
	c.Assert(err, check.IsNil)
	c.Assert(b, check.Equals, false)
}

func (s *S) TestList(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, check.Equals, "GET")
		c.Assert(r.URL.Path, check.Equals, "/hosts/")
		hosts := []*host.Host{{Name: "host1"}, {Name: "host2"}}
		json, err := json.Marshal(hosts)
		c.Assert(err, check.IsNil)
		w.Write(json)
	}))
	defer ts.Close()
	store := NewRestStore(ts.URL+"/hosts/", nil)
	names, err := store.List()
	c.Assert(err, check.IsNil)
	c.Assert(names, check.DeepEquals, []string{"host1", "host2"})
}

func (s *S) TestLoad(c *check.C) {
	expectedHost := &host.Host{
		Name:       "host1",
		DriverName: "my-driver",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, check.Equals, "GET")
		c.Assert(r.URL.Path, check.Equals, "/hosts/host1")
		json, err := json.Marshal(expectedHost)
		c.Assert(err, check.IsNil)
		w.Write(json)
	}))
	defer ts.Close()
	store := NewRestStore(ts.URL+"/hosts/", nil)
	h, err := store.Load("host1")
	c.Assert(err, check.IsNil)
	c.Assert(h, check.DeepEquals, expectedHost)
}

func (s *S) TestRemove(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, check.Equals, "DELETE")
		c.Assert(r.URL.Path, check.Equals, "/hosts/host1")
	}))
	defer ts.Close()
	store := NewRestStore(ts.URL+"/hosts/", nil)
	err := store.Remove("host1")
	c.Assert(err, check.IsNil)
}

func (s *S) TestSave(c *check.C) {
	expectedHost := &host.Host{
		Name:       "host1",
		DriverName: "my-driver",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, check.Equals, "POST")
		c.Assert(r.URL.Path, check.Equals, "/hosts/host1")
		var host *host.Host
		err := json.NewDecoder(r.Body).Decode(&host)
		c.Assert(err, check.IsNil)
		c.Assert(host, check.DeepEquals, expectedHost)
	}))
	defer ts.Close()
	store := NewRestStore(ts.URL+"/hosts/", nil)
	err := store.Save(expectedHost)
	c.Assert(err, check.IsNil)
}
