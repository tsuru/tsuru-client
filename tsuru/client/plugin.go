// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec"
)

type PluginInstall struct{}

func (PluginInstall) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plugin-install",
		Usage:   "plugin-install <plugin-name> <plugin-url>",
		Desc:    `Downloads the plugin file. It will be copied to [[$HOME/.tsuru/plugins]].`,
		MinArgs: 2,
	}
}

func (c *PluginInstall) Run(context *cmd.Context, client *cmd.Client) error {
	pluginsDir := cmd.JoinWithUserDir(".tsuru", "plugins")
	err := filesystem().MkdirAll(pluginsDir, 0755)
	if err != nil {
		return err
	}
	pluginName := context.Args[0]
	pluginURL := context.Args[1]
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", pluginName)
	file, err := filesystem().OpenFile(pluginPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	resp, err := http.Get(pluginURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	n, err := file.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return errors.New("Failed to install plugin.")
	}
	fmt.Fprintf(context.Stdout, `Plugin "%s" successfully installed!`+"\n", pluginName)
	return nil
}

type PluginRemove struct{}

func (PluginRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plugin-remove",
		Usage:   "plugin-remove <plugin-name>",
		Desc:    "Removes a previously installed tsuru plugin.",
		MinArgs: 1,
	}
}

func (c *PluginRemove) Run(context *cmd.Context, client *cmd.Client) error {
	pluginName := context.Args[0]
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", pluginName)
	err := filesystem().Remove(pluginPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `Plugin "%s" successfully removed!`+"\n", pluginName)
	return nil
}

type PluginList struct{}

func (PluginList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plugin-list",
		Usage:   "plugin-list",
		Desc:    "List installed tsuru plugins.",
		MinArgs: 0,
	}
}

func (c *PluginList) Run(context *cmd.Context, client *cmd.Client) error {
	pluginsPath := cmd.JoinWithUserDir(".tsuru", "plugins")
	plugins, _ := ioutil.ReadDir(pluginsPath)
	for _, p := range plugins {
		fmt.Println(p.Name())
	}
	return nil
}

func RunPlugin(context *cmd.Context) error {
	context.RawOutput()
	pluginName := context.Args[0]
	if os.Getenv("TSURU_PLUGIN_NAME") == pluginName {
		return cmd.ErrLookup
	}
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", pluginName)
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return cmd.ErrLookup
	}
	target, err := cmd.GetTarget()
	if err != nil {
		return err
	}
	token, err := cmd.ReadToken()
	if err != nil {
		return err
	}
	envs := os.Environ()
	tsuruEnvs := []string{
		"TSURU_TARGET=" + target,
		"TSURU_TOKEN=" + token,
		"TSURU_PLUGIN_NAME=" + pluginName,
	}
	envs = append(envs, tsuruEnvs...)
	opts := exec.ExecuteOptions{
		Cmd:    pluginPath,
		Args:   context.Args[1:],
		Stdout: context.Stdout,
		Stderr: context.Stderr,
		Stdin:  context.Stdin,
		Envs:   envs,
	}
	return Executor().Execute(opts)
}
