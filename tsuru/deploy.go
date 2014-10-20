// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tsuru/tsuru/cmd"
)

type deploy struct {
	cmd.GuessingCommand
}

func (c *deploy) Info() *cmd.Info {
	desc := `Deploys set of files and/or directories to tsuru server. Some examples of calls are:

tsuru deploy .
tsuru deploy myfile.jar Procfile
`
	return &cmd.Info{
		Name:    "app-deploy",
		Usage:   "app-deploy [-a/--app <appname>] <file-or-dir-1> [file-or-dir-2] ... [file-or-dir-n]",
		Desc:    desc,
		MinArgs: 1,
	}
}

func (c *deploy) Run(context *cmd.Context, client *cmd.Client) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "archive.tar.gz")
	if err != nil {
		return err
	}
	err = targz(context, file, context.Args...)
	if err != nil {
		return err
	}
	writer.Close()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	url, err := cmd.GetURL("/apps/" + appName + "/deploy")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "multipart/form-data; boundary="+writer.Boundary())
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
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if fi.IsDir() {
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

func addDir(writer *tar.Writer, path string) error {
	dir, err := os.Open(path)
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
	header.Name = path
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
			err = addDir(writer, filepath.Join(path, fi.Name()))
		} else {
			err = addFile(writer, filepath.Join(path, fi.Name()))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func addFile(writer *tar.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	header.Name = path
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
