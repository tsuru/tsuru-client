// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tsuru/gnuflag"
	tsuruapp "github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
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

type appDeployList struct {
	cmd.GuessingCommand
}

func (c *appDeployList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "app-deploy-list",
		Usage: "app-deploy-list [-a/--app <appname>]",
		Desc:  "List information about deploys for an application.",
	}
}

func (c *appDeployList) Run(context *cmd.Context, client *cmd.Client) error {
	appName, err := c.Guess()
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
	table := cmd.NewTable()
	table.Headers = cmd.Row([]string{"Image (Rollback)", "Origin", "User", "Date (Duration)", "Error"})
	for _, deploy := range deploys {
		timestamp := deploy.Timestamp.Local().Format(time.Stamp)
		seconds := deploy.Duration / time.Second
		minutes := seconds / 60
		seconds = seconds % 60
		if deploy.Origin == "git" {
			if len(deploy.Commit) > 7 {
				deploy.Commit = deploy.Commit[:7]
			}
			deploy.Origin = fmt.Sprintf("git (%s)", deploy.Commit)
		}
		timestamp = fmt.Sprintf("%s (%02d:%02d)", timestamp, minutes, seconds)
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
		table.AddRow(cmd.Row(rowData))
	}
	context.Stdout.Write(table.Bytes())
	return nil
}

type appDeploy struct {
	cmd.GuessingCommand
	image string
	fs    *gnuflag.FlagSet
}

func (c *appDeploy) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		image := "The image to deploy in app"
		c.fs.StringVar(&c.image, "image", "", image)
		c.fs.StringVar(&c.image, "i", "", image)
	}
	return c.fs
}

func (c *appDeploy) Info() *cmd.Info {
	desc := `Deploys set of files and/or directories to tsuru server. Some examples of
calls are:

::

    $ tsuru app-deploy .
    $ tsuru app-deploy myfile.jar Procfile
    $ tsuru app-deploy mysite
    $ tsuru app-deploy -i http://registry.mysite.com:5000/image-name
`
	return &cmd.Info{
		Name:    "app-deploy",
		Usage:   "app-deploy [-a/--app <appname>] [-i/--image <image_url>] <file-or-dir-1> [file-or-dir-2] ... [file-or-dir-n]",
		Desc:    desc,
		MinArgs: 0,
	}
}

func (c *appDeploy) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL("/apps/" + appName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
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
	url, err = cmd.GetURL(fmt.Sprintf("/apps/%s/deploy?origin=%s", appName, origin))
	if err != nil {
		return err
	}
	var body bytes.Buffer
	request, err = http.NewRequest("POST", url, &body)
	if err != nil {
		return err
	}
	if c.image != "" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, err = body.WriteString(fmt.Sprintf("image=%s", c.image))
		if err != nil {
			return err
		}
	} else {
		writer := multipart.NewWriter(&body)
		var file io.Writer
		file, err = writer.CreateFormFile("file", "archive.tar.gz")
		if err != nil {
			return err
		}
		err = targz(context, file, context.Args...)
		if err != nil {
			return err
		}
		writer.Close()
		request.Header.Set("Content-Type", "multipart/form-data; boundary="+writer.Boundary())
	}
	var buf bytes.Buffer
	respBody := firstWriter{Writer: io.MultiWriter(context.Stdout, &buf)}
	go func() {
		fmt.Fprint(context.Stdout, "Uploading files..")
		for buf.Len() == 0 {
			fmt.Fprint(context.Stdout, ".")
			time.Sleep(2e9)
		}
	}()
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(&respBody, resp.Body)
	if err != nil {
		return err
	}
	if strings.HasSuffix(buf.String(), "\nOK\n") {
		return nil
	}
	return cmd.ErrAbortCommand
}

func targz(ctx *cmd.Context, destination io.Writer, filepaths ...string) error {
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	for _, path := range filepaths {
		if path == ".." {
			fmt.Fprintf(ctx.Stderr, "Warning: skipping %q", path)
			continue
		}
		fi, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			if len(filepaths) == 1 && path != "." {
				return singleDir(ctx, destination, path)
			}
			err = addDir(tarWriter, path)
		} else {
			err = addFile(tarWriter, path)
		}
		if err != nil {
			return err
		}
	}
	err := tarWriter.Close()
	if err != nil {
		return err
	}
	gzipWriter := gzip.NewWriter(destination)
	defer gzipWriter.Close()
	_, err = io.Copy(gzipWriter, &buf)
	return err
}

func singleDir(ctx *cmd.Context, destination io.Writer, path string) error {
	old, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(old)
	err = os.Chdir(path)
	if err != nil {
		return err
	}
	return targz(ctx, destination, ".")
}

func addDir(writer *tar.Writer, dirpath string) error {
	dir, err := os.Open(dirpath)
	if err != nil {
		return err
	}
	defer dir.Close()
	fi, err := dir.Stat()
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	header.Name = dirpath
	err = writer.WriteHeader(header)
	if err != nil {
		return err
	}
	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			err = addDir(writer, path.Join(dirpath, fi.Name()))
		} else {
			err = addFile(writer, path.Join(dirpath, fi.Name()))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func addFile(writer *tar.Writer, filepath string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := os.Lstat(filepath)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		var target string
		target, err = os.Readlink(filepath)
		if err != nil {
			return err
		}
		return addSymlink(writer, filepath, target)
	}
	header, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	header.Name = filepath
	err = writer.WriteHeader(header)
	if err != nil {
		return err
	}
	n, err := io.Copy(writer, f)
	if err != nil {
		return err
	}
	if n != fi.Size() {
		return io.ErrShortWrite
	}
	return nil
}

func addSymlink(writer *tar.Writer, symlink, target string) error {
	fi, err := os.Lstat(symlink)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	header.Name = symlink
	header.Linkname = target
	err = writer.WriteHeader(header)
	if err != nil {
		return err
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

type appDeployRollback struct {
	cmd.GuessingCommand
	cmd.ConfirmationCommand
	fs *gnuflag.FlagSet
}

func (c *appDeployRollback) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = cmd.MergeFlagSet(
			c.GuessingCommand.Flags(),
			c.ConfirmationCommand.Flags(),
		)
	}
	return c.fs
}

func (c *appDeployRollback) Info() *cmd.Info {
	desc := "Deploys an existing image for an app. You can list available images with `tsuru app-deploy-list`."
	return &cmd.Info{
		Name:    "app-deploy-rollback",
		Usage:   "app-deploy-rollback [-a/--app appname] [-y/--assume-yes] <image-name>",
		Desc:    desc,
		MinArgs: 1,
	}
}

func (c *appDeployRollback) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	imgName := context.Args[0]
	if !c.Confirm(context, fmt.Sprintf("Are you sure you want to rollback app %q to image %q?", appName, imgName)) {
		return nil
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/deploy/rollback?origin=%s", appName, "rollback"))
	if err != nil {
		return err
	}
	body := strings.NewReader("image=" + imgName)
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	w := tsuruIo.NewStreamWriter(context.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, response.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return fmt.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}
