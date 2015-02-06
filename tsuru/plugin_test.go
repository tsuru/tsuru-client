// Copyright 2015 tsuru-client authors. All rights reserved.
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

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec/exectest"
	"github.com/tsuru/tsuru/fs/fstest"
	"launchpad.net/gocheck"
)

func (s *S) TestPluginInstallInfo(c *gocheck.C) {
	c.Assert(pluginInstall{}.Info(), gocheck.NotNil)
}

func (s *S) TestPluginInstall(c *gocheck.C) {
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
	c.Assert(err, gocheck.IsNil)
	pluginsPath := cmd.JoinWithUserDir(".tsuru", "plugins")
	hasAction := rfs.HasAction(fmt.Sprintf("mkdirall %s with mode 0755", pluginsPath))
	c.Assert(hasAction, gocheck.Equals, true)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	hasAction = rfs.HasAction(fmt.Sprintf("openfile %s with mode 0755", pluginPath))
	c.Assert(hasAction, gocheck.Equals, true)
	f, err := rfs.Open(pluginPath)
	c.Assert(err, gocheck.IsNil)
	data, err := ioutil.ReadAll(f)
	c.Assert(err, gocheck.IsNil)
	c.Assert("fakeplugin\n", gocheck.Equals, string(data))
	expected := `Plugin "myplugin" successfully installed!` + "\n"
	c.Assert(expected, gocheck.Equals, stdout.String())
}

func (s *S) TestPluginInstallIsACommand(c *gocheck.C) {
	var _ cmd.Command = &pluginInstall{}
}

func (s *S) TestPluginInfo(c *gocheck.C) {
	c.Assert(plugin{}.Info(), gocheck.NotNil)
}

func (s *S) TestPlugin(c *gocheck.C) {
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
	c.Assert(err, gocheck.IsNil)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{"a", "b"}), gocheck.Equals, true)
	c.Assert(buf.String(), gocheck.Equals, "hello world")
	commands := fexec.GetCommands(pluginPath)
	c.Assert(commands, gocheck.HasLen, 1)
	target, err := cmd.ReadTarget()
	c.Assert(err, gocheck.IsNil)
	token, err := cmd.ReadToken()
	c.Assert(err, gocheck.IsNil)
	envs := os.Environ()
	tsuruEnvs := []string{
		fmt.Sprintf("TSURU_TARGET=%s/", target),
		fmt.Sprintf("TSURU_TOKEN=%s", token),
		"TSURU_PLUGIN_NAME=myplugin",
	}
	envs = append(envs, tsuruEnvs...)
	c.Assert(commands[0].GetEnvs(), gocheck.DeepEquals, envs)
}

func (s *S) TestPluginWithArgs(c *gocheck.C) {
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
	c.Assert(err, gocheck.IsNil)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{"ble", "bla"}), gocheck.Equals, true)
}

func (s *S) TestPluginIsACommand(c *gocheck.C) {
	var _ cmd.Command = &plugin{}
}

func (s *S) TestPluginRemoveInfo(c *gocheck.C) {
	c.Assert(pluginRemove{}.Info(), gocheck.NotNil)
}

func (s *S) TestPluginRemove(c *gocheck.C) {
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
	c.Assert(err, gocheck.IsNil)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	hasAction := rfs.HasAction(fmt.Sprintf("remove %s", pluginPath))
	c.Assert(hasAction, gocheck.Equals, true)
	expected := `Plugin "myplugin" successfully removed!` + "\n"
	c.Assert(expected, gocheck.Equals, stdout.String())
}

func (s *S) TestPluginRemoveIsACommand(c *gocheck.C) {
	var _ cmd.Command = &pluginRemove{}
}

func (s *S) TestPluginListInfo(c *gocheck.C) {
	c.Assert(pluginList{}.Info(), gocheck.NotNil)
}

func (s *S) TestPluginListIsACommand(c *gocheck.C) {
	var _ cmd.Command = &pluginList{}
}
