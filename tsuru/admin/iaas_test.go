// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/ajg/form"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/iaas"
	"gopkg.in/check.v1"
)

func (s *S) TestMachineListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	m1 := iaas.Machine{Id: "id1", Address: "addr1", Iaas: "iaas1", CreationParams: map[string]string{
		"param1": "value1",
	}}
	m2 := iaas.Machine{Id: "id2", Address: "addr2", Iaas: "iaas2", CreationParams: map[string]string{
		"param1": "value1",
		"param2": "value2",
	}}
	data, err := json.Marshal([]iaas.Machine{m1, m2})
	c.Assert(err, check.IsNil)
	expected := `+-----+-------+---------+-----------------+--------------------+
| Id  | IaaS  | Address | Creation Params | Matching Templates |
+-----+-------+---------+-----------------+--------------------+
| id1 | iaas1 | addr1   | param1=value1   | tmpl1              |
+-----+-------+---------+-----------------+--------------------+
| id2 | iaas2 | addr2   | param1=value1   | tmpl1              |
|     |       |         | param2=value2   | tmpl2              |
+-----+-------+---------+-----------------+--------------------+
`
	templates, err := json.Marshal([]iaas.Template{
		{Name: "tmpl1", Data: iaas.TemplateDataList{
			{Name: "param1", Value: "value1"},
		}},
		{Name: "tmpl2", Data: iaas.TemplateDataList{
			{Name: "param1", Value: "value1"},
			{Name: "param2", Value: "value2"},
		}},
	})
	c.Assert(err, check.IsNil)
	trans := &cmdtest.MultiConditionalTransport{ConditionalTransports: []cmdtest.ConditionalTransport{
		{
			Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
			CondFunc: func(req *http.Request) bool {
				return strings.HasSuffix(req.URL.Path, "/iaas/machines") && req.Method == "GET"
			},
		},
		{
			Transport: cmdtest.Transport{Message: string(templates), Status: http.StatusOK},
			CondFunc: func(req *http.Request) bool {
				return strings.HasSuffix(req.URL.Path, "/iaas/templates") && req.Method == "GET"
			},
		},
	}}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := MachineList{}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestMachineDestroyRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"myid1"},
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/iaas/machines/myid1") && req.Method == http.MethodDelete
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := MachineDestroy{}
	command.Flags().Parse(true, []string{"-y"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Machine successfully destroyed.\n")
}

func (s *S) TestTemplateListRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	tpl1 := iaas.Template{Name: "machine1", IaaSName: "ec2", Data: iaas.TemplateDataList{
		iaas.TemplateData{Name: "region", Value: "us-east-1"},
		iaas.TemplateData{Name: "type", Value: "m1.small"},
	}}
	tpl2 := iaas.Template{Name: "tpl1", IaaSName: "ec2", Data: iaas.TemplateDataList{
		iaas.TemplateData{Name: "region", Value: "xxxx"},
		iaas.TemplateData{Name: "type", Value: "l1.large"},
	}}
	data, err := json.Marshal([]iaas.Template{tpl1, tpl2})
	c.Assert(err, check.IsNil)
	expected := `+----------+------+------------------+
| Name     | IaaS | Params           |
+----------+------+------------------+
| machine1 | ec2  | region=us-east-1 |
|          |      | type=m1.small    |
+----------+------+------------------+
| tpl1     | ec2  | region=xxxx      |
|          |      | type=l1.large    |
+----------+------+------------------+
`
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/iaas/templates") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := TemplateList{}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTemplateListCountRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	tpl1 := iaas.Template{Name: "tpl1", IaaSName: "ec2", Data: iaas.TemplateDataList{
		iaas.TemplateData{Name: "region", Value: "us-east-1"},
		iaas.TemplateData{Name: "type", Value: "m1.small"},
	}}
	tpl2 := iaas.Template{Name: "tpl2", IaaSName: "ec2", Data: iaas.TemplateDataList{
		iaas.TemplateData{Name: "region", Value: "xxxx"},
		iaas.TemplateData{Name: "type", Value: "l1.large"},
	}}
	data, err := json.Marshal([]iaas.Template{tpl1, tpl2})
	c.Assert(err, check.IsNil)
	expected := `+------+------+------------------+------------+
| Name | IaaS | Params           | # Machines |
+------+------+------------------+------------+
| tpl1 | ec2  | region=us-east-1 | 2          |
|      |      | type=m1.small    |            |
+------+------+------------------+------------+
| tpl2 | ec2  | region=xxxx      | 1          |
|      |      | type=l1.large    |            |
+------+------+------------------+------------+
`
	machines, err := json.Marshal([]iaas.Machine{
		{CreationParams: map[string]string{"region": "xxxx", "type": "l1.large", "extra": "xpto"}},
		{CreationParams: map[string]string{"region": "us-east-1", "type": "m1.small"}},
		{CreationParams: map[string]string{"region": "us-east-1", "type": "m1.small", "extra": "xpto"}},
	})
	trans := &cmdtest.MultiConditionalTransport{ConditionalTransports: []cmdtest.ConditionalTransport{
		{
			Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
			CondFunc: func(req *http.Request) bool {
				return strings.HasSuffix(req.URL.Path, "/iaas/templates") && req.Method == "GET"
			},
		},
		{
			Transport: cmdtest.Transport{Message: string(machines), Status: http.StatusOK},
			CondFunc: func(req *http.Request) bool {
				return strings.HasSuffix(req.URL.Path, "/iaas/machines") && req.Method == "GET"
			},
		},
	}}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := TemplateList{}
	command.Flags().Parse(true, []string{"--count"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestTemplateAddCmdRun(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"my-tpl", "ec2", "zone=xyz", "image=ami-something"}, Stdout: &buf}
	expectedBody := iaas.Template{
		Name:     "my-tpl",
		IaaSName: "ec2",
		Data: []iaas.TemplateData{
			{Name: "zone", Value: "xyz"},
			{Name: "image", Value: "ami-something"},
		},
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			var paramTemplate iaas.Template
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			err = dec.DecodeValues(&paramTemplate, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(paramTemplate, check.DeepEquals, expectedBody)
			path := strings.HasSuffix(req.URL.Path, "/iaas/templates")
			method := req.Method == "POST"
			return path && method
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := TemplateAdd{}
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "Template successfully added.\n")
}

func (s *S) TestTemplateRemoveCmdRun(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"my-tpl"}, Stdout: &buf, Stderr: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/iaas/templates/my-tpl") && req.Method == http.MethodDelete
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := TemplateRemove{}
	cmd.Flags().Parse(true, []string{"-y"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "Template successfully removed.\n")
}

func (s *S) TestTemplateUpdateCmdRun(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"my-tpl", "zone=", "image=ami-something"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			var template iaas.Template
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			err = dec.DecodeValues(&template, req.Form)
			c.Assert(err, check.IsNil)
			expected := iaas.Template{
				Name: "my-tpl",
				Data: iaas.TemplateDataList{
					iaas.TemplateData{Name: "zone", Value: ""},
					iaas.TemplateData{Name: "image", Value: "ami-something"},
				},
			}
			c.Assert(template, check.DeepEquals, expected)
			path := strings.HasSuffix(req.URL.Path, "/iaas/templates/my-tpl")
			method := req.Method == "PUT"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && method && contentType
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := TemplateUpdate{iaasName: ""}
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "Template successfully updated.\n")
}

