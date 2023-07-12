// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type OutputType string

const (
	// every OutputType should be mapped inside PrintInfo()
	JSON       OutputType = "JSON"
	PrettyJSON OutputType = "PrettyJSON"
	YAML       OutputType = "YAML"
	Table      OutputType = "Table"
)

func FormatAs(s string) OutputType {
	switch strings.ToLower(s) {
	case "json":
		return JSON
	case "pretty-json", "prettyjson":
		return PrettyJSON
	case "yaml":
		return YAML
	case "table":
		return Table
	default:
		return Table
	}
}

func Print(out io.Writer, data any, format OutputType) error {
	switch format {
	case JSON:
		return PrintJSON(out, data)
	case PrettyJSON:
		return PrintPrettyJSON(out, data)
	case YAML:
		return PrintYAML(out, data)
	case Table:
		return PrintTable(out, data)
	default:
		return fmt.Errorf("unknown format: %q", format)
	}
}

func PrintJSON(out io.Writer, data any) error {
	if data == nil {
		return nil
	}
	dataByte, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error converting to json: %w", err)
	}
	fmt.Fprintln(out, string(dataByte))
	return nil
}

func PrintPrettyJSON(out io.Writer, data any) error {
	if data == nil {
		return nil
	}
	dataByte, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error converting to json: %w", err)
	}
	fmt.Fprintln(out, string(dataByte))
	return nil
}

func PrintYAML(out io.Writer, data any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// yaml.v3 panics a lot: https://github.com/go-yaml/yaml/issues/954
			err = fmt.Errorf("error converting to yaml (panic): %v", r)
		}
	}()

	if data == nil {
		return nil
	}
	dataByte, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("error converting to yaml: %w", err)
	}
	_, err = out.Write(dataByte)
	return err
}
