// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	v2 "github.com/tsuru/tsuru-client/tsuru/cmd/v2"
	"github.com/tsuru/tsuru/exec"
)

type Plugin struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type PluginManifest struct {
	SchemaVersion  string                 `json:"SchemaVersion"`
	Metadata       PluginManifestMetadata `json:"Metadata"`
	URLPerPlatform map[string]string      `json:"UrlPerPlatform"`
}

type PluginManifestMetadata struct {
	Name    string `json:"Name"`
	Version string `json:"Version"`
}

func (p *Plugin) Validate() error {
	if p.Name == "" && p.URL == "" {
		return fmt.Errorf("zero value plugin (no Name nor URL)")
	}
	if p.Name == "" {
		return fmt.Errorf("plugin.Name must not be empty (url: %q)", p.URL)
	}
	if p.URL == "" {
		return fmt.Errorf("plugin.URL must not be empty (name: %q)", p.Name)
	}
	return nil
}

type PluginInstall struct{}

func (PluginInstall) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plugin-install",
		Usage:   "<plugin-name> <plugin-url>",
		Desc:    `Downloads the plugin file. It will be copied to [[$HOME/.tsuru/plugins]].`,
		MinArgs: 2,
	}
}

func (c *PluginInstall) Run(context *cmd.Context) error {
	pluginsDir := config.JoinWithUserDir(".tsuru", "plugins")
	err := config.Filesystem().MkdirAll(pluginsDir, 0755)
	if err != nil {
		return err
	}
	pluginName := context.Args[0]
	pluginURL := context.Args[1]
	if err := installPlugin(pluginName, pluginURL, 0); err != nil {
		return fmt.Errorf("error installing plugin %q: %w", pluginName, err)
	}

	fmt.Fprintf(context.Stdout, `Plugin "%s" successfully installed!`+"\n", pluginName)
	return nil
}

func installPlugin(pluginName, pluginURL string, level int) error {
	if level > 1 { // Avoid infinite recursion
		return fmt.Errorf("infinite Recursion detected, check if manifest.json is correct")
	}
	tmpDir, err := config.Filesystem().MkdirTemp(config.JoinWithUserDir(".tsuru", "plugins"), "tmpdir-*")
	if err != nil {
		return fmt.Errorf("could not create a tmpdir: %w", err)
	}
	defer config.Filesystem().RemoveAll(tmpDir)

	resp, err := http.Get(pluginURL)
	if err != nil {
		return fmt.Errorf("could not GET %q: %w", pluginURL, err)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("invalid status code reading plugin: %d - %q", resp.StatusCode, string(data))
	}

	// try to unmarshall manifest
	manifest := PluginManifest{}
	if err = json.Unmarshal(data, &manifest); err == nil {
		platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH) // get platform information
		if url, ok := manifest.URLPerPlatform[platform]; ok {
			return installPlugin(pluginName, url, level+1)
		}
		return fmt.Errorf("no plugin URL found for platform: %s", platform)
	}

	// Try to extract .tar.gz first, then .zip. Fallbacks to copy the content
	extractErr := extractTarGz(tmpDir, bytes.NewReader(data))
	if extractErr != nil {
		extractErr = extractZip(tmpDir, bytes.NewReader(data))
	}
	if extractErr != nil {
		file, err := config.Filesystem().OpenFile(filepath.Join(tmpDir, pluginName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		n, err := file.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write file content: %w", err)
		}
		if n != len(data) {
			return fmt.Errorf("incomplete write")
		}
	}

	executablePath := findExecutablePlugin(tmpDir, pluginName)
	if executablePath == "" {
		return fmt.Errorf("the downloaded plugin content is invalid")
	}

	if fstat, err1 := config.Filesystem().Stat(executablePath); err1 == nil {
		fmode := fstat.Mode()
		os.Chmod(executablePath, fmode|0111) // make this file executable
	}

	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", pluginName)
	if extractErr == nil {
		if _, err := config.Filesystem().Stat(pluginPath); err == nil {
			config.Filesystem().RemoveAll(pluginPath)
		}
		if err := config.Filesystem().Rename(tmpDir, pluginPath); err != nil {
			return fmt.Errorf("could not move tmpDir: %w", err)
		}
		os.Chmod(pluginPath, 0755) // this is a directory with an executable inside
	} else {
		if err := copyFile(executablePath, pluginPath); err != nil {
			return fmt.Errorf("could not write plugin file: %w", err)
		}
	}

	return nil
}

