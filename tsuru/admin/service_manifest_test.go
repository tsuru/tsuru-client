package admin

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
)

const manifestFixture = `{
  "enabled": true,
  "strict_actions": true,
  "legacy_compat": false,
  "operations": [
    {
      "method": "POST",
      "path": "/rules/{ruleId}/sync",
      "action": "rules.sync"
    },
    {
      "method": "GET",
      "path": "/rules",
      "action": "rules.list"
    }
  ],
  "future_field": "preserved"
}`

func setupManifestClient(t *testing.T, transport http.RoundTripper) {
	t.Helper()
	t.Setenv("TSURU_TARGET", "http://localhost")
	previous := tsuruHTTP.AuthenticatedClient
	t.Cleanup(func() { tsuruHTTP.AuthenticatedClient = previous })
	(&S{}).setupFakeTransport(transport)
}

func TestServiceManifestGetInfo(t *testing.T) {
	info := (&ServiceManifestGet{}).Info()
	require.Equal(t, "service-manifest-get", info.Name)
	require.Equal(t, "<service> [--json | --yaml] [-o/--output FILE]", info.Usage)
	require.Equal(t, 1, info.MinArgs)
	require.Equal(t, 1, info.MaxArgs)
	require.NotEmpty(t, info.Desc)
}

func TestServiceManifestSetInfo(t *testing.T) {
	info := (&ServiceManifestSet{}).Info()
	require.Equal(t, "service-manifest-set", info.Name)
	require.Equal(t, "<service> <filename>", info.Usage)
	require.Equal(t, 2, info.MinArgs)
	require.Equal(t, 2, info.MaxArgs)
	require.NotEmpty(t, info.Desc)
}

func TestServiceManifestGetHumanOutput(t *testing.T) {
	previous := tablecli.TableConfig.UseTabWriter
	tablecli.TableConfig.UseTabWriter = false
	t.Cleanup(func() { tablecli.TableConfig.UseTabWriter = previous })
	setupManifestClient(t, cmdtest.Transport{Status: http.StatusOK, Message: manifestFixture})
	var stdout bytes.Buffer
	err := (&ServiceManifestGet{}).Run(&cmd.Context{Args: []string{"mysql"}, Stdout: &stdout})
	require.NoError(t, err)
	expected := `Service: mysql
Enabled: true
Strict actions: true
Legacy compatibility: false

+--------+----------------------+------------+
| Method | Path                 | Action     |
+--------+----------------------+------------+
| POST   | /rules/{ruleId}/sync | rules.sync |
| GET    | /rules               | rules.list |
+--------+----------------------+------------+
`
	require.Equal(t, expected, stdout.String())
}

func TestServiceManifestGet(t *testing.T) {
	for _, test := range []struct {
		name, response string
		flags          []string
		contains       []string
	}{
		{"human", manifestFixture, nil, []string{"Service: mysql", "Enabled: true", "Strict actions: true", "Legacy compatibility: false", "Method", "Path", "Action", "POST", "/rules/{ruleId}/sync", "rules.sync"}},
		{"absent", "null", nil, []string{"No manifest configured."}},
		{"empty operations", `{"enabled":false,"operations":[]}`, nil, []string{"Enabled: false", "No operations configured."}},
		{"json", manifestFixture, []string{"--json"}, []string{"\n  \"enabled\": true", `"strict_actions": true`, `"legacy_compat": false`}},
		{"yaml", manifestFixture, []string{"--yaml"}, []string{"enabled: true", "strict_actions: true", "legacy_compat: false"}},
		{"absent json", "null", []string{"--json"}, []string{"null\n"}},
		{"absent yaml", "null", []string{"--yaml"}, []string{"null\n"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			setupManifestClient(t, &cmdtest.ConditionalTransport{
				Transport: cmdtest.Transport{Status: http.StatusOK, Message: test.response},
				CondFunc: func(r *http.Request) bool {
					called = true
					return r.Method == http.MethodGet && r.URL.Path == "/1.31/services/mysql/manifest"
				},
			})
			var stdout bytes.Buffer
			command := &ServiceManifestGet{}
			require.NoError(t, command.Flags().Parse(test.flags))
			require.NoError(t, command.Run(&cmd.Context{Args: []string{"mysql"}, Stdout: &stdout}))
			require.True(t, called)
			for _, expected := range test.contains {
				require.Contains(t, stdout.String(), expected)
			}
			if test.name == "human" {
				require.Less(t, strings.Index(stdout.String(), "rules.sync"), strings.Index(stdout.String(), "rules.list"))
			}
		})
	}
}

