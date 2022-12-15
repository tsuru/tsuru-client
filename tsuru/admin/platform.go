// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package admin

import (
	"bytes"
	"context"
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
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	"github.com/tsuru/tsuru/cmd"
)

type platform struct {
	Name     string `json:"name"`
	Disabled bool   `json:"disabled"`
}

type PlatformList struct {
	fs         *gnuflag.FlagSet
	simplified bool
	json       bool
}

func (p *PlatformList) Run(context *cmd.Context, client *cmd.Client) error {
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
	if resp.StatusCode == http.StatusNoContent {
		fmt.Fprintln(context.Stdout, "No platforms available.")
		return nil
	}
	err = json.NewDecoder(resp.Body).Decode(&platforms)
	if err != nil {
		return err
	}
	sort.Slice(platforms, func(i, j int) bool {
		return platforms[i].Name < platforms[j].Name
	})

	if p.simplified {
		for _, p := range platforms {
			fmt.Fprintln(context.Stdout, p.Name)
		}
		return nil
	}

	if p.json {
		return formatter.JSON(context.Stdout, platforms)
	}

	tbl := tablecli.NewTable()
	tbl.Headers = tablecli.Row{"Name", "Status"}
	tbl.LineSeparator = false
	for _, p := range platforms {
		status := "enabled"
		if p.Disabled {
			status = "disabled"
		}
		tbl.AddRow(tablecli.Row{
			p.Name,
			status,
		})
	}
	fmt.Fprint(context.Stdout, tbl.String())

	return nil
}

func (c *PlatformList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("platform-list", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.simplified, "q", false, "Display only platform name")
		c.fs.BoolVar(&c.json, "json", false, "Display in JSON format")

	}
	return c.fs
}
func (*PlatformList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "platform-list",
		Usage:   "platform list",
		Desc:    "Lists the available platforms. All platforms displayed in this list may be used to create new apps (see app-create).",
		MinArgs: 0,
	}
}

type PlatformAdd struct {
	dockerfile string
	image      string
	fs         *gnuflag.FlagSet
}

func (p *PlatformAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "platform-add",
		Usage: "platform add <platform name> [--dockerfile/-d Dockerfile] [--image/-i image]",
		Desc: `Adds a new platform to tsuru.

The name of the image can be automatically inferred in case you're using an
official platform. Check https://github.com/tsuru/platforms for a list of
official platforms and instructions on how to create a custom platform.

Examples:

	[[tsuru platform add java # uses official tsuru/java image from docker hub]]
	[[tsuru platform add java -i registry.company.com/tsuru/java # uses custom Java image]]
	[[tsuru platform add java -d /data/projects/java/Dockerfile # uses local Dockerfile]]
	[[tsuru platform add java -d https://platforms.com/java/Dockerfile # uses remote Dockerfile]]`,
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
	if err != nil {
		return err
	}
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

type PlatformUpdate struct {
	dockerfile string
	image      string
	disable    bool
	enable     bool
	fs         *gnuflag.FlagSet
}

func (p *PlatformUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "platform-update",
		Usage: "platform update <platform name> [--dockerfile/-d Dockerfile] [--disable/--enable] [--image/-i image]",
		Desc: `Updates a platform in tsuru.

The name of the image can be automatically inferred in case you're using an
official platform. Check https://github.com/tsuru/platforms for a list of
official platforms.

The flags --enable and --disable can be used for enabling or disabling a
platform.

Examples:

[[tsuru platform update java # uses official tsuru/java image from docker hub]]
[[tsuru platform update java -i registry.company.com/tsuru/java # uses custom Java image]]
[[tsuru platform update java -d /data/projects/java/Dockerfile # uses local Dockerfile]]
[[tsuru platform update java -d https://platforms.com/java/Dockerfile # uses remote Dockerfile]]`,
		MinArgs: 1,
	}
}

func (p *PlatformUpdate) Flags() *gnuflag.FlagSet {
	dockerfileMessage := "URL or path to the Dockerfile used for building the image of the platform"
	if p.fs == nil {
		p.fs = gnuflag.NewFlagSet("platform-update", gnuflag.ExitOnError)
		p.fs.StringVar(&p.dockerfile, "dockerfile", "", dockerfileMessage)
		p.fs.StringVar(&p.dockerfile, "d", "", dockerfileMessage)
		p.fs.BoolVar(&p.disable, "disable", false, "Disable the platform")
		p.fs.BoolVar(&p.enable, "enable", false, "Enable the platform")
		msg := "Name of the prebuilt Docker image"
		p.fs.StringVar(&p.image, "image", "", msg)
		p.fs.StringVar(&p.image, "i", "", msg)
	}
	return p.fs
}

func (p *PlatformUpdate) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	name := context.Args[0]
	if p.disable && p.enable {
		return errors.New("Conflicting options: --enable and --disable")
	}
	var disable string
	if p.enable {
		disable = "false"
	}
	if p.disable {
		disable = "true"
	}
	var body bytes.Buffer
	implicitImage := !p.disable && !p.enable && p.dockerfile == "" && p.image == ""
	writer, err := serializeDockerfile(context.Args[0], &body, p.dockerfile, p.image, implicitImage)
	if err != nil {
		return err
	}
	writer.WriteField("disabled", disable)
	writer.Close()
	url, err := cmd.GetURL(fmt.Sprintf("/platforms/%s", name))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", url, &body)
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

type PlatformRemove struct {
	cmd.ConfirmationCommand
}

func (p *PlatformRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "platform-remove",
		Usage: "platform remove <platform name> [-y]",
		Desc: `Remove a platform from tsuru. This command will fail if there are application
still using the platform.`,
		MinArgs: 1,
	}
}

func (p *PlatformRemove) Run(context *cmd.Context, client *cmd.Client) error {
	name := context.Args[0]
	if !p.Confirm(context, fmt.Sprintf(`Are you sure you want to remove "%s" platform?`, name)) {
		return nil
	}
	url, err := cmd.GetURL("/platforms/" + name)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		fmt.Fprintf(context.Stdout, "Failed to remove platform!\n")
		return err
	}
	fmt.Fprintf(context.Stdout, "Platform successfully removed!\n")
	return nil
}

type PlatformInfo struct {
	fs   *gnuflag.FlagSet
	json bool
}

func (c *PlatformInfo) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("platform-info", gnuflag.ExitOnError)
		c.fs.BoolVar(&c.json, "json", false, "Display platform in JSON Format")
	}
	return c.fs
}

func (p *PlatformInfo) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "platform-info",
		Usage:   "platform info <platform name>",
		Desc:    `Shows information about a specific platform.`,
		MinArgs: 1,
	}
}

func (c PlatformInfo) Run(ctx *cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	ctx.RawOutput()
	name := ctx.Args[0]
	info, resp, err := apiClient.PlatformApi.PlatformInfo(context.TODO(), name)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	defer resp.Body.Close()

	if c.json {
		return formatter.JSON(ctx.Stdout, info)
	}

	var status string
	if info.Platform.Disabled {
		status = "disabled"
	} else {
		status = "enabled"
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Name: %s\n", info.Platform.Name))
	buf.WriteString(fmt.Sprintf("Status: %s\n", status))
	buf.WriteString("Images:\n")
	sort.Sort(sort.Reverse(sort.StringSlice(info.Images)))
	for _, img := range info.Images {
		buf.WriteString(fmt.Sprintf(" - %s\n", img))
	}
	ctx.Stdout.Write(buf.Bytes())
	return nil
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
