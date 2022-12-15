// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tablecli"
	"github.com/tsuru/tsuru-client/tsuru/formatter"
	tsuruapp "github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
	"github.com/tsuru/tsuru/safe"
	terminal "golang.org/x/term"
)

const (
	deployOutputBufferSize = 4096

	clearLineEscape = "\033[2K\r"
)

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
	cmd.AppNameMixIn

	flagsApplied bool
	json         bool
}

func (c *AppDeployList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-deploy-list",
		Usage: "app deploy list [-a/--app <appname>]",
		Desc:  "List information about deploys for an application.",
	}
}

func (c *AppDeployList) Flags() *gnuflag.FlagSet {
	fs := c.AppNameMixIn.Flags()
	if !c.flagsApplied {
		fs.BoolVar(&c.json, "json", false, "Show JSON")

		c.flagsApplied = true
	}
	return fs
}

func (c *AppDeployList) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL(fmt.Sprintf("/deploys?app=%s&limit=10", appName))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusNoContent {
		fmt.Fprintf(context.Stdout, "App %s has no deploy.\n", appName)
		return nil
	}
	defer response.Body.Close()
	result, err := ioutil.ReadAll(response.Body)
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
					rowData[i] = cmd.Colorfy(el, "red", "", "")
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
	cmd.AppNameMixIn
	image   string
	message string
	eventID string
	fs      *gnuflag.FlagSet
	m       sync.Mutex
	deployVersionArgs
	filesOnly bool
}

func (c *AppDeploy) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		image := "The image to deploy in app"
		c.fs.StringVar(&c.image, "image", "", image)
		c.fs.StringVar(&c.image, "i", "", image)
		message := "A message describing this deploy"
		c.fs.StringVar(&c.message, "message", "", message)
		c.fs.StringVar(&c.message, "m", "", message)
		filesOnly := "Enables single file deployment into the root of the app's tree"
		c.fs.BoolVar(&c.filesOnly, "f", false, filesOnly)
		c.fs.BoolVar(&c.filesOnly, "files-only", false, filesOnly)
		c.deployVersionArgs.flags(c.fs)
	}
	return c.fs
}

func (c *AppDeploy) Info() *cmd.Info {
	desc := `Deploys set of files and/or directories to tsuru server. Some examples of
calls are:

::

	$ tsuru app deploy .
	$ tsuru app deploy myfile.jar Procfile
	$ tsuru app deploy -f directory/main.go directory/Procfile
	$ tsuru app deploy mysite
	$ tsuru app deploy -i http://registry.mysite.com:5000/image-name
`
	return &cmd.Info{
		Name:    "app-deploy",
		Usage:   "app deploy [-a/--app <appname>] [-i/--image <image_url>] [-m/--message <message>] [--new-version] [--override-old-versions] [-f/--files-only] <file-or-dir-1> [file-or-dir-2] ... [file-or-dir-n]",
		Desc:    desc,
		MinArgs: 0,
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
	stream := tsuruIo.NewStreamWriter(&firstWriter{Writer: context.Stdout}, nil)
	encoderWriter := &safeWriter{w: &tsuruIo.SimpleJsonMessageEncoderWriter{Encoder: json.NewEncoder(stream)}}
	respBody := io.MultiWriter(encoderWriter, buf)
	return respBody
}

func (c *AppDeploy) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	if c.image == "" && len(context.Args) == 0 {
		return errors.New("You should provide at least one file or a docker image to deploy.\n")
	}
	if c.image != "" && len(context.Args) > 0 {
		return errors.New("You can't deploy files and docker image at the same time.\n")
	}
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL("/apps/" + appName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	origin := "app-deploy"
	if c.image != "" {
		origin = "image"
	}
	values := url.Values{}
	values.Set("origin", origin)
	c.deployVersionArgs.values(values)
	if c.message != "" {
		values.Set("message", c.message)
	}
	u, err = cmd.GetURL(fmt.Sprintf("/apps/%s/deploy", appName))
	if err != nil {
		return err
	}
	body := safe.NewBuffer(nil)
	request, err = http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}
	buf := safe.NewBuffer(nil)
	c.m.Lock()
	respBody := prepareUploadStreams(context, buf)
	c.m.Unlock()
	if c.image != "" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		values.Set("image", c.image)
		_, err = body.WriteString(values.Encode())
		if err != nil {
			return err
		}
		fmt.Fprint(context.Stdout, "Deploying image...")
	} else {
		if err = uploadFiles(context, c.filesOnly, request, buf, body, values); err != nil {
			return err
		}
	}
	c.m.Lock()
	resp, err := client.Do(request)
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

