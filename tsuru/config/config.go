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
	lastChanges   time.Time // latest unsaved changes time
}

func newDefaultConf() *ConfigType {
	return &ConfigType{
		SchemaVersion: SchemaVersion,
		lastChanges:   time.Now().UTC(),
	}
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
		nowTimeStr := time.Now().UTC().Format("2006-01-02_15:04:05")
		backupFilePath := configPath + "." + nowTimeStr + ".bak"
		fmt.Fprintf(stderr, "Error parsing %q: %v\n", configPath, err)
		fmt.Fprintf(stderr, "Backing up current file to %q. A new configuration will be saved.\n", backupFilePath)
		filesystem().Rename(configPath, backupFilePath)
		return newDefaultConf()
	}

	config.lastChanges = config.LastUpdate
	return &config
}

func getConfig() *ConfigType {
	if config == nil {
		config = bootstrapConfig()
	}
	return config
}

func (c *ConfigType) hasChanges() bool {
	if c == nil {
		return false
	}
	return c.LastUpdate.Before(c.lastChanges)
}

func SaveChangesNoPrint() error {
	c := getConfig()
	if !c.hasChanges() {
		return nil
	}

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

func SaveChanges() {
	if err := SaveChangesNoPrint(); err != nil {
		fmt.Println("Warning:", err)
	}
}
