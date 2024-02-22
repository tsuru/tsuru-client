package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tsuru/gnuflag"
	tsuruClientApp "github.com/tsuru/tsuru-client/tsuru/app"
	"github.com/tsuru/tsuru-client/tsuru/config"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/safe"
)

type AppBuild struct {
	tsuruClientApp.AppNameMixIn
	tag       string
	fs        *gnuflag.FlagSet
	filesOnly bool
}

func (c *AppBuild) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = c.AppNameMixIn.Flags()
		tag := "The image tag"
		c.fs.StringVar(&c.tag, "tag", "", tag)
		c.fs.StringVar(&c.tag, "t", "", tag)
		filesOnly := "Enables single file build into the root of the app's tree"
		c.fs.BoolVar(&c.filesOnly, "f", false, filesOnly)
		c.fs.BoolVar(&c.filesOnly, "files-only", false, filesOnly)
	}
	return c.fs
}

func (c *AppBuild) Info() *cmd.Info {
	desc := `Build a container image following the app deploy's workflow - but do not change anything on the running application on Tsuru.
You can deploy this container image to the app later.

Files specified in the ".tsuruignore" file are skipped - similar to ".gitignore".

Examples:
  To build using app's platform build process (just sending source code or configurations):
    Uploading all files within the current directory
      $ tsuru app build -a <APP> .

    Uploading all files within a specific directory
      $ tsuru app build -a <APP> mysite/

    Uploading specific files
      $ tsuru app build -a <APP> ./myfile.jar ./Procfile

    Uploading specific files but ignoring their directory trees
      $ tsuru app build -a <APP> --files-only ./my-code/main.go ./tsuru_stuff/Procfile
`
	return &cmd.Info{
		Name:    "app-build",
		Usage:   "app build [-a/--app <appname>] [-t/--tag <image_tag>] [-f/--files-only] <file-or-dir-1> [file-or-dir-2] ... [file-or-dir-n]",
		Desc:    desc,
		MinArgs: 0,
	}
}

func (c *AppBuild) Run(context *cmd.Context) error {
	context.RawOutput()
	if c.tag == "" {
		return errors.New("You should provide one tag to build the image.\n")
	}
	if len(context.Args) == 0 {
		return errors.New("You should provide at least one file to build the image.\n")
	}

	appName, err := c.AppName()
	if err != nil {
		return err
	}
	u, err := config.GetURL("/apps/" + appName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	_, err = tsuruHTTP.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	values := url.Values{}
	values.Set("tag", c.tag)
	u, err = config.GetURLVersion("1.5", fmt.Sprintf("/apps/%s/build", appName))
	if err != nil {
		return err
	}
	body := safe.NewBuffer(nil)
	request, err = http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}
	buf := safe.NewBuffer(nil)
	respBody := prepareUploadStreams(context, buf)

	var archive bytes.Buffer
	err = Archive(&archive, c.filesOnly, context.Args, DefaultArchiveOptions(nil))
	if err != nil {
		return err
	}

	if err = uploadFiles(context, request, buf, body, values, &archive); err != nil {
		return err
	}
	resp, err := tsuruHTTP.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return cmd.ErrAbortCommand
	}
	_, err = io.Copy(respBody, resp.Body)
	if err != nil {
		return err
	}
	if strings.HasSuffix(buf.String(), "\nOK\n") {
		return nil
	}
	return cmd.ErrAbortCommand
}

func uploadFiles(context *cmd.Context, request *http.Request, buf *safe.Buffer, body *safe.Buffer, values url.Values, archive io.Reader) error {
	if archive == nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, err := body.WriteString(values.Encode())
		return err
	}

	writer := multipart.NewWriter(body)
	for k := range values {
		writer.WriteField(k, values.Get(k))
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())

	f, err := writer.CreateFormFile("file", "archive.tar.gz")
	if err != nil {
		return err
	}

	if _, err = io.Copy(f, archive); err != nil {
		return err
	}

	if err = writer.Close(); err != nil {
		return err
	}

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
			fmt.Fprintf(context.Stdout, "\rUploading files (%0.2fMB)... %0.2f%%", fullSize/megabyte, percent)
			if remaining > 0 {
				fmt.Fprintf(context.Stdout, " (%0.2fMB/s)", speed)
			}
			if remaining == 0 && buf.Len() == 0 {
				fmt.Fprintf(context.Stdout, " Processing%s", strings.Repeat(".", count))
				count++
			}
			time.Sleep(2 * time.Second)
		}
	}()
	return nil
}

func buildWithContainerFile(appName, path string, filesOnly bool, files []string, stderr io.Writer) (string, io.Reader, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to stat the file %s: %w", path, err)
	}

	var containerfile []byte

	switch {
	case fi.IsDir():
		path, err = guessingContainerFile(appName, path)
		if err != nil {
			return "", nil, fmt.Errorf("failed to guess the container file (can you specify the container file passing --dockerfile ./path/to/Dockerfile?): %w", err)
		}

		fallthrough

	case fi.Mode().IsRegular():
		containerfile, err = os.ReadFile(path)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

	default:
		return "", nil, fmt.Errorf("invalid file type")
	}

	if len(files) == 0 { // no additional files set, using the dockerfile dir
		files = []string{filepath.Dir(path)}
	}

	var buildContext bytes.Buffer
	err = Archive(&buildContext, filesOnly, files, DefaultArchiveOptions(stderr))
	if err != nil {
		return "", nil, err
	}

	return string(containerfile), &buildContext, nil
}

func guessingContainerFile(app, dir string) (string, error) {
	validNames := []string{
		fmt.Sprintf("Dockerfile.%s", app),
		fmt.Sprintf("Containerfile.%s", app),
		"Dockerfile.tsuru",
		"Containerfile.tsuru",
		"Dockerfile",
		"Containerfile",
	}

	for _, name := range validNames {
		path := filepath.Join(dir, name)

		fi, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}

		if err != nil {
			return "", err
		}

		if fi.Mode().IsRegular() {
			return path, nil
		}
	}

	return "", errors.New("container file not found")
}
