// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
	"launchpad.net/gnuflag"
)

var appflag = &gnuflag.Flag{
	Name:     "app",
	Usage:    "The name of the app.",
	Value:    nil,
	DefValue: "",
}

var appshortflag = &gnuflag.Flag{
	Name:     "a",
	Usage:    "The name of the app.",
	Value:    nil,
	DefValue: "",
}

func (s *S) TestAppCreateInfo(c *check.C) {
	c.Assert((&appCreate{}).Info(), check.NotNil)
}

func (s *S) TestAppCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app-info to check the status of the app and its units.
Your repository for "ble" project is "git@tsuru.plataformas.glb.com:ble.git"` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			defer req.Body.Close()
			body, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			expected := map[string]interface{}{
				"name":      "ble",
				"platform":  "django",
				"teamOwner": "",
				"plan":      map[string]interface{}{"name": ""},
				"pool":      "",
			}
			result := map[string]interface{}{}
			err = json.Unmarshal(body, &result)
			c.Assert(expected, check.DeepEquals, result)
			return req.Method == "POST" && req.URL.Path == "/apps"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := appCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreateTeamOwner(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app-info to check the status of the app and its units.
Your repository for "ble" project is "git@tsuru.plataformas.glb.com:ble.git"` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			defer req.Body.Close()
			body, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			expected := map[string]interface{}{
				"name":      "ble",
				"platform":  "django",
				"teamOwner": "team",
				"plan":      map[string]interface{}{"name": ""},
				"pool":      "",
			}
			result := map[string]interface{}{}
			err = json.Unmarshal(body, &result)
			c.Assert(expected, check.DeepEquals, result)
			return req.Method == "POST" && req.URL.Path == "/apps"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := appCreate{}
	command.Flags().Parse(true, []string{"-t", "team"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreatePlan(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app-info to check the status of the app and its units.
Your repository for "ble" project is "git@tsuru.plataformas.glb.com:ble.git"` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			defer req.Body.Close()
			body, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			expected := map[string]interface{}{
				"name":      "ble",
				"platform":  "django",
				"teamOwner": "",
				"plan":      map[string]interface{}{"name": "myplan"},
				"pool":      "",
			}
			result := map[string]interface{}{}
			err = json.Unmarshal(body, &result)
			c.Assert(expected, check.DeepEquals, result)
			return req.Method == "POST" && req.URL.Path == "/apps"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := appCreate{}
	command.Flags().Parse(true, []string{"-p", "myplan"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreatePool(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app-info to check the status of the app and its units.
Your repository for "ble" project is "git@tsuru.plataformas.glb.com:ble.git"` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			defer req.Body.Close()
			body, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			expected := map[string]interface{}{
				"name":      "ble",
				"platform":  "django",
				"teamOwner": "",
				"plan":      map[string]interface{}{"name": ""},
				"pool":      "mypool",
			}
			result := map[string]interface{}{}
			err = json.Unmarshal(body, &result)
			c.Assert(expected, check.DeepEquals, result)
			return req.Method == "POST" && req.URL.Path == "/apps"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := appCreate{}
	command.Flags().Parse(true, []string{"-o", "mypool"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreateNoRepository(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success"}`
	expected := `App "ble" has been created!
Use app-info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			defer req.Body.Close()
			body, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			expected := map[string]interface{}{
				"name":      "ble",
				"platform":  "django",
				"teamOwner": "",
				"plan":      map[string]interface{}{"name": ""},
				"pool":      "",
			}
			result := map[string]interface{}{}
			err = json.Unmarshal(body, &result)
			c.Assert(expected, check.DeepEquals, result)
			return req.Method == "POST" && req.URL.Path == "/apps"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := appCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreateWithInvalidFramework(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"invalidapp", "lombra"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusInternalServerError}}, nil, manager)
	command := appCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestAppCreateFlags(c *check.C) {
	command := appCreate{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse(true, []string{"-p", "myplan"})
	plan := flagset.Lookup("plan")
	usage := "The plan used to create the app"
	c.Check(plan, check.NotNil)
	c.Check(plan.Name, check.Equals, "plan")
	c.Check(plan.Usage, check.Equals, usage)
	c.Check(plan.Value.String(), check.Equals, "myplan")
	c.Check(plan.DefValue, check.Equals, "")
	splan := flagset.Lookup("p")
	c.Check(splan, check.NotNil)
	c.Check(splan.Name, check.Equals, "p")
	c.Check(splan.Usage, check.Equals, usage)
	c.Check(splan.Value.String(), check.Equals, "myplan")
	c.Check(splan.DefValue, check.Equals, "")
	flagset.Parse(true, []string{"-t", "team"})
	usage = "Team owner app"
	teamOwner := flagset.Lookup("team")
	c.Check(teamOwner, check.NotNil)
	c.Check(teamOwner.Name, check.Equals, "team")
	c.Check(teamOwner.Usage, check.Equals, usage)
	c.Check(teamOwner.Value.String(), check.Equals, "team")
	c.Check(teamOwner.DefValue, check.Equals, "")
	teamOwner = flagset.Lookup("t")
	c.Check(teamOwner, check.NotNil)
	c.Check(teamOwner.Name, check.Equals, "t")
	c.Check(teamOwner.Usage, check.Equals, usage)
	c.Check(teamOwner.Value.String(), check.Equals, "team")
	c.Check(teamOwner.DefValue, check.Equals, "")
}

func (s *S) TestAppRemove(c *check.C) {
	var stdout, stderr bytes.Buffer
	expectedOut := "-- removed --"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	expected := `Are you sure you want to remove app "ble"? (y/n) `
	context := cmd.Context{
		Args:   []string{"ble"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: string(result), Status: http.StatusOK}}, nil, manager)
	command := appRemove{}
	command.Flags().Parse(true, []string{"-a", "ble"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected+expectedOut)
}

func (s *S) TestAppRemoveWithoutAsking(c *check.C) {
	var stdout, stderr bytes.Buffer
	expectedOut := "-- removed --"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	context := cmd.Context{
		Args:   []string{"ble"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: string(result), Status: http.StatusOK}}, nil, manager)
	command := appRemove{}
	command.Flags().Parse(true, []string{"-a", "ble", "-y"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestAppRemoveFlags(c *check.C) {
	command := appRemove{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse(true, []string{"-a", "ashamed", "-y"})
	app := flagset.Lookup("app")
	c.Check(app, check.NotNil)
	c.Check(app.Name, check.Equals, "app")
	c.Check(app.Usage, check.Equals, "The name of the app.")
	c.Check(app.Value.String(), check.Equals, "ashamed")
	c.Check(app.DefValue, check.Equals, "")
	sapp := flagset.Lookup("a")
	c.Check(sapp, check.NotNil)
	c.Check(sapp.Name, check.Equals, "a")
	c.Check(sapp.Usage, check.Equals, "The name of the app.")
	c.Check(sapp.Value.String(), check.Equals, "ashamed")
	c.Check(sapp.DefValue, check.Equals, "")
	assume := flagset.Lookup("assume-yes")
	c.Check(assume, check.NotNil)
	c.Check(assume.Name, check.Equals, "assume-yes")
	c.Check(assume.Usage, check.Equals, "Don't ask for confirmation.")
	c.Check(assume.Value.String(), check.Equals, "true")
	c.Check(assume.DefValue, check.Equals, "false")
	sassume := flagset.Lookup("y")
	c.Check(sassume, check.NotNil)
	c.Check(sassume.Name, check.Equals, "y")
	c.Check(sassume.Usage, check.Equals, "Don't ask for confirmation.")
	c.Check(sassume.Value.String(), check.Equals, "true")
	c.Check(sassume.DefValue, check.Equals, "false")
}

func (s *S) TestAppRemoveWithoutArgs(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Please use the -a/--app flag to specify which app you want to remove."
	context := cmd.Context{
		Args:   nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: ""}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := appRemove{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, expected)
}

func (s *S) TestAppRemoveWithoutConfirmation(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Are you sure you want to remove app "ble"? (y/n) Abort.` + "\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("n\n"),
	}
	command := appRemove{}
	command.Flags().Parse(true, []string{"--app", "ble"})
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRemoveInfo(c *check.C) {
	c.Assert((&appRemove{}).Info(), check.NotNil)
}

func (s *S) TestAppInfo(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7}`
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: myapp.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Quota: 0/unlimited

Units: 3
+--------+---------+
| Unit   | State   |
+--------+---------+
| app1/0 | started |
| app1/1 | started |
| app1/2 | pending |
+--------+---------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"-a/--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

type transportFunc func(req *http.Request) (resp *http.Response, err error)

func (fn transportFunc) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return fn(req)
}

func (s *S) TestAppInfoWithQuota(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: myapp.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Quota: 3/40 units

Units: 3
+--------+---------+
| Unit   | State   |
+--------+---------+
| app1/0 | started |
| app1/1 | started |
| app1/2 | pending |
+--------+---------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		var body string
		if req.URL.Path == "/apps/app1/quota" {
			body = `{"Limit":40,"InUse":3}`
		} else if req.URL.Path == "/apps/app1" {
			body = `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7}`
		}
		return &http.Response{
			Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoLock(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "lock": {"locked": true, "owner": "admin@example.com", "reason": "DELETE /apps/rbsample/units", "acquiredate": "2012-04-01T10:32:00Z"}}`
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: myapp.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Lock:
 Acquired in: %s
 Owner: admin@example.com
 Running: DELETE /apps/rbsample/units
Quota: 0/unlimited

Units: 3
+--------+---------+
| Unit   | State   |
+--------+---------+
| app1/0 | started |
| app1/1 | started |
| app1/2 | pending |
+--------+---------+

`
	expected = fmt.Sprintf(expected, time.Date(2012, time.April, 1, 10, 32, 0, 0, time.UTC))
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"-a/--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoManyProcesses(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{
  "name": "app1",
  "teamowner": "myteam",
  "cname": [
    ""
  ],
  "ip": "myapp.tsuru.io",
  "platform": "php",
  "repository": "git@git.com:php.git",
  "state": "dead",
  "units": [
    {
      "Ip": "10.10.10.10",
      "ID": "app1/0",
      "Status": "started",
      "ProcessName": "web"
    },
    {
      "Ip": "9.9.9.9",
      "ID": "app1/1",
      "Status": "started",
      "ProcessName": "worker"
    },
    {
      "Ip": "",
      "ID": "app1/2",
      "Status": "pending",
      "ProcessName": "worker"
    }
  ],
  "teams": [
    "tsuruteam",
    "crane"
  ],
  "owner": "myapp_owner",
  "deploys": 7
}`
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: myapp.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Quota: 0/unlimited

Units [web]: 1
+--------+---------+
| Unit   | State   |
+--------+---------+
| app1/0 | started |
+--------+---------+

Units [worker]: 2
+--------+---------+
| Unit   | State   |
+--------+---------+
| app1/1 | started |
| app1/2 | pending |
+--------+---------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"-a/--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoNoUnits(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","ip":"app1.tsuru.io","teamowner":"myteam","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7}`
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: app1.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Quota: 0/unlimited

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"-a/--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoEmptyUnit(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"x","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Name":"","Status":""}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7}`
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: myapp.tsuru.io
Owner: myapp_owner
Team owner: x
Deploys: 7
Pool:
Quota: 0/unlimited

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"-a/--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithoutArgs(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"secret","teamowner":"myteam","ip":"secret.tsuru.io","platform":"ruby","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"secret/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"secret/1","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7}`
	expected := `Application: secret
Repository: git@git.com:php.git
Platform: ruby
Teams: tsuruteam, crane
Address: secret.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Quota: 0/unlimited

Units: 2
+----------+---------+
| Unit     | State   |
+----------+---------+
| secret/0 | started |
| secret/1 | pending |
+----------+---------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/apps/secret" && req.Method == "GET" || req.URL.Path == "/apps/secret/quota" && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := appInfo{GuessingCommand: guessCommand}
	command.Flags().Parse(true, nil)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoCName(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","ip":"myapp.tsuru.io","cname":["yourapp.tsuru.io"],"platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7}`
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: yourapp.tsuru.io, myapp.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Quota: 0/unlimited

Units: 3
+--------+---------+
| Unit   | State   |
+--------+---------+
| app1/0 | started |
| app1/1 | started |
| app1/2 | pending |
+--------+---------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"-a/--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithServices(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: myapp.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Quota: 0/unlimited

Units: 3
+--------+---------+
| Unit   | State   |
+--------+---------+
| app1/0 | started |
| app1/1 | started |
| app1/2 | pending |
+--------+---------+

Service instances: 1
+----------+------------+
| Service  | Instance   |
+----------+------------+
| redisapi | myredisapi |
+----------+------------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		var body string
		if req.URL.Path == "/apps/app1" {
			body = `{"name":"app1","teamowner":"myteam","ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7}`
		} else if req.URL.Path == "/services/instances" && req.URL.RawQuery == "app=app1" {
			body = `[{"service":"redisapi","instances":["myredisapi"]},
					 {"service":"mongodb", "instances":[]}]`
		}
		return &http.Response{
			Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithPlan(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "plan":{"name": "test",  "memory": 536870912, "swap": 268435456, "cpushare": 100, "router": "imemine", "default": false}}`
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: myapp.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Quota: 0/unlimited

Units: 3
+--------+---------+
| Unit   | State   |
+--------+---------+
| app1/0 | started |
| app1/1 | started |
| app1/2 | pending |
+--------+---------+

App Plan:
+------+-----------+-----------+-----------+---------+---------+
| Name | Memory    | Swap      | Cpu Share | Router  | Default |
+------+-----------+-----------+-----------+---------+---------+
| test | 536870912 | 268435456 | 100       | imemine | false   |
+------+-----------+-----------+-----------+---------+---------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"-a/--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithServicesAndPlan(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Application: app1
Repository: git@git.com:php.git
Platform: php
Teams: tsuruteam, crane
Address: myapp.tsuru.io
Owner: myapp_owner
Team owner: myteam
Deploys: 7
Pool:
Quota: 0/unlimited

Units: 3
+--------+---------+
| Unit   | State   |
+--------+---------+
| app1/0 | started |
| app1/1 | started |
| app1/2 | pending |
+--------+---------+

Service instances: 1
+----------+------------+
| Service  | Instance   |
+----------+------------+
| redisapi | myredisapi |
+----------+------------+

App Plan:
+------+-----------+-----------+-----------+---------+---------+
| Name | Memory    | Swap      | Cpu Share | Router  | Default |
+------+-----------+-----------+-----------+---------+---------+
| test | 536870912 | 268435456 | 100       | freeeee | false   |
+------+-----------+-----------+-----------+---------+---------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		var body string
		if req.URL.Path == "/apps/app1" {
			body = `{"name":"app1","teamowner":"myteam","ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7,"plan":{"name": "test",  "memory": 536870912, "swap": 268435456, "cpushare": 100, "router": "freeeee", "default": false}}`
		} else if req.URL.Path == "/services/instances" && req.URL.RawQuery == "app=app1" {
			body = `[{"service":"redisapi","instances":["myredisapi"]},
					 {"service":"mongodb", "instances":[]}]`
		}
		return &http.Response{
			Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := appInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoInfo(c *check.C) {
	c.Assert((&appInfo{}).Info(), check.NotNil)
}

func (s *S) TestAppInfoFlags(c *check.C) {
	command := appInfo{}
	flagset := command.Flags()
	flag := flagset.Lookup("app")
	flag.Value = nil
	c.Assert(flag, check.DeepEquals, appflag)
}

func (s *S) TestAppGrant(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Team "cobrateam" was added to the "games" app` + "\n"
	context := cmd.Context{
		Args:   []string{"cobrateam"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := appGrant{}
	command.Flags().Parse(true, []string{"--app", "games"})
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppGrantWithoutFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Team "cobrateam" was added to the "fights" app` + "\n"
	context := cmd.Context{
		Args:   []string{"cobrateam"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := &cmdtest.FakeGuesser{Name: "fights"}
	command := appGrant{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, nil)
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppGrantInfo(c *check.C) {
	c.Assert((&appGrant{}).Info(), check.NotNil)
}

func (s *S) TestAppRevoke(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Team "cobrateam" was removed from the "games" app` + "\n"
	context := cmd.Context{
		Args:   []string{"cobrateam"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := appRevoke{}
	command.Flags().Parse(true, []string{"--app", "games"})
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRevokeWithoutFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Team "cobrateam" was removed from the "fights" app` + "\n"
	context := cmd.Context{
		Args:   []string{"cobrateam"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := &cmdtest.FakeGuesser{Name: "fights"}
	command := appRevoke{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, nil)
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRevokeInfo(c *check.C) {
	c.Assert((&appRevoke{}).Info(), check.NotNil)
}

func (s *S) TestAppList(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-------------------------+-------------+
| Application | Units State Summary     | Address     |
+-------------+-------------------------+-------------+
| app1        | 1 of 1 units in-service | 10.10.10.10 |
+-------------+-------------------------+-------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListDisplayAppsInAlphabeticalOrder(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.11","name":"sapp","units":[{"ID":"sapp1/0","Status":"started"}]},{"ip":"10.10.10.10","name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-------------------------+-------------+
| Application | Units State Summary     | Address     |
+-------------+-------------------------+-------------+
| app1        | 1 of 1 units in-service | 10.10.10.10 |
+-------------+-------------------------+-------------+
| sapp        | 1 of 1 units in-service | 10.10.10.11 |
+-------------+-------------------------+-------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListUnitIsntAvailable(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","name":"app1","units":[{"ID":"app1/0","Status":"pending"}]}]`
	expected := `+-------------+-------------------------+-------------+
| Application | Units State Summary     | Address     |
+-------------+-------------------------+-------------+
| app1        | 0 of 1 units in-service | 10.10.10.10 |
+-------------+-------------------------+-------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListUnitWithoutName(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","name":"app1","units":[{"Name":"","Status":"pending"}]}]`
	expected := `+-------------+-------------------------+-------------+
| Application | Units State Summary     | Address     |
+-------------+-------------------------+-------------+
| app1        | 0 of 0 units in-service | 10.10.10.10 |
+-------------+-------------------------+-------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListCName(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","cname":["app1.tsuru.io"],"name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-------------------------+---------------+
| Application | Units State Summary     | Address       |
+-------------+-------------------------+---------------+
| app1        | 1 of 1 units in-service | app1.tsuru.io |
|             |                         | 10.10.10.10   |
+-------------+-------------------------+---------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := appList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListFiltering(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","cname":["app1.tsuru.io"],"name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-------------------------+---------------+
| Application | Units State Summary     | Address       |
+-------------+-------------------------+---------------+
| app1        | 1 of 1 units in-service | app1.tsuru.io |
|             |                         | 10.10.10.10   |
+-------------+-------------------------+---------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	var request *http.Request
	transport := cmdtest.ConditionalTransport{
		CondFunc: func(r *http.Request) bool {
			request = r
			return true
		},
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := appList{}
	command.Flags().Parse(true, []string{"-p", "python", "--locked", "--user", "glenda@tsuru.io", "-t", "tsuru", "--name", "myapp", "--pool", "pool"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
	queryString := url.Values(map[string][]string{
		"platform":  {"python"},
		"locked":    {"true"},
		"owner":     {"glenda@tsuru.io"},
		"teamowner": {"tsuru"},
		"name":      {"myapp"},
		"pool":      {"pool"},
	})
	c.Assert(request.URL.Query(), check.DeepEquals, queryString)
}

func (s *S) TestAppListFilteringMe(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","cname":["app1.tsuru.io"],"name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-------------------------+---------------+
| Application | Units State Summary     | Address       |
+-------------+-------------------------+---------------+
| app1        | 1 of 1 units in-service | app1.tsuru.io |
|             |                         | 10.10.10.10   |
+-------------+-------------------------+---------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	var request *http.Request
	transport := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				CondFunc: func(r *http.Request) bool {
					return true
				},
				Transport: cmdtest.Transport{Message: `{"Email":"gopher@tsuru.io","Teams":[]}`, Status: http.StatusOK},
			},
			{
				CondFunc: func(r *http.Request) bool {
					request = r
					return true
				},
				Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
			},
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := appList{}
	command.Flags().Parse(true, []string{"-u", "me"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
	queryString := url.Values(map[string][]string{"owner": {"gopher@tsuru.io"}})
	c.Assert(request.URL.Query(), check.DeepEquals, queryString)
}

func (s *S) TestAppListWithFlagQ(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","name":"app1","units":[{"ID":"app1/0","Status":"started"}]},{"ip":"10.10.10.11","name":"app2","units":[{"ID":"app2/0","Status":"started"}]},{"ip":"10.10.10.12","cname":["app3.tsuru.io"],"name":"app3","units":[{"ID":"app3/0","Status":"started"}]}]`
	expected := `app1
app2
app3
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	var request *http.Request
	transport := cmdtest.ConditionalTransport{
		CondFunc: func(r *http.Request) bool {
			request = r
			return true
		},
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := appList{}
	command.Flags().Parse(true, []string{"-q"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListInfo(c *check.C) {
	c.Assert((&appList{}).Info(), check.NotNil)
}

func (s *S) TestAppListIsACommand(c *check.C) {
	var _ cmd.Command = &appList{}
}

func (s *S) TestAppRestart(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "-- restarted --"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.URL.Query().Get("process"), check.Equals, "web")
			return req.URL.Path == "/apps/handful_of_nothing/restart" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := appRestart{}
	command.Flags().Parse(true, []string{"--app", "handful_of_nothing", "--process", "web"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestAppRestartWithoutTheFlag(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "-- restarted --"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.URL.Path == "/apps/motorbreath/restart" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "motorbreath"}
	command := appRestart{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, nil)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestAppRestartInfo(c *check.C) {
	c.Assert((&appRestart{}).Info(), check.NotNil)
}

func (s *S) TestAppRestartIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &appRestart{}
}

func (s *S) TestAddCName(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"death.evergrey.mycompany.com"},
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Restarted", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			var m map[string][]string
			err := json.NewDecoder(req.Body).Decode(&m)
			c.Assert(err, check.IsNil)
			c.Assert(m["cname"], check.DeepEquals, []string{"death.evergrey.mycompany.com"})
			return req.URL.Path == "/apps/death/cname" &&
				req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := cnameAdd{}
	command.Flags().Parse(true, []string{"-a", "death"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "cname successfully defined.\n")
}

func (s *S) TestAddCNameWithoutTheFlag(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"corey.evergrey.mycompany.com"},
	}
	fake := &cmdtest.FakeGuesser{Name: "corey"}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Restarted", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			var m map[string][]string
			err := json.NewDecoder(req.Body).Decode(&m)
			c.Assert(err, check.IsNil)
			c.Assert(m["cname"], check.DeepEquals, []string{"corey.evergrey.mycompany.com"})
			return req.URL.Path == "/apps/corey/cname" &&
				req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&cnameAdd{cmd.GuessingCommand{G: fake}}).Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "cname successfully defined.\n")
}

func (s *S) TestAddCNameFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"masterplan.evergrey.mycompany.com"},
	}
	trans := &cmdtest.Transport{Message: "Invalid cname", Status: http.StatusPreconditionFailed}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := cnameAdd{}
	command.Flags().Parse(true, []string{"-a", "masterplan"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "Invalid cname")
}

func (s *S) TestAddCNameInfo(c *check.C) {
	c.Assert((&cnameAdd{}).Info(), check.NotNil)
}

func (s *S) TestAddCNameIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &cnameAdd{}
}

func (s *S) TestRemoveCName(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Restarted", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.URL.Path == "/apps/death/cname" && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := cnameRemove{}
	command.Flags().Parse(true, []string{"--app", "death"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "cname successfully undefined.\n")
}

func (s *S) TestRemoveCNameWithoutTheFlag(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := &cmdtest.FakeGuesser{Name: "corey"}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Restarted", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.URL.Path == "/apps/corey/cname" && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&cnameRemove{cmd.GuessingCommand{G: fake}}).Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "cname successfully undefined.\n")
}

func (s *S) TestRemoveCNameInfo(c *check.C) {
	c.Assert((&cnameRemove{}).Info(), check.NotNil)
}

func (s *S) TestRemoveCNameIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &cnameRemove{}
}

func (s *S) TestAppStartInfo(c *check.C) {
	c.Assert((&appStart{}).Info(), check.NotNil)
}

func (s *S) TestSetTeamOwnerInfo(c *check.C) {
	c.Assert((&TeamOwnerSet{}).Info(), check.NotNil)
}

func (s *S) TestSetTeamOwner(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"test"},
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "app's owner team successfully changed.", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.URL.Path == "/apps/app-fake/team-owner" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := TeamOwnerSet{}
	command.Flags().Parse(true, []string{"--app", "app-fake"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "app's owner team successfully changed.\n")
}

func (s *S) TestAppStart(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Started", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.URL.Query().Get("process"), check.Equals, "worker")
			return req.URL.Path == "/apps/handful_of_nothing/start" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := appStart{}
	command.Flags().Parse(true, []string{"--app", "handful_of_nothing", "--process", "worker"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "Started")
}

func (s *S) TestAppStartWithoutTheFlag(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Started", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.URL.Path == "/apps/motorbreath/start" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "motorbreath"}
	command := appStart{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, nil)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "Started")
}

func (s *S) TestAppStartIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &appStart{}
}

func (s *S) TestUnitAvailable(c *check.C) {
	u := &unit{Status: "started"}
	c.Assert(u.Available(), check.Equals, true)
	u = &unit{Status: "down"}
	c.Assert(u.Available(), check.Equals, false)
}

func (s *S) TestAppStopInfo(c *check.C) {
	c.Assert((&appStop{}).Info(), check.NotNil)
}

func (s *S) TestAppStop(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Stopped", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.URL.Query().Get("process"), check.Equals, "worker")
			return req.URL.Path == "/apps/handful_of_nothing/stop" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := appStop{}
	command.Flags().Parse(true, []string{"--app", "handful_of_nothing", "--process", "worker"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "Stopped")
}

func (s *S) TestAppStopWithoutTheFlag(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Stopped", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			return req.URL.Path == "/apps/motorbreath/stop" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	fake := &cmdtest.FakeGuesser{Name: "motorbreath"}
	command := appStop{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, nil)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "Stopped")
}

func (s *S) TestAppStopIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &appStop{}
}

func (s *S) TestUnitAdd(c *check.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	context := cmd.Context{
		Args:   []string{"3"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "-- added unit --"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.FormValue("process"), check.Equals, "p1")
			c.Assert(req.FormValue("units"), check.Equals, "3")
			return req.URL.Path == "/apps/radio/units" && req.Method == "PUT"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := unitAdd{}
	command.Flags().Parse(true, []string{"-a", "radio", "-p", "p1"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestUnitAddFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"3"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	msg := io.SimpleJsonMessage{Error: "errored msg"}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: string(result), Status: 200}}, nil, manager)
	command := unitAdd{}
	command.Flags().Parse(true, []string{"-a", "radio"})
	err = command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "errored msg")
}

func (s *S) TestUnitAddInfo(c *check.C) {
	c.Assert((&unitAdd{}).Info(), check.NotNil)
}

func (s *S) TestUnitAddIsFlaggedACommand(c *check.C) {
	var _ cmd.FlaggedCommand = &unitAdd{}
}

func (s *S) TestUnitRemove(c *check.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	context := cmd.Context{
		Args:   []string{"2"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "-- removed unit --"
	msg := io.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.FormValue("process"), check.Equals, "web1")
			c.Assert(req.FormValue("units"), check.Equals, "2")
			return req.URL.Path == "/apps/vapor/units" && req.Method == "DELETE"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := unitRemove{}
	command.Flags().Parse(true, []string{"-a", "vapor", "-p", "web1"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "-- removed unit --")
}

func (s *S) TestUnitRemoveFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"1"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{
		Transport: &cmdtest.Transport{Message: "Failed to remove.", Status: 500},
	}, nil, manager)
	command := unitRemove{}
	command.Flags().Parse(true, []string{"-a", "vapor"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "Failed to remove.")
}

func (s *S) TestUnitRemoveInfo(c *check.C) {
	c.Assert((&unitRemove{}).Info(), check.NotNil)
}

func (s *S) TestUnitRemoveIsACommand(c *check.C) {
	var _ cmd.Command = &unitRemove{}
}

func (s *S) TestAppChangePoolRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	var called bool
	context := cmd.Context{
		Args:   []string{"new-pool"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			b, err := ioutil.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			c.Assert(string(b), check.Equals, "new-pool")
			return req.URL.Path == "/apps/radio/pool" && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := appPoolChange{}
	command.Flags().Parse(true, []string{"-a", "radio"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := "Pool successfully changed!\n"
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppChangePoolInfo(c *check.C) {
	c.Assert((&appPoolChange{}).Info(), check.NotNil)
}
