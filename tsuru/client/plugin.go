// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec"
)

type Plugin struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (p *Plugin) Validate() error {
	if p.Name == "" && p.URL == "" {
		return fmt.Errorf("Zero value plugin (no Name nor URL)")
	}
	if p.Name == "" {
		return fmt.Errorf("Plugin.Name must not be empty (url: %q)", p.URL)
	}
	if p.URL == "" {
		return fmt.Errorf("Plugin.URL must not be empty (name: %q)", p.Name)
	}
	return nil
}

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
	if err := installPlugin(pluginName, pluginURL); err != nil {
		return err
	}

	fmt.Fprintf(context.Stdout, `Plugin "%s" successfully installed!`+"\n", pluginName)
	return nil
}

func installPlugin(pluginName, pluginURL string) error {
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
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("Invalid status code reading plugin: %d - %q", resp.StatusCode, string(data))
	}
	n, err := file.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return errors.New("Failed to install plugin.")
	}
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
		pluginPath += ".*"
		results, _ := filepath.Glob(pluginPath)
		if len(results) != 1 {
			return cmd.ErrLookup
		}
		pluginPath = results[0]
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

type PluginBundle struct {
	fs  *gnuflag.FlagSet
	url string
}

type BundleMetadata struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type BundleUrlPlatform struct {
	Darwin_ARM_64  *string `json:"darwin/arm64,omitempty"`
	Darwin_x86_64  *string `json:"darwin/x86_64,omitempty"`
	Linux_i386     *string `json:"linux/i386,omitempty"`
	Linux_x86_64   *string `json:"linux/x86_64,omitempty"`
	Windows_i386   *string `json:"windows/i386,omitempty"`
	Windows_x86_64 *string `json:"windows/x86_64,omitempty"`
}

type BundleManifest struct {
	SchemaVersion  string            `json:"schemaVersion,omitempty"`
	Metadata       BundleMetadata    `json:"metadata,omitempty"`
	Plugins        []Plugin          `json:"plugins,omitempty"`
	UrlPerPlatform BundleUrlPlatform `json:"urlPerPlatform,omitempty"`
}

func (PluginBundle) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "plugin-bundle",
		Usage: "plugin-bundle --url <bundle-url>",
		Desc:  `Syncs multiple plugins using a remote manifest containing a list of plugins.`,
	}
}

func (c *PluginBundle) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("plugin-bundle", gnuflag.ExitOnError)
		c.fs.StringVar(&c.url, "url", "", "URL for the remote plugin-bundle JSON manifest")
	}
	return c.fs
}

func (c *PluginBundle) Run(context *cmd.Context, client *cmd.Client) error {
	pluginsDir := cmd.JoinWithUserDir(".tsuru", "plugins")
	err := filesystem().MkdirAll(pluginsDir, 0755)
	if err != nil {
		return err
	}

	if c.url == "" {
		return fmt.Errorf("--url <url> is mandatory. See --help for usage")
	}

	manifestUrl := c.url
	resp, err := http.Get(manifestUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("Invalid status code reading plugin bundle: %d - %q", resp.StatusCode, string(data))
	}

	bundleManifest := BundleManifest{}
	if err := json.Unmarshal(data, &bundleManifest); err != nil {
		return fmt.Errorf("Error reading JSON manifest. Wrong syntax: %w", err)
	}

	// validate manifest structure
	for _, plugin := range bundleManifest.Plugins {
		if err := plugin.Validate(); err != nil {
			return fmt.Errorf("Error reading JSON manifest. Wrong plugin syntax: %w", err)
		}
	}

	var successfulPlugins []string
	failedPlugins := make(map[string]string)
	for _, plugin := range bundleManifest.Plugins {
		if err := installPlugin(plugin.Name, plugin.URL); err != nil {
			failedPlugins[plugin.Name] = fmt.Sprintf("%v", err)
		} else {
			successfulPlugins = append(successfulPlugins, plugin.Name)
		}
	}

	installPlatformPlugings(bundleManifest.Metadata.Name, bundleManifest.UrlPerPlatform)

	fmt.Fprintf(context.Stdout, "Successfully installed %d plugins: %s\n", len(successfulPlugins), strings.Join(successfulPlugins, ", "))
	if len(failedPlugins) > 0 {
		fmt.Fprintf(context.Stdout, "Failed to install %d plugins:\n", len(failedPlugins))
		for name, errStr := range failedPlugins {
			fmt.Fprintf(context.Stdout, "  %s: %s\n", name, errStr)
		}
		return fmt.Errorf("Bundle install has finished with errors.")
	}
	return nil
}

func installPlatformPlugings(name string, platforms BundleUrlPlatform) {
	if platforms.Darwin_ARM_64 != nil {
		installPlugin(name, *platforms.Darwin_ARM_64)
	}
	if platforms.Darwin_x86_64 != nil {
		installPlugin(name, *platforms.Darwin_x86_64)
	}
	if platforms.Linux_i386 != nil {
		installPlugin(name, *platforms.Linux_i386)
	}
	if platforms.Linux_x86_64 != nil {
		installPlugin(name, *platforms.Linux_i386)
	}
	if platforms.Windows_i386 != nil {
		installPlugin(name, *platforms.Linux_i386)
	}
	if platforms.Windows_x86_64 != nil {
		installPlugin(name, *platforms.Linux_i386)
	}
}
