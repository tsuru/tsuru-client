package client

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAliases(t *testing.T) {
	a := Aliases{}
	src := []string{"app", "deploy"}
	target := []string{"deploy"}

	a.Add(src, target)
	if len(a) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(a))
	}

	found, ok := a.Has(src)
	if !ok {
		t.Fatal("alias not found")
	}
	if !reflect.DeepEqual(found.Target, target) {
		t.Errorf("expected target %v, got %v", target, found.Target)
	}

	a.Remove(src)
	if len(a) != 0 {
		t.Errorf("expected 0 aliases, got %d", len(a))
	}
}

func TestConfigFilePath(t *testing.T) {
	os.Setenv("TSURU_CONFIG_FILE", "/tmp/tsuru/config.json")
	defer os.Unsetenv("TSURU_CONFIG_FILE")

	path, err := configFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/tmp/tsuru/config.json" {
		t.Errorf("expected /tmp/tsuru/config.json, got %s", path)
	}
}

func TestSaveAndGetConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tsuru-test")
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loadedConfig, err := GetConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(loadedConfig.Aliases) != 1 {
		t.Errorf("expected 1 alias, got %d", len(loadedConfig.Aliases))
	}
	if !reflect.DeepEqual(loadedConfig.Aliases[0].Target, []string{"cmd"}) {
		t.Errorf("expected alias cmd, got %v", loadedConfig.Aliases[0].Target)
	}
}
