// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/provision/pool"
	"gopkg.in/check.v1"
)

func decodeJSONBody(c *check.C, req *http.Request, opts interface{}) {
	err := json.NewDecoder(req.Body).Decode(&opts)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAddPoolToTheSchedulerCmd(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/pools")
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := AddPoolToSchedulerCmd{}
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAddPublicPool(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			opts := new(addOpts)
			decodeJSONBody(c, req, opts)
			url := strings.HasSuffix(req.URL.Path, "/pools")
			name := opts.Name == "test"
			public := opts.Public == true
			def := opts.DefaultPool == false

			return url && name && public && def
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := AddPoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"-p"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAddDefaultPool(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools")
			opts := new(addOpts)
			decodeJSONBody(c, req, opts)
			name := opts.Name == "test"
			public := opts.Public == false
			def := opts.DefaultPool == true
			return url && name && public && def
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"-d"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAddPoolWithProvisioner(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools")
			opts := new(addOpts)
			decodeJSONBody(c, req, opts)
			name := opts.Name == "test"
			public := opts.Public == false
			def := opts.DefaultPool == false
			prov := opts.Provisioner == "kub"
			return url && name && public && def && prov
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"--provisioner", "kub"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAddPoolWithLabels(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			opts := new(addOpts)
			decodeJSONBody(c, req, opts)
			if v, ok := opts.Labels["test-key"]; ok {
				if strings.Compare(v, "test-value") == 0 {
					return true
				}
			}
			return false
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := AddPoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"--labels", "test-key=test-value"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestFailToAddMoreThanOneDefaultPool(c *check.C) {
	var buf bytes.Buffer
	stdin := bytes.NewBufferString("no")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			opts := new(addOpts)
			decodeJSONBody(c, req, opts)
			name := opts.Name == "test"
			public := opts.Public == false
			def := opts.DefaultPool == true
			url := strings.HasSuffix(req.URL.Path, "/pools")
			return name && public && def && url
		},
	}
	manager := cmd.Manager{}
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf, Stdin: stdin}
	client := cmd.NewClient(&http.Client{Transport: &transportError}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"-d"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := "WARNING: Default pool already exist. Do you want change to test pool? (y/n) Pool add aborted.\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestForceToOverwriteDefaultPool(c *check.C) {
	var buf bytes.Buffer
	stdin := bytes.NewBufferString("no")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			opts := new(addOpts)
			decodeJSONBody(c, req, opts)
			name := opts.Name == "test"
			public := opts.Public == false
			def := opts.DefaultPool == true
			force := opts.ForceDefault == true
			return name && public && def && force
		},
	}
	manager := cmd.Manager{}
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf, Stdin: stdin}
	client := cmd.NewClient(&http.Client{Transport: &transportError}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"-d"})
	command.Flags().Parse(true, []string{"-f"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAskOverwriteDefaultPool(c *check.C) {
	var buf bytes.Buffer
	var called int
	stdin := bytes.NewBufferString("yes")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			called++
			opts := new(addOpts)
			decodeJSONBody(c, req, opts)
			name := opts.Name == "test"
			public := opts.Public == false
			def := opts.DefaultPool == true
			url := opts.ForceDefault == false
			return url && name && public && def
		},
	}
	transportOk := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK, Message: ""},
		CondFunc: func(req *http.Request) bool {
			called++
			opts := new(addOpts)
			decodeJSONBody(c, req, opts)
			return opts.ForceDefault == true
		},
	}
	multiTransport := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{transportError, transportOk},
	}
	context := cmd.Context{
		Args:   []string{"test"},
		Stdout: &buf,
		Stdin:  stdin,
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: &multiTransport}, nil, &manager)
	command := AddPoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"-d"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, 2)
	expected := "WARNING: Default pool already exist. Do you want change to test pool? (y/n) Pool successfully registered.\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestUpdatePoolToTheSchedulerCmd(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			opts := new(updateOpts)
			decodeJSONBody(c, req, opts)
			def := opts.DefaultPool == nil
			var public bool
			if opts.Public != nil {
				public = *opts.Public == true
			}
			labels := opts.Labels == nil
			force := opts.ForceDefault == false
			url := strings.HasSuffix(req.URL.Path, "/pools/poolTest")
			method := req.Method == "PUT"
			return public && method && url && force && def && labels
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := UpdatePoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"--public", "true"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestUpdatePoolAddLabels(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	pool := tsuru.Pool{
		Name: "poolTest",
	}
	data, err := json.Marshal(pool)
	c.Assert(err, check.IsNil)
	trans := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					c.Assert(req.Method, check.Equals, http.MethodGet)
					c.Assert(req.URL.Path, check.Equals, "/pools/poolTest")
					return true
				},
			},
			{
				Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					url := strings.HasSuffix(req.URL.Path, "/pools/poolTest")
					method := req.Method == "PUT"
					opts := new(updateOpts)
					decodeJSONBody(c, req, opts)
					expected := map[string]string{"test-key": "test-value"}
					c.Assert(opts.Labels, check.DeepEquals, expected)
					return url && method
				},
			},
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, &manager)
	cmd := UpdatePoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"--add-labels", "test-key=test-value"})
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestUpdatePoolFailRemoveUnexistingLabels(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	pool := tsuru.Pool{
		Name: "poolTest",
	}
	data, err := json.Marshal(pool)
	c.Assert(err, check.IsNil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			c.Assert(req.Method, check.Equals, http.MethodGet)
			c.Assert(req.URL.Path, check.Equals, "/pools/poolTest")
			return true
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, &manager)
	cmd := UpdatePoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"--remove-labels", "test-key"})
	err = cmd.Run(&context, client)
	c.Assert(err.Error(), check.Equals, "key test-key does not exist in pool labelset, can't delete an unexisting key")
}

