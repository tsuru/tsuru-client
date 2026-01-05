// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/cezarsa/form"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	volumeTypes "github.com/tsuru/tsuru/types/volume"
	"gopkg.in/check.v1"
)

func (s *S) TestVolumeListInfo(c *check.C) {
	c.Assert((&VolumeList{}).Info(), check.NotNil)
}

func (s *S) TestVolumeList(c *check.C) {
	var stdout, stderr bytes.Buffer
	response := `[
		{"Name":"vag-nfs","Pool":"kubepool","Plan":{"Name":"nfs","Opts":{"access-modes":"ReadWriteMany","plugin":"nfs"}},"TeamOwner":"admin","Status":"","Binds":[{"ID":{"App":"myapp","MountPoint":"/mymnt","Volume":"vag-nfs"},"ReadOnly":false},{"ID":{"App":"myapp","MountPoint":"/mymnt1","Volume":"vag-nfs"},"ReadOnly":false}],"Opts":{"capacity":"1Gi","path":"/home/vagrant/nfstest","server":"192.168.50.4"}},
		{"Name":"other","Pool":"swarmpool","Plan":{"Name":"nfs","Opts":{"driver":"local"}},"TeamOwner":"admin","Status":"","Binds":null,"Opts":{"o":"addr=192.168.50.1,rw","device":":/exports/dir"}}
]`
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: response, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/volumes") && req.Method == "GET"
		},
	}
	s.setupFakeTransport(trans)
	err := (&VolumeList{}).Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, `+---------+------+-----------+-------+
| Name    | Plan | Pool      | Team  |
+---------+------+-----------+-------+
| other   | nfs  | swarmpool | admin |
+---------+------+-----------+-------+
| vag-nfs | nfs  | kubepool  | admin |
+---------+------+-----------+-------+
`)
}

func (s *S) TestVolumeListEmpty(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/volumes") && req.Method == "GET"
		},
	}
	s.setupFakeTransport(trans)
	err := (&VolumeList{}).Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "No volumes available.\n")
}

func (s *S) TestVolumeInfoInfo(c *check.C) {
	c.Assert((&VolumeInfo{}).Info(), check.NotNil)
}

func (s *S) TestVolumeInfo(c *check.C) {
	var stdout, stderr bytes.Buffer
	response := `
		{"Name":"vol1","Pool":"kubepool","Plan":{"Name":"nfs","Opts":{"access-modes":"ReadWriteMany","plugin":"nfs"}},"TeamOwner":"admin","Status":"","Binds":[{"ID":{"App":"myapp","MountPoint":"/mymnt","Volume":"vag-nfs"},"ReadOnly":false},{"ID":{"App":"myapp","MountPoint":"/mymnt1","Volume":"vag-nfs"},"ReadOnly":false}],"Opts":{"capacity":"1Gi","path":"/home/vagrant/nfstest","server":"192.168.50.4"}}`
	ctx := cmd.Context{
		Args:   []string{"vol1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: response, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/volumes/vol1") && req.Method == "GET"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeInfo{}
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, `Name: vol1
Plan: nfs
Pool: kubepool
Team: admin

Binds:
+-------+------------+------+
| App   | MountPoint | Mode |
+-------+------------+------+
| myapp | /mymnt     | rw   |
+-------+------------+------+
| myapp | /mymnt1    | rw   |
+-------+------------+------+

Plan Opts:
+--------------+---------------+
| Key          | Value         |
+--------------+---------------+
| access-modes | ReadWriteMany |
+--------------+---------------+
| plugin       | nfs           |
+--------------+---------------+

Opts:
+----------+-----------------------+
| Key      | Value                 |
+----------+-----------------------+
| capacity | 1Gi                   |
+----------+-----------------------+
| path     | /home/vagrant/nfstest |
+----------+-----------------------+
| server   | 192.168.50.4          |
+----------+-----------------------+
`)
}

func (s *S) TestVolumeInfoEmpty(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"vol"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/volumes/vol") && req.Method == "GET"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeInfo{}
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "No volumes available.\n")
}

func (s *S) TestVolumePlansListInfo(c *check.C) {
	c.Assert((&VolumePlansList{}).Info(), check.NotNil)
}

