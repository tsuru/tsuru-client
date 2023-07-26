// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package printer

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
)

func TestFormatAs(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		s        string
		expected OutputFormat
		hasError bool
	}{
		{"compactjson", CompactJSON, false},
		{"compact-json", CompactJSON, false},
		{"json", PrettyJSON, false},
		{"pretty-json", PrettyJSON, false},
		{"prettyjson", PrettyJSON, false},
		{"yaml", YAML, false},
		{"table", Table, false},
		{"invalid", Table, true},
	} {
		got, err := FormatAs(test.s)
		if test.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, test.expected, got)
	}
}

func TestPrintJSON(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		data     interface{}
		expected string
	}{
		{"test", `"test"`},
		{[]byte("test"), `"dGVzdA=="`},
		{&bytes.Buffer{}, "{}"},
		{map[string]string{"key": "value"}, `{"key":"value"}`},
		{map[string]interface{}{"key": map[string]string{"subkey": "value"}}, `{"key":{"subkey":"value"}}`},
		{struct {
			Key   string
			Value int
		}{"mykey", 42}, `{"Key":"mykey","Value":42}`},
		{tsuru.Plan{Name: "myplan", Memory: 1024, Cpumilli: 1000}, `{"name":"myplan","memory":1024,"cpumilli":1000,"override":{}}`},
		{(*tsuru.Plan)(nil), "null"},
	} {
		w := &bytes.Buffer{}
		err := PrintJSON(w, test.data)
		assert.NoError(t, err)
		assert.EqualValues(t, test.expected+"\n", w.String())
	}

	// Empty result (passing nil)
	for _, test := range []struct {
		data interface{}
	}{
		{nil},
	} {
		w := &bytes.Buffer{}
		err := PrintJSON(w, test.data)
		assert.NoError(t, err)
		assert.EqualValues(t, "", w.String())
	}

	// Error cases
	for _, test := range []struct {
		data interface{}
	}{
		{func() {}},
		{make(chan bool)},
	} {
		w := &bytes.Buffer{}
		err := PrintJSON(w, test.data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error converting to json")
	}
}

func TestPrintPrettyJSON(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		data     interface{}
		expected string
	}{
		{"test", `"test"`},
		{[]byte("test"), `"dGVzdA=="`},
		{&bytes.Buffer{}, "{}"},
		{map[string]string{"key": "value"}, "{\n  \"key\": \"value\"\n}"},
		{map[string]interface{}{"key": map[string]string{"subkey": "value"}}, `{
  "key": {
    "subkey": "value"
  }
}`},
		{struct {
			Key   string
			Value int
		}{"mykey", 42}, `{
  "Key": "mykey",
  "Value": 42
}`},
		{tsuru.Plan{Name: "myplan", Memory: 1024, Cpumilli: 1000}, `{
  "name": "myplan",
  "memory": 1024,
  "cpumilli": 1000,
  "override": {}
}`},
		{(*tsuru.Plan)(nil), "null"},
	} {
		w := &bytes.Buffer{}
		err := PrintPrettyJSON(w, test.data)
		assert.NoError(t, err)
		assert.EqualValues(t, test.expected+"\n", w.String())
	}

	// Empty result (passing nil)
	for _, test := range []struct {
		data interface{}
	}{
		{nil},
	} {
		w := &bytes.Buffer{}
		err := PrintPrettyJSON(w, test.data)
		assert.NoError(t, err)
		assert.EqualValues(t, "", w.String())
	}

	// Error cases
	for _, test := range []struct {
		data interface{}
	}{
		{func() {}},
		{make(chan bool)},
	} {
		w := &bytes.Buffer{}
		err := PrintPrettyJSON(w, test.data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error converting to json")
	}
}

type yamlMarshalerError struct{}

func (yamlMarshalerError) MarshalYAML() (interface{}, error) {
	return nil, errors.New("error")
}
func TestPrintYAML(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		data     interface{}
		expected string
	}{
		{"test", `test`},
		{[]byte("test"), `- 116
- 101
- 115
- 116`},
		{&bytes.Buffer{}, "{}"},
		{map[string]string{"key": "value"}, "key: value"},
		{map[string]interface{}{"key": map[string]string{"subkey": "value"}}, "key:\n    subkey: value"},
		{struct {
			Key   string
			Value int
		}{"mykey", 42}, "key: mykey\nvalue: 42"},
		{tsuru.Plan{Name: "myplan", Memory: 1024, Cpumilli: 1000}, `name: myplan
memory: 1024
cpumilli: 1000
default: false
override:
    memory: null
    cpumilli: null`},
		{(*tsuru.Plan)(nil), "null"},
	} {
		w := &bytes.Buffer{}
		err := PrintYAML(w, test.data)
		assert.NoError(t, err)
		assert.EqualValues(t, test.expected+"\n", w.String())
	}

	// Empty result (passing nil)
	for _, test := range []struct {
		data interface{}
	}{
		{nil},
	} {
		w := &bytes.Buffer{}
		err := PrintYAML(w, test.data)
		assert.NoError(t, err)
		assert.EqualValues(t, "", w.String())
	}

	// Error cases

	for _, test := range []struct {
		data interface{}
	}{
		{func() {}},
		{make(chan bool)},
		{yamlMarshalerError{}},
	} {
		w := &bytes.Buffer{}
		err := PrintYAML(w, test.data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error converting to yaml")
	}
}