func (s *S) TestUpdatePoolRemoveAllLabels(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	pool := tsuru.Pool{
		Name:   "poolTest",
		Labels: map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
	}
	data, err := json.Marshal(pool)
	c.Assert(err, check.IsNil)
	trans := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					c.Assert(req.Method, check.Equals, http.MethodGet)
					c.Assert(req.URL.Path, check.Equals, "/pools/poolTest")
					return true
				},
			},
			{
				Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					url := strings.HasSuffix(req.URL.Path, "/pools/poolTest")
					method := req.Method == "PUT"
					opts := new(updateOpts)
					decodeJSONBody(c, req, opts)
					expected := map[string]string{}
					c.Assert(opts.Labels, check.DeepEquals, expected)
					return url && method
				},
			},
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, &manager)
	cmd := UpdatePoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"--remove-labels", "k1", "--remove-labels", "k2", "--remove-labels", "k3"})
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestUpdatePoolRemoveAllLabelsThenAddNewOnes(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	pool := tsuru.Pool{
		Name:   "poolTest",
		Labels: map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
	}
	data, err := json.Marshal(pool)
	c.Assert(err, check.IsNil)
	trans := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					c.Assert(req.Method, check.Equals, http.MethodGet)
					c.Assert(req.URL.Path, check.Equals, "/pools/poolTest")
					return true
				},
			},
			{
				Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					url := strings.HasSuffix(req.URL.Path, "/pools/poolTest")
					method := req.Method == "PUT"
					opts := new(updateOpts)
					decodeJSONBody(c, req, opts)
					c.Assert(opts.Labels, check.DeepEquals, map[string]string{"new-key": "new-value"})
					return url && method
				},
			},
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, &manager)
	cmd := UpdatePoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"--remove-labels", "k1", "--remove-labels", "k2", "--remove-labels", "k3", "--add-labels", "new-key=new-value"})
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestUpdatePoolWithLabelsAddAndRemoveLabels(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	pool := tsuru.Pool{
		Name:   "poolTest",
		Labels: map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
	}
	data, err := json.Marshal(pool)
	c.Assert(err, check.IsNil)
	trans := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					c.Assert(req.Method, check.Equals, http.MethodGet)
					c.Assert(req.URL.Path, check.Equals, "/pools/poolTest")
					return true
				},
			},
			{
				Transport: cmdtest.Transport{Message: string(data), Status: http.StatusOK},
				CondFunc: func(req *http.Request) bool {
					url := strings.HasSuffix(req.URL.Path, "/pools/poolTest")
					method := req.Method == "PUT"
					opts := new(updateOpts)
					decodeJSONBody(c, req, opts)
					expected := map[string]string{"k1": "v1", "k3": "v3", "k4": "v4"}
					c.Assert(opts.Labels, check.DeepEquals, expected)
					return url && method
				},
			},
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, &manager)
	cmd := UpdatePoolToSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"--add-labels", "k4=v4", "--remove-labels", "k2"})
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestFailToUpdateMoreThanOneDefaultPool(c *check.C) {
	var buf bytes.Buffer
	stdin := bytes.NewBufferString("no")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			opts := new(updateOpts)
			decodeJSONBody(c, req, opts)
			var def bool
			if opts.DefaultPool != nil {
				def = *opts.DefaultPool == true
			}
			public := opts.Public == nil
			force := opts.ForceDefault == false
			labels := opts.Labels == nil
			url := strings.HasSuffix(req.URL.Path, "/pools/test")
			return def && url && public && force && labels
		},
	}
	manager := cmd.Manager{}
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf, Stdin: stdin}
	client := cmd.NewClient(&http.Client{Transport: &transportError}, nil, &manager)
	command := UpdatePoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"--default=true"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := "WARNING: Default pool already exist. Do you want change to test pool? (y/n) Pool update aborted.\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestForceToOverwriteDefaultPoolInUpdate(c *check.C) {
	var buf bytes.Buffer
	stdin := bytes.NewBufferString("no")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			opts := new(updateOpts)
			decodeJSONBody(c, req, opts)
			var def bool
			if opts.DefaultPool != nil {
				def = *opts.DefaultPool == true
			}
			public := opts.Public == nil
			labels := opts.Labels == nil
			force := opts.ForceDefault == true
			url := strings.HasSuffix(req.URL.Path, "/pools/test")
			method := req.Method == "PUT"
			return url && force && def && public && method && labels
		},
	}
	manager := cmd.Manager{}
	context := cmd.Context{Args: []string{"test"}, Stdout: &buf, Stdin: stdin}
	client := cmd.NewClient(&http.Client{Transport: &transportError}, nil, &manager)
	command := UpdatePoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"--default=true"})
	command.Flags().Parse(true, []string{"-f"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestAskOverwriteDefaultPoolInUpdate(c *check.C) {
	var buf bytes.Buffer
	var called int
	stdin := bytes.NewBufferString("yes")
	transportError := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusPreconditionFailed, Message: "Default pool already exist."},
		CondFunc: func(req *http.Request) bool {
			called++
			opts := new(updateOpts)
			decodeJSONBody(c, req, opts)
			var def bool
			if opts.DefaultPool != nil {
				def = *opts.DefaultPool == true
			}
			public := opts.Public == nil
			force := opts.ForceDefault == false
			url := strings.HasSuffix(req.URL.Path, "/pools/test")
			return url && def && public && force
		},
	}
	transportOk := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK, Message: ""},
		CondFunc: func(req *http.Request) bool {
			called++
			url := strings.HasSuffix(req.URL.Path, "/pools/test")
			opts := new(updateOpts)
			decodeJSONBody(c, req, opts)
			force := opts.ForceDefault == true
			return url && force
		},
	}
	multiTransport := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{transportError, transportOk},
	}
	context := cmd.Context{
		Args:   []string{"test"},
		Stdout: &buf,
		Stdin:  stdin,
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: &multiTransport}, nil, &manager)
	command := UpdatePoolToSchedulerCmd{}
	command.Flags().Parse(true, []string{"--default=true"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, 2)
	expected := "WARNING: Default pool already exist. Do you want change to test pool? (y/n) Pool successfully updated.\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestRemovePoolFromTheSchedulerCmd(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"poolTest"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools/poolTest")
			method := req.Method == http.MethodDelete
			return method && url
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := RemovePoolFromSchedulerCmd{}
	cmd.Flags().Parse(true, []string{"-y"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestRemovePoolFromTheSchedulerCmdConfirmation(c *check.C) {
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{"poolX"},
		Stdout: &stdout,
		Stdin:  strings.NewReader("n\n"),
	}
	command := RemovePoolFromSchedulerCmd{}
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Are you sure you want to remove \"poolX\" pool? (y/n) Abort.\n")
}

