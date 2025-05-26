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

func (s *S) TestMetadataSetBothJobAndAppFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label=some-value"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-j", "somejob"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "please use only one of the -a/--app and -j/--job flags")
}

func (s *S) TestMetadataGetAppRun(c *check.C) {
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
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := MetadataGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestMetadataGetJobRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `{
		"job": {
			"name": "somejob",
			"metadata": {
				"annotations": [{
					"name": "my-annotation",
					"value": "some long value"
				}],
				"labels": [{
					"name": "logging.globoi.com/backup",
					"value": "true"
				}]
			}
		}
	}`
	result := "Labels:\n" +
		"\tlogging.globoi.com/backup: true\n" +
		"Annotations:\n" +
		"\tmy-annotation: some long value\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := MetadataGet{}
	command.Flags().Parse(true, []string{"-j", "somejob"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestMetadataGetAppRunWithProcesses(c *check.C) {
	var stdout, stderr bytes.Buffer
	jsonResult := `{
			"name": "somejob",
			"metadata": {
				"annotations": [{
					"name": "my-annotation",
					"value": "some long value"
				}],
				"labels": [{
					"name": "logging.globoi.com/backup",
					"value": "true"
				}]
			},
			"processes": [
				{
					"name": "web",
					"metadata": {
						"labels": [{
							"name": "logging.globoi.com/sampling",
							"value": "0.1"
						}]
					}
				}
			]
	}`
	result := "Metadata for app:\n" +
		"Labels:\n" +
		"\tlogging.globoi.com/backup: true\n" +
		"Annotations:\n" +
		"\tmy-annotation: some long value\n\n" +
		"Metadata for process: \"web\"\n" +
		"Labels:\n" +
		"\tlogging.globoi.com/sampling: 0.1\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Message: jsonResult, Status: http.StatusOK})
	command := MetadataGet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, result)
}

func (s *S) TestMetadataSetInfo(c *check.C) {
	c.Assert((&MetadataSet{}).Info(), check.NotNil)
}

func (s *S) TestMetadataSetRunJobWithLabel(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label=some-value"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "job \"somejob\" has been updated!\n"

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/jobs/somejob")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)
			var payload map[string]interface{}
			err = json.Unmarshal(data, &payload)
			c.Assert(err, check.IsNil)
			c.Assert(payload, check.DeepEquals, map[string]interface{}{
				"container": map[string]interface{}{},
				"metadata":  map[string]interface{}{"labels": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "value": "some-value"}}},
				"name":      "somejob"})
			return url && method
		},
	}
	s.setupFakeTransport(trans)

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-j", "somejob", "-t", "label"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataSetRunAppWithLabel(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label=some-value"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "app \"someapp\" has been updated!\n"

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
	s.setupFakeTransport(trans)

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "label"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataSetRunAppWithProcess(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label=some-value"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "app \"someapp\" has been updated!\n"

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
				"metadata":     map[string]interface{}{},
				"processes": []interface{}{
					map[string]interface{}{
						"name": "web",
						"metadata": map[string]interface{}{
							"labels": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "value": "some-value"}},
						},
					},
				},
			})
			return url && method
		},
	}
	s.setupFakeTransport(trans)

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "label", "-p", "web"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataSetRunJobWithAnnotations(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label=some-value, that is really long"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "job \"somejob\" has been updated!\n"

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/jobs/somejob")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)

			var payload map[string]interface{}
			err = json.Unmarshal(data, &payload)
			c.Assert(err, check.IsNil)
			c.Assert(payload, check.DeepEquals, map[string]interface{}{
				"container": map[string]interface{}{},
				"metadata":  map[string]interface{}{"annotations": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "value": "some-value, that is really long"}}},
				"name":      "somejob"})
			return url && method
		},
	}
	s.setupFakeTransport(trans)

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-j", "somejob", "-t", "annotation"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataSetRunAppWithAnnotations(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label=some-value, that is really long"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "app \"someapp\" has been updated!\n"

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
	s.setupFakeTransport(trans)

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "annotation"})
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)

	command := MetadataSet{}
	command.Flags().Parse(true, []string{"-a", "someapp"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "a type is required: label or annotation")
}

func (s *S) TestMetadataSetAppSupportsNoRestart(c *check.C) {
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

func (s *S) TestMetadataUnsetRunJobWithLabel(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "job \"somejob\" has been updated!\n"

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/jobs/somejob")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)

			var payload map[string]interface{}
			err = json.Unmarshal(data, &payload)
			c.Assert(err, check.IsNil)
			c.Assert(payload, check.DeepEquals, map[string]interface{}{
				"container": map[string]interface{}{},
				"metadata":  map[string]interface{}{"labels": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "delete": true}}},
				"name":      "somejob"})
			return url && method
		},
	}
	s.setupFakeTransport(trans)

	command := MetadataUnset{}
	command.Flags().Parse(true, []string{"-j", "somejob", "-t", "label"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataUnsetRunAppWithLabel(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "app \"someapp\" has been updated!\n"

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
	s.setupFakeTransport(trans)

	command := MetadataUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "label"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataUnsetRunAppWithProcess(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "app \"someapp\" has been updated!\n"

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
				"metadata":     map[string]interface{}{},
				"processes": []interface{}{
					map[string]interface{}{
						"name": "worker",
						"metadata": map[string]interface{}{
							"labels": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "delete": true}},
						},
					},
				},
			})
			return url && method
		},
	}
	s.setupFakeTransport(trans)

	command := MetadataUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "label", "-p", "worker"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataUnsetRunJobWithAnnotations(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "job \"somejob\" has been updated!\n"

	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			url := strings.HasSuffix(req.URL.Path, "/jobs/somejob")
			method := req.Method == "PUT"
			data, err := io.ReadAll(req.Body)
			c.Assert(err, check.IsNil)

			var payload map[string]interface{}
			err = json.Unmarshal(data, &payload)
			c.Assert(err, check.IsNil)
			c.Assert(payload, check.DeepEquals, map[string]interface{}{
				"container": map[string]interface{}{},
				"metadata":  map[string]interface{}{"annotations": []interface{}{map[string]interface{}{"name": "test.tsuru.io/label", "delete": true}}},
				"name":      "somejob"})
			return url && method
		},
	}
	s.setupFakeTransport(trans)

	command := MetadataUnset{}
	command.Flags().Parse(true, []string{"-j", "somejob", "-t", "annotation"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}

func (s *S) TestMetadataUnsetRunAppWithAnnotations(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"test.tsuru.io/label"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expectedOut := "app \"someapp\" has been updated!\n"

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
	s.setupFakeTransport(trans)

	command := MetadataUnset{}
	command.Flags().Parse(true, []string{"-a", "someapp", "-t", "annotation"})
	err := command.Run(&context)
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
	s.setupFakeTransport(trans)

	command := MetadataUnset{}
	command.Flags().Parse(true, []string{"-j", "somejob"})
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "a type is required: label or annotation")
}

func (s *S) TestMetadateUnsetAppSupportsNoRestart(c *check.C) {
	command := MetadataUnset{}
	flagset := command.Flags()
	flagset.Parse(true, []string{"--no-restart"})
	noRestart := flagset.Lookup("no-restart")
	c.Check(noRestart, check.NotNil)
	c.Check(noRestart.Name, check.Equals, "no-restart")
	c.Check(noRestart.Value.String(), check.Equals, "true")
	c.Check(noRestart.DefValue, check.Equals, "false")
}
