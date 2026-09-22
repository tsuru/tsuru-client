package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
)

type AppConfig struct {
	Aliases Aliases `json:"aliases"`
}

type Aliases []*Alias

type Alias struct {
	Source []string `json:"source"`
	Target []string `json:"target"`
}

// MatchesSource checks if the provided source matches the alias source.
func (a *Alias) MatchesSource(source []string) bool {
	for i, item := range a.Source {
		if item != source[i] {
			return false
		}
	}
	return true
}

// Has returns the alias matching the provided source, if it exists.
func (a *Aliases) Has(source []string) (*Alias, bool) {
	for _, alias := range *a {
		if alias.MatchesSource(source) {
			return alias, true
		}
	}

	return nil, false
}

// Add will add an alias to the config, updating the entry if it already exists.
func (a *Aliases) Add(source, target []string) {
	if len(source) == 0 || len(target) == 0 {
		return
	}

	alias, exists := a.Has(source)
	if exists {
		alias.Target = target
		return
	}

	alias = &Alias{
		Source: source,
		Target: target,
	}
	*a = append(*a, alias)
}

// Remove will remove an alias to the config, if it exists.
func (a *Aliases) Remove(source []string) {
	for i, alias := range *a {
		if alias.MatchesSource(source) {
			*a = append((*a)[:i], (*a)[i+1:]...)
			break
		}
	}
}

func configFilePath() (string, error) {
	if envPath := os.Getenv("TSURU_CONFIG_FILE"); envPath != "" {
		return envPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(home, ".config", "tsuru")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configPath, "config.json"), nil
}

func GetConfig() (*AppConfig, error) {
	config := AppConfig{
		Aliases: make(Aliases, 0),
	}

	configPath, err := configFilePath()
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &config, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (config *AppConfig) Save() error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, content, 0644)
}

type ConfigAlias struct {
	fs     *pflag.FlagSet
	delete bool
}

func (c *ConfigAlias) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("config-alias", pflag.ExitOnError)

		desc := "remove the provided alias instead of adding it"
		c.fs.BoolVarP(&c.delete, "delete", "d", false, desc)
	}
	return c.fs
}

func (c *ConfigAlias) Run(ctx *cmd.Context) error {
	config, err := GetConfig()
	if err != nil {
		return err
	}

	if c.delete {
		source := strings.Split(ctx.Args[0], " ")
		config.Aliases.Remove(source)
	} else {
		source := strings.Split(ctx.Args[0], " ")
		target := strings.Split(ctx.Args[1], " ")
		config.Aliases.Add(source, target)
	}

	return config.Save()
}

func (ConfigAlias) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "config-alias",
		Usage: "<alias> <command> [--delete/-d]",
		Desc: `Add an alias to a tsuru command. This allows you to create custom shortcuts for tsuru commands, making it easier to use the CLI.
Examples:

# make 'deploy' map to 'app deploy'
$ tsuru config alias "app deploy" "deploy"
$ tsuru deploy <app-name>

# create a shorthand for 'app info':
$ tsuru config alias "app info" "ai"
$ tsuru ai <app-name>
`,
	}
}