func (c *AppDeploy) Cancel(ctx cmd.Context, cli *cmd.Client) error {
	apiClient, err := client.ClientFromEnvironment(&tsuru.Configuration{
		HTTPClient: cli.HTTPClient,
	})
	if err != nil {
		return err
	}
	c.m.Lock()
	defer c.m.Unlock()
	if c.eventID == "" {
		return errors.New("event ID not available yet")
	}
	fmt.Fprintln(ctx.Stdout, cmd.Colorfy("Warning: the deploy is still RUNNING in the background!", "red", "", "bold"))
	fmt.Fprint(ctx.Stdout, "Are you sure you want to cancel this deploy? (Y/n) ")
	var answer string
	fmt.Fscanf(ctx.Stdin, "%s", &answer)
	if strings.ToLower(answer) != "y" && answer != "" {
		return fmt.Errorf("aborted")
	}
	_, err = apiClient.EventApi.EventCancel(context.Background(), c.eventID, tsuru.EventCancelArgs{Reason: "Canceled on client."})
	return err
}

type tarMaker struct {
	ctx    *cmd.Context
	isTerm bool
}

func newTarMaker(ctx *cmd.Context) tarMaker {
	isTerm := false
	if desc, ok := ctx.Stdin.(interface {
		Fd() uintptr
	}); ok {
		fd := int(desc.Fd())
		isTerm = terminal.IsTerminal(fd)
	}
	return tarMaker{
		ctx:    ctx,
		isTerm: isTerm,
	}
}

func (m tarMaker) targz(destination io.Writer, filesOnly bool, filepaths ...string) error {
	fmt.Fprint(m.ctx.Stdout, "Generating tar.gz...")
	defer fmt.Fprintf(m.ctx.Stdout, "%sGenerating tar.gz. Done!\n", clearLineEscape)
	ign, err := ignore.CompileIgnoreFile(".tsuruignore")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	gzipWriter := gzip.NewWriter(destination)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, path := range filepaths {
		if path == ".." {
			fmt.Fprintf(m.ctx.Stderr, "Warning: skipping %q", path)
			continue
		}
		var fi os.FileInfo
		fi, err = os.Lstat(path)
		if err != nil {
			return err
		}
		var wd string
		wd, err = os.Getwd()
		if err != nil {
			return err
		}
		fiName := filepath.Join(wd, fi.Name())
		if ign != nil && ign.MatchesPath(fiName) {
			continue
		}
		if fi.IsDir() {
			dir := wd
			dirFilesOnly := filesOnly || len(filepaths) == 1
			if dirFilesOnly {
				dir = path
				path = "."
			}
			err = inDir(func() error {
				return m.addDir(tarWriter, path, ign, dirFilesOnly)
			}, dir)
		} else {
			err = m.addFile(tarWriter, path, filesOnly)
		}
		if err != nil {
			return err
		}
	}
	err = tarWriter.Close()
	if err != nil {
		return err
	}
	return gzipWriter.Close()
}

func inDir(fn func() error, path string) error {
	old, err := os.Getwd()
	if err != nil {
		return err
	}
	err = os.Chdir(path)
	if err != nil {
		return err
	}
	defer os.Chdir(old)
	return fn()
}

func (m tarMaker) addDir(writer *tar.Writer, dirpath string, ign *ignore.GitIgnore, filesOnly bool) error {
	dir, err := os.Open(dirpath)
	if err != nil {
		return err
	}
	defer dir.Close()
	if !filesOnly {
		var fi os.FileInfo
		fi, err = dir.Stat()
		if err != nil {
			return err
		}
		var header *tar.Header
		header, err = tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		header.Name = dirpath
		err = writer.WriteHeader(header)
		if err != nil {
			return err
		}
	}
	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	for _, fi := range fis {
		fiName := filepath.Join(wd, fi.Name())
		if dirpath != "." {
			fiName = filepath.Join(wd, dirpath, fi.Name())
		}
		if ign != nil && ign.MatchesPath(fiName) {
			continue
		}
		if fi.IsDir() {
			err = m.addDir(writer, path.Join(dirpath, fi.Name()), ign, false)
		} else {
			err = m.addFile(writer, path.Join(dirpath, fi.Name()), filesOnly)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (m tarMaker) addFile(writer *tar.Writer, filepath string, filesOnly bool) error {
	fi, err := os.Lstat(filepath)
	if err != nil {
		return err
	}
	var linkName string
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		var target string
		target, err = os.Readlink(filepath)
		if err != nil {
			return err
		}
		linkName = target
	}
	header, err := tar.FileInfoHeader(fi, linkName)
	if err != nil {
		return err
	}
	header.Name = filepath
	if filesOnly {
		header.Name = path.Base(filepath)
	}
	if m.isTerm {
		fmt.Fprintf(m.ctx.Stdout, "%sGenerating tar.gz... adding %s", clearLineEscape, header.Name)
	}
	err = writer.WriteHeader(header)
	if err != nil {
		return err
	}
	if linkName != "" {
		return nil
	}
	f, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(writer, f)
	if err != nil {
		return err
	}
	if n != fi.Size() {
		return io.ErrShortWrite
	}
	return nil
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
	cmd.AppNameMixIn
	cmd.ConfirmationCommand
	deployVersionArgs
	fs *gnuflag.FlagSet
}

func (c *AppDeployRollback) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = cmd.MergeFlagSet(
			c.AppNameMixIn.Flags(),
			c.ConfirmationCommand.Flags(),
		)
		c.deployVersionArgs.flags(c.fs)
	}
	return c.fs
}

