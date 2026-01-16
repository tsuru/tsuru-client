// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru/exec/exectest"
	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestPluginInstallInfo(c *check.C) {
	c.Assert(PluginInstall{}.Info(), check.NotNil)
}

func (s *S) TestPluginInstallWithManifest(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "fakeplugin")
	}))
	defer ts.Close()
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResp := fmt.Sprintf(`{
			"SchemaVersion":"1.0",
			"Metadata": {"Name": "myplugin", "Version": "0.33.1"},
			"URLPerPlatform": {
			  "%s/%s": "%s"
			}
		  }`, runtime.GOOS, runtime.GOARCH, ts.URL)
		fmt.Fprintln(w, jsonResp)
	}))

	defer ts2.Close()
	rfs := fstest.RecordingFs{}
	config.SetFileSystem(&rfs)
	defer func() {
		config.ResetFileSystem()
	}()
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", ts2.URL},
		Stdout: &stdout,
	}
	command := PluginInstall{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	pluginsPath := config.JoinWithUserDir(".tsuru", "plugins")
	hasAction := rfs.HasAction(fmt.Sprintf("mkdirall %s with mode 0755", pluginsPath))
	c.Assert(hasAction, check.Equals, true)
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	hasAction = rfs.HasAction(fmt.Sprintf("openfile %s with mode 0755", pluginPath))
	c.Assert(hasAction, check.Equals, true)
	f, err := rfs.Open(pluginPath)
	c.Assert(err, check.IsNil)
	data, err := io.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, "fakeplugin\n")
	expected := `Plugin "myplugin" successfully installed!` + "\n"
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPluginInstall(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "fakeplugin")
	}))
	defer ts.Close()
	rfs := fstest.RecordingFs{}
	config.SetFileSystem(&rfs)
	defer func() {
		config.ResetFileSystem()
	}()
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", ts.URL},
		Stdout: &stdout,
	}
	command := PluginInstall{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	pluginsPath := config.JoinWithUserDir(".tsuru", "plugins")
	hasAction := rfs.HasAction(fmt.Sprintf("mkdirall %s with mode 0755", pluginsPath))
	c.Assert(hasAction, check.Equals, true)
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	hasAction = rfs.HasAction(fmt.Sprintf("openfile %s with mode 0755", pluginPath))
	c.Assert(hasAction, check.Equals, true)
	f, err := rfs.Open(pluginPath)
	c.Assert(err, check.IsNil)
	data, err := io.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, "fakeplugin\n")
	expected := `Plugin "myplugin" successfully installed!` + "\n"
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPluginInstallError(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("my err"))
	}))
	defer ts.Close()
	rfs := fstest.RecordingFs{}
	config.SetFileSystem(&rfs)
	defer func() {
		config.ResetFileSystem()
	}()
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin", ts.URL},
		Stdout: &stdout,
	}
	command := PluginInstall{}
	err := command.Run(&context)
	c.Assert(err, check.ErrorMatches, `error installing plugin "myplugin": invalid status code reading plugin: 500 - "my err"`)
}

func (s *S) TestPluginInstallIsACommand(c *check.C) {
	var _ cmd.Command = &PluginInstall{}
}

func (s *S) TestPluginExtractTarGz(c *check.C) {
	rfs := fstest.RecordingFs{}
	config.SetFileSystem(&rfs)
	defer func() {
		config.ResetFileSystem()
	}()
	tmpDir, err := config.Filesystem().MkdirTemp("", "")
	c.Assert(err, check.IsNil)

	tarGzFile, err := os.ReadFile("./testdata/archivedplugins/myplugin.tar.gz")
	c.Assert(err, check.IsNil)

	err = extractTarGz(tmpDir, bytes.NewReader(tarGzFile))
	c.Assert(err, check.IsNil)

	expectedFilepath := filepath.Join(tmpDir, "myplugin", "myplugin.txt")
	resultFile, err := config.Filesystem().Open(expectedFilepath)
	c.Assert(err, check.IsNil)
	resultContent, err := io.ReadAll(resultFile)
	c.Assert(err, check.IsNil)
	c.Assert(string(resultContent), check.Equals, "It worked")
}