func TestServiceManifestExportRoundTrip(t *testing.T) {
	for _, test := range []struct {
		filename string
		flags    []string
		json     bool
	}{
		{"manifest.json", nil, true},
		{"manifest.yaml", nil, false},
		{"manifest.yml", nil, false},
		{"manifest.YAML", nil, false},
		{"manifest.data", []string{"--json"}, true},
		{"manifest.json", []string{"--yaml"}, false},
	} {
		t.Run(test.filename+strings.Join(test.flags, ""), func(t *testing.T) {
			transport := &cmdtest.MultiConditionalTransport{
				ConditionalTransports: []cmdtest.ConditionalTransport{
					{
						Transport: cmdtest.Transport{Status: http.StatusOK, Message: manifestFixture},
						CondFunc: func(r *http.Request) bool {
							return r.Method == http.MethodGet && r.URL.Path == "/1.31/services/mysql/manifest"
						},
					},
					{
						Transport: cmdtest.Transport{Status: http.StatusOK},
						CondFunc: func(r *http.Request) bool {
							data, err := io.ReadAll(r.Body)
							require.NoError(t, err)
							require.JSONEq(t, manifestFixture, string(data))
							return r.Method == http.MethodPut && r.URL.Path == "/1.31/services/mysql/manifest" && r.Header.Get("Content-Type") == "application/json"
						},
					},
				},
			}
			setupManifestClient(t, transport)
			filename := filepath.Join(t.TempDir(), test.filename)
			require.NoError(t, os.WriteFile(filename, []byte("previous longer contents must be replaced"), 0o600))
			var stdout bytes.Buffer
			get := &ServiceManifestGet{}
			require.NoError(t, get.Flags().Parse(append(test.flags, "-o", filename)))
			require.NoError(t, get.Run(&cmd.Context{Args: []string{"mysql"}, Stdout: &stdout}))
			require.Empty(t, stdout.String())
			data, err := os.ReadFile(filename)
			require.NoError(t, err)
			if test.json {
				require.True(t, json.Valid(data))
			} else {
				require.Contains(t, string(data), "strict_actions: true")
				data, err = yaml.YAMLToJSON(data)
				require.NoError(t, err)
			}
			require.JSONEq(t, manifestFixture, string(data))
			require.NoError(t, (&ServiceManifestSet{}).Run(&cmd.Context{Args: []string{"mysql", filename}, Stdout: &stdout}))
			require.Equal(t, "Manifest for service \"mysql\" successfully updated.\n", stdout.String())
			require.Empty(t, transport.ConditionalTransports)
		})
	}
}

func TestServiceManifestInvalidInput(t *testing.T) {
	for _, data := range []string{"", "# comment", "null", "[]", "true", "text", "{", "enabled: [", `{"enabled":"true"}`, "strict_actions: wrong", "legacy_compat: 1", "operations: wrong", "operations:\n- method: 42"} {
		t.Run(data, func(t *testing.T) {
			called := false
			setupManifestClient(t, &cmdtest.ConditionalTransport{
				Transport: cmdtest.Transport{Status: http.StatusOK},
				CondFunc: func(r *http.Request) bool {
					called = true
					return true
				},
			})
			filename := filepath.Join(t.TempDir(), "input")
			require.NoError(t, os.WriteFile(filename, []byte(data), 0o600))
			var stdout bytes.Buffer
			err := (&ServiceManifestSet{}).Run(&cmd.Context{Args: []string{"mysql", filename}, Stdout: &stdout})
			require.ErrorContains(t, err, "invalid manifest")
			require.False(t, called)
			require.Empty(t, stdout.String())
		})
	}
}

func TestServiceManifestFailures(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			message := `{"service":"mysql","conflicts":[{"action":"rules.sync","roles":["operator"]}]}`
			setupManifestClient(t, cmdtest.Transport{Status: status, Message: message})
			filename := filepath.Join(t.TempDir(), "manifest.json")
			require.NoError(t, os.WriteFile(filename, []byte(manifestFixture), 0o600))
			var stdout bytes.Buffer
			err := (&ServiceManifestSet{}).Run(&cmd.Context{Args: []string{"mysql", filename}, Stdout: &stdout})
			require.EqualError(t, tsuruHTTP.UnwrapErr(err), message)
			err = (&ServiceManifestGet{}).Run(&cmd.Context{Args: []string{"mysql"}, Stdout: &stdout})
			require.EqualError(t, tsuruHTTP.UnwrapErr(err), message)
			require.Empty(t, stdout.String())
		})
	}
	t.Run("transport", func(t *testing.T) {
		setupManifestClient(t, &cmdtest.ConditionalTransport{
			CondFunc: func(*http.Request) bool { return false },
		})
		err := (&ServiceManifestGet{}).Run(&cmd.Context{Args: []string{"mysql"}, Stdout: io.Discard})
		require.EqualError(t, tsuruHTTP.UnwrapErr(err), "condition failed")
	})
	t.Run("invalid response preserves file", func(t *testing.T) {
		setupManifestClient(t, cmdtest.Transport{Status: http.StatusOK, Message: "invalid"})
		filename := filepath.Join(t.TempDir(), "manifest.json")
		require.NoError(t, os.WriteFile(filename, []byte("original"), 0o600))
		get := &ServiceManifestGet{output: filename}
		require.Error(t, get.Run(&cmd.Context{Args: []string{"mysql"}, Stdout: io.Discard}))
		data, err := os.ReadFile(filename)
		require.NoError(t, err)
		require.Equal(t, "original", string(data))
	})
	t.Run("file errors", func(t *testing.T) {
		setupManifestClient(t, cmdtest.Transport{Status: http.StatusOK, Message: manifestFixture})
		filename := filepath.Join(t.TempDir(), "missing", "manifest.json")
		require.Error(t, (&ServiceManifestSet{}).Run(&cmd.Context{Args: []string{"mysql", filename}, Stdout: io.Discard}))
		require.Error(t, (&ServiceManifestGet{output: filename}).Run(&cmd.Context{Args: []string{"mysql"}, Stdout: io.Discard}))
	})
}

func TestServiceManifestInvalidFlags(t *testing.T) {
	called := false
	setupManifestClient(t, &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			called = true
			return true
		},
	})
	for _, flags := range [][]string{{"--json", "--yaml"}, {"-o", "manifest.txt"}} {
		get := &ServiceManifestGet{}
		require.NoError(t, get.Flags().Parse(flags))
		require.Error(t, get.Run(&cmd.Context{Args: []string{"mysql"}, Stdout: io.Discard}))
		require.False(t, called)
	}
}