func (c *AppDeployRollback) Info() *cmd.Info {
	desc := "Deploys an existing image for an app. You can list available images with `tsuru app deploy list`."
	return &cmd.Info{
		Name:    "app-deploy-rollback",
		Usage:   "app deploy rollback [-a/--app appname] [-y/--assume-yes] <image-name>",
		Desc:    desc,
		MinArgs: 1,
		MaxArgs: 1,
	}
}

func (c *AppDeployRollback) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	imgName := context.Args[0]
	if !c.Confirm(context, fmt.Sprintf("Are you sure you want to rollback app %q to image %q?", appName, imgName)) {
		return nil
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/deploy/rollback", appName))
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("origin", "rollback")
	v.Set("image", imgName)
	c.deployVersionArgs.values(v)
	request, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(context.Stdout, response)
}

type AppDeployRebuild struct {
	cmd.AppNameMixIn
	deployVersionArgs
	fs *gnuflag.FlagSet
}

func (c *AppDeployRebuild) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		c.deployVersionArgs.flags(c.fs)
	}
	return c.fs
}

func (c *AppDeployRebuild) Info() *cmd.Info {
	desc := "Rebuild and deploy the last app image."
	return &cmd.Info{
		Name:    "app-deploy-rebuild",
		Usage:   "app deploy rebuild [-a/--app appname]",
		Desc:    desc,
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *AppDeployRebuild) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := cmd.GetURLVersion("1.3", fmt.Sprintf("/apps/%s/deploy/rebuild", appName))
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("origin", "rebuild")
	c.deployVersionArgs.values(v)
	request, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(context.Stdout, response)
}

type AppDeployRollbackUpdate struct {
	cmd.AppNameMixIn
	image   string
	reason  string
	disable bool
	fs      *gnuflag.FlagSet
}

func (c *AppDeployRollbackUpdate) Info() *cmd.Info {
	desc := `Locks an existing image of an app. You can list images with "tsuru app deploy list -a <appName>"

::

	The [-i/--image] flag is the name of an app image.

	The [-d/--disable] flag disables an image rollback. The default behavior (omitting this flag) is to enable it.

	The [-r/--reason] flag lets the user tell why this action was needed.
`
	return &cmd.Info{
		Name:    "app-deploy-rollback-update",
		Usage:   "app deploy rollback update [-a/--app appName] [-i/--image imageName] [-d/--disable] [-r/--reason reason]",
		Desc:    desc,
		MinArgs: 0,
		MaxArgs: 0,
	}
}

func (c *AppDeployRollbackUpdate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		image := "The image name for an app version"
		c.fs.StringVar(&c.image, "image", "", image)
		c.fs.StringVar(&c.image, "i", "", image)
		reason := "The reason why the rollback has to be disabled"
		c.fs.StringVar(&c.reason, "reason", "", reason)
		c.fs.StringVar(&c.reason, "r", "", reason)
		disable := "Enables or disables the rollback on a specific image version"
		c.fs.BoolVar(&c.disable, "disable", false, disable)
		c.fs.BoolVar(&c.disable, "d", false, disable)
	}
	return c.fs
}

func (c *AppDeployRollbackUpdate) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := cmd.GetURL(fmt.Sprintf("/apps/%s/deploy/rollback/update", appName))
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
	_, err = client.Do(request)
	return err
}

type deployVersionArgs struct {
	newVersion       bool
	overrideVersions bool
}

func (c *deployVersionArgs) flags(fs *gnuflag.FlagSet) {
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
