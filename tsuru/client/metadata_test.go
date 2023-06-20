package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestMetadataGetInfo(c *check.C) {
	c.Assert((&MetadataGet{}).Info(), check.NotNil)
}

func (s *S) TestMetadataGetRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `{"metadata":{"annotations":[{"name":"my-annotation","value":"some long value"}],"labels":[{"name":"logging.globoi.com/backup","value":"true"}]}}`
	result := "Labels:\n" +
		"\tlogging.globoi.com/backup: true\n" +
		"Annotations:\n" +
		"\tmy-annotation: some long value\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: jsonResult, Status: http.StatusOK}}, nil, manager)
	command := MetadataGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestMetadataSetInfo(c *check.C) {
	c.Assert((&MetadataSet{}).Info(), check.NotNil)
}

func (s *S) TestMetadataSetRunWithLabel(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label=some-value"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "App \"someapp\" has been updated!\n"

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/someapp")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)

			var payload map[string]interface{}
			err = json.Unmarshal(data, &payload)
			c.Assert(err, check.IsNil)
			c.Assert(payload, check.DeepEquals, map[string]interface{}{
				"planoverride": map[string]interface{}{},
				"metadata": map[string]interface{}{
					"labels": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "value": "some-value"}},
				},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "label"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataSetRunWithAnnotations(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label=some-value, that is really long"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "App \"someapp\" has been updated!\n"

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/someapp")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)

			var payload map[string]interface{}
			err = json.Unmarshal(data, &payload)
			c.Assert(err, check.IsNil)
			c.Assert(payload, check.DeepEquals, map[string]interface{}{
				"planoverride": map[string]interface{}{},
				"metadata": map[string]interface{}{
					"annotations": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "value": "some-value, that is really long"}},
				},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "annotation"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataSetFailsWithoutType(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label=some-value"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "A type is required: label or annotation")
}

func (s *S) TestMetadateSetSupportsNoRestart(c *check.C) {
	command := MetadataSet{}
	flagset := command.Flags()
	flagset.Parse(true, []string{"--no-restart"})
	noRestart := flagset.Lookup("no-restart")
	c.Check(noRestart, check.NotNil)
	c.Check(noRestart.Name, check.Equals, "no-restart")
	c.Check(noRestart.Value.String(), check.Equals, "true")
	c.Check(noRestart.DefValue, check.Equals, "false")
}

func (s *S) TestMetadataUnsetInfo(c *check.C) {
	c.Assert((&MetadataUnset{}).Info(), check.NotNil)
}

func (s *S) TestMetadataUnsetRunWithLabel(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "App \"someapp\" has been updated!\n"

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/someapp")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)

			var payload map[string]interface{}
			err = json.Unmarshal(data, &payload)
			c.Assert(err, check.IsNil)
			c.Assert(payload, check.DeepEquals, map[string]interface{}{
				"planoverride": map[string]interface{}{},
				"metadata": map[string]interface{}{
					"labels": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "delete": true}},
				},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)

	command := MetadataUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "label"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataUnsetRunWithAnnotations(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "App \"someapp\" has been updated!\n"

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/apps/someapp")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)

			var payload map[string]interface{}
			err = json.Unmarshal(data, &payload)
			c.Assert(err, check.IsNil)
			c.Assert(payload, check.DeepEquals, map[string]interface{}{
				"planoverride": map[string]interface{}{},
				"metadata": map[string]interface{}{
					"annotations": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "delete": true}},
				},
			})
			return url && method
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)

	command := MetadataUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "annotation"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataUnsetFailsWithoutType(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)

	command := MetadataUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "A type is required: label or annotation")
}

func (s *S) TestMetadateUnsetSupportsNoRestart(c *check.C) {
	command := MetadataUnset{}
	flagset := command.Flags()
	flagset.Parse(true, []string{"--no-restart"})
	noRestart := flagset.Lookup("no-restart")
	c.Check(noRestart, check.NotNil)
	c.Check(noRestart.Name, check.Equals, "no-restart")
	c.Check(noRestart.Value.String(), check.Equals, "true")
	c.Check(noRestart.DefValue, check.Equals, "false")
}
