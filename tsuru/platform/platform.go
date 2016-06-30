// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
)

type platform struct {
	Name     string
	Disabled bool
}

type PlatformList struct{}

func (PlatformList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/platforms")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	var platforms []platform
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&platforms)
	if err != nil {
		return err
	}
	if len(platforms) == 0 {
		fmt.Fprintln(context.Stdout, "No platforms available.")
		return nil
	}
	platformNames := make([]string, len(platforms))
	for i, p := range platforms {
		platformNames[i] = p.Name
		if p.Disabled {
			platformNames[i] += " (disabled)"
		}
	}
	sort.Strings(platformNames)
	for _, p := range platformNames {
		fmt.Fprintf(context.Stdout, "- %s\n", p)
	}
	return nil
}

func (PlatformList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "platform-list",
		Usage:   "platform-list",
		Desc:    "Lists the available platforms. All platforms displayed in this list may be used to create new apps (see app-create).",
		MinArgs: 0,
	}
}

type PlatformAdd struct {
	name       string
	dockerfile string
	image      string
	fs         *gnuflag.FlagSet
}

func (p *PlatformAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "platform-add",
		Usage: "platform-add <platform name> [--dockerfile/-d Dockerfile] [--image/-i image]",
		Desc: `Adds a new platform to tsuru.

The name of the image can be automatically inferred in case you're using an
official platform. Check https://github.com/tsuru/platforms for a list of
official platforms and instructions on how to create a custom platform.

Examples:

	[[tsuru-admin platform-add java # uses official tsuru/java image from docker hub]]
	[[tsuru-admin platform-add java -i registry.company.com/tsuru/java # uses custom Java image]]
	[[tsuru-admin platform-add java -d /data/projects/java/Dockerfile # uses local Dockerfile]]
	[[tsuru-admin platform-add java -d https://platforms.com/java/Dockerfile # uses remote Dockerfile]]`,
		MinArgs: 1,
	}
}

func (p *PlatformAdd) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	var body bytes.Buffer
	writer, err := serializeDockerfile(context.Args[0], &body, p.dockerfile, p.image, true)
	if err != nil {
		return err
	}
	writer.WriteField("name", context.Args[0])
	writer.Close()
	url, err := cmd.GetURL("/platforms")
	request, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return cmd.StreamJSONResponse(context.Stdout, response)
}

func (p *PlatformAdd) Flags() *gnuflag.FlagSet {
	dockerfileMessage := "URL or path to the Dockerfile used for building the image of the platform"
	if p.fs == nil {
		p.fs = gnuflag.NewFlagSet("", gnuflag.ExitOnError)
		p.fs.StringVar(&p.dockerfile, "dockerfile", "", dockerfileMessage)
		p.fs.StringVar(&p.dockerfile, "d", "", dockerfileMessage)
		msg := "Name of the prebuilt Docker image"
		p.fs.StringVar(&p.image, "image", "", msg)
		p.fs.StringVar(&p.image, "i", "", msg)
	}
	return p.fs
}

func serializeDockerfile(name string, w io.Writer, dockerfile, image string, useImplicit bool) (*multipart.Writer, error) {
	if dockerfile != "" && image != "" {
		return nil, errors.New("Conflicting options: --image and --dockerfile")
	}
	writer := multipart.NewWriter(w)
	var dockerfileContent []byte
	if image != "" {
		dockerfileContent = []byte("FROM " + image)
	} else if dockerfile != "" {
		dockerfileURL, err := url.Parse(dockerfile)
		if err != nil {
			return nil, err
		}
		switch dockerfileURL.Scheme {
		case "http", "https":
			dockerfileContent, err = downloadDockerfile(dockerfile)
		default:
			dockerfileContent, err = ioutil.ReadFile(dockerfile)
		}
		if err != nil {
			return nil, err
		}
	} else if useImplicit {
		dockerfileContent = []byte("FROM tsuru/" + name)
	} else {
		return writer, nil
	}
	fileWriter, err := writer.CreateFormFile("dockerfile_content", "Dockerfile")
	if err != nil {
		return nil, err
	}
	fileWriter.Write(dockerfileContent)
	return writer, nil
}

func downloadDockerfile(dockerfileURL string) ([]byte, error) {
	resp, err := http.Get(dockerfileURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