func (s *S) TestPluginExtractZip(c *check.C) {
	rfs := fstest.RecordingFs{}
	config.SetFileSystem(&rfs)
	defer func() {
		config.ResetFileSystem()
	}()

	tmpDir, err := config.Filesystem().MkdirTemp("", "")
	c.Assert(err, check.IsNil)

	zipFile, err := os.ReadFile("./testdata/archivedplugins/myplugin.zip")
	c.Assert(err, check.IsNil)

	err = extractZip(tmpDir, bytes.NewReader(zipFile))
	c.Assert(err, check.IsNil)

	expectedFilepath := filepath.Join(tmpDir, "myplugin", "myplugin.txt")
	resultFile, err := config.Filesystem().Open(expectedFilepath)
	c.Assert(err, check.IsNil)
	resultContent, err := io.ReadAll(resultFile)
	c.Assert(err, check.IsNil)
	c.Assert(string(resultContent), check.Equals, "It worked")
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
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{"a", "b"}), check.Equals, true)
	c.Assert(buf.String(), check.Equals, "hello world")
	commands := fexec.GetCommands(pluginPath)
	c.Assert(commands, check.HasLen, 1)
	target, err := config.GetTarget()
	c.Assert(err, check.IsNil)
	token, err := config.ReadTokenV1()
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
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", "myplugin")
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
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", "otherplugin.exe")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{"a", "b"}), check.Equals, true)
	c.Assert(buf.String(), check.Equals, "hello world")
	commands := fexec.GetCommands(pluginPath)
	c.Assert(commands, check.HasLen, 1)
	target, err := config.GetTarget()
	c.Assert(err, check.IsNil)
	token, err := config.ReadTokenV1()
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
	config.SetFileSystem(&rfs)
	defer func() {
		config.ResetFileSystem()
	}()
	var stdout bytes.Buffer
	context := cmd.Context{
		Args:   []string{"myplugin"},
		Stdout: &stdout,
	}
	command := PluginRemove{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	hasAction := rfs.HasAction(fmt.Sprintf("remove %s", pluginPath))
	c.Assert(hasAction, check.Equals, true)
	expected := `Plugin "myplugin" successfully removed!` + "\n"
	c.Assert(stdout.String(), check.Equals, expected)
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
	config.SetFileSystem(&rfs)
	defer func() {
		config.ResetFileSystem()
	}()
	var stdout bytes.Buffer
	context := cmd.Context{Stdout: &stdout}
	command := PluginBundle{}
	command.Flags().Parse([]string{"--url", ts.URL})

	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	pluginsPath := config.JoinWithUserDir(".tsuru", "plugins")
	hasAction := rfs.HasAction(fmt.Sprintf("mkdirall %s with mode 0755", pluginsPath))
	c.Assert(hasAction, check.Equals, true)
	plugin1Path := config.JoinWithUserDir(".tsuru", "plugins", "testfake1")
	hasAction = rfs.HasAction(fmt.Sprintf("openfile %s with mode 0755", plugin1Path))
	c.Assert(hasAction, check.Equals, true)
	plugin2Path := config.JoinWithUserDir(".tsuru", "plugins", "testfake2")
	hasAction = rfs.HasAction(fmt.Sprintf("openfile %s with mode 0755", plugin2Path))
	c.Assert(hasAction, check.Equals, true)

	f, err := rfs.Open(plugin1Path)
	c.Assert(err, check.IsNil)
	data, err := io.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, "fakeplugin1\n")

	f, err = rfs.Open(plugin2Path)
	c.Assert(err, check.IsNil)
	data, err = io.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert(string(data), check.Equals, "fakeplugin2\n")

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
	config.SetFileSystem(&rfs)
	defer func() {
		config.ResetFileSystem()
	}()
	var stdout bytes.Buffer
	context := cmd.Context{Stdout: &stdout}
	command := PluginBundle{}
	command.Flags().Parse([]string{"--url", ts.URL})

	err := command.Run(&context)
	c.Assert(err, check.ErrorMatches, `invalid status code reading plugin bundle: 500 - "my err"`)
}

func (s *S) TestPluginBundleErrorNoFlags(c *check.C) {
	var stdout bytes.Buffer
	context := cmd.Context{Stdout: &stdout}

	command := PluginBundle{}
	command.Flags().Parse([]string{})
	err := command.Run(&context)
	c.Assert(err, check.ErrorMatches, `--url <url> is mandatory. See --help for usage`)
}

func (s *S) TestPluginBundleIsACommand(c *check.C) {
	var _ cmd.Command = &PluginBundle{}
}

