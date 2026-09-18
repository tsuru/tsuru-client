package formatter

import (
	"io"

	"github.com/ghodss/yaml"
)

func YAML(writer io.Writer, data []byte) error {
	output, err := yaml.JSONToYAML(data)
	if err != nil {
		return err
	}
	if _, err := writer.Write(output); err != nil {
		return err
	}
	return nil
}
