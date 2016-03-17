// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec/exectest"
	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestPluginInstallInfo(c *check.C) {
	c.Assert(pluginInstall{}.Info(), check.NotNil)
}

func (s *S) TestPluginInstall(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "fakeplugin")
	}))
	defer ts.Close()
	rfs := fstest.RecordingFs{}
	fsystem = &rfs
	defer func() {
		fsystem = nil
	}()
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", ts.URL},
		Stdout: &stdout,
	}
	client := cmd.NewClient(nil, nil, manager)
	command := pluginInstall{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	pluginsPath := cmd.JoinWithUserDir(".tsuru", "plugins")
	hasAction := rfs.HasAction(fmt.Sprintf("mkdirall %s with mode 0755", pluginsPath))
	c.Assert(hasAction, check.Equals, true)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	hasAction = rfs.HasAction(fmt.Sprintf("openfile %s with mode 0755", pluginPath))
	c.Assert(hasAction, check.Equals, true)
	f, err := rfs.Open(pluginPath)
	c.Assert(err, check.IsNil)
	data, err := ioutil.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert("fakeplugin\n", check.Equals, string(data))
	expected := `Plugin "myplugin" successfully installed!` + "\n"
	c.Assert(expected, check.Equals, stdout.String())
}

func (s *S) TestPluginInstallIsACommand(c *check.C) {
	var _ cmd.Command = &pluginInstall{}
}

func (s *S) TestPlugin(c *check.C) {
	// Kids, do not try this at $HOME
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempHome, _ := filepath.Abs("testdata")
	os.Setenv("HOME", tempHome)

	fexec := exectest.FakeExecutor{
		Output: map[string][][]byte{
			"a b": {[]byte("hello world")},
		},
	}
	execut = &fexec
	defer func() {
		execut = nil
	}()
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", "a", "b"},
		Stdout: &buf,
		Stderr: &buf,
	}
	client := cmd.NewClient(nil, nil, manager)
	command := plugin{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{"a", "b"}), check.Equals, true)
	c.Assert(buf.String(), check.Equals, "hello world")
	commands := fexec.GetCommands(pluginPath)
	c.Assert(commands, check.HasLen, 1)
	target, err := cmd.GetTarget()
	c.Assert(err, check.IsNil)
	token, err := cmd.ReadToken()
	c.Assert(err, check.IsNil)
	envs := os.Environ()
	tsuruEnvs := []string{
		fmt.Sprintf("TSURU_TARGET=%s", target),
		fmt.Sprintf("TSURU_TOKEN=%s", token),
		"TSURU_PLUGIN_NAME=myplugin",
	}
	envs = append(envs, tsuruEnvs...)
	c.Assert(commands[0].GetEnvs(), check.DeepEquals, envs)
}

func (s *S) TestPluginWithArgs(c *check.C) {
	// Kids, do not try this at $HOME
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempHome, _ := filepath.Abs("testdata")
	os.Setenv("HOME", tempHome)

	fexec := exectest.FakeExecutor{}
	execut = &fexec
	defer func() {
		execut = nil
	}()
	context := cmd.Context{
		Args: []string{"myplugin", "ble", "bla"},
	}
	client := cmd.NewClient(nil, nil, manager)
	command := plugin{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{"ble", "bla"}), check.Equals, true)
}

func (s *S) TestPluginLoop(c *check.C) {
	os.Setenv("TSURU_PLUGIN_NAME", "myplugin")
	defer os.Unsetenv("TSURU_PLUGIN_NAME")
	fexec := exectest.FakeExecutor{
		Output: map[string][][]byte{
			"a b": {[]byte("hello world")},
		},
	}
	execut = &fexec
	defer func() {
		execut = nil
	}()
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", "a", "b"},
		Stdout: &buf,
		Stderr: &buf,
	}
	client := cmd.NewClient(nil, nil, manager)
	command := plugin{}
	err := command.Run(&context, client)
	c.Assert(err, check.Equals, cmd.ErrLookup)
}

func (s *S) TestPluginCommandNotFound(c *check.C) {
	fexec := exectest.ErrorExecutor{Err: os.ErrNotExist}
	execut = &fexec
	defer func() {
		execut = nil
	}()
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", "a", "b"},
		Stdout: &buf,
		Stderr: &buf,
	}
	client := cmd.NewClient(nil, nil, manager)
	command := plugin{}
	err := command.Run(&context, client)
	c.Assert(err, check.Equals, cmd.ErrLookup)
}

func (s *S) TestPluginRemoveInfo(c *check.C) {
	c.Assert(pluginRemove{}.Info(), check.NotNil)
}

func (s *S) TestPluginRemove(c *check.C) {
	rfs := fstest.RecordingFs{}
	fsystem = &rfs
	defer func() {
		fsystem = nil
	}()
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin"},
		Stdout: &stdout,
	}
	client := cmd.NewClient(nil, nil, manager)
	command := pluginRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	hasAction := rfs.HasAction(fmt.Sprintf("remove %s", pluginPath))
	c.Assert(hasAction, check.Equals, true)
	expected := `Plugin "myplugin" successfully removed!` + "\n"
	c.Assert(expected, check.Equals, stdout.String())
}

func (s *S) TestPluginRemoveIsACommand(c *check.C) {
	var _ cmd.Command = &pluginRemove{}
}

func (s *S) TestPluginListInfo(c *check.C) {
	c.Assert(pluginList{}.Info(), check.NotNil)
}

func (s *S) TestPluginListIsACommand(c *check.C) {
	var _ cmd.Command = &pluginList{}
}
