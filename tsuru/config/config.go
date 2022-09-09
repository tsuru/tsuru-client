package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tsuru/tsuru/cmd"
)

var (
	config        *ConfigType
	configPath    string        = cmd.JoinWithUserDir(".tsuru", "config.json")
	SchemaVersion string        = "0.1"
	stdout        io.ReadWriter = os.Stdout
	stderr        io.ReadWriter = os.Stderr
)

type ConfigType struct {
	SchemaVersion string
	LastUpdate    time.Time
	hasChanges    bool
}

func newDefaultConf() *ConfigType {
	return &ConfigType{
		SchemaVersion: SchemaVersion,
		LastUpdate:    time.Now().UTC(),
		hasChanges:    true,
	}
}

func bootstrapConfig() *ConfigType {
	file, err := filesystem().Open(configPath)
	if err != nil {
		return newDefaultConf()
	}
	defer file.Close()

	nowTimeStr := time.Now().UTC().Format("2006-01-02_15:04:05")
	backupFilePath := configPath + "." + nowTimeStr + ".bak"

	rawContent, err := io.ReadAll(file)
	if err != nil {
		fmt.Fprintf(stderr, "Could not read %q: %v\nContinuing without config", configPath, err)
		return nil
	}

	config := ConfigType{}
	if err := json.Unmarshal(rawContent, &config); err != nil {
		fmt.Fprintf(stderr, "Error parsing %q: %v\n", configPath, err)
		fmt.Fprintf(stderr, "Backing up current file to %q. A new configuration will be saved.\n", backupFilePath)
		filesystem().Rename(configPath, backupFilePath)
		return newDefaultConf()
	}
	return &config
}

func Config() *ConfigType {
	if config == nil {
		config = bootstrapConfig()
	}
	return config
}

func (c *ConfigType) HasChanges() bool {
	if c != nil && c.hasChanges {
		return true
	}
	return false
}

func (c *ConfigType) SaveChanges() error {
	if !c.HasChanges() {
		return nil
	}

	file, err := filesystem().OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
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