func (s *S) TestVolumePlansList(c *check.C) {
	var stdout, stderr bytes.Buffer
	response := `{
	"kubernetes": [{"Name":"nfs","Opts":{"access-modes":"ReadWriteMany","plugin":"nfs"}}, {"Name":"ebs","Opts":{"storage-class":"myebs"}}],
	"swarm": [{"Name":"nfs","Opts":{"driver":"local", "opt": [{"type": "nfs"}]}}]
}`
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: response, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/volumeplans") && req.Method == "GET"
		},
	}
	s.setupFakeTransport(trans)
	err := (&VolumePlansList{}).Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, `+------+-------------+-----------------------------+
| Plan | Provisioner | Opts                        |
+------+-------------+-----------------------------+
| ebs  | kubernetes  | storage-class: myebs        |
+------+-------------+-----------------------------+
| nfs  | kubernetes  | access-modes: ReadWriteMany |
|      |             | plugin: nfs                 |
+------+-------------+-----------------------------+
| nfs  | swarm       | driver: local               |
|      |             | opt: [map[type:nfs]]        |
+------+-------------+-----------------------------+
`)
}

func (s *S) TestVolumePlansListEmpty(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/volumeplans") && req.Method == "GET"
		},
	}
	s.setupFakeTransport(trans)
	err := (&VolumePlansList{}).Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, `+------+-------------+------+
| Plan | Provisioner | Opts |
+------+-------------+------+
`)
}

func (s *S) TestVolumeCreateInfo(c *check.C) {
	c.Assert((&VolumeCreate{}).Info(), check.NotNil)
}

func (s *S) TestVolumeCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"vol1", "plan1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusCreated},
		CondFunc: func(r *http.Request) bool {
			r.ParseForm()
			dec := form.NewDecoder(nil)
			dec.IgnoreCase(true)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			var vol volumeTypes.Volume
			err := dec.DecodeValues(&vol, r.Form)
			c.Assert(err, check.IsNil)
			c.Assert(vol, check.DeepEquals, volumeTypes.Volume{
				Name:      "vol1",
				Plan:      volumeTypes.VolumePlan{Name: "plan1"},
				TeamOwner: "team1",
				Pool:      "pool1",
				Opts:      map[string]string{"a": "1", "b": "2"},
			})
			return strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeCreate{}
	command.Flags().Parse(true, []string{"-t", "team1", "-p", "pool1", "-o", "a=1", "-o", "b=2"})
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully created.\n")
}

func (s *S) TestVolumeUpdateInfo(c *check.C) {
	c.Assert((&VolumeUpdate{}).Info(), check.NotNil)
}

func (s *S) TestVolumeUpdate(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"vol1", "plan1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			r.ParseForm()
			dec := form.NewDecoder(nil)
			dec.IgnoreCase(true)
			dec.IgnoreUnknownKeys(true)
			dec.UseJSONTags(false)
			var vol volumeTypes.Volume
			err := dec.DecodeValues(&vol, r.Form)
			c.Assert(err, check.IsNil)
			c.Assert(vol, check.DeepEquals, volumeTypes.Volume{
				Name:      "vol1",
				Plan:      volumeTypes.VolumePlan{Name: "plan1"},
				TeamOwner: "team1",
				Pool:      "pool1",
				Opts:      map[string]string{"a": "1", "b": "2"},
			})
			return strings.HasSuffix(r.URL.Path, "/volumes/vol1") && r.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeUpdate{}
	command.Flags().Parse(true, []string{"-t", "team1", "-p", "pool1", "-o", "a=1", "-o", "b=2"})
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully updated.\n")
}

func (s *S) TestVolumeDeleteInfo(c *check.C) {
	c.Assert((&VolumeDelete{}).Info(), check.NotNil)
}

func (s *S) TestVolumeDelete(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"vol1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			return strings.HasSuffix(r.URL.Path, "/volumes/vol1") && r.Method == "DELETE"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeDelete{}
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully deleted.\n")
}

func (s *S) TestVolumeBindInfo(c *check.C) {
	c.Assert((&VolumeBind{}).Info(), check.NotNil)
}

