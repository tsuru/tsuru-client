package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tsuru/tsuru/cmd"
)

var (
	privConfig                     *ConfigType
	configPath                     string           = cmd.JoinWithUserDir(".tsuru", "config.json")
	SchemaVersion                  string           = "0.1"
	stdout                         io.ReadWriter    = os.Stdout
	stderr                         io.ReadWriter    = os.Stderr
	nowUTC                         func() time.Time = func() time.Time { return time.Now().UTC() } // so we can test time-dependent sh!t
	defaultLocalTimeout            time.Duration    = 1 * time.Second
	defaultForceCheckAfterDuration time.Duration    = 72 * time.Hour
)

// ConfigType is the main config, serialized to ~/.tsuru/config.json
type ConfigType struct {
	SchemaVersion   string
	LastUpdate      time.Time
	originalContent []byte // used to detect changes

	// ---- public confs ----
	ClientSelfUpdater ClientSelfUpdater
}

func newDefaultConf() *ConfigType {
	return &ConfigType{
		SchemaVersion: SchemaVersion,
		ClientSelfUpdater: ClientSelfUpdater{
			ForceCheckAfter: nowUTC().Add(defaultForceCheckAfterDuration),
		},
	}
}

func (c *ConfigType) saveOriginalContent() {
	originalContent, _ := json.Marshal(c)
	c.originalContent = originalContent
}

func bootstrapConfig() *ConfigType {
	file, err := filesystem().Open(configPath)
	if os.IsNotExist(err) {
		return newDefaultConf()
	}
	if err != nil {
		fmt.Fprintf(stderr, "Could not read %q: %v\nContinuing without config", configPath, err)
		return nil
	}
	defer file.Close()

	rawContent, err := io.ReadAll(file)
	if err != nil {
		fmt.Fprintf(stderr, "Could not read %q: %v\nContinuing without config", configPath, err)
		return nil
	}

	config := ConfigType{}
	if err := json.Unmarshal(rawContent, &config); err != nil {
		nowTimeStr := nowUTC().Format("2006-01-02_15:04:05")
		backupFilePath := configPath + "." + nowTimeStr + ".bak"
		fmt.Fprintf(stderr, "Error parsing %q: %v\n", configPath, err)
		fmt.Fprintf(stderr, "Backing up current file to %q. A new configuration will be saved.\n", backupFilePath)
		filesystem().Rename(configPath, backupFilePath)
		return newDefaultConf()
	}

	config.saveOriginalContent()
	return &config
}

// GetConfig() returns a *ConfigType singleton.
func GetConfig() *ConfigType {
	if privConfig == nil {
		privConfig = bootstrapConfig()
	}
	return privConfig
}

func (c *ConfigType) hasChanges() bool {
	if c == nil {
		return false
	}
	jsonConfig, _ := json.Marshal(c)
	return !bytes.Equal(c.originalContent, jsonConfig)
}

func SaveChangesNoPrint() error {
	c := GetConfig()
	if !c.hasChanges() {
		return nil
	}
	c.LastUpdate = nowUTC()

	file, err := filesystem().OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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
	timeout := defaultLocalTimeout
	if timeoutStr := os.Getenv("TSURU_CLIENT_LOCAL_TIMEOUT"); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = duration
		}
	}

	c := make(chan bool, 1)
	go func() {
		if err := SaveChangesNoPrint(); err != nil {
			fmt.Fprintf(stderr, "Warning: Could not save config file: %v\n", err)
		}
		c <- true
	}()

	select {
	case <-c:
	case <-time.After(timeout):
		fmt.Fprintln(stderr, "Warning: Could not save config within the specified timeout. (check filesystem and/or change TSURU_CLIENT_LOCAL_TIMEOUT env)")
	}
}

// ClientSelfUpdater saves configuration regarding self updating the client
type ClientSelfUpdater struct {
	LastCheck       time.Time
	ForceCheckAfter time.Time
	SnoozeUntil     time.Time
}
