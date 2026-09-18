package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	serviceTypes "github.com/tsuru/tsuru/types/service"
)

type ServiceManifestGet struct {
	fs     *pflag.FlagSet
	json   bool
	yaml   bool
	output string
}

func (*ServiceManifestGet) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-manifest-get",
		Usage: "<service> [--json | --yaml] [-o/--output FILE]",
		Desc: `Shows the stored service permissions manifest. Requires API 1.31 or later.
Without flags, displays settings and an operations table.
Use --json or --yaml for structured stdout, or --output to export a file.
The output format is inferred from .json, .yaml or .yml unless explicitly selected.
Other file extensions require --json or --yaml. Existing files are overwritten.

Examples:
  tsuru service manifest get my-service
  tsuru service manifest get my-service --json
  tsuru service manifest get my-service --yaml
  tsuru service manifest get my-service -o manifest.yaml`,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *ServiceManifestGet) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("service-manifest-get", pflag.ContinueOnError)
		c.fs.BoolVar(&c.json, "json", false, "Show JSON")
		c.fs.BoolVar(&c.yaml, "yaml", false, "Show YAML")
		c.fs.StringVarP(&c.output, "output", "o", "", "Write the manifest to a file")
	}
	return c.fs
}

func (c *ServiceManifestGet) outputFormat() (string, error) {
	if c.json && c.yaml {
		return "", fmt.Errorf("--json and --yaml cannot be used together")
	}
	if c.json {
		return "json", nil
	}
	if c.yaml {
		return "yaml", nil
	}
	if c.output == "" {
		return "human", nil
	}
	switch strings.ToLower(filepath.Ext(c.output)) {
	case ".json":
		return "json", nil
	case ".yaml", ".yml":
		return "yaml", nil
	default:
		return "", fmt.Errorf("output filename must end in .json, .yaml or .yml, or specify --json or --yaml")
	}
}

func (c *ServiceManifestGet) Run(ctx *cmd.Context) error {
	if len(ctx.Args) != 1 {
		return &cmd.UsageError{Err: fmt.Errorf("expected exactly one service argument")}
	}
	format, err := c.outputFormat()
	if err != nil {
		return err
	}
	resp, err := serviceManifestRequest(http.MethodGet, ctx.Args[0], nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var raw json.RawMessage
	if err = json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return err
	}
	var manifest *serviceTypes.ServiceManifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if format == "human" {
		return showServiceManifest(ctx.Stdout, ctx.Args[0], manifest)
	}
	var output bytes.Buffer
	if format == "json" {
		err = formatter.JSON(&output, raw)
	} else {
		err = formatter.YAML(&output, raw)
	}
	if err != nil {
		return err
	}
	if c.output != "" {
		return os.WriteFile(c.output, output.Bytes(), 0o644)
	}
	_, err = ctx.Stdout.Write(output.Bytes())
	return err
}

func showServiceManifest(w io.Writer, name string, manifest *serviceTypes.ServiceManifest) error {
	var output bytes.Buffer
	fmt.Fprintf(&output, "Service: %s\n", name)
	if manifest == nil {
		fmt.Fprintln(&output, "No manifest configured.")
	} else {
		fmt.Fprintf(&output, "Enabled: %t\nStrict actions: %t\nLegacy compatibility: %t\n\n", manifest.Enabled, manifest.StrictActions, manifest.LegacyCompat)
		if len(manifest.Operations) == 0 {
			fmt.Fprintln(&output, "No operations configured.")
		} else {
			table := tablecli.NewTable()
			table.Headers = tablecli.Row{"Method", "Path", "Action"}
			for _, op := range manifest.Operations {
				table.AddRow(tablecli.Row{op.Method, op.Path, op.Action})
			}
			output.Write(table.Bytes())
		}
	}
	_, err := w.Write(output.Bytes())
	return err
}

type ServiceManifestSet struct{}

func (*ServiceManifestSet) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "service-manifest-set",
		Usage: "<service> <filename>",
		Desc: `Replaces the stored service permissions manifest with a JSON or YAML file.
Requires API 1.31 or later. Operation validation and grant conflicts are checked by the server.
This is the permissions manifest, separate from the service-create/service-update configuration.

Examples:
  tsuru service manifest set my-service manifest.json
  tsuru service manifest set my-service manifest.yaml

Example YAML:
  enabled: true
  strict_actions: true
  legacy_compat: false
  operations:
    - method: POST
      path: /rules/{ruleId}/sync
      action: rules.sync

Equivalent JSON:
  {"enabled":true,"strict_actions":true,"legacy_compat":false,"operations":[{"method":"POST","path":"/rules/{ruleId}/sync","action":"rules.sync"}]}`,
		MinArgs: 2,
		MaxArgs: 2,
	}
}

func (*ServiceManifestSet) Run(ctx *cmd.Context) error {
	if len(ctx.Args) != 2 {
		return &cmd.UsageError{Err: fmt.Errorf("expected a service and an input filename")}
	}
	data, err := os.ReadFile(ctx.Args[1])
	if err != nil {
		return err
	}
	data, err = parseServiceManifest(data)
	if err != nil {
		return fmt.Errorf("invalid manifest %q: %w", ctx.Args[1], err)
	}
	resp, err := serviceManifestRequest(http.MethodPut, ctx.Args[0], bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = fmt.Fprintf(ctx.Stdout, "Manifest for service %q successfully updated.\n", ctx.Args[0])
	return err
}

func parseServiceManifest(data []byte) ([]byte, error) {
	// Since JSON is a subset of YAML, passing JSON through this method should be a no-op.
	converted, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, err
	}
	if len(converted) == 0 || converted[0] != '{' {
		return nil, fmt.Errorf("manifest must be an object")
	}
	var manifest serviceTypes.ServiceManifest
	if err := json.Unmarshal(converted, &manifest); err != nil {
		return nil, err
	}
	return converted, nil
}

func serviceManifestRequest(method, service string, body io.Reader) (*http.Response, error) {
	u, err := config.GetURLVersion("1.31", "/services/"+url.PathEscape(service)+"/manifest")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return tsuruHTTP.AuthenticatedClient.Do(req)
}