func (s *S) TestVolumeBind(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"vol1", "/mnt"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			r.ParseForm()
			c.Assert(r.FormValue("App"), check.Equals, "myapp")
			c.Assert(r.FormValue("MountPoint"), check.Equals, "/mnt")
			return strings.HasSuffix(r.URL.Path, "/volumes/vol1/bind") && r.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeBind{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully bound.\n")
}

func (s *S) TestVolumeBindNoRestart(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"vol1", "/mnt"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			r.ParseForm()
			c.Assert(r.FormValue("App"), check.Equals, "myapp")
			c.Assert(r.FormValue("MountPoint"), check.Equals, "/mnt")
			c.Assert(r.FormValue("NoRestart"), check.Equals, "true")
			return strings.HasSuffix(r.URL.Path, "/volumes/vol1/bind") && r.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeBind{}
	command.Flags().Parse(true, []string{"-a", "myapp", "--no-restart"})
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully bound.\n")
}

func (s *S) TestVolumeBindRO(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"vol1", "/mnt"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			r.ParseForm()
			c.Assert(r.FormValue("App"), check.Equals, "myapp")
			c.Assert(r.FormValue("MountPoint"), check.Equals, "/mnt")
			c.Assert(r.FormValue("ReadOnly"), check.Equals, "true")
			return strings.HasSuffix(r.URL.Path, "/volumes/vol1/bind") && r.Method == "POST"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeBind{}
	command.Flags().Parse(true, []string{"-a", "myapp", "--readonly"})
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully bound.\n")
}

func (s *S) TestVolumeUnbindInfo(c *check.C) {
	c.Assert((&VolumeUnbind{}).Info(), check.NotNil)
}

func (s *S) TestVolumeUnbind(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"vol1", "/mnt"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			r.ParseForm()
			c.Assert(r.FormValue("App"), check.Equals, "myapp")
			c.Assert(r.FormValue("MountPoint"), check.Equals, "/mnt")
			return strings.HasSuffix(r.URL.Path, "/volumes/vol1/bind") && r.Method == "DELETE"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeUnbind{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully unbound.\n")
}

func (s *S) TestVolumeUnbindNoRestart(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Args:   []string{"vol1", "/mnt"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			r.ParseForm()
			c.Assert(r.FormValue("App"), check.Equals, "myapp")
			c.Assert(r.FormValue("MountPoint"), check.Equals, "/mnt")
			c.Assert(r.FormValue("NoRestart"), check.Equals, "true")
			return strings.HasSuffix(r.URL.Path, "/volumes/vol1/bind") && r.Method == "DELETE"
		},
	}
	s.setupFakeTransport(trans)
	command := &VolumeUnbind{}
	command.Flags().Parse(true, []string{"-a", "myapp", "--no-restart"})
	err := command.Run(&ctx)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully unbound.\n")
}

func (s *S) TestVolumeClientSideFilter(c *check.C) {
	volumes := []volumeTypes.Volume{

		{
			Name:      "gcp-volume-01",
			Pool:      "gcp-pool-01",
			Plan:      volumeTypes.VolumePlan{Name: "big"},
			TeamOwner: "their-team",
		},
		{
			Name:      "gcp-volume-02",
			Pool:      "gcp-pool-02",
			Plan:      volumeTypes.VolumePlan{Name: "small"},
			TeamOwner: "my-team",
		},
		{
			Name:      "aws-volume-01",
			Pool:      "aws-pool-01",
			Plan:      volumeTypes.VolumePlan{Name: "small"},
			TeamOwner: "their-team",
		},
	}

	filters := []volumeFilter{
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
			pool: "gcp-pool-02",
		},

		{
			plan: "small",
		},

		{
			teamOwner: "my-team",
		},
	}

	expectedResults := [][]string{
		{"gcp-volume-01", "gcp-volume-02"},
		{"aws-volume-01"},
		{"aws-volume-01"},
		{"gcp-volume-02"},
		{"gcp-volume-02", "aws-volume-01"},
		{"gcp-volume-02"},
	}

	for i := range filters {
		cl := VolumeList{
			filter: filters[i],
		}

		filteredVolumes := cl.clientSideFilter(volumes)
		result := []string{}
		for _, volume := range filteredVolumes {
			result = append(result, volume.Name)
		}

		if !c.Check(result, check.DeepEquals, expectedResults[i]) {
			c.Errorf("Failed to test case: %d, %#v", i, filters[i])
		}
	}
}
