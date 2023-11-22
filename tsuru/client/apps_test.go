// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	tsuruIo "github.com/tsuru/tsuru/io"
	check "gopkg.in/check.v1"
)

var appflag = &gnuflag.Flag{
	Name:     "app",
	Usage:    "The name of the app.",
	Value:    nil,
	DefValue: "",
}

func (s *S) TestAppCreateInfo(c *check.C) {
	c.Assert((&AppCreate{}).Info(), check.NotNil)
}

func (s *S) TestAppCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			name := r.FormValue("name") == "ble"
			platform := r.FormValue("platform") == "django"
			teamOwner := r.FormValue("teamOwner") == ""
			plan := r.FormValue("plan") == ""
			pool := r.FormValue("pool") == ""
			description := r.FormValue("description") == ""
			r.ParseForm()
			tags := r.Form["tag"] == nil
			router := r.FormValue("router") == ""
			method := r.Method == "POST"
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(r.URL.Path, "/apps")
			return method && url && name && platform && teamOwner && plan && pool && description && tags && contentType && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreateEmptyPlatform(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			name := r.FormValue("name") == "ble"
			platform := r.FormValue("platform") == ""
			teamOwner := r.FormValue("teamOwner") == ""
			plan := r.FormValue("plan") == ""
			pool := r.FormValue("pool") == ""
			description := r.FormValue("description") == ""
			r.ParseForm()
			tags := r.Form["tag"] == nil
			router := r.FormValue("router") == ""
			method := r.Method == "POST"
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(r.URL.Path, "/apps")
			return method && url && name && platform && teamOwner && plan && pool && description && tags && contentType && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreateTeamOwner(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			name := r.FormValue("name") == "ble"
			platform := r.FormValue("platform") == "django"
			teamOwner := r.FormValue("teamOwner") == "team"
			plan := r.FormValue("plan") == ""
			pool := r.FormValue("pool") == ""
			description := r.FormValue("description") == ""
			r.ParseForm()
			tags := r.Form["tag"] == nil
			router := r.FormValue("router") == ""
			method := r.Method == "POST"
			url := strings.HasSuffix(r.URL.Path, "/apps")
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return method && url && name && platform && teamOwner && plan && pool && description && tags && contentType && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppCreate{}
	command.Flags().Parse(true, []string{"-t", "team"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreatePlan(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			name := r.FormValue("name") == "ble"
			platform := r.FormValue("platform") == "django"
			teamOwner := r.FormValue("teamOwner") == ""
			plan := r.FormValue("plan") == "myplan"
			pool := r.FormValue("pool") == ""
			router := r.FormValue("router") == ""
			description := r.FormValue("description") == ""
			r.ParseForm()
			tags := r.Form["tag"] == nil
			method := r.Method == "POST"
			url := strings.HasSuffix(r.URL.Path, "/apps")
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return method && url && name && platform && teamOwner && plan && pool && description && tags && contentType && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppCreate{}
	command.Flags().Parse(true, []string{"-p", "myplan"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreatePool(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			name := r.FormValue("name") == "ble"
			platform := r.FormValue("platform") == "django"
			teamowner := r.FormValue("teamowner") == ""
			plan := r.FormValue("plan") == ""
			pool := r.FormValue("pool") == "mypool"
			router := r.FormValue("router") == ""
			description := r.FormValue("description") == ""
			r.ParseForm()
			tags := r.Form["tag"] == nil
			method := r.Method == "POST"
			url := strings.HasSuffix(r.URL.Path, "/apps")
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return method && url && name && platform && teamowner && plan && pool && description && tags && contentType && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppCreate{}
	command.Flags().Parse(true, []string{"-o", "mypool"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreateRouterOpts(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			name := r.FormValue("name") == "ble"
			platform := r.FormValue("platform") == "django"
			teamowner := r.FormValue("teamowner") == ""
			plan := r.FormValue("plan") == ""
			pool := r.FormValue("pool") == ""
			description := r.FormValue("description") == ""
			r.ParseForm()
			tags := r.Form["tag"] == nil
			router := r.FormValue("router") == ""
			c.Assert(r.FormValue("routeropts.a"), check.Equals, "1")
			c.Assert(r.FormValue("routeropts.b"), check.Equals, "2")
			method := r.Method == "POST"
			url := strings.HasSuffix(r.URL.Path, "/apps")
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return method && url && name && platform && teamowner && plan && pool && description && tags && contentType && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppCreate{}
	command.Flags().Parse(true, []string{"--router-opts", "a=1", "--router-opts", "b=2"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreateNoRepository(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success"}`
	expected := `App "ble" has been created!
Use app info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			name := r.FormValue("name") == "ble"
			platform := r.FormValue("platform") == "django"
			teamowner := r.FormValue("teamowner") == ""
			plan := r.FormValue("plan") == ""
			pool := r.FormValue("pool") == ""
			router := r.FormValue("router") == ""
			description := r.FormValue("description") == ""
			r.ParseForm()
			tags := r.Form["tag"] == nil
			method := r.Method == "POST"
			url := strings.HasSuffix(r.URL.Path, "/apps")
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return method && url && name && platform && teamowner && plan && pool && description && tags && contentType && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppCreate{}
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
	command := AppCreate{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestAppCreateWithTags(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			r.ParseForm()
			name := r.FormValue("name") == "ble"
			platform := r.FormValue("platform") == "django"
			teamOwner := r.FormValue("teamOwner") == ""
			plan := r.FormValue("plan") == ""
			pool := r.FormValue("pool") == ""
			description := r.FormValue("description") == ""
			tags := len(r.Form["tag"]) == 2 && r.Form["tag"][0] == "tag1" && r.Form["tag"][1] == "tag2"
			router := r.FormValue("router") == ""
			method := r.Method == "POST"
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(r.URL.Path, "/apps")
			return method && url && name && platform && teamOwner && plan && pool && description && tags && contentType && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppCreate{}
	command.Flags().Parse(true, []string{"--tag", "tag1", "--tag", "tag2"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreateWithEmptyTag(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"status":"success", "repository_url":"git@tsuru.plataformas.glb.com:ble.git"}`
	expected := `App "ble" has been created!
Use app info to check the status of the app and its units.` + "\n"
	context := cmd.Context{
		Args:   []string{"ble", "django"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			r.ParseForm()
			name := r.FormValue("name") == "ble"
			platform := r.FormValue("platform") == "django"
			teamOwner := r.FormValue("teamOwner") == ""
			plan := r.FormValue("plan") == ""
			pool := r.FormValue("pool") == ""
			description := r.FormValue("description") == ""
			tags := len(r.Form["tag"]) == 1 && r.Form["tag"][0] == ""
			router := r.FormValue("router") == ""
			method := r.Method == "POST"
			contentType := r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			url := strings.HasSuffix(r.URL.Path, "/apps")
			return method && url && name && platform && teamOwner && plan && pool && description && tags && contentType && router
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := AppCreate{}
	command.Flags().Parse(true, []string{"--tag", ""})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppCreateFlags(c *check.C) {
	command := AppCreate{}
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
	flagset.Parse(true, []string{"-r", "router"})
	usage = "The router used by the app"
	router := flagset.Lookup("router")
	c.Check(router, check.NotNil)
	c.Check(router.Name, check.Equals, "router")
	c.Check(router.Usage, check.Equals, usage)
	c.Check(router.Value.String(), check.Equals, "router")
	c.Check(router.DefValue, check.Equals, "")
	router = flagset.Lookup("r")
	c.Check(router, check.NotNil)
	c.Check(router.Name, check.Equals, "r")
	c.Check(router.Usage, check.Equals, usage)
	c.Check(router.Value.String(), check.Equals, "router")
	c.Check(router.DefValue, check.Equals, "")
	flagset.Parse(true, []string{"--tag", "tag1", "--tag", "tag2"})
	usage = "App tag"
	tag := flagset.Lookup("tag")
	c.Check(tag, check.NotNil)
	c.Check(tag.Name, check.Equals, "tag")
	c.Check(tag.Usage, check.Equals, usage)
	c.Check(tag.Value.String(), check.Equals, "[\"tag1\",\"tag2\"]")
	c.Check(tag.DefValue, check.Equals, "[]")
	tag = flagset.Lookup("g")
	c.Check(tag, check.NotNil)
	c.Check(tag.Name, check.Equals, "g")
	c.Check(tag.Usage, check.Equals, usage)
	c.Check(tag.Value.String(), check.Equals, "[\"tag1\",\"tag2\"]")
	c.Check(tag.DefValue, check.Equals, "[]")
	flagset.Parse(true, []string{"--router-opts", "opt1=val1", "--router-opts", "opt2=val2"})
	routerOpts := flagset.Lookup("router-opts")
	c.Check(routerOpts, check.NotNil)
	c.Check(routerOpts.Name, check.Equals, "router-opts")
	c.Check(routerOpts.Usage, check.Equals, "Router options")
	c.Check(routerOpts.Value.String(), check.Equals, "{\"opt1\":\"val1\",\"opt2\":\"val2\"}")
	c.Check(routerOpts.DefValue, check.Equals, "{}")
}

func (s *S) TestAppUpdateInfo(c *check.C) {
	c.Assert((&AppUpdate{}).Info(), check.NotNil)
}

func (s *S) TestAppUpdate(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := fmt.Sprintf("App %q has been updated!\n", "ble")
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/ble")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			c.Assert(err, check.IsNil)
			c.Assert(result, check.DeepEquals, map[string]interface{}{
				"description":  "description of my app",
				"platform":     "python",
				"tags":         []interface{}{"tag 1", "tag 2"},
				"planoverride": map[string]interface{}{},
				"metadata":     map[string]interface{}{},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppUpdate{}
	err := command.Flags().Parse(true, []string{"-d", "description of my app", "-a", "ble", "-l", "python", "-g", "tag 1", "-g", "tag 2"})
	c.Assert(err, check.IsNil)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppUpdateImageReset(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := fmt.Sprintf("App %q has been updated!\n", "img")
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/img")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			c.Assert(err, check.IsNil)
			c.Assert(result, check.DeepEquals, map[string]interface{}{
				"imageReset":   true,
				"planoverride": map[string]interface{}{},
				"metadata":     map[string]interface{}{},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppUpdate{}
	command.Flags().Parse(true, []string{"-a", "img", "-i"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppUpdateWithoutTags(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := fmt.Sprintf("App %q has been updated!\n", "ble")
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/ble")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			c.Assert(err, check.IsNil)
			c.Assert(result, check.DeepEquals, map[string]interface{}{
				"description":  "description",
				"planoverride": map[string]interface{}{},
				"metadata":     map[string]interface{}{},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppUpdate{}
	command.Flags().Parse(true, []string{"-d", "description", "-a", "ble"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppUpdateWithEmptyTag(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := fmt.Sprintf("App %q has been updated!\n", "ble")
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/ble")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			c.Assert(err, check.IsNil)
			c.Assert(result, check.DeepEquals, map[string]interface{}{
				"description":  "description",
				"tags":         []interface{}{""},
				"planoverride": map[string]interface{}{},
				"metadata":     map[string]interface{}{},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppUpdate{}
	command.Flags().Parse(true, []string{"-d", "description", "-a", "ble", "-g", ""})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppUpdateWithCPUAndMemory(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := fmt.Sprintf("App %q has been updated!\n", "ble")
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/ble")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			c.Assert(err, check.IsNil)
			c.Assert(result, check.DeepEquals, map[string]interface{}{
				"planoverride": map[string]interface{}{
					"cpumilli": float64(100),
					"memory":   float64(1 * 1024 * 1024 * 1024),
				},
				"metadata": map[string]interface{}{},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppUpdate{}
	command.Flags().Parse(true, []string{"-a", "ble", "--cpu", "100m", "--memory", "1Gi"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppUpdateWithCPUBurst(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := fmt.Sprintf("App %q has been updated!\n", "ble")
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/ble")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			c.Assert(err, check.IsNil)
			c.Assert(result, check.DeepEquals, map[string]interface{}{
				"planoverride": map[string]interface{}{
					"cpuBurst": float64(1.3),
				},
				"metadata": map[string]interface{}{},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppUpdate{}
	command.Flags().Parse(true, []string{"-a", "ble", "--cpu-burst-factor", "1.3"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppUpdateWithInvalidCPUBurst(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/ble")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			c.Assert(err, check.IsNil)
			c.Assert(result, check.DeepEquals, map[string]interface{}{
				"planoverride": map[string]interface{}{
					"cpuBurst": float64(1.3),
				},
				"metadata": map[string]interface{}{},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppUpdate{}
	command.Flags().Parse(true, []string{"-a", "ble", "--cpu-burst-factor", "0.5"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "Invalid factor, please use a value greater equal 1")
}

func (s *S) TestAppUpdateWithoutArgs(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Please use the -a/--app flag to specify which app you want to update."
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/secret")
			method := req.Method == "PUT"
			description := req.FormValue("description") == "description of my app"
			req.ParseForm()
			tags := req.Form["tag"] == nil
			return url && method && description && tags
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppUpdate{}
	command.Flags().Parse(true, []string{"-d", "description of my app"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, expected)
}

func (s *S) TestAppUpdateFlags(c *check.C) {
	command := AppUpdate{}
	flagset := command.Flags()
	c.Assert(flagset, check.NotNil)
	flagset.Parse(true, []string{"-d", "description of my app"})
	appdescription := flagset.Lookup("description")
	c.Check(appdescription, check.NotNil)
	c.Check(appdescription.Name, check.Equals, "description")
	c.Check(appdescription.Value.String(), check.Equals, "description of my app")
	c.Check(appdescription.DefValue, check.Equals, "")
	sdescription := flagset.Lookup("d")
	c.Check(sdescription, check.NotNil)
	c.Check(sdescription.Name, check.Equals, "d")
	c.Check(sdescription.Value.String(), check.Equals, "description of my app")
	c.Check(sdescription.DefValue, check.Equals, "")
	flagset.Parse(true, []string{"-p", "my plan"})
	plan := flagset.Lookup("plan")
	c.Check(plan, check.NotNil)
	c.Check(plan.Name, check.Equals, "plan")
	c.Check(plan.Value.String(), check.Equals, "my plan")
	c.Check(plan.DefValue, check.Equals, "")
	splan := flagset.Lookup("p")
	c.Check(splan, check.NotNil)
	c.Check(splan.Name, check.Equals, "p")
	c.Check(splan.Value.String(), check.Equals, "my plan")
	c.Check(splan.DefValue, check.Equals, "")
	flagset.Parse(true, []string{"-o", "myPool"})
	pool := flagset.Lookup("pool")
	c.Check(pool, check.NotNil)
	c.Check(pool.Name, check.Equals, "pool")
	c.Check(pool.Value.String(), check.Equals, "myPool")
	c.Check(pool.DefValue, check.Equals, "")
	spool := flagset.Lookup("o")
	c.Check(spool, check.NotNil)
	c.Check(spool.Name, check.Equals, "o")
	c.Check(spool.Value.String(), check.Equals, "myPool")
	c.Check(spool.DefValue, check.Equals, "")
	flagset.Parse(true, []string{"-t", "newowner"})
	teamOwner := flagset.Lookup("team-owner")
	c.Check(teamOwner, check.NotNil)
	c.Check(teamOwner.Name, check.Equals, "team-owner")
	c.Check(teamOwner.Value.String(), check.Equals, "newowner")
	c.Check(teamOwner.DefValue, check.Equals, "")
	steamOwner := flagset.Lookup("t")
	c.Check(steamOwner, check.NotNil)
	c.Check(steamOwner.Name, check.Equals, "t")
	c.Check(steamOwner.Value.String(), check.Equals, "newowner")
	c.Check(steamOwner.DefValue, check.Equals, "")
	flagset.Parse(true, []string{"-g", "tag"})
	tag := flagset.Lookup("tag")
	c.Check(tag, check.NotNil)
	c.Check(tag.Name, check.Equals, "tag")
	c.Check(tag.Value.String(), check.Equals, "[\"tag\"]")
	c.Check(tag.DefValue, check.Equals, "[]")
	tag = flagset.Lookup("g")
	c.Check(tag, check.NotNil)
	c.Check(tag.Name, check.Equals, "g")
	c.Check(tag.Value.String(), check.Equals, "[\"tag\"]")
	c.Check(tag.DefValue, check.Equals, "[]")

	flagset.Parse(true, []string{"--no-restart"})
	noRestart := flagset.Lookup("no-restart")
	c.Check(noRestart, check.NotNil)
	c.Check(noRestart.Name, check.Equals, "no-restart")
	c.Check(noRestart.Value.String(), check.Equals, "true")
	c.Check(noRestart.DefValue, check.Equals, "false")
}

func (s *S) TestAppRemove(c *check.C) {
	var stdout, stderr bytes.Buffer
	expectedOut := "-- removed --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
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
	command := AppRemove{}
	command.Flags().Parse(true, []string{"-a", "ble"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected+expectedOut)
}

func (s *S) TestAppRemoveWithoutAsking(c *check.C) {
	var stdout, stderr bytes.Buffer
	expectedOut := "-- removed --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	context := cmd.Context{
		Args:   []string{"ble"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: string(result), Status: http.StatusOK}}, nil, manager)
	command := AppRemove{}
	command.Flags().Parse(true, []string{"-a", "ble", "-y"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestAppRemoveFlags(c *check.C) {
	command := AppRemove{}
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
	command := AppRemove{}
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
	command := AppRemove{}
	command.Flags().Parse(true, []string{"--app", "ble"})
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRemoveInfo(c *check.C) {
	c.Assert((&AppRemove{}).Info(), check.NotNil)
}

func (s *S) TestAppInfo(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","Address":{"Host": "10.8.7.6:3333"}}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started","Address":{"Host": "10.8.7.6:3323"}}, {"Ip":"","ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb"}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+----------+------+
| Name   | Status  | Host     | Port |
+--------+---------+----------+------+
| app1/2 | pending |          |      |
| app1/0 | started | 10.8.7.6 | 3333 |
| app1/1 | started | 10.8.7.6 | 3323 |
+--------+---------+----------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoSimplified(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","pool": "dev-a", "provisioner": "kubernetes", "cluster": "mycluster", "teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","ProcessName": "web","Address":{"Host": "10.8.7.6:3333"}, "ready": true, "routable": true}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started","ProcessName": "web","Address":{"Host": "10.8.7.6:3323"}, "ready": true, "routable": true}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb", "plan":{"name": "test",  "memory": 536870912, "cpumilli": 100, "default": false}}`
	expected := `Application: app1
Created by: myapp_owner
Platform: php
Plan: test
Pool: dev-a (kubernetes | cluster: mycluster)
Router: planb
Teams: myteam (owner), tsuruteam, crane
Cluster External Addresses: myapp.tsuru.io
Units: 2
+---------+-------+----------+---------------+------------+
| Process | Ready | Restarts | Avg CPU (abs) | Avg Memory |
+---------+-------+----------+---------------+------------+
| web     | 2/2   | 0        | 0%            | 0Mi        |
+---------+-------+----------+---------------+------------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1", "-s"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoKubernetes(c *check.C) {
	var stdout, stderr bytes.Buffer
	t0 := time.Now().UTC().Format(time.RFC3339)
	t1 := time.Now().Add(time.Hour * -1).UTC().Format(time.RFC3339)
	t2 := time.Now().Add(time.Hour * -1 * 24 * 30).UTC().Format(time.RFC3339)

	result := fmt.Sprintf(`{
		"name":"app1",
		"teamowner":"myteam",
		"cname":[""],"ip":"myapp.tsuru.io",
		"provisioner": "kubernetes",
		"platform":"php",
		"repository":"git@git.com:php.git",
		"state":"dead",
		"cluster": "kube-cluster-dev",
		"pool": "dev-a",
		"units":[
			{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","Address":{"Host": "10.8.7.6:3333"}, "ready": true, "restarts": 10, "createdAt": "%s"},
			{"Ip":"9.9.9.9","ID":"app1/1","Status":"started","Address":{"Host": "10.8.7.6:3323"}, "ready": true, "restarts": 0, "createdAt": "%s"},
			{"Ip":"","ID":"app1/2","Status":"pending", "ready": false, "createdAt": "%s"}
		],
		"unitsMetrics": [
			{
				"ID": "app1/0",
				"CPU": "900m",
				"Memory": "2000000Ki"
			},
			{
				"ID": "app1/1",
				"CPU": "800m",
				"Memory": "3000000Ki"
			},
			{
				"ID": "app1/2",
				"CPU": "80m",
				"Memory": "300Ki"
			}
		],
		"teams": ["tsuruteam","crane"],
		"owner": "myapp_owner",
		"deploys": 7,
		"router": "planb"
	}`, t0, t1, t2)
	expected := `Application: app1
Platform: php
Provisioner: kubernetes
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Cluster: kube-cluster-dev
Pool: dev-a
Quota: 0/0 units

Units: 3
+--------+----------+---------+----------+-----+-----+--------+
| Name   | Host     | Status  | Restarts | Age | CPU | Memory |
+--------+----------+---------+----------+-----+-----+--------+
| app1/2 |          | pending |          | 30d | 8%  | 0Mi    |
| app1/0 | 10.8.7.6 | ready   | 10       | 0s  | 90% | 1953Mi |
| app1/1 | 10.8.7.6 | ready   | 0        | 60m | 80% | 2929Mi |
+--------+----------+---------+----------+-----+-----+--------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoMultipleAddresses(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started","Address":{"Host": "10.8.7.6:3333"},"Addresses":[{"Host": "10.8.7.6:3333"}, {"Host": "10.8.7.6:4444"}]}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started","Address":{"Host": "10.8.7.6:3323"}}, {"Ip":"","ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb"}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+----------+------------+
| Name   | Status  | Host     | Port       |
+--------+---------+----------+------------+
| app1/2 | pending |          |            |
| app1/0 | started | 10.8.7.6 | 3333, 4444 |
| app1/1 | started | 10.8.7.6 | 3323       |
+--------+---------+----------+------------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoMultipleRouters(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `
{
	"name": "app1",
	"teamowner": "myteam",
	"cname": [
		"cname1"
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
			"Address": {
				"Host": "10.8.7.6:3333"
			}
		},
		{
			"Ip": "9.9.9.9",
			"ID": "app1/1",
			"Status": "started",
			"Address": {
				"Host": "10.8.7.6:3323"
			}
		},
		{
			"Ip": "",
			"ID": "app1/2",
			"Status": "pending"
		}
	],
	"teams": [
		"tsuruteam",
		"crane"
	],
	"owner": "myapp_owner",
	"deploys": 7,
	"router": "planb",
	"routers": [
		{"name": "r1", "type": "r", "opts": {"a": "b", "x": "y"}, "address": "addr1"},
		{"name": "r2", "addresses": ["addr2", "addr9"], "status": "ready"},
		{"name": "r3", "type": "r3", "address": "addr3", "status": "not ready", "status-detail": "something happening"}
	]
}`
	expected := `Application: app1
Platform: php
Teams: myteam (owner), tsuruteam, crane
External Addresses: cname1 (cname), addr1, addr2, addr9, addr3
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+----------+------+
| Name   | Status  | Host     | Port |
+--------+---------+----------+------+
| app1/2 | pending |          |      |
| app1/0 | started | 10.8.7.6 | 3333 |
| app1/1 | started | 10.8.7.6 | 3323 |
+--------+---------+----------+------+

Routers:
+------+------+-----------+--------------------------------+
| Name | Opts | Addresses | Status                         |
+------+------+-----------+--------------------------------+
| r1   | a: b | addr1     |                                |
|      | x: y |           |                                |
+------+------+-----------+--------------------------------+
| r2   |      | addr2     | ready                          |
|      |      | addr9     |                                |
+------+------+-----------+--------------------------------+
| r3   |      | addr3     | not ready: something happening |
+------+------+-----------+--------------------------------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithDescription(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"ID":"app1/0","Status":"started"}, {"ID":"app1/1","Status":"started"}, {"ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "description": "My app", "router": "planb"}`
	expected := `Application: app1
Description: My app
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithTags(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "tags": ["tag 1", "tag 2", "tag 3"], "router": "planb"}`
	expected := `Application: app1
Tags: tag 1, tag 2, tag 3
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+-------------+------+
| Name   | Status  | Host        | Port |
+--------+---------+-------------+------+
| app1/2 | pending |             |      |
| app1/0 | started | 10.10.10.10 |      |
| app1/1 | started | 9.9.9.9     |      |
+--------+---------+-------------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithRouterOpts(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"ID":"app1/0","Status":"started"}, {"ID":"app1/1","Status":"started"}, {"ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "routeropts": {"opt1": "val1", "opt2": "val2"}, "router": "planb"}`
	expected := `Application: app1
Platform: php
Router: planb (opt1=val1, opt2=val2)
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
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
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 3/40 units

Units: 3
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		body := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"ID":"app1/0","Status":"started"}, {"ID":"app1/1","Status":"started"}, {"ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb", "quota": {"inUse": 3, "limit": 40}}`
		return &http.Response{
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoLock(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"ID":"app1/0","Status":"started"}, {"ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "lock": {"locked": true, "owner": "admin@example.com", "reason": "DELETE /apps/rbsample/units", "acquiredate": "2012-04-01T10:32:00Z"}, "router": "planb"}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Lock:
 Acquired in: %s
 Owner: admin@example.com
 Running: DELETE /apps/rbsample/units
Quota: 0/0 units

Units: 3
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

`
	expected = fmt.Sprintf(expected, time.Date(2012, time.April, 1, 10, 32, 0, 0, time.UTC))
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
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
  "deploys": 7,
  "router": "planb"
}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units [process web]: 1
+--------+---------+-------------+------+
| Name   | Status  | Host        | Port |
+--------+---------+-------------+------+
| app1/0 | started | 10.10.10.10 |      |
+--------+---------+-------------+------+

Units [process worker]: 2
+--------+---------+---------+------+
| Name   | Status  | Host    | Port |
+--------+---------+---------+------+
| app1/2 | pending |         |      |
| app1/1 | started | 9.9.9.9 |      |
+--------+---------+---------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoManyVersions(c *check.C) {
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
      "ID": "app1/0",
      "Status": "started",
	  "ProcessName": "web",
	  "Version": 1,
	  "Routable": false
    },
    {
      "ID": "app1/1",
      "Status": "started",
	  "ProcessName": "worker",
	  "Version": 1,
	  "Routable": false
    },
    {
      "ID": "app1/2",
      "Status": "pending",
	  "ProcessName": "worker",
	  "Version": 1,
	  "Routable": false
	},
	{
      "ID": "app1/3",
      "Status": "started",
	  "ProcessName": "web",
	  "Version": 2,
	  "Routable": true
    },
    {
      "ID": "app1/4",
      "Status": "started",
	  "ProcessName": "worker",
	  "Version": 2,
	  "Routable": true
    }
  ],
  "teams": [
    "tsuruteam",
    "crane"
  ],
  "owner": "myapp_owner",
  "deploys": 7,
  "router": "planb"
}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units [process web] [version 1]: 1
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
+--------+---------+------+------+

Units [process worker] [version 1]: 2
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

Units [process web] [version 2] [routable]: 1
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/3 | started |      |      |
+--------+---------+------+------+

Units [process worker] [version 2] [routable]: 1
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/4 | started |      |      |
+--------+---------+------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithAutoScale(c *check.C) {
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
      "ID": "app1/0",
      "Status": "started",
      "ProcessName": "web"
    },
    {
      "ID": "app1/1",
      "Status": "started",
      "ProcessName": "worker"
    }
  ],
  "teams": [
    "tsuruteam",
    "crane"
  ],
  "owner": "myapp_owner",
  "deploys": 7,
  "router": "planb",
  "autoscale": [
    {
      "process":"web",
      "minUnits":1,
      "maxUnits":10,
      "averageCPU":"500m",
      "version":10
    },
    {
      "process":"worker",
      "minUnits":2,
      "maxUnits":5,
      "averageCPU":"2",
      "version":10
    }
  ]
}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units [process web]: 1
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
+--------+---------+------+------+

Units [process worker]: 1
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/1 | started |      |      |
+--------+---------+------+------+

Auto Scale:
+--------------+-----+-----+------------+
| Process      | Min | Max | Target CPU |
+--------------+-----+-----+------------+
| web (v10)    | 1   | 10  | 50%        |
| worker (v10) | 2   | 5   | 200%       |
+--------------+-----+-----+------------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoNoUnits(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","ip":"app1.tsuru.io","teamowner":"myteam","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb"}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: app1.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoEmptyUnit(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"x","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"Name":"","Status":""}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb"}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: x (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithoutArgs(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"secret","teamowner":"myteam","ip":"secret.tsuru.io","platform":"ruby","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"","ID":"secret/0","Status":"started"}, {"Ip":"","ID":"secret/1","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb", "quota": {"inUse": 0, "limit": -1}}`
	expected := `Application: secret
Platform: ruby
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: secret.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/unlimited

Units: 2
+----------+---------+------+------+
| Name     | Status  | Host | Port |
+----------+---------+------+------+
| secret/0 | started |      |      |
| secret/1 | pending |      |      |
+----------+---------+------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/apps/secret") && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"-a", "secret"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoCName(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","ip":"myapp.tsuru.io","cname":["yourapp.tsuru.io"],"platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"ID":"app1/0","Status":"started"}, {"ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb"}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: yourapp.tsuru.io (cname), myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithServices(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

Service instances: 1
+----------+-----------------+
| Service  | Instance (Plan) |
+----------+-----------------+
| redisapi | myredisapi      |
+----------+-----------------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		body := `{"name":"app1","teamowner":"myteam","ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"ID":"app1/0","Status":"started"}, {"ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb", "serviceInstanceBinds": [{"service": "redisapi", "instance": "myredisapi"}]}`
		return &http.Response{
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithServicesTwoService(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+-------------+------+
| Name   | Status  | Host        | Port |
+--------+---------+-------------+------+
| app1/2 | pending |             |      |
| app1/0 | started | 10.10.10.10 |      |
| app1/1 | started | 9.9.9.9     |      |
+--------+---------+-------------+------+

Service instances: 2
+----------+-----------------+
| Service  | Instance (Plan) |
+----------+-----------------+
| mongodb  | mongoapi        |
| redisapi | myredisapi      |
+----------+-----------------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		body := `{"name":"app1","teamowner":"myteam","ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb", "serviceInstanceBinds": [{"service": "redisapi", "instance": "myredisapi"}, {"service": "mongodb", "instance": "mongoapi"}]}`
		return &http.Response{
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithPlan(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"app1","teamowner":"myteam","cname":[""],"ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead", "units":[{"ID":"app1/0","Status":"started"}, {"ID":"app1/1","Status":"started"}, {"ID":"app1/2","Status":"pending"}],"teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "plan":{"name": "test",  "memory": 536870912, "cpumilli": 100, "default": false}, "router": "planb"}`
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

App Plan:
+------+-----+--------+
| Name | CPU | Memory |
+------+-----+--------+
| test | 10% | 512Mi  |
+------+-----+--------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithServicesAndPlan(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

Service instances: 1
+----------+-----------------+
| Service  | Instance (Plan) |
+----------+-----------------+
| redisapi | myredisapi      |
+----------+-----------------+

App Plan:
+------+-----+--------+
| Name | CPU | Memory |
+------+-----+--------+
| test | 10% | 512Mi  |
+------+-----+--------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		body := `{"name":"app1","teamowner":"myteam","ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"ID":"app1/0","Status":"started"}, {"ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7,"plan":{"name": "test",  "memory": 536870912, "cpumilli": 100, "default": false}, "router": "planb", "serviceInstanceBinds": [{"service": "redisapi", "instance": "myredisapi"}]}`
		return &http.Response{
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithServicesAndPlanAssociated(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------+---------+------+------+
| Name   | Status  | Host | Port |
+--------+---------+------+------+
| app1/0 | started |      |      |
| app1/1 | started |      |      |
| app1/2 | pending |      |      |
+--------+---------+------+------+

Service instances: 1
+----------+-------------------+
| Service  | Instance (Plan)   |
+----------+-------------------+
| redisapi | myredisapi (test) |
+----------+-------------------+

App Plan:
+------+-----+--------+
| Name | CPU | Memory |
+------+-----+--------+
| test | 10% | 512Mi  |
+------+-----+--------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		body := `{"name":"app1","teamowner":"myteam","ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"ID":"app1/0","Status":"started"}, {"ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7,"plan":{"name": "test",  "memory": 536870912, "cpumilli": 100, "default": false}, "router": "planb", "serviceInstanceBinds": [{"service": "redisapi", "instance": "myredisapi", "plan": "test"}]}`
		return &http.Response{
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoShortensHexIDs(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: app1.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 3
+--------------------+---------+------+------+
| Name               | Status  | Host | Port |
+--------------------+---------+------+------+
| abcea3             | started |      |      |
| abcea389cbae       | started |      |      |
| my_long_non_hex_id | started |      |      |
+--------------------+---------+------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	infoData := `{
    "name": "app1",
    "teamowner": "myteam",
    "ip": "app1.tsuru.io",
    "platform": "php",
    "repository": "git@git.com:php.git",
    "units": [
        {
            "ID": "abcea389cbaebce89abc9a",
            "Status": "started"
        },
        {
            "ID": "abcea3",
            "Status": "started"
        },
        {
            "ID": "my_long_non_hex_id",
            "Status": "started"
        }
    ],
    "Teams": [
        "tsuruteam",
        "crane"
    ],
    "owner": "myapp_owner",
    "deploys": 7,
    "router": "planb"
}`
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{
			Body:       io.NopCloser(bytes.NewBufferString(infoData)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithInternalAddresses(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `{"name":"powerapp","teamowner":"powerteam","cname":[""],"ip":"monster.tsuru.io","platform":"assembly","repository":"git@git.com:app.git","state":"dead", "units":[{"Ip":"9.9.9.9","ID":"app1/1","Status":"started","Address":{"Host": "10.8.7.6:3323"}}],"teams":["tsuruzers"], "owner": "myapp_owner", "deploys": 7, "router": "", "internalAddresses":[{"domain":"test.cluster.com","port":80,"protocol":"TCP","process": "web","version":"2"}, {"domain":"test.cluster.com","port":443,"protocol":"TCP","process":"jobs","version":"3"}]}`
	expected := `Application: powerapp
Platform: assembly
Router:
Teams: powerteam (owner), tsuruzers
External Addresses: monster.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 0/0 units

Units: 1
+--------+---------+----------+------+
| Name   | Status  | Host     | Port |
+--------+---------+----------+------+
| app1/1 | started | 10.8.7.6 | 3323 |
+--------+---------+----------+------+

Cluster internal addresses:
+------------------+---------+---------+---------+
| Domain           | Port    | Process | Version |
+------------------+---------+---------+---------+
| test.cluster.com | 80/TCP  | web     | 2       |
| test.cluster.com | 443/TCP | jobs    | 3       |
+------------------+---------+---------+---------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoWithVolume(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Application: app1
Platform: php
Router: planb
Teams: myteam (owner), tsuruteam, crane
External Addresses: myapp.tsuru.io
Created by: myapp_owner
Deploys: 7
Pool:
Quota: 3/40 units

Units: 3
+--------+---------+-------------+------+
| Name   | Status  | Host        | Port |
+--------+---------+-------------+------+
| app1/2 | pending |             |      |
| app1/0 | started | 10.10.10.10 |      |
| app1/1 | started | 9.9.9.9     |      |
+--------+---------+-------------+------+

Service instances: 1
+----------+-------------------+
| Service  | Instance (Plan)   |
+----------+-------------------+
| redisapi | myredisapi (test) |
+----------+-------------------+

Volumes: 1
+------+------------+------+
| Name | MountPoint | Mode |
+------+------------+------+
| vol1 | /vol1      | rw   |
+------+------------+------+

`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := transportFunc(func(req *http.Request) (resp *http.Response, err error) {
		body := `{"name":"app1","teamowner":"myteam","ip":"myapp.tsuru.io","platform":"php","repository":"git@git.com:php.git","state":"dead","units":[{"Ip":"10.10.10.10","ID":"app1/0","Status":"started"}, {"Ip":"9.9.9.9","ID":"app1/1","Status":"started"}, {"Ip":"","ID":"app1/2","Status":"pending"}],"Teams":["tsuruteam","crane"], "owner": "myapp_owner", "deploys": 7, "router": "planb", "quota": {"limit":40, "inUse":3}, "volumeBinds": [{"ID":{"App":"app1","MountPoint":"/vol1","Volume":"vol1"},"ReadOnly":false}], "serviceInstanceBinds": [{"service": "redisapi", "instance": "myredisapi", "plan": "test"}]}`
		return &http.Response{
			Body:       io.NopCloser(bytes.NewBufferString(body)),
			StatusCode: http.StatusOK,
		}, nil
	})
	client := cmd.NewClient(&http.Client{Transport: transport}, nil, manager)
	command := AppInfo{}
	command.Flags().Parse(true, []string{"--app", "app1"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppInfoInfo(c *check.C) {
	c.Assert((&AppInfo{}).Info(), check.NotNil)
}

func (s *S) TestAppInfoFlags(c *check.C) {
	command := AppInfo{}
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
	command := AppGrant{}
	command.Flags().Parse(true, []string{"--app", "games"})
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppGrantInfo(c *check.C) {
	c.Assert((&AppGrant{}).Info(), check.NotNil)
}

func (s *S) TestAppRevoke(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `Team "cobrateam" was removed from the "games" app` + "\n"
	context := cmd.Context{
		Args:   []string{"cobrateam"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppRevoke{}
	command.Flags().Parse(true, []string{"--app", "games"})
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppRevokeInfo(c *check.C) {
	c.Assert((&AppRevoke{}).Info(), check.NotNil)
}

func (s *S) TestAppList(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-----------+-------------+
| Application | Units     | Address     |
+-------------+-----------+-------------+
| app1        | 1 started | 10.10.10.10 |
+-------------+-----------+-------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListDisplayAppsInAlphabeticalOrder(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.11","name":"sapp","units":[{"ID":"sapp1/0","Status":"started"}]},{"ip":"10.10.10.10","name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-----------+-------------+
| Application | Units     | Address     |
+-------------+-----------+-------------+
| app1        | 1 started | 10.10.10.10 |
+-------------+-----------+-------------+
| sapp        | 1 started | 10.10.10.11 |
+-------------+-----------+-------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListUnitIsntAvailable(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","name":"app1","units":[{"ID":"app1/0","Status":"pending"}]}]`
	expected := `+-------------+-----------+-------------+
| Application | Units     | Address     |
+-------------+-----------+-------------+
| app1        | 1 pending | 10.10.10.10 |
+-------------+-----------+-------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListErrorFetchingUnits(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","name":"app1","units":[],"Error": "timeout"}]`
	expected := `+-------------+----------------------+-------------+
| Application | Units                | Address     |
+-------------+----------------------+-------------+
| app1        | error fetching units | 10.10.10.10 |
+-------------+----------------------+-------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListErrorFetchingUnitsVerbose(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","name":"app1","units":[],"Error": "timeout"}]`
	expected := "*************************** <Request uri=\"/1.0/apps?\"> **********************************\n" +
		"GET /1.0/apps? HTTP/1.1\r\n" +
		"Host: localhost:8080\r\n" +
		"Connection: close\r\n" +
		"Authorization: bearer sometoken\r\n" +
		"X-Tsuru-Verbosity: 1\r\n" +
		"\r\n" +
		"*************************** </Request uri=\"/1.0/apps?\"> **********************************\n" +
		"+-------------+-------------------------------+-------------+\n" +
		"| Application | Units                         | Address     |\n" +
		"+-------------+-------------------------------+-------------+\n" +
		"| app1        | error fetching units: timeout | 10.10.10.10 |\n" +
		"+-------------+-------------------------------+-------------+\n"
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{
		Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK},
	}, &context, manager)
	client.Verbosity = 1
	command := AppList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListUnitWithoutID(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","name":"app1","units":[{"ID":"","Status":"pending"}, {"ID":"unit2","Status":"stopped"}]}]`
	expected := `+-------------+-----------+-------------+
| Application | Units     | Address     |
+-------------+-----------+-------------+
| app1        | 1 stopped | 10.10.10.10 |
+-------------+-----------+-------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListCName(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","cname":["app1.tsuru.io"],"name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-----------+-----------------------+
| Application | Units     | Address               |
+-------------+-----------+-----------------------+
| app1        | 1 started | app1.tsuru.io (cname) |
|             |           | 10.10.10.10           |
+-------------+-----------+-----------------------+
`
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListFiltering(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","cname":["app1.tsuru.io"],"name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-----------+-----------------------+
| Application | Units     | Address               |
+-------------+-----------+-----------------------+
| app1        | 1 started | app1.tsuru.io (cname) |
|             |           | 10.10.10.10           |
+-------------+-----------+-----------------------+
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
	command := AppList{}
	command.Flags().Parse(true, []string{"-p", "python", "--locked", "--user", "glenda@tsuru.io", "-t", "tsuru", "--name", "myapp", "--pool", "pool", "--status", "started", "--tag", "tag a", "--tag", "tag b"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
	queryString := url.Values(map[string][]string{
		"platform":  {"python"},
		"locked":    {"true"},
		"owner":     {"glenda@tsuru.io"},
		"teamOwner": {"tsuru"},
		"name":      {"myapp"},
		"pool":      {"pool"},
		"status":    {"started"},
		"tag":       {"tag a", "tag b"},
	})
	c.Assert(request.URL.Query(), check.DeepEquals, queryString)
}

func (s *S) TestAppListFilteringMe(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","cname":["app1.tsuru.io"],"name":"app1","units":[{"ID":"app1/0","Status":"started"}]}]`
	expected := `+-------------+-----------+-----------------------+
| Application | Units     | Address               |
+-------------+-----------+-----------------------+
| app1        | 1 started | app1.tsuru.io (cname) |
|             |           | 10.10.10.10           |
+-------------+-----------+-----------------------+
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
	command := AppList{}
	command.Flags().Parse(true, []string{"-u", "me"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
	queryString := url.Values(map[string][]string{"owner": {"gopher@tsuru.io"}})
	c.Assert(request.URL.Query(), check.DeepEquals, queryString)
}

func (s *S) TestAppListSortByCountAndStatus(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"ip":"10.10.10.10","cname":["app1.tsuru.io"],"name":"app1","units":[{"ID":"app1/0","Status":"starting"},{"ID":"app1/1","Status":"stopped"},{"ID":"app1/2","Status":"asleep"},{"ID":"app1/3","Status":"started"},{"ID":"app1/4","Status":"started"},{"ID":"app1/5","Status":"stopped"}]}]`
	expected := `+-------------+------------+-----------------------+
| Application | Units      | Address               |
+-------------+------------+-----------------------+
| app1        | 2 started  | app1.tsuru.io (cname) |
|             | 2 stopped  | 10.10.10.10           |
|             | 1 asleep   |                       |
|             | 1 starting |                       |
+-------------+------------+-----------------------+
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
	command := AppList{}
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
	transport := cmdtest.ConditionalTransport{
		CondFunc: func(r *http.Request) bool {
			return true
		},
		Transport: cmdtest.Transport{Message: result, Status: http.StatusOK},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := AppList{}
	command.Flags().Parse(true, []string{"-q"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppListWithFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `[{"name":"app1","platform":"python","pool":"pool2"},{"name":"app2","platform":"python","pool":"pool2"},{"name":"app3","platform":"go","pool":"pool1"}]`
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
	command := AppList{}
	command.Flags().Parse(true, []string{"-p", "python", "-q"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
	queryString := url.Values(map[string][]string{
		"platform":   {"python"},
		"simplified": {"true"},
	})
	c.Assert(request.URL.Query(), check.DeepEquals, queryString)
}

func (s *S) TestAppListInfo(c *check.C) {
	c.Assert((&AppList{}).Info(), check.NotNil)
}

func (s *S) TestAppListIsACommand(c *check.C) {
	var _ cmd.Command = &AppList{}
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
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.FormValue("process"), check.Equals, "web")
			return strings.HasSuffix(req.URL.Path, "/apps/handful_of_nothing/restart") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppRestart{}
	command.Flags().Parse(true, []string{"--app", "handful_of_nothing", "--process", "web"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestAppRestartInfo(c *check.C) {
	c.Assert((&AppRestart{}).Info(), check.NotNil)
}

func (s *S) TestAppRestartIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &AppRestart{}
}

func (s *S) TestAddCName(c *check.C) {
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
			cname := req.FormValue("cname") == "death.evergrey.mycompany.com"
			method := req.Method == "POST"
			url := strings.HasSuffix(req.URL.Path, "/apps/death/cname")
			contentType := req.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
			return method && url && cname && contentType
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := CnameAdd{}
	err := command.Flags().Parse(true, []string{"-a", "death", "death.evergrey.mycompany.com"})
	c.Assert(err, check.IsNil)
	context.Args = command.Flags().Args()
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "cname successfully defined.\n")
}

func (s *S) TestAddCNameFailure(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.Transport{Message: "Invalid cname", Status: http.StatusPreconditionFailed}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := CnameAdd{}
	err := command.Flags().Parse(true, []string{"-a", "masterplan", "masterplan.evergrey.mycompany.com"})
	c.Assert(err, check.IsNil)

	context.Args = command.Flags().Args()
	err = command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "Invalid cname")
}

func (s *S) TestAddCNameInfo(c *check.C) {
	c.Assert((&CnameAdd{}).Info(), check.NotNil)
}

func (s *S) TestAddCNameIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &CnameAdd{}
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
			method := req.Method == http.MethodDelete
			url := strings.HasSuffix(req.URL.Path, "/apps/death/cname")
			return method && url
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := CnameRemove{}
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
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Restarted", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			method := req.Method == http.MethodDelete
			url := strings.HasSuffix(req.URL.Path, "/apps/corey/cname")
			return method && url
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	cmd := &CnameRemove{}
	cmd.Flags().Parse(true, []string{"-a", "corey"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "cname successfully undefined.\n")
}

func (s *S) TestRemoveCNameInfo(c *check.C) {
	c.Assert((&CnameRemove{}).Info(), check.NotNil)
}

func (s *S) TestRemoveCNameIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &CnameRemove{}
}

func (s *S) TestAppStartInfo(c *check.C) {
	c.Assert((&AppStart{}).Info(), check.NotNil)
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
	expectedOut := "-- started --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.FormValue("process"), check.Equals, "worker")
			return strings.HasSuffix(req.URL.Path, "/apps/handful_of_nothing/start") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppStart{}
	command.Flags().Parse(true, []string{"--app", "handful_of_nothing", "--process", "worker"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestAppStartIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &AppStart{}
}

func (s *S) TestUnitPort(c *check.C) {
	var tests = []struct {
		unit *unit
		port string
	}{
		{&unit{Address: &url.URL{Host: "localhost:4040"}}, "4040"},
		{&unit{Address: &url.URL{Host: "localhost"}}, ""},
		{&unit{}, ""},
	}
	for _, t := range tests {
		c.Check(t.unit.Port(), check.Equals, t.port)
	}
}

func (s *S) TestUnitHost(c *check.C) {
	var tests = []struct {
		unit *unit
		host string
	}{
		{&unit{Address: &url.URL{Host: "localhost:4040"}}, "localhost"},
		{&unit{}, ""},
	}
	for _, t := range tests {
		c.Check(t.unit.Host(), check.Equals, t.host)
	}
}

func (s *S) TestAppStopInfo(c *check.C) {
	c.Assert((&AppStop{}).Info(), check.NotNil)
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
	expectedOut := "-- stopped --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			c.Assert(req.FormValue("process"), check.Equals, "worker")
			return strings.HasSuffix(req.URL.Path, "/apps/handful_of_nothing/stop") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppStop{}
	command.Flags().Parse(true, []string{"--app", "handful_of_nothing", "--process", "worker"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestAppStopIsAFlaggedCommand(c *check.C) {
	var _ cmd.FlaggedCommand = &AppStop{}
}