func (s *S) TestAddTeamsToPoolCmdRun(c *check.C) {
	var buf bytes.Buffer
	ctx := cmd.Context{Stdout: &buf, Args: []string{"pool1", "team1", "team2"}}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools/pool1/team")
			method := req.Method == "POST"
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			teams := req.Form["team"]
			c.Assert(teams, check.DeepEquals, []string{"team1", "team2"})
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return url && method && contentType
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	err := AddTeamsToPoolCmd{}.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestRemoveTeamsFromPoolCmdRun(c *check.C) {
	var buf bytes.Buffer
	ctx := cmd.Context{Stdout: &buf, Args: []string{"pool1", "team1"}}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/pools/pool1/team")
			method := req.Method == http.MethodDelete
			rq := req.URL.RawQuery == "team=team1"
			return url && method && rq
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	err := RemoveTeamsFromPoolCmd{}.Run(&ctx, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestPoolConstraintList(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Stdout: &buf}
	constraints := []pool.PoolConstraint{
		{PoolExpr: "*", Field: "router", Values: []string{"routerA", "routerB"}},
		{PoolExpr: "dev", Field: "team", Values: []string{"*"}, Blacklist: true},
	}
	json, err := json.Marshal(constraints)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(json), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/constraints")
			method := req.Method == "GET"
			return method && url
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &cmd.Manager{})
	cmd := PoolConstraintList{}
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, `+-----------------+--------+-----------------+-----------+
| Pool Expression | Field  | Values          | Blacklist |
+-----------------+--------+-----------------+-----------+
| *               | router | routerA,routerB | false     |
| dev             | team   | *               | true      |
+-----------------+--------+-----------------+-----------+
`)
}

