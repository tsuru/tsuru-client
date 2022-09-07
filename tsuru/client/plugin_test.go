// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

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
	c.Assert(PluginInstall{}.Info(), check.NotNil)
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
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := PluginInstall{}
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

func (s *S) TestPluginInstallError(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("my err"))
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
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := PluginInstall{}
	err := command.Run(&context, client)
	c.Assert(err, check.ErrorMatches, `Invalid status code reading plugin: 500 - "my err"`)
}

func (s *S) TestPluginInstallIsACommand(c *check.C) {
	var _ cmd.Command = &PluginInstall{}
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
	Execut = &fexec
	defer func() {
		Execut = nil
	}()
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", "a", "b"},
		Stdout: &buf,
		Stderr: &buf,
	}
	err := RunPlugin(&context)
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
	Execut = &fexec
	defer func() {
		Execut = nil
	}()
	context := cmd.Context{Args: []string{"myplugin", "ble", "bla"}}
	err := RunPlugin(&context)
	c.Assert(err, check.IsNil)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{"ble", "bla"}), check.Equals, true)
}

func (s *S) TestPluginTryNameWithAnyExtension(c *check.C) {
	// Kids, do not try this at $HOME
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempHome, _ := filepath.Abs("testdata")
	os.Setenv("HOME", tempHome)

	fexec := exectest.FakeExecutor{
		Output: map[string][][]byte{
			"a b": {[]byte("hello world")},
		},
	}
	Execut = &fexec
	defer func() {
		Execut = nil
	}()
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"otherplugin", "a", "b"},
		Stdout: &buf,
		Stderr: &buf,
	}
	err := RunPlugin(&context)
	c.Assert(err, check.IsNil)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "otherplugin.exe")
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
		"TSURU_PLUGIN_NAME=otherplugin",
	}
	envs = append(envs, tsuruEnvs...)
	c.Assert(commands[0].GetEnvs(), check.DeepEquals, envs)
}

func (s *S) TestPluginLoop(c *check.C) {
	os.Setenv("TSURU_PLUGIN_NAME", "myplugin")
	defer os.Unsetenv("TSURU_PLUGIN_NAME")
	fexec := exectest.FakeExecutor{
		Output: map[string][][]byte{
			"a b": {[]byte("hello world")},
		},
	}
	Execut = &fexec
	defer func() {
		Execut = nil
	}()
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", "a", "b"},
		Stdout: &buf,
		Stderr: &buf,
	}
	err := RunPlugin(&context)
	c.Assert(err, check.Equals, cmd.ErrLookup)
}

func (s *S) TestPluginCommandNotFound(c *check.C) {
	fexec := exectest.ErrorExecutor{Err: os.ErrNotExist}
	Execut = &fexec
	defer func() {
		Execut = nil
	}()
	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", "a", "b"},
		Stdout: &buf,
		Stderr: &buf,
	}
	err := RunPlugin(&context)
	c.Assert(err, check.Equals, cmd.ErrLookup)
}

func (s *S) TestPluginRemoveInfo(c *check.C) {
	c.Assert(PluginRemove{}.Info(), check.NotNil)
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
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := PluginRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	hasAction := rfs.HasAction(fmt.Sprintf("remove %s", pluginPath))
	c.Assert(hasAction, check.Equals, true)
	expected := `Plugin "myplugin" successfully removed!` + "\n"
	c.Assert(expected, check.Equals, stdout.String())
}

func (s *S) TestPluginRemoveIsACommand(c *check.C) {
	var _ cmd.Command = &PluginRemove{}
}

func (s *S) TestPluginListInfo(c *check.C) {
	c.Assert(PluginList{}.Info(), check.NotNil)
}

func (s *S) TestPluginListIsACommand(c *check.C) {
	var _ cmd.Command = &PluginList{}
}

func (s *S) TestPluginBundleInfo(c *check.C) {
	c.Assert(PluginBundle{}.Info(), check.NotNil)
}

func (s *S) TestPluginBundle(c *check.C) {
	tsFake1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "fakeplugin1")
	}))
	defer tsFake1.Close()
	tsFake2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "fakeplugin2")
	}))
	defer tsFake2.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"plugins":[{"name":"testfake1","url":"%s"},{"name":"testfake2","url":"%s"}]}`, tsFake1.URL, tsFake2.URL)
	}))
	defer ts.Close()

	rfs := fstest.RecordingFs{}
	fsystem = &rfs
	defer func() {
		fsystem = nil
	}()
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{ts.URL},
		Stdout: &stdout,
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := PluginBundle{}

	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	pluginsPath := cmd.JoinWithUserDir(".tsuru", "plugins")
	hasAction := rfs.HasAction(fmt.Sprintf("mkdirall %s with mode 0755", pluginsPath))
	c.Assert(hasAction, check.Equals, true)
	plugin1Path := cmd.JoinWithUserDir(".tsuru", "plugins", "testfake1")
	hasAction = rfs.HasAction(fmt.Sprintf("openfile %s with mode 0755", plugin1Path))
	c.Assert(hasAction, check.Equals, true)
	plugin2Path := cmd.JoinWithUserDir(".tsuru", "plugins", "testfake2")
	hasAction = rfs.HasAction(fmt.Sprintf("openfile %s with mode 0755", plugin2Path))
	c.Assert(hasAction, check.Equals, true)

	f, err := rfs.Open(plugin1Path)
	c.Assert(err, check.IsNil)
	data, err := ioutil.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert("fakeplugin1\n", check.Equals, string(data))

	f, err = rfs.Open(plugin2Path)
	c.Assert(err, check.IsNil)
	data, err = ioutil.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert("fakeplugin2\n", check.Equals, string(data))

	expected := `Successfully installed 2 plugins: testfake1, testfake2` + "\n"
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPluginBundleError(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("my err"))
	}))
	defer ts.Close()
	rfs := fstest.RecordingFs{}
	fsystem = &rfs
	defer func() {
		fsystem = nil
	}()
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{ts.URL},
		Stdout: &stdout,
	}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	command := PluginBundle{}
	err := command.Run(&context, client)
	c.Assert(err, check.ErrorMatches, `Invalid status code reading plugin bundle: 500 - "my err"`)
}

func (s *S) TestPluginBundleIsACommand(c *check.C) {
	var _ cmd.Command = &PluginBundle{}
}
