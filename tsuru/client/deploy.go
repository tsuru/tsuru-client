// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	tsuruClientApp "github.com/tsuru/tsuru-client/tsuru/app"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	v2 "github.com/tsuru/tsuru-client/tsuru/cmd/v2"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	tsuruapp "github.com/tsuru/tsuru/app"
	tsuruIo "github.com/tsuru/tsuru/io"
	"github.com/tsuru/tsuru/safe"
)

const deployOutputBufferSize = 4096

type deployList []tsuruapp.DeployData

func (dl deployList) Len() int {
	return len(dl)
}
func (dl deployList) Swap(i, j int) {
	dl[i], dl[j] = dl[j], dl[i]
}
func (dl deployList) Less(i, j int) bool {
	return dl[i].Timestamp.Before(dl[j].Timestamp)
}

type AppDeployList struct {
	tsuruClientApp.AppNameMixIn

	flagsApplied bool
	json         bool
}

func (c *AppDeployList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-deploy-list",
		Usage: "[<appname>]",
		Desc:  "List information about deploys for an application.",
	}
}

func (c *AppDeployList) Flags() *pflag.FlagSet {
	fs := c.AppNameMixIn.Flags()
	if !c.flagsApplied {
		fs.BoolVar(&c.json, standards.FlagJSON, false, "Show JSON")

		c.flagsApplied = true
	}
	return fs
}

func (c *AppDeployList) Run(context *cmd.Context) error {
	appName, err := c.AppNameByArgsAndFlag(context.Args)
	if err != nil {
		return err
	}
	url, err := config.GetURL(fmt.Sprintf("/deploys?app=%s&limit=10", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusNoContent {
		fmt.Fprintf(context.Stdout, "App %s has no deploy.\n", appName)
		return nil
	}
	defer response.Body.Close()
	result, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var deploys []tsuruapp.DeployData
	err = json.Unmarshal(result, &deploys)
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(deployList(deploys)))

	if c.json {
		return formatter.JSON(context.Stdout, deploys)
	}

	table := tablecli.NewTable()
	table.Headers = tablecli.Row([]string{"Image (Rollback)", "Origin", "User", "Date (Duration)", "Error"})
	for _, deploy := range deploys {
		timestamp := formatter.FormatDateAndDuration(deploy.Timestamp, &deploy.Duration)
		if deploy.Origin == "git" {
			if len(deploy.Commit) > 7 {
				deploy.Commit = deploy.Commit[:7]
			}
			deploy.Origin = fmt.Sprintf("git (%s)", deploy.Commit)
		}
		if deploy.CanRollback {
			deploy.Image += " (*)"
		}
		rowData := []string{deploy.Image, deploy.Origin, deploy.User, timestamp, deploy.Error}
		if deploy.Error != "" {
			for i, el := range rowData {
				if el != "" {
					rowData[i] = color.RedString(el)
				}
			}
		}
		table.LineSeparator = true
		table.AddRow(tablecli.Row(rowData))
	}
	context.Stdout.Write(table.Bytes())
	return nil
}

var _ cmd.Cancelable = &AppDeploy{}

type AppDeploy struct {
	tsuruClientApp.AppNameMixIn
	image      string
	message    string
	dockerfile string
	eventID    string
	fs         *pflag.FlagSet
	m          sync.Mutex
	deployVersionArgs
	filesOnly bool
}

func (c *AppDeploy) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.fs.SortFlags = false
		image := "The image to deploy in app"
		c.fs.StringVarP(&c.image, "image", "i", "", image)

		message := "A message describing this deploy"
		c.fs.StringVarP(&c.message, "message", "m", "", message)
		filesOnly := "Enables single file deployment into the root of the app's tree"
		c.fs.BoolVarP(&c.filesOnly, "files-only", "f", false, filesOnly)
		c.flags(c.fs)
		c.fs.StringVar(&c.dockerfile, "dockerfile", "", "Container file")
	}
	return c.fs
}