func (s *S) TestTemplateUpdateIaaS(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"my-tpl", "zone=us"}, Stdout: &buf}
	expectedBody := iaas.Template{
		Name:     "my-tpl",
		IaaSName: "ec2",
		Data: iaas.TemplateDataList{
			iaas.TemplateData{Name: "zone", Value: "us"},
		},
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			var template iaas.Template
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			err = dec.DecodeValues(&template, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(template, check.DeepEquals, expectedBody)
			path := strings.HasSuffix(req.URL.Path, "/iaas/templates/my-tpl")
			method := req.Method == "PUT"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && method && contentType
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := TemplateUpdate{}
	cmd.Flags().Parse(true, []string{"-i", "ec2"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "Template successfully updated.\n")
}

func (s *S) TestTemplateFailToUpdateIaaS(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"my-tpl"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedBody := iaas.Template{Name: "my-tpl", IaaSName: "notvalidiaas"}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusConflict},
		CondFunc: func(req *http.Request) bool {
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			var template iaas.Template
			dec := form.NewDecoder(nil)
			dec.IgnoreUnknownKeys(true)
			err = dec.DecodeValues(&template, req.Form)
			c.Assert(err, check.IsNil)
			c.Assert(template, check.DeepEquals, expectedBody)
			path := strings.HasSuffix(req.URL.Path, "/iaas/templates/my-tpl")
			method := req.Method == "PUT"
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return path && method && contentType
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := TemplateUpdate{}
	cmd.Flags().Parse(true, []string{"-i", "notvalidiaas"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "")
	c.Assert(stderr.String(), check.Equals, "Failed to update template.\n")
}

func (s *S) TestTemplateCopyCmdRun(c *check.C) {
	var buf bytes.Buffer
	tpl1 := iaas.Template{Name: "tpl1", IaaSName: "ec2", Data: iaas.TemplateDataList{
		iaas.TemplateData{Name: "region", Value: "xxxx"},
		iaas.TemplateData{Name: "type", Value: "l1.large"},
	}}
	data, err := json.Marshal([]iaas.Template{tpl1})
	c.Assert(err, check.IsNil)
	context := cmd.Context{Args: []string{"my-tpl", "tpl1", "zone=xyz", "type=l2.large"}, Stdout: &buf}
	expectedBody := iaas.Template{
		Name:     "my-tpl",
		IaaSName: "ec2",
		Data: []iaas.TemplateData{
			{Name: "region", Value: "xxxx"},
			{Name: "type", Value: "l2.large"},
			{Name: "zone", Value: "xyz"},
		},
	}
	trans := &cmdtest.MultiConditionalTransport{ConditionalTransports: []cmdtest.ConditionalTransport{
		{
			Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
			CondFunc: func(req *http.Request) bool {
				c.Assert(strings.HasSuffix(req.URL.Path, "/iaas/templates"), check.Equals, true)
				c.Assert(req.Method, check.Equals, "GET")
				return true
			},
		},
		{
			Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
			CondFunc: func(req *http.Request) bool {
				err = req.ParseForm()
				c.Assert(err, check.IsNil)
				var paramTemplate iaas.Template
				dec := form.NewDecoder(nil)
				dec.IgnoreUnknownKeys(true)
				err = dec.DecodeValues(&paramTemplate, req.Form)
				c.Assert(err, check.IsNil)
				sort.Slice(paramTemplate.Data, func(i, j int) bool {
					return paramTemplate.Data[i].Name < paramTemplate.Data[j].Name
				})
				c.Assert(paramTemplate, check.DeepEquals, expectedBody)
				path := strings.HasSuffix(req.URL.Path, "/iaas/templates")
				method := req.Method == "POST"
				return path && method
			},
		},
	}}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := TemplateCopy{}
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "Template successfully copied.\n")
}