func findExecutablePlugin(basePath, pluginName string) (execPath string) {
	testPathGlobs := []string{
		filepath.Join(basePath, pluginName),
		filepath.Join(basePath, pluginName, pluginName),
		filepath.Join(basePath, pluginName, pluginName+".*"),
		filepath.Join(basePath, pluginName+".*"),
	}
	for _, pathGlob := range testPathGlobs {
		var fStat fs.FileInfo
		var err error
		execPath = pathGlob
		if fStat, err = config.Filesystem().Stat(pathGlob); err != nil {
			files, _ := filepath.Glob(pathGlob)
			if len(files) != 1 {
				continue
			}
			execPath = files[0]
			fStat, err = config.Filesystem().Stat(execPath)
		}
		if err != nil || fStat.IsDir() || !fStat.Mode().IsRegular() {
			continue
		}
		return execPath
	}
	return ""
}

func FindPlugins() []string {
	basePath := config.JoinWithUserDir(".tsuru", "plugins")

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil
	}
	var plugins []string
	for _, entry := range entries {
		entryPath := filepath.Join(basePath, entry.Name())
		fStat, err := config.Filesystem().Stat(entryPath)
		if err != nil {
			continue
		}
		if fStat.IsDir() {
			execPath := findExecutablePlugin(entryPath, entry.Name())
			if execPath != "" {
				plugins = append(plugins, entry.Name())
			}
		} else if fStat.Mode().IsRegular() {
			plugins = append(plugins, entry.Name())
		}
	}
	return plugins
}

func copyFile(src, dst string) error {
	sourceFile, err := config.Filesystem().Open(src)
	if err != nil {
		return fmt.Errorf("failed to open src file: %w", err)
	}
	defer sourceFile.Close()
	sourceStat, err := config.Filesystem().Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	targetFile, err := config.Filesystem().OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to open dest file: %w", err)
	}
	defer targetFile.Close()

	n, err := io.Copy(targetFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	if n != sourceStat.Size() {
		return fmt.Errorf("incomplete write! This file may be corrupted")
	}
	return nil
}

func extractTarGz(basePath string, gzipStream io.Reader) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(uncompressedStream)
	var header *tar.Header
	for {
		header, err = tarReader.Next()
		if err != nil {
			break
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err = config.Filesystem().Mkdir(filepath.Join(basePath, header.Name), fs.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("ExtractTarGz: Mkdir() failed: %w", err)
			}
		case tar.TypeReg:
			outFile, err1 := config.Filesystem().OpenFile(filepath.Join(basePath, header.Name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.FileMode(header.Mode))
			if err1 != nil {
				return fmt.Errorf("ExtractTarGz: Create() failed: %w", err1)
			}

			if _, err = io.Copy(outFile, tarReader); err != nil {
				// outFile.Close error omitted as Copy error is more interesting at this point
				outFile.Close()
				return fmt.Errorf("ExtractTarGz: Copy() failed: %w", err)
			}
			if err = outFile.Close(); err != nil {
				return fmt.Errorf("ExtractTarGz: Close() failed: %w", err)
			}
		default:
			return fmt.Errorf("ExtractTarGz: unsupported type: %b in %s", header.Typeflag, header.Name)
		}
	}
	if err != io.EOF {
		return fmt.Errorf("ExtractTarGz: Next() failed: %w", err)
	}
	return nil
}

func extractZip(basePath string, source io.Reader) error {
	zipData, err := io.ReadAll(source)
	if err != nil {
		return fmt.Errorf("could not read from source: %w", err)
	}
	br := bytes.NewReader(zipData)
	z, err := zip.NewReader(br, int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("could not read zip from source: %w", err)
	}

	for _, f := range z.File {
		fPath := filepath.Join(basePath, f.Name)
		if f.FileInfo().IsDir() {
			config.Filesystem().MkdirAll(fPath, f.Mode().Perm())
			continue
		}

		freader, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open %q from zip: %w", f.Name, err)
		}
		defer freader.Close()

		fDest, err := config.Filesystem().OpenFile(fPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode().Perm())
		if err != nil {
			return fmt.Errorf("could not open %q for writing: %w", fPath, err)
		}
		defer fDest.Close()

		if _, err := io.Copy(fDest, freader); err != nil {
			return fmt.Errorf("could not write content to %q: %w", fPath, err)
		}
	}
	return nil
}

type PluginRemove struct{}

func (PluginRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plugin-remove",
		Usage:   "<plugin-name>",
		Desc:    "Removes a previously installed tsuru plugin.",
		MinArgs: 1,
	}
}