func (c *AppDeploy) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-deploy",
		Usage: "[--app <app name>] [--image <container image name>] [--dockerfile <container image file>] [--message <message>] [--files-only] [--new-version] [--override-old-versions] [file-or-dir ...]",
		Desc: `Deploy the source code and/or configurations to the application on Tsuru.

Files specified in the ".tsuruignore" file are skipped - similar to ".gitignore". It also honors ".dockerignore" file if deploying with container file (--dockerfile).

Examples:
  To deploy using app's platform build process (just sending source code and/or configurations):
    Uploading all files within the current directory
      $ tsuru app deploy -a <APP> .

    Uploading all files within a specific directory
      $ tsuru app deploy -a <APP> mysite/

    Uploading specific files
      $ tsuru app deploy -a <APP> ./myfile.jar ./Procfile

    Uploading specific files (ignoring their base directories)
      $ tsuru app deploy -a <APP> --files-only ./my-code/main.go ./tsuru_stuff/Procfile

  To deploy using a container image:
    $ tsuru app deploy -a <APP> --image registry.example.com/my-company/app:v42

  To deploy using container file ("docker build" mode):
    Sending the the current directory as container build context - uses Dockerfile file as container image instructions:
      $ tsuru app deploy -a <APP> --dockerfile .

    Sending a specific container file and specific directory as container build context:
      $ tsuru app deploy -a <APP> --dockerfile ./Dockerfile.other ./other/
`,
	}
}

type safeWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *safeWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}

func prepareUploadStreams(context *cmd.Context, buf *safe.Buffer) io.Writer {
	context.Stdout = &safeWriter{w: context.Stdout}

	fw := &firstWriter{Writer: context.Stdout}

	if v2.ColorStream() {
		encoderWriter := &safeWriter{w: formatter.NewColoredStreamWriter(fw)}
		return io.MultiWriter(encoderWriter, buf)
	}

	stream := tsuruIo.NewStreamWriter(fw, nil)
	encoderWriter := &safeWriter{w: &tsuruIo.SimpleJsonMessageEncoderWriter{Encoder: json.NewEncoder(stream)}}
	return io.MultiWriter(encoderWriter, buf)
}

func (c *AppDeploy) Run(context *cmd.Context) error {
	context.RawOutput()

	if c.image == "" && c.dockerfile == "" && len(context.Args) == 0 {
		return errors.New("you should provide at least one file, Docker image name or Dockerfile to deploy")
	}

	if c.image != "" && len(context.Args) > 0 {
		return errors.New("you can't deploy files and docker image at the same time")
	}

	if c.image != "" && c.dockerfile != "" {
		return errors.New("you can't deploy container image and container file at same time")
	}

	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}

	values := url.Values{}

	origin := "app-deploy"
	if c.image != "" {
		origin = "image"
	}
	values.Set("origin", origin)

	if c.message != "" {
		values.Set("message", c.message)
	}

	c.values(values)

	u, err := config.GetURL(fmt.Sprintf("/apps/%s/deploy", appName))
	if err != nil {
		return err
	}

	body := safe.NewBuffer(nil)
	request, err := http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}

	buf := safe.NewBuffer(nil)

	c.m.Lock()
	respBody := prepareUploadStreams(context, buf)
	c.m.Unlock()

	var archive io.Reader

	if c.image != "" {
		fmt.Fprintln(context.Stdout, "Deploying container image...")
		values.Set("image", c.image)
	}

	if c.dockerfile != "" {
		fmt.Fprintln(context.Stdout, "Deploying with Dockerfile...")

		var dockerfile string
		dockerfile, archive, err = buildWithContainerFile(appName, c.dockerfile, c.filesOnly, context.Args, nil)
		if err != nil {
			return err
		}

		values.Set("dockerfile", dockerfile)
	}

	if c.image == "" && c.dockerfile == "" {
		fmt.Fprintln(context.Stdout, "Deploying using app's platform...")

		var buffer bytes.Buffer
		err = Archive(&buffer, c.filesOnly, context.Args, DefaultArchiveOptions(nil))
		if err != nil {
			return err
		}

		archive = &buffer
	}

	if err = uploadFiles(context, request, buf, body, values, archive); err != nil {
		return err
	}

	c.m.Lock()
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		c.m.Unlock()
		return err
	}
	defer resp.Body.Close()
	c.eventID = resp.Header.Get("X-Tsuru-Eventid")
	c.m.Unlock()

	var readBuffer [deployOutputBufferSize]byte
	var readErr error
	for readErr == nil {
		var read int
		read, readErr = resp.Body.Read(readBuffer[:])
		if read == 0 {
			continue
		}
		c.m.Lock()
		written, writeErr := respBody.Write(readBuffer[:read])
		c.m.Unlock()
		if written < read {
			return fmt.Errorf("short write processing output, read: %d, written: %d", read, written)
		}
		if writeErr != nil {
			return fmt.Errorf("error writing response: %v", writeErr)
		}
	}
	if readErr != io.EOF {
		return fmt.Errorf("error reading response: %v", readErr)
	}
	if strings.HasSuffix(buf.String(), "\nOK\n") {
		return nil
	}
	return cmd.ErrAbortCommand
}

