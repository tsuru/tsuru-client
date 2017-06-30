package defaultconfig

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

//go:generate bash -c "rm -f configs.go && go run ./generator/main.go -o configs.go"

func DefaultTsuruConfig() map[string]interface{} {
	conf := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(Tsuru), conf)
	if err != nil {
		panic(fmt.Sprintf("invalid default config for tsuru api: %s", err))
	}
	return conf
}