func (s *S) TestPoolConstraintSetDefaultFlags(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"*", "router", "myrouter", "myrouter2"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			constraint := new(pool.PoolConstraint)
			decodeJSONBody(c, req, constraint)
			url := strings.HasSuffix(req.URL.Path, "/constraints")
			c.Assert(constraint, check.DeepEquals, &pool.PoolConstraint{
				PoolExpr:  "*",
				Field:     "router",
				Values:    []string{"myrouter", "myrouter2"},
				Blacklist: false,
			})
			method := req.Method == "PUT"
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			append := req.FormValue("append") == ""
			return method && append && url
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &cmd.Manager{})
	cmd := PoolConstraintSet{}
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestPoolConstraintSet(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"*", "router", "myrouter"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			constraint := new(pool.PoolConstraint)
			decodeJSONBody(c, req, constraint)
			url := strings.HasSuffix(req.URL.Path, "/constraints")
			c.Assert(constraint, check.DeepEquals, &pool.PoolConstraint{
				PoolExpr:  "*",
				Field:     "router",
				Values:    []string{"myrouter"},
				Blacklist: true,
			})
			method := req.Method == "PUT"
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			append := req.FormValue("append") == "true"
			return method && append && url
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &cmd.Manager{})
	cmd := PoolConstraintSet{}
	cmd.Flags().Parse(true, []string{"--blacklist", "--append"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}

func (s *S) TestPoolConstraintSetEmptyValues(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Args: []string{"*", "router"}, Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			constraint := new(pool.PoolConstraint)
			decodeJSONBody(c, req, constraint)
			url := strings.HasSuffix(req.URL.Path, "/constraints")
			c.Assert(constraint, check.DeepEquals, &pool.PoolConstraint{
				PoolExpr:  "*",
				Field:     "router",
				Blacklist: false,
			})
			method := req.Method == "PUT"
			err := req.ParseForm()
			c.Assert(err, check.IsNil)
			append := req.FormValue("append") == ""
			return method && append && url
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &cmd.Manager{})
	cmd := PoolConstraintSet{}
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
}