func (c *AppDeploy) Cancel(ctx cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	c.m.Lock()
	defer c.m.Unlock()
	ctx.RawOutput()
	if c.eventID == "" {
		return errors.New("event ID not available yet")
	}
	fmt.Fprintln(ctx.Stdout, color.New(color.FgRed, color.Bold).Sprint("Warning: the deploy is still RUNNING in the background!"))
	fmt.Fprint(ctx.Stdout, "Are you sure you want to cancel this deploy? (Y/n) ")
	var answer string
	fmt.Fscanf(ctx.Stdin, "%s", &answer)
	if strings.ToLower(answer) != "y" && answer != "" {
		return fmt.Errorf("aborted")
	}
	_, err = apiClient.EventApi.EventCancel(context.Background(), c.eventID, tsuru.EventCancelArgs{Reason: "Canceled on client."})
	return err
}

type firstWriter struct {
	io.Writer
	once sync.Once
}

func (w *firstWriter) Write(p []byte) (int, error) {
	w.once.Do(func() {
		w.Writer.Write([]byte(" ok\n"))
	})
	return w.Writer.Write(p)
}

type AppDeployRollback struct {
	tsuruClientApp.AppNameMixIn
	cmd.ConfirmationCommand
	deployVersionArgs
	fs *pflag.FlagSet
}

func (c *AppDeployRollback) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = mergeFlagSet(
			c.AppNameMixIn.Flags(),
			c.ConfirmationCommand.Flags(),
		)
		c.flags(c.fs)
	}
	return c.fs
}

func (c *AppDeployRollback) Info() *cmd.Info {
	desc := "Deploys an existing image for an app. You can list available images with `tsuru app deploy list`."
	return &cmd.Info{
		Name:    "app-deploy-rollback",
		Usage:   "[-a/--app appname] [-y/--assume-yes] <image-name>",
		Desc:    desc,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *AppDeployRollback) Run(context *cmd.Context) error {
	appName, err := c.AppNameByFlag()
	if err != nil {
		return err
	}
	imgName := context.Args[0]
	if !c.Confirm(context, fmt.Sprintf("Are you sure you want to rollback app %q to image %q?", appName, imgName)) {
		return nil
	}
	u, err := config.GetURL(fmt.Sprintf("/apps/%s/deploy/rollback", appName))
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("origin", "rollback")
	v.Set("image", imgName)
	c.values(v)
	request, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return err
	}
	return formatter.StreamJSONResponse(context.Stdout, response)
}

type AppDeployRollbackUpdate struct {
	tsuruClientApp.AppNameMixIn
	image   string
	reason  string
	disable bool
	fs      *pflag.FlagSet
}

