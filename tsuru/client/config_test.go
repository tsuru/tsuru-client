package client

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	check "gopkg.in/check.v1"
)

func (s *S) TestAliases(c *check.C) {
	a := Aliases{}
	src := []string{"deploy"}
	target := []string{"app", "deploy"}

	a.Add(src, target)
	c.Assert(a, check.HasLen, 1)

	found, ok := a.Has(src)
	c.Assert(ok, check.Equals, true)
	c.Assert(found.Source, check.DeepEquals, src)
	c.Assert(found.Target, check.DeepEquals, target)

	found, ok = a.Has([]string{"unexistent"})
	c.Assert(ok, check.Equals, false)
	c.Assert(found, check.IsNil)

	override := []string{"app", "shell"}
	a.Add(src, override)
	found, ok = a.Has(src)
	c.Assert(ok, check.Equals, true)
	c.Assert(found.Source, check.DeepEquals, src)
	c.Assert(found.Target, check.DeepEquals, override)

	a.Remove(src)
	c.Assert(a, check.HasLen, 0)
}

func (s *S) TestConfigFilePath(c *check.C) {
	homeDir, err := os.UserHomeDir()
	c.Assert(err, check.IsNil)
	expectedPath := filepath.Join(homeDir, ".config", "tsuru", "config.json")
	path, err := configFilePath()
	c.Assert(err, check.IsNil)
	c.Assert(expectedPath, check.Equals, path)

	tmpDir, err := os.MkdirTemp("", "tsuru-test")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(tmpDir)
	expectedPath = filepath.Join(tmpDir, "config.json")
	os.Setenv("TSURU_CONFIG_FILE", expectedPath)
	defer os.Unsetenv("TSURU_CONFIG_FILE")
	path, err = configFilePath()
	c.Assert(err, check.IsNil)
	c.Assert(path, check.Equals, expectedPath)
}

func (s *S) TestSaveAndGetConfig(c *check.C) {
	tmpDir, err := os.MkdirTemp("", "tsuru-test")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	os.Setenv("TSURU_CONFIG_FILE", configPath)
	defer os.Unsetenv("TSURU_CONFIG_FILE")

	config := AppConfig{
		Aliases: Aliases{
			{Source: []string{"test"}, Target: []string{"cmd"}},
		},
	}

	err = config.Save()
	c.Assert(err, check.IsNil)

	loadedConfig, err := GetConfig()
	c.Assert(err, check.IsNil)
	c.Assert(loadedConfig.Aliases, check.HasLen, 1)
	c.Assert(loadedConfig.Aliases[0].Target, check.DeepEquals, []string{"cmd"})
}

func (s *S) TestAliasInfo(c *check.C) {
	var cmd ConfigAlias
	c.Assert(cmd.Info(), check.NotNil)
}

func (s *S) TestAliasRun(c *check.C) {
	tmpDir, err := os.MkdirTemp("", "tsuru-test")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(tmpDir)
	configPath := filepath.Join(tmpDir, "config.json")
	os.Setenv("TSURU_CONFIG_FILE", configPath)
	defer os.Unsetenv("TSURU_CONFIG_FILE")

	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"deploy", "app deploy"},
	}
	command := ConfigAlias{}
	command.Flags().Parse([]string{})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)

	content, err := os.ReadFile(configPath)
	c.Assert(err, check.IsNil)
	var appConfig AppConfig
	err = json.Unmarshal(content, &appConfig)
	c.Assert(err, check.IsNil)

	c.Assert(appConfig.Aliases, check.HasLen, 1)
	c.Assert(appConfig.Aliases[0].Source, check.DeepEquals, []string{"deploy"})
	c.Assert(appConfig.Aliases[0].Target, check.DeepEquals, []string{"app", "deploy"})

	context.Args = []string{"deploy", "override"}
	command = ConfigAlias{}
	command.Flags().Parse([]string{})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)

	content, err = os.ReadFile(configPath)
	c.Assert(err, check.IsNil)
	err = json.Unmarshal(content, &appConfig)
	c.Assert(err, check.IsNil)

	c.Assert(appConfig.Aliases, check.HasLen, 1)
	c.Assert(appConfig.Aliases[0].Source, check.DeepEquals, []string{"deploy"})
	c.Assert(appConfig.Aliases[0].Target, check.DeepEquals, []string{"override"})
}

func (s *S) TestAliasRemove(c *check.C) {
	tmpDir, err := os.MkdirTemp("", "tsuru-test")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(tmpDir)
	configPath := filepath.Join(tmpDir, "config.json")
	os.Setenv("TSURU_CONFIG_FILE", configPath)
	defer os.Unsetenv("TSURU_CONFIG_FILE")

	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"deploy", "app deploy"},
	}
	command := ConfigAlias{}
	command.Flags().Parse([]string{})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)

	content, err := os.ReadFile(configPath)
	c.Assert(err, check.IsNil)
	var appConfig AppConfig
	err = json.Unmarshal(content, &appConfig)
	c.Assert(err, check.IsNil)

	c.Assert(appConfig.Aliases, check.HasLen, 1)
	c.Assert(appConfig.Aliases[0].Source, check.DeepEquals, []string{"deploy"})

	context.Args = []string{"deploy"}
	command = ConfigAlias{}
	command.Flags().Parse([]string{"--delete"})
	err = command.Run(&context)
	c.Assert(err, check.IsNil)

	content, err = os.ReadFile(configPath)
	c.Assert(err, check.IsNil)
	err = json.Unmarshal(content, &appConfig)
	c.Assert(err, check.IsNil)

	c.Assert(appConfig.Aliases, check.HasLen, 0)
}
