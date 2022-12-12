// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
)

func (s *S) TestPlatformList(c *check.C) {
	var buf bytes.Buffer
	var called bool
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status:  http.StatusOK,
			Message: `[{"Name":"ruby"},{"Name":"python"}]`,
		},
		CondFunc: func(r *http.Request) bool {
			called = true
			return r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/platforms")
		},
	}
	context := cmd.Context{Stdout: &buf}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, s.manager)
	err := (&PlatformList{simplified: true}).Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := `python
ruby` + "\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestPlatformListWithDisabledPlatforms(c *check.C) {
	var buf bytes.Buffer
	var called bool
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Status:  http.StatusOK,
			Message: `[{"Name":"ruby"},{"Name":"python"},{"Name":"ruby20", "Disabled":true}]`,
		},
		CondFunc: func(r *http.Request) bool {
			called = true
			return r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/platforms")
		},
	}
	context := cmd.Context{Stdout: &buf}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, s.manager)
	err := (&PlatformList{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	expected := `+--------+----------+
| Name   | Status   |
+--------+----------+
| python | enabled  |
| ruby   | enabled  |
| ruby20 | disabled |
+--------+----------+` + "\n"
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestPlatformListEmpty(c *check.C) {
	var buf bytes.Buffer
	transport := cmdtest.Transport{
		Status: http.StatusNoContent,
	}
	context := cmd.Context{Stdout: &buf}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, s.manager)
	err := (&PlatformList{}).Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(buf.String(), check.Equals, "No platforms available.\n")
}

func (s *S) TestPlatformListInfo(c *check.C) {
	c.Assert((&PlatformList{}).Info(), check.NotNil)
}

func (s *S) TestPlatformListIsACommand(c *check.C) {
	var _ cmd.Command = &PlatformList{}
}

func (s *S) TestPlatformAddInfo(c *check.C) {
	c.Assert((&PlatformAdd{}).Info(), check.NotNil)
}
func (s *S) TestPlatformAddRun(c *check.C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("FROM tsuru/java"))
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"teste"},
	}
	expectedMsg := "--something--\nPlatform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			file, header, transErr := req.FormFile("dockerfile_content")
			c.Assert(transErr, check.IsNil)
			defer file.Close()
			c.Assert(header.Filename, check.Equals, "Dockerfile")
			data, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(string(data), check.Equals, "FROM tsuru/java")
			return strings.HasSuffix(req.URL.Path, "/platforms") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformAdd{}
	command.Flags().Parse(true, []string{"--dockerfile", server.URL})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformAddRunLocalDockerFile(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"teste"},
	}
	expectedMsg := "--something--\nPlatform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			file, header, transErr := req.FormFile("dockerfile_content")
			c.Assert(transErr, check.IsNil)
			defer file.Close()
			c.Assert(header.Filename, check.Equals, "Dockerfile")
			data, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(string(data), check.Equals, "FROM\ttsuru/java\nRUN\ttrue\n")
			return strings.HasSuffix(req.URL.Path, "/platforms") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformAdd{}
	command.Flags().Parse(true, []string{"--dockerfile", "testdata/Dockerfile"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformAddPrebuiltImage(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"teste"},
	}
	expectedMsg := "--something--\nPlatform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			file, header, transErr := req.FormFile("dockerfile_content")
			c.Assert(transErr, check.IsNil)
			defer file.Close()
			c.Assert(header.Filename, check.Equals, "Dockerfile")
			data, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(string(data), check.Equals, "FROM tsuru/python")
			return strings.HasSuffix(req.URL.Path, "/platforms") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformAdd{}
	command.Flags().Parse(true, []string{"--image", "tsuru/python"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformAddRunImplicitDockerfile(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"teste"},
	}
	expectedMsg := "--something--\nPlatform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			file, header, transErr := req.FormFile("dockerfile_content")
			c.Assert(transErr, check.IsNil)
			defer file.Close()
			c.Assert(header.Filename, check.Equals, "Dockerfile")
			data, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(string(data), check.Equals, "FROM tsuru/teste")
			return strings.HasSuffix(req.URL.Path, "/platforms") && req.Method == "POST"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformAdd{}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformAddRunFlagsConflict(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"teste"},
	}
	client := cmd.NewClient(&http.Client{}, nil, s.manager)
	command := PlatformAdd{}
	command.Flags().Parse(true, []string{"--image", "tsuru/python", "--dockerfile", "testdata/Dockerfile"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "Conflicting options: --image and --dockerfile")
}

func (s *S) TestPlatformAddFlagSet(c *check.C) {
	message := "URL or path to the Dockerfile used for building the image of the platform"
	command := PlatformAdd{}
	flagset := command.Flags()
	flagset.Parse(true, []string{"--dockerfile", "dockerfile", "-i", "tsuru/python"})

	dockerfile := flagset.Lookup("dockerfile")
	c.Check(dockerfile.Name, check.Equals, "dockerfile")
	c.Check(dockerfile.Usage, check.Equals, message)
	c.Check(dockerfile.DefValue, check.Equals, "")

	sdockerfile := flagset.Lookup("d")
	c.Check(sdockerfile.Name, check.Equals, "d")
	c.Check(sdockerfile.Usage, check.Equals, message)
	c.Check(sdockerfile.DefValue, check.Equals, "")

	image := flagset.Lookup("image")
	c.Check(image.Name, check.Equals, "image")
	c.Check(image.Usage, check.Equals, "Name of the prebuilt Docker image")
	c.Check(image.DefValue, check.Equals, "")

	simage := flagset.Lookup("i")
	c.Check(simage.Name, check.Equals, "i")
	c.Check(simage.Usage, check.Equals, "Name of the prebuilt Docker image")
	c.Check(simage.DefValue, check.Equals, "")
}

func (s *S) TestPlatformUpdateInfo(c *check.C) {
	c.Assert((&PlatformUpdate{}).Info(), check.NotNil)
}

func (s *S) TestPlatformUpdateFlagSet(c *check.C) {
	dockerfileMessage := "URL or path to the Dockerfile used for building the image of the platform"
	command := PlatformUpdate{}
	flagset := command.Flags()
	flagset.Parse(true, []string{"--dockerfile", "dockerfile"})

	dockerfile := flagset.Lookup("dockerfile")
	c.Check(dockerfile.Name, check.Equals, "dockerfile")
	c.Check(dockerfile.Usage, check.Equals, dockerfileMessage)
	c.Check(dockerfile.DefValue, check.Equals, "")

	sdockerfile := flagset.Lookup("d")
	c.Check(sdockerfile.Name, check.Equals, "d")
	c.Check(sdockerfile.Usage, check.Equals, dockerfileMessage)
	c.Check(sdockerfile.DefValue, check.Equals, "")

	image := flagset.Lookup("image")
	c.Check(image.Name, check.Equals, "image")
	c.Check(image.Usage, check.Equals, "Name of the prebuilt Docker image")
	c.Check(image.DefValue, check.Equals, "")

	simage := flagset.Lookup("i")
	c.Check(simage.Name, check.Equals, "i")
	c.Check(simage.Usage, check.Equals, "Name of the prebuilt Docker image")
	c.Check(simage.DefValue, check.Equals, "")
}

func (s *S) TestPlatformUpdateRun(c *check.C) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("FROM tsuru/java"))
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expectedMsg := "--something--\nPlatform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			file, header, transErr := req.FormFile("dockerfile_content")
			c.Assert(transErr, check.IsNil)
			defer file.Close()
			c.Assert(header.Filename, check.Equals, "Dockerfile")
			data, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(string(data), check.Equals, "FROM tsuru/java")
			return strings.HasSuffix(req.URL.Path, "/platforms/"+name) && req.Method == "PUT"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformUpdate{}
	command.Flags().Parse(true, []string{"--dockerfile", server.URL})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformUpdateRunLocalDockerfile(c *check.C) {
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expectedMsg := "--something--\nPlatform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			file, header, transErr := req.FormFile("dockerfile_content")
			c.Assert(transErr, check.IsNil)
			defer file.Close()
			c.Assert(header.Filename, check.Equals, "Dockerfile")
			data, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(string(data), check.Equals, "FROM\ttsuru/java\nRUN\ttrue\n")
			return strings.HasSuffix(req.URL.Path, "/platforms/"+name) && req.Method == "PUT"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformUpdate{}
	command.Flags().Parse(true, []string{"--dockerfile", "testdata/Dockerfile"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformUpdateRunPrebuiltImage(c *check.C) {
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expectedMsg := "--something--\nPlatform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			file, header, transErr := req.FormFile("dockerfile_content")
			c.Assert(transErr, check.IsNil)
			defer file.Close()
			c.Assert(header.Filename, check.Equals, "Dockerfile")
			data, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(string(data), check.Equals, "FROM tsuru/python")
			return strings.HasSuffix(req.URL.Path, "/platforms/"+name) && req.Method == "PUT"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformUpdate{}
	command.Flags().Parse(true, []string{"--image", "tsuru/python"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformUpdateRunImplicitImage(c *check.C) {
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expectedMsg := "--something--\nPlatform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			file, header, transErr := req.FormFile("dockerfile_content")
			c.Assert(transErr, check.IsNil)
			defer file.Close()
			c.Assert(header.Filename, check.Equals, "Dockerfile")
			data, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(string(data), check.Equals, "FROM tsuru/teste")
			return strings.HasSuffix(req.URL.Path, "/platforms/"+name) && req.Method == "PUT"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformUpdate{}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformUpdateWithFlagDisableTrue(c *check.C) {
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expectedMsg := "Platform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			_, _, err = req.FormFile("dockerfile_content")
			c.Assert(err, check.NotNil)
			c.Assert(req.FormValue("disabled"), check.Equals, "true")
			return strings.HasSuffix(req.URL.Path, "/platforms/"+name) && req.Method == "PUT"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformUpdate{}
	command.Flags().Parse(true, []string{"--disable"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformUpdateWithFlagEnabledTrue(c *check.C) {
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expectedMsg := "Platform successfully updated!\n"
	msg := io.SimpleJsonMessage{Message: expectedMsg}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			_, _, err = req.FormFile("dockerfile_content")
			c.Assert(err, check.NotNil)
			c.Assert(req.FormValue("disabled"), check.Equals, "false")
			return strings.HasSuffix(req.URL.Path, "/platforms/"+name) && req.Method == "PUT"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformUpdate{}
	command.Flags().Parse(true, []string{"--enable"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}

func (s *S) TestPlatformUpdateImageAndDockerfile(c *check.C) {
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expected := "Conflicting options: --image and --dockerfile"
	client := cmd.NewClient(&http.Client{}, nil, s.manager)
	command := PlatformUpdate{}
	command.Flags().Parse(true, []string{"--image", "tsuru/python", "--dockerfile", "testdata/Dockerfile"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, expected)
}

func (s *S) TestPlatformUpdateEnableAndDisable(c *check.C) {
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expected := "Conflicting options: --enable and --disable"
	client := cmd.NewClient(&http.Client{}, nil, s.manager)
	command := PlatformUpdate{}
	command.Flags().Parse(true, []string{"--disable", "--enable"})
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, expected)
}

func (s *S) TestPlatformRemoveInfo(c *check.C) {
	c.Assert((&PlatformRemove{}).Info(), check.NotNil)
}

func (s *S) TestPlatformRemoveRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expected := "Platform successfully removed!\n"
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/platforms/"+name) && req.Method == http.MethodDelete
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformRemove{}
	command.Flags().Parse(true, []string{"-y"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestPlatformRemoveConfirmation(c *check.C) {
	var stdout bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stdin:  strings.NewReader("n\n"),
		Args:   []string{"something"},
	}
	command := PlatformRemove{}
	err := command.Run(&context, nil)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "Are you sure you want to remove \"something\" platform? (y/n) Abort.\n")
}

func (s *S) TestPlatformInfoInfo(c *check.C) {
	c.Assert((&PlatformInfo{}).Info(), check.NotNil)
}

func (s *S) TestPlatformInfoRun(c *check.C) {
	var stdout, stderr bytes.Buffer
	name := "teste"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{name},
	}
	expectedMsg := "Name: teste\nStatus: enabled\nImages:\n - tsuru/teste:v2\n - tsuru/teste:v1\n"
	res := struct {
		Platform platform
		Images   []string
	}{
		Platform: platform{Name: name, Disabled: false},
		Images:   []string{"tsuru/teste:v1", "tsuru/teste:v2"},
	}
	result, err := json.Marshal(res)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: string(result),
			Status:  http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return strings.HasSuffix(req.URL.Path, "/platforms/"+name) && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, s.manager)
	command := PlatformInfo{}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expectedMsg)
}