func (c *AppDeployRollbackUpdate) Info() *cmd.Info {
	desc := `Disables an existing image of an app. You can list images with "tsuru app deploy list -a <appName>"

::

	The [-i/--image] flag is the name of an app image.

	The [-d/--disable] flag disables an image rollback. The default behavior (omitting this flag) is to enable it.

	The [-r/--reason] flag lets the user tell why this action was needed.
`
	return &cmd.Info{
		Name:  "app-deploy-rollback-update",
		Usage: "[appName] [-i/--image imageName] [-d/--disable] [-r/--reason reason]",
		Desc:  desc,

		MaxArgs: 1,
	}
}

func (c *AppDeployRollbackUpdate) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		image := "The image name for an app version"
		c.fs.StringVarP(&c.image, "image", "i", "", image)
		reason := "The reason why the rollback has to be disabled"
		c.fs.StringVarP(&c.reason, "reason", "r", "", reason)
		disable := "Enables or disables the rollback on a specific image version"
		c.fs.BoolVarP(&c.disable, "disable", "d", false, disable)
	}
	return c.fs
}

func (c *AppDeployRollbackUpdate) Run(context *cmd.Context) error {
	appName, err := c.AppNameByArgsAndFlag(context.Args)
	if err != nil {
		return err
	}
	u, err := config.GetURLVersion("1.4", fmt.Sprintf("/apps/%s/deploy/rollback/update", appName))
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("image", c.image)
	v.Set("reason", c.reason)
	v.Set("origin", "rollback")
	v.Set("disable", strconv.FormatBool(c.disable))
	request, err := http.NewRequest(http.MethodPut, u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, err = tsuruHTTP.AuthenticatedClient.Do(request)
	return err
}

type deployVersionArgs struct {
	newVersion       bool
	overrideVersions bool
}

func (c *deployVersionArgs) flags(fs *pflag.FlagSet) {
	newVersion := "Creates a new version for the current deployment while preserving existing versions"
	fs.BoolVar(&c.newVersion, "new-version", false, newVersion)
	overrideVersions := "Force replace all deployed versions by this new deploy"
	fs.BoolVar(&c.overrideVersions, "override-old-versions", false, overrideVersions)
}

func (c *deployVersionArgs) values(values url.Values) {
	if c.newVersion {
		values.Set("new-version", strconv.FormatBool(c.newVersion))
	}
	if c.overrideVersions {
		values.Set("override-versions", strconv.FormatBool(c.overrideVersions))
	}
}

var _ cmd.Cancelable = &JobDeploy{}

type JobDeploy struct {
	jobName    string
	image      string
	message    string
	dockerfile string
	eventID    string
	fs         *pflag.FlagSet
	m          sync.Mutex
}

func (c *JobDeploy) Flags() *pflag.FlagSet {
	if c.fs == nil {
		c.fs = pflag.NewFlagSet("", pflag.ExitOnError)
		c.fs.StringVarP(&c.jobName, standards.FlagJob, standards.ShortFlagJob, "", "The name of the job.")

		image := "The image to deploy in job"
		c.fs.StringVarP(&c.image, "image", "i", "", image)

		message := "A message describing this deploy"
		c.fs.StringVarP(&c.message, "message", "m", "", message)
		c.fs.StringVar(&c.dockerfile, "dockerfile", "", "Container file")
	}
	return c.fs
}

func (c *JobDeploy) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "job-deploy",
		Usage: "[--job <job name>] [--image <container image name>] [--dockerfile <container image file>] [--message <message>]",
		Desc: `Deploy the source code and/or configurations to a Job on Tsuru.

Files specified in the ".tsuruignore" file are skipped - similar to ".gitignore". It also honors ".dockerignore" file if deploying with container file (--dockerfile).

Examples:
  To deploy using a container image:
    $ tsuru job deploy -j <JOB> --image registry.example.com/my-company/my-job:v42

  To deploy using container file ("docker build" mode):
    Sending the the current directory as container build context - uses Dockerfile file as container image instructions:
      $ tsuru job deploy -j <JOB> --dockerfile .

    Sending a specific container file and specific directory as container build context:
      $ tsuru job deploy -j <JOB> --dockerfile ./Dockerfile.other ./other/
`,
	}
}

