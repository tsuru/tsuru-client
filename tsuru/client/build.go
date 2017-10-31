package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
	"github.com/tsuru/tsuru/safe"
)

type AppBuild struct {
	cmd.GuessingCommand
	tag string
	fs  *gnuflag.FlagSet
}

func (c *AppBuild) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.GuessingCommand.Flags()
		tag := "The image tag"
		c.fs.StringVar(&c.tag, "tag", "", tag)
		c.fs.StringVar(&c.tag, "t", "", tag)
	}
	return c.fs
}

func (c *AppBuild) Info() *cmd.Info {
	desc := `Builds a tsuru app image respecting .tsuruignore file. Some examples of calls are:

::

		$ tsuru app-build -a myapp -t mytag .
		$ tsuru app-build -a myapp -t latest myfile.jar Procfile
`
	return &cmd.Info{
		Name:    "app-build",
		Usage:   "app-build [-a/--app <appname>] [-t/--tag <image_tag>] <file-or-dir-1> [file-or-dir-2] ... [file-or-dir-n]",
		Desc:    desc,
		MinArgs: 0,
	}
}

func (c *AppBuild) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	if c.tag == "" {
		return errors.New("You should provide one tag to build the image.\n")
	}
	if len(context.Args) == 0 {
		return errors.New("You should provide at least one file to build the image.\n")
	}
	appName, err := c.Guess()
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
	values := url.Values{}
	values.Set("tag", c.tag)
	u, err = cmd.GetURLVersion("1.5", fmt.Sprintf("/apps/%s/build", appName))
	if err != nil {
		return err
	}
	body := safe.NewBuffer(nil)
	request, err = http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}
	buf := safe.NewBuffer(nil)
	stream := tsuruIo.NewStreamWriter(context.Stdout, nil)
	safeStdout := &safeWriter{w: &tsuruIo.SimpleJsonMessageEncoderWriter{Encoder: json.NewEncoder(stream)}}
	respBody := firstWriter{Writer: io.MultiWriter(safeStdout, buf)}
	if err = uploadFiles(context, request, buf, safeStdout, body, values); err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(&respBody, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(safeStdout, buf.String())
	if resp.StatusCode != http.StatusOK {
		return cmd.ErrAbortCommand
	}
	return nil
}

func uploadFiles(context *cmd.Context, request *http.Request, buf *safe.Buffer, safeStdout *safeWriter, body *safe.Buffer, values url.Values) error {
	writer := multipart.NewWriter(body)
	for k := range values {
		writer.WriteField(k, values.Get(k))
	}
	file, err := writer.CreateFormFile("file", "archive.tar.gz")
	if err != nil {
		return err
	}
	ignoreSet := make(map[string]struct{})
	ignorePatterns, _ := readTsuruIgnore()
	for _, pattern := range ignorePatterns {
		ignSet, errProc := processTsuruIgnore(pattern, context.Args...)
		if errProc != nil {
			return errProc
		}
		for k, v := range ignSet {
			ignoreSet[k] = v
		}
	}
	err = targz(context, file, ignoreSet, context.Args...)
	if err != nil {
		return err
	}
	writer.Close()
	request.Header.Set("Content-Type", "multipart/form-data; boundary="+writer.Boundary())
	fullSize := float64(body.Len())
	megabyte := 1024.0 * 1024.0
	fmt.Fprintf(context.Stdout, "Uploading files (%0.2fMB)... ", fullSize/megabyte)
	count := 0
	go func() {
		t0 := time.Now()
		lastTransferred := 0.0
		for buf.Len() == 0 {
			remaining := body.Len()
			transferred := fullSize - float64(remaining)
			speed := ((transferred - lastTransferred) / megabyte) / (float64(time.Since(t0)) / float64(time.Second))
			t0 = time.Now()
			lastTransferred = transferred
			percent := (transferred / fullSize) * 100.0
			fmt.Fprintf(safeStdout, "\rUploading files (%0.2fMB)... %0.2f%%", fullSize/megabyte, percent)
			if remaining > 0 {
				fmt.Fprintf(safeStdout, " (%0.2fMB/s)", speed)
			}
			if remaining == 0 && buf.Len() == 0 {
				fmt.Fprintf(safeStdout, " Processing%s", strings.Repeat(".", count))
				count++
			}
			time.Sleep(2e9)
		}
	}()
	return nil
}