func (c *PluginRemove) Run(context *cmd.Context) error {
	pluginName := context.Args[0]
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", pluginName)
	err := config.Filesystem().Remove(pluginPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(context.Stdout, `Plugin "%s" successfully removed!`+"\n", pluginName)
	return nil
}

type PluginList struct{}

func (PluginList) Info() *cmd.Info {
	return &cmd.Info{
		Name: "plugin-list",
		Desc: "List installed tsuru plugins.",
	}
}

func (c *PluginList) Run(context *cmd.Context) error {
	pluginsPath := config.JoinWithUserDir(".tsuru", "plugins")
	plugins, _ := os.ReadDir(pluginsPath)
	for _, p := range plugins {
		fmt.Fprintln(context.Stdout, p.Name())
	}
	return nil
}

func RunPlugin(context *cmd.Context) error {
	context.RawOutput()

	if len(context.Args) == 0 {
		return cmd.ErrLookup
	}
	pluginName := context.Args[0]

	return runPlugin(context, pluginName, context.Args[1:])
}

func runPlugin(context *cmd.Context, pluginName string, args []string) error {
	if os.Getenv("TSURU_PLUGIN_NAME") == pluginName {
		return cmd.ErrLookup
	}
	pluginPath := findExecutablePlugin(config.JoinWithUserDir(".tsuru", "plugins"), pluginName)
	if pluginPath == "" {
		return cmd.ErrLookup
	}
	target, err := config.GetTarget()
	if err != nil {
		return err
	}
	token, err := config.DefaultTokenProvider.Token()
	if err != nil {
		return err
	}
	envs := os.Environ()
	tsuruEnvs := []string{
		"TSURU_TARGET=" + target,
		"TSURU_TOKEN=" + token,
		"TSURU_PLUGIN_NAME=" + pluginName,
		"TSURU_TABLE_COLOR=" + v2.TableColor(),
		"TSURU_TABLE_UTF8=" + strconv.FormatBool(v2.TableUTF8()),
	}

	if v2.ColorDisabled() {
		tsuruEnvs = append(tsuruEnvs, "NO_COLOR=1")
	}

	pager, pagerFound := v2.Pager()
	if pagerFound {
		tsuruEnvs = append(tsuruEnvs, "TSURU_PAGER="+pager)
	}

	envs = append(envs, tsuruEnvs...)
	opts := exec.ExecuteOptions{
		Cmd:    pluginPath,
		Args:   args,
		Stdout: context.Stdout,
		Stderr: context.Stderr,
		Stdin:  context.Stdin,
		Envs:   envs,
	}
	return Executor().Execute(opts)
}

var _ cmd.Command = &ExecutePlugin{}

type ExecutePlugin struct {
	PluginName string
}

func (e ExecutePlugin) Info() *cmd.Info {
	return &cmd.Info{
		Name:    e.PluginName,
		Usage:   e.PluginName,
		Desc:    "Executes the " + e.PluginName + " plugin.",
		MinArgs: cmd.ArbitraryArgs,

		GroupID:             "plugin",
		OnlyAppendOnRoot:    true,
		DisableFlagParsing:  true,
		SilenceUsage:        true,
		ParseFirstFlagsOnly: true,
	}
}

func (c *ExecutePlugin) Run(context *cmd.Context) error {
	context.RawOutput()
	return runPlugin(context, c.PluginName, context.Args)
}

type PluginBundle struct {
	fs  *pflag.FlagSet
	url string
}
type BundleManifest struct {
	Plugins []Plugin
}

func (PluginBundle) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "plugin-bundle",
		Usage: "--url <bundle-url>",
		Desc:  `Syncs multiple plugins using a remote manifest containing a list of plugins.`,
	}
}

func (c *PluginBundle) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("plugin-bundle", pflag.ExitOnError)
		c.fs.StringVar(&c.url, "url", "", "URL for the remote plugin-bundle JSON manifest")
	}
	return c.fs
}

func (c *PluginBundle) Run(context *cmd.Context) error {
	pluginsDir := config.JoinWithUserDir(".tsuru", "plugins")
	err := config.Filesystem().MkdirAll(pluginsDir, 0755)
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("invalid status code reading plugin bundle: %d - %q", resp.StatusCode, string(data))
	}

	bundleManifest := BundleManifest{}
	if err := json.Unmarshal(data, &bundleManifest); err != nil {
		return fmt.Errorf("error reading JSON manifest. Wrong syntax: %w", err)
	}

	// validate manifest structure
	for _, plugin := range bundleManifest.Plugins {
		if err := plugin.Validate(); err != nil {
			return fmt.Errorf("error reading JSON manifest. Wrong plugin syntax: %w", err)
		}
	}

	var successfulPlugins []string
	failedPlugins := make(map[string]string)
	for _, plugin := range bundleManifest.Plugins {
		if err := installPlugin(plugin.Name, plugin.URL, 0); err != nil {
			failedPlugins[plugin.Name] = fmt.Sprintf("%v", err)
		} else {
			successfulPlugins = append(successfulPlugins, plugin.Name)
		}
	}

	fmt.Fprintf(context.Stdout, "Successfully installed %d plugins: %s\n", len(successfulPlugins), strings.Join(successfulPlugins, ", "))
	if len(failedPlugins) > 0 {
		fmt.Fprintf(context.Stdout, "Failed to install %d plugins:\n", len(failedPlugins))
		for name, errStr := range failedPlugins {
			fmt.Fprintf(context.Stdout, "  %s: %s\n", name, errStr)
		}
		return fmt.Errorf("bundle install has finished with errors")
	}
	return nil
}