func (s *S) TestFindPlugins(c *check.C) {
	// Kids, do not try this at $HOME
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempHome, _ := filepath.Abs("testdata")
	os.Setenv("HOME", tempHome)

	plugins := FindPlugins()
	c.Assert(plugins, check.HasLen, 2)
	c.Assert(plugins, check.DeepEquals, []string{"myplugin", "otherplugin.exe"})
}

func (s *S) TestFindPluginsEmptyDir(c *check.C) {
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempDir := c.MkDir()
	os.Setenv("HOME", tempDir)

	plugins := FindPlugins()
	c.Assert(plugins, check.IsNil)
}

func (s *S) TestFindPluginsNonExistentDir(c *check.C) {
	defer os.Setenv("HOME", os.Getenv("HOME"))
	os.Setenv("HOME", "/nonexistent/path/that/does/not/exist")

	plugins := FindPlugins()
	c.Assert(plugins, check.IsNil)
}

func (s *S) TestFindPluginsWithSubdirectory(c *check.C) {
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempDir := c.MkDir()
	os.Setenv("HOME", tempDir)

	pluginsDir := filepath.Join(tempDir, ".tsuru", "plugins")
	err := os.MkdirAll(pluginsDir, 0755)
	c.Assert(err, check.IsNil)

	// Create a regular plugin file
	pluginFile := filepath.Join(pluginsDir, "regular-plugin")
	f, err := os.Create(pluginFile)
	c.Assert(err, check.IsNil)
	f.Close()

	// Create a subdirectory with executable plugin
	subDir := filepath.Join(pluginsDir, "subplugin")
	err = os.MkdirAll(subDir, 0755)
	c.Assert(err, check.IsNil)

	// Create executable in subdirectory with same name as directory
	subPluginExec := filepath.Join(subDir, "subplugin")
	f, err = os.OpenFile(subPluginExec, os.O_CREATE|os.O_WRONLY, 0755)
	c.Assert(err, check.IsNil)
	f.Close()

	plugins := FindPlugins()
	c.Assert(len(plugins), check.Equals, 2)
	c.Assert(plugins, check.DeepEquals, []string{"regular-plugin", "subplugin"})
}

func (s *S) TestExecutePluginInfo(c *check.C) {
	plugin := ExecutePlugin{PluginName: "myplugin"}
	info := plugin.Info()
	c.Assert(info, check.NotNil)
	c.Assert(info.Name, check.Equals, "myplugin")
	c.Assert(info.Usage, check.Equals, "myplugin")
	c.Assert(info.Desc, check.Equals, "Executes the myplugin plugin.")
	c.Assert(info.MinArgs, check.Equals, cmd.ArbitraryArgs)
	c.Assert(info.GroupID, check.Equals, "plugin")
	c.Assert(info.OnlyAppendOnRoot, check.Equals, true)
	c.Assert(info.DisableFlagParsing, check.Equals, true)
	c.Assert(info.SilenceUsage, check.Equals, true)
}

func (s *S) TestExecutePluginRun(c *check.C) {
	// Kids, do not try this at $HOME
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempHome, _ := filepath.Abs("testdata")
	os.Setenv("HOME", tempHome)

	fexec := exectest.FakeExecutor{
		Output: map[string][][]byte{
			"arg1 arg2": {[]byte("plugin output")},
		},
	}
	Execut = &fexec
	defer func() {
		Execut = nil
	}()

	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{"arg1", "arg2"},
		Stdout: &buf,
		Stderr: &buf,
	}
	plugin := ExecutePlugin{PluginName: "myplugin"}
	err := plugin.Run(&context)
	c.Assert(err, check.IsNil)
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{"arg1", "arg2"}), check.Equals, true)
}

func (s *S) TestExecutePluginRunWithNoArgs(c *check.C) {
	// Kids, do not try this at $HOME
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempHome, _ := filepath.Abs("testdata")
	os.Setenv("HOME", tempHome)

	fexec := exectest.FakeExecutor{}
	Execut = &fexec
	defer func() {
		Execut = nil
	}()

	var buf bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &buf,
		Stderr: &buf,
	}
	plugin := ExecutePlugin{PluginName: "myplugin"}
	err := plugin.Run(&context)
	c.Assert(err, check.IsNil)
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{}), check.Equals, true)
}