func (c *JobDeploy) Run(context *cmd.Context) error {
	context.RawOutput()

	if c.jobName == "" {
		return errors.New(`the name of the job is required.

Use the --job/-j flag to specify it`)
	}

	if c.image == "" && c.dockerfile == "" {
		return errors.New("you should provide at least one between Docker image name or Dockerfile to deploy")
	}

	if c.image != "" && len(context.Args) > 0 {
		return errors.New("you can't deploy files and docker image at the same time")
	}

	if c.image != "" && c.dockerfile != "" {
		return errors.New("you can't deploy container image and container file at same time")
	}

	values := url.Values{}

	origin := "job-deploy"
	if c.image != "" {
		origin = "image"
	}
	values.Set("origin", origin)

	if c.message != "" {
		values.Set("message", c.message)
	}

	u, err := config.GetURLVersion("1.23", "/jobs/"+c.jobName+"/deploy")
	if err != nil {
		return err
	}

	body := safe.NewBuffer(nil)
	request, err := http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}

	buf := safe.NewBuffer(nil)

	c.m.Lock()
	respBody := prepareUploadStreams(context, buf)
	c.m.Unlock()

	var archive io.Reader

	if c.image != "" {
		fmt.Fprintln(context.Stdout, "Deploying container image...")
		values.Set("image", c.image)
	}

	if c.dockerfile != "" {
		fmt.Fprintln(context.Stdout, "Deploying with Dockerfile...")

		var dockerfile string
		dockerfile, archive, err = buildWithContainerFile(c.jobName, c.dockerfile, false, context.Args, nil)
		if err != nil {
			return err
		}

		values.Set("dockerfile", dockerfile)
	}

	if err = uploadFiles(context, request, buf, body, values, archive); err != nil {
		return err
	}

	c.m.Lock()
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		c.m.Unlock()
		return err
	}
	defer resp.Body.Close()
	c.eventID = resp.Header.Get("X-Tsuru-Eventid")
	c.m.Unlock()

	var readBuffer [deployOutputBufferSize]byte
	var readErr error
	for readErr == nil {
		var read int
		read, readErr = resp.Body.Read(readBuffer[:])
		if read == 0 {
			continue
		}
		c.m.Lock()
		written, writeErr := respBody.Write(readBuffer[:read])
		c.m.Unlock()
		if written < read {
			return fmt.Errorf("short write processing output, read: %d, written: %d", read, written)
		}
		if writeErr != nil {
			return fmt.Errorf("error writing response: %v", writeErr)
		}
	}
	if readErr != io.EOF {
		return fmt.Errorf("error reading response: %v", readErr)
	}
	if strings.HasSuffix(buf.String(), "\nDeploy finished with success!\n") {
		return nil
	}
	return cmd.ErrAbortCommand
}

func (c *JobDeploy) Cancel(ctx cmd.Context) error {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return err
	}
	c.m.Lock()
	defer c.m.Unlock()
	ctx.RawOutput()
	if c.eventID == "" {
		return errors.New("event ID not available yet")
	}
	fmt.Fprintln(ctx.Stdout, color.New(color.FgRed, color.Bold).Sprint("Warning: the deploy is still RUNNING in the background!"))
	fmt.Fprint(ctx.Stdout, "Are you sure you want to cancel this deploy? (Y/n) ")
	var answer string
	fmt.Fscanf(ctx.Stdin, "%s", &answer)
	if strings.ToLower(answer) != "y" && answer != "" {
		return fmt.Errorf("aborted")
	}
	_, err = apiClient.EventApi.EventCancel(context.Background(), c.eventID, tsuru.EventCancelArgs{Reason: "Canceled on client."})
	return err
}
