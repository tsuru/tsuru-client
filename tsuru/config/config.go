// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	defaultLocalTimeout      time.Duration = 1 * time.Second
	defaultLatestManifestURL string        = "https://github.com/tsuru/tsuru-client/releases/latest/download/metadata.json"
)

var (
	privConfig         *ConfigType
	configPath         string           = JoinWithUserDir(".tsuru", "config.json")
	SchemaVersion      string           = "0.1"
	stdout             io.ReadWriter    = os.Stdout
	stderr             io.ReadWriter    = os.Stderr
	nowUTC             func() time.Time = func() time.Time { return time.Now().UTC() } // so we can test time-dependent features
	clientLocalTimeout time.Duration
)

func init() {
	clientLocalTimeout = defaultLocalTimeout
	if timeoutStr := os.Getenv("TSURU_CLIENT_LOCAL_TIMEOUT"); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			clientLocalTimeout = duration
		} else {
			fmt.Fprintf(stderr, "ERROR: TSURU_CLIENT_LOCAL_TIMEOUT could not be parsed. Using default: %q\n", defaultLocalTimeout)
		}
	}
}

// ConfigType is the main config, serialized to ~/.tsuru/config.json
type ConfigType struct {
	SchemaVersion   string
	LastUpdate      time.Time
	originalContent string // used to detect changes

	// ---- public confs ----
	ClientSelfUpdater ClientSelfUpdater
}

func newDefaultConf() *ConfigType {
	return &ConfigType{
		SchemaVersion: SchemaVersion,
		ClientSelfUpdater: ClientSelfUpdater{
			LatestManifestURL: defaultLatestManifestURL,
		},
	}
}

func bootstrapConfig() *ConfigType {
	file, err := Filesystem().Open(configPath)
	if os.IsNotExist(err) {
		return newDefaultConf()
	}
	if err != nil {
		fmt.Fprintf(stderr, "Could not read %q: %v\nContinuing without config\n", configPath, err)
		return nil
	}
	defer file.Close()

	rawContent, err := io.ReadAll(file)
	if err != nil {
		fmt.Fprintf(stderr, "Could not read %q: %v\nContinuing without config\n", configPath, err)
		return nil
	}

	config := ConfigType{}
	if err := json.Unmarshal(rawContent, &config); err != nil {
		nowTimeStr := nowUTC().Format("2006-01-02_15:04:05")
		backupFilePath := configPath + "." + nowTimeStr + ".bak"
		fmt.Fprintf(stderr, "Error parsing %q: %v\n", configPath, err)
		fmt.Fprintf(stderr, "Backing up current file to %q. A new configuration will be saved.\n", backupFilePath)
		if err := Filesystem().Rename(configPath, backupFilePath); err != nil {
			fmt.Fprintf(stderr, "Error renaming the file: %v\n", err)
		}
		return newDefaultConf()
	}

	config.originalContent = string(rawContent)

	// Convert any older config
	// ...

	// Mandatory fields
	if config.ClientSelfUpdater.LatestManifestURL == "" {
		config.ClientSelfUpdater.LatestManifestURL = defaultLatestManifestURL
	}

	return &config
}

// GetConfig() returns a *ConfigType singleton.
func GetConfig() *ConfigType {
	if privConfig == nil {
		privConfig = bootstrapConfig()
	}

	return privConfig
}

func (c *ConfigType) hasChanges() (bool, error) {
	if c == nil {
		return false, nil
	}
	jsonConfig, err := json.Marshal(c)
	if err != nil {
		return false, fmt.Errorf("Configuration (ConfigType) could not be marshaled to JSON: %w", err)
	}
	return c.originalContent != string(jsonConfig), nil
}

func SaveChangesNoPrint() error {
	c := GetConfig()
	hasChanges, err := c.hasChanges()
	if err != nil {
		return err
	}
	if !hasChanges {
		return nil
	}
	c.LastUpdate = nowUTC()

	file, err := Filesystem().OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Could not open file %q for write: %w", configPath, err)
	}
	jsonData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("Could not convert config to JSON: %w", err)
	}
	_, err = file.Write(jsonData)
	if err != nil {
		return fmt.Errorf("Got errors when saving config: %w", err)
	}
	return nil
}

// SaveChangesWithTimeout will try to save changes on ~/.tsuru/config.json and
// it will timeout after 1s (default). Timeout is overridden from env TSURU_CLIENT_LOCAL_TIMEOUT
func SaveChangesWithTimeout() {
	c := make(chan bool, 1)
	go func() {
		if err := SaveChangesNoPrint(); err != nil {
			fmt.Fprintf(stderr, "Warning: Could not save config file: %v\n", err)
		}
		c <- true
	}()

	select {
	case <-c:
	case <-time.After(clientLocalTimeout):
		fmt.Fprintln(stderr, "Warning: Could not save config within the specified timeout. (check filesystem and/or change TSURU_CLIENT_LOCAL_TIMEOUT env)")
	}
}

// ClientSelfUpdater saves configuration regarding self updating the client
type ClientSelfUpdater struct {
	LatestManifestURL string
	LastCheck         time.Time
}
