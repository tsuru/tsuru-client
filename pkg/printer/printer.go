// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type OutputFormat string

const (
	// every OutputType should be mapped inside PrintInfo()
	CompactJSON OutputFormat = "compatc-json"
	PrettyJSON  OutputFormat = "json"
	YAML        OutputFormat = "yaml"
	Table       OutputFormat = "table"
)

var _ pflag.Value = (*OutputFormat)(nil)

func OutputFormatCompletionHelp() []string {
	return []string{
		CompactJSON.ToString() + "\toutput as compact JSON format (no newlines)",
		PrettyJSON.ToString() + "\toutput as JSON (PrettyJSON)",
		YAML.ToString() + "\toutput as YAML",
		Table.ToString() + "\toutput as Human readable table",
	}
}

func (o OutputFormat) ToString() string {
	return string(o)
}
func (o *OutputFormat) String() string {
	return string(*o)
}
func (o *OutputFormat) Set(v string) error {
	var err error
	*o, err = FormatAs(v)
	return err
}
func (e *OutputFormat) Type() string {
	return "OutputFormat"
}

func FormatAs(s string) (OutputFormat, error) {
	switch strings.ToLower(s) {
	case "compact-json", "compactjson":
		return CompactJSON, nil
	case "json", "pretty-json", "prettyjson":
		return PrettyJSON, nil
	case "yaml":
		return YAML, nil
	case "table":
		return Table, nil
	default:
		return Table, fmt.Errorf("must be one of: json, compact-json, yaml, table")
	}
}

// Print will print the data in the given format.
// If the format is not supported, it will return an error.
// If the format is Table, it will try to convert the data to a human readable format. (see pkg/converter)
func Print(out io.Writer, data any, format OutputFormat) error {
	switch format {
	case CompactJSON:
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
