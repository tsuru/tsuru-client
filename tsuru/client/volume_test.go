// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/ajg/form"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/volume"
	"gopkg.in/check.v1"
)

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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&VolumeList{}).Run(&ctx, client)
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&VolumeList{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "No volumes available.\n")
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeInfo{}
	err := command.Run(&ctx, client)
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeInfo{}
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "No volumes available.\n")
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&VolumePlansList{}).Run(&ctx, client)
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&VolumePlansList{}).Run(&ctx, client)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, `+------+-------------+------+
| Plan | Provisioner | Opts |
+------+-------------+------+
`)
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
			var vol volume.Volume
			err := dec.DecodeValues(&vol, r.Form)
			c.Assert(err, check.IsNil)
			c.Assert(vol, check.DeepEquals, volume.Volume{
				Name:      "vol1",
				Plan:      volume.VolumePlan{Name: "plan1"},
				TeamOwner: "team1",
				Pool:      "pool1",
				Opts:      map[string]string{"a": "1", "b": "2"},
			})
			return strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeCreate{}
	command.Flags().Parse(true, []string{"-t", "team1", "-p", "pool1", "-o", "a=1", "-o", "b=2"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully created.\n")
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
			var vol volume.Volume
			err := dec.DecodeValues(&vol, r.Form)
			c.Assert(err, check.IsNil)
			c.Assert(vol, check.DeepEquals, volume.Volume{
				Name:      "vol1",
				Plan:      volume.VolumePlan{Name: "plan1"},
				TeamOwner: "team1",
				Pool:      "pool1",
				Opts:      map[string]string{"a": "1", "b": "2"},
			})
			return strings.HasSuffix(r.URL.Path, "/volumes/vol1") && r.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeUpdate{}
	command.Flags().Parse(true, []string{"-t", "team1", "-p", "pool1", "-o", "a=1", "-o", "b=2"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully updated.\n")
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeDelete{}
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully deleted.\n")
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeBind{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&ctx, client)
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeBind{}
	command.Flags().Parse(true, []string{"-a", "myapp", "--no-restart"})
	err := command.Run(&ctx, client)
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeBind{}
	command.Flags().Parse(true, []string{"-a", "myapp", "--readonly"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully bound.\n")
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeUnbind{}
	command.Flags().Parse(true, []string{"-a", "myapp"})
	err := command.Run(&ctx, client)
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
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := &VolumeUnbind{}
	command.Flags().Parse(true, []string{"-a", "myapp", "--no-restart"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.IsNil)
	result := stdout.String()
	c.Assert(result, check.Equals, "Volume successfully unbound.\n")
}
