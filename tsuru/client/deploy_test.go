package client

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	tsuruIo "github.com/tsuru/tsuru/io"
	"gopkg.in/check.v1"
)

func (s *S) TestDeployInfo(c *check.C) {
	var cmd AppDeploy
	c.Assert(cmd.Info(), check.NotNil)
}

func (s *S) TestDeployRun(c *check.C) {
	calledTimes := 0
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: bytes.NewBufferString("")}
	err := targz(&ctx, &buf, nil, "testdata", "..")
	c.Assert(err, check.IsNil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			if req.Body != nil {
				defer req.Body.Close()
			}
			if calledTimes == 1 {
				return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/secret")
			}
			file, _, transErr := req.FormFile("file")
			c.Assert(transErr, check.IsNil)
			content, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(content, check.DeepEquals, buf.Bytes())
			c.Assert(req.Header.Get("Content-Type"), check.Matches, "multipart/form-data; boundary=.*")
			c.Assert(req.FormValue("origin"), check.Equals, "app-deploy")
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/secret/deploy")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	cmd := AppDeploy{GuessingCommand: guessCommand}
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(calledTimes, check.Equals, 2)
}

func (s *S) TestDeployImage(c *check.C) {
	calledTimes := 0
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			if req.Body != nil {
				defer req.Body.Close()
			}
			if calledTimes == 1 {
				return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/secret")
			}
			image := req.FormValue("image")
			c.Assert(image, check.Equals, "registr.com/image-to-deploy")
			c.Assert(req.Header.Get("Content-Type"), check.Equals, "application/x-www-form-urlencoded")
			c.Assert(req.FormValue("origin"), check.Equals, "image")
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/secret/deploy")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	cmd := AppDeploy{GuessingCommand: guessCommand}
	cmd.Flags().Parse(true, []string{"-i", "registr.com/image-to-deploy"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(calledTimes, check.Equals, 2)
}

func (s *S) TestDeployRunWithMessage(c *check.C) {
	calledTimes := 0
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: bytes.NewBufferString("")}
	err := targz(&ctx, &buf, nil, "testdata", "..")
	c.Assert(err, check.IsNil)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "deploy worked\nOK\n", Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			if req.Body != nil {
				defer req.Body.Close()
			}
			if calledTimes == 1 {
				return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/secret")
			}
			file, _, transErr := req.FormFile("file")
			c.Assert(transErr, check.IsNil)
			content, transErr := ioutil.ReadAll(file)
			c.Assert(transErr, check.IsNil)
			c.Assert(content, check.DeepEquals, buf.Bytes())
			c.Assert(req.Header.Get("Content-Type"), check.Matches, "multipart/form-data; boundary=.*")
			c.Assert(req.FormValue("origin"), check.Equals, "app-deploy")
			c.Assert(req.FormValue("message"), check.Equals, "my awesome deploy")
			return req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/apps/secret/deploy")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	cmd := AppDeploy{GuessingCommand: guessCommand}
	cmd.Flags().Parse(true, []string{"-m", "my awesome deploy"})
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(calledTimes, check.Equals, 2)
}

func (s *S) TestDeployAuthNotOK(c *check.C) {
	calledTimes := 0
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "Forbidden", Status: http.StatusForbidden},
		CondFunc: func(req *http.Request) bool {
			calledTimes++
			return req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/apps/secret")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.ErrorMatches, "Forbidden")
	c.Assert(calledTimes, check.Equals, 1)
}

func (s *S) TestDeployRunNotOK(c *check.C) {
	trans := cmdtest.Transport{Message: "deploy worked\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.Equals, cmd.ErrAbortCommand)
}

func (s *S) TestDeployRunFileNotFound(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"/tmp/something/that/doesn't/really/exist/im/sure"},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
}

func (s *S) TestDeployRunWithoutArgsAndImage(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "You should provide at least one file or a docker image to deploy.\n")
}

func (s *S) TestDeployRunWithArgsAndImage(c *check.C) {
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	trans := cmdtest.Transport{Message: "OK\n", Status: http.StatusOK}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	command.Flags().Parse(true, []string{"-i", "registr.com/image-to-deploy"})
	err := command.Run(&ctx, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "You can't deploy files and docker image at the same time.\n")
}

func (s *S) TestDeployRunRequestFailure(c *check.C) {
	trans := cmdtest.Transport{Message: "app not found\n", Status: http.StatusNotFound}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"testdata", ".."},
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeploy{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "app not found\n")
}

func (s *S) TestTargz(c *check.C) {
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: &buf}
	var gzipBuf, tarBuf bytes.Buffer
	err := targz(&ctx, &gzipBuf, nil, "testdata/deploy", "..")
	c.Assert(err, check.IsNil)
	gzipReader, err := gzip.NewReader(&gzipBuf)
	c.Assert(err, check.IsNil)
	_, err = io.Copy(&tarBuf, gzipReader)
	c.Assert(err, check.IsNil)
	tarReader := tar.NewReader(&tarBuf)
	var headers []string
	var contents []string
	for header, err := tarReader.Next(); err == nil; header, err = tarReader.Next() {
		headers = append(headers, header.Name)
		if !header.FileInfo().IsDir() {
			content, err := ioutil.ReadAll(tarReader)
			c.Assert(err, check.IsNil)
			contents = append(contents, string(content))
		}
	}
	expected := []string{
		"testdata/deploy", "testdata/deploy/directory", "testdata/deploy/directory/file.txt",
		"testdata/deploy/file1.txt", "testdata/deploy/file2.txt",
	}
	sort.Strings(expected)
	sort.Strings(headers)
	c.Assert(headers, check.DeepEquals, expected)
	expectedContents := []string{"wat\n", "something happened\n", "twice\n"}
	sort.Strings(expectedContents)
	sort.Strings(contents)
	c.Assert(contents, check.DeepEquals, expectedContents)
	c.Assert(buf.String(), check.Equals, `Warning: skipping ".."`)
}

func (s *S) TestTargzSingleDirectory(c *check.C) {
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: &buf}
	var gzipBuf, tarBuf bytes.Buffer
	err := targz(&ctx, &gzipBuf, nil, "testdata/deploy")
	c.Assert(err, check.IsNil)
	gzipReader, err := gzip.NewReader(&gzipBuf)
	c.Assert(err, check.IsNil)
	_, err = io.Copy(&tarBuf, gzipReader)
	c.Assert(err, check.IsNil)
	tarReader := tar.NewReader(&tarBuf)
	var headers []string
	var contents []string
	for header, err := tarReader.Next(); err == nil; header, err = tarReader.Next() {
		headers = append(headers, header.Name)
		if !header.FileInfo().IsDir() {
			content, err := ioutil.ReadAll(tarReader)
			c.Assert(err, check.IsNil)
			contents = append(contents, string(content))
		}
	}
	expected := []string{".", "directory", "directory/file.txt", "file1.txt", "file2.txt"}
	sort.Strings(expected)
	sort.Strings(headers)
	c.Assert(headers, check.DeepEquals, expected)
	expectedContents := []string{"wat\n", "something happened\n", "twice\n"}
	sort.Strings(expectedContents)
	sort.Strings(contents)
	c.Assert(contents, check.DeepEquals, expectedContents)
}

func (s *S) TestTargzSymlink(c *check.C) {
	if runtime.GOOS == "windows" {
		c.Skip("no symlink support on windows")
	}
	var buf bytes.Buffer
	ctx := cmd.Context{Stderr: &buf}
	var gzipBuf, tarBuf bytes.Buffer
	err := targz(&ctx, &gzipBuf, nil, "testdata-symlink", "..")
	c.Assert(err, check.IsNil)
	gzipReader, err := gzip.NewReader(&gzipBuf)
	c.Assert(err, check.IsNil)
	_, err = io.Copy(&tarBuf, gzipReader)
	c.Assert(err, check.IsNil)
	tarReader := tar.NewReader(&tarBuf)
	var headers [][]string
	for header, err := tarReader.Next(); err == nil; header, err = tarReader.Next() {
		if header.Linkname != "" {
			headers = append(headers, []string{header.Name, header.Linkname})
		}
	}
	expected := [][]string{{"testdata-symlink/link", "test"}}
	c.Assert(headers, check.DeepEquals, expected)
}

func (s *S) TestTargzFailure(c *check.C) {
	var stderr bytes.Buffer
	ctx := cmd.Context{Stderr: &stderr}
	var buf bytes.Buffer
	err := targz(&ctx, &buf, nil, "/tmp/something/that/definitely/doesn't/exist/right", "testdata")
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Matches, ".*(no such file or directory|cannot find the path specified).*")
}

func (s *S) TestDeployListInfo(c *check.C) {
	var cmd AppDeployList
	c.Assert(cmd.Info(), check.NotNil)
}

func (s *S) TestAppDeployList(c *check.C) {
	var stdout, stderr bytes.Buffer
	result := `
[
  {
    "ID": "54c92d91a46ec0e78501d86b",
    "App": "test",
    "Timestamp": "2015-01-27T18:42:25.725Z",
    "Duration": 18709653486,
    "Commit": "54c92d91a46ec0e78501d86b54c92d91a46ec0e78501d86b",
    "Error": "",
    "Image": "tsuru/app-test:v3",
    "User": "admin@example.com",
    "Origin": "git",
    "CanRollback": true
  },
  {
    "ID": "54c922d0a46ec0e78501d84e",
    "App": "test",
    "Timestamp": "2015-01-28T18:56:32.583Z",
    "Duration": 18781564759,
    "Commit": "",
    "Error": "",
    "Image": "tsuru/app-test:v2",
    "User": "admin@example.com",
    "Origin": "app-deploy",
    "CanRollback": true
  },
  {
    "ID": "54c918a7a46ec0e78501d831",
    "App": "test",
    "Timestamp": "2015-01-28T19:13:11.498Z",
    "Duration": 26064205176,
    "Commit": "",
    "Error": "my-error",
    "Image": "tsuru/app-test:v1",
    "User": "",
    "Origin": "rollback",
    "CanRollback": false
  }
]
`
	timestamps := []string{
		"2015-01-27T18:42:25.725Z",
		"2015-01-28T18:56:32.583Z",
		"2015-01-28T19:13:11.498Z",
	}
	var formatted []string
	for _, t := range timestamps {
		parsed, _ := time.Parse(time.RFC3339, t)
		formatted = append(formatted, parsed.Local().Format(time.Stamp))
	}
	red := "\x1b[0;31;10m"
	reset := "\x1b[0m"
	expected := `+-----------------------+---------------+-------------------+-------------------------+----------+
| Image (Rollback)      | Origin        | User              | Date (Duration)         | Error    |
+-----------------------+---------------+-------------------+-------------------------+----------+
| ` + red + `tsuru/app-test:v1` + reset + `     | ` + red + `rollback` + reset + `      |                   | ` + red + formatted[2] + ` (00:26)` + reset + ` | ` + red + `my-error` + reset + ` |
+-----------------------+---------------+-------------------+-------------------------+----------+
| tsuru/app-test:v2 (*) | app-deploy    | admin@example.com | ` + formatted[1] + ` (00:18) |          |
+-----------------------+---------------+-------------------+-------------------------+----------+
| tsuru/app-test:v3 (*) | git (54c92d9) | admin@example.com | ` + formatted[0] + ` (00:18) |          |
+-----------------------+---------------+-------------------+-------------------------+----------+
`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: result, Status: http.StatusOK}}, nil, manager)
	command := AppDeployList{}
	command.Flags().Parse(true, []string{"--app", "test"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestDeployRunAppWithouDeploy(c *check.C) {
	trans := cmdtest.Transport{Message: "", Status: http.StatusNoContent}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := cmdtest.FakeGuesser{Name: "secret"}
	guessCommand := cmd.GuessingCommand{G: &fake}
	command := AppDeployList{GuessingCommand: guessCommand}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "App secret has no deploy.\n")
}

func (s *S) TestAppDeployRollbackInfo(c *check.C) {
	c.Assert((&AppDeployRollback{}).Info(), check.NotNil)
}

func (s *S) TestAppDeployRollback(c *check.C) {
	var (
		called         bool
		stdout, stderr bytes.Buffer
	)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"my-image"},
	}
	expectedOut := "-- deployed --"
	msg := tsuruIo.SimpleJsonMessage{Message: expectedOut}
	result, err := json.Marshal(msg)
	c.Assert(err, check.IsNil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			called = true
			method := req.Method == "POST"
			path := strings.HasSuffix(req.URL.Path, "/apps/arrakis/deploy/rollback")
			image := req.FormValue("image") == "my-image"
			rollback := req.FormValue("origin") == "rollback"
			return method && path && image && rollback
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := AppDeployRollback{}
	command.Flags().Parse(true, []string{"--app", "arrakis", "-y"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(called, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, expectedOut)
}
func (s *S) TestProcessTsuruIgnore(c *check.C) {
	dir, _ := os.Getwd()
	tests := []struct {
		name    string
		pattern string
		dirPath []string
		want    []string
	}{
		{
			pattern: "*.txt",
			dirPath: []string{filepath.Join(dir, "testdata", "deploy")},
			want:    []string{filepath.Join(dir, "testdata/deploy/file1.txt"), filepath.Join(dir, "testdata/deploy/file2.txt"), filepath.Join(dir, "testdata/deploy/directory/file.txt")},
		},
	}
	for _, tt := range tests {
		got, err := processTsuruIgnore(tt.pattern, tt.dirPath...)
		c.Assert(err, check.IsNil)
		c.Assert(got, check.DeepEquals, tt.want)
	}
}

func (s *S) TestIgnoreFiles(c *check.C) {
	wd, _ := os.Getwd()
	var buf bytes.Buffer
	var ignore []string
	ignorePatterns := []string{"*.txt"}
	for _, pattern := range ignorePatterns {
		ignPats, _ := processTsuruIgnore(pattern, filepath.Join(wd, "testdata/deploy2"))
		ignore = append(ignore, ignPats...)
	}
	ctx := cmd.Context{Stderr: &buf}
	var gzipBuf, tarBuf bytes.Buffer
	err := targz(&ctx, &gzipBuf, ignore, "testdata/deploy2")
	c.Assert(err, check.IsNil)
	gzipReader, err := gzip.NewReader(&gzipBuf)
	c.Assert(err, check.IsNil)
	_, err = io.Copy(&tarBuf, gzipReader)
	c.Assert(err, check.IsNil)
	tarReader := tar.NewReader(&tarBuf)
	var headers []string
	var contents []string
	for header, err := tarReader.Next(); err == nil; header, err = tarReader.Next() {
		headers = append(headers, header.Name)
		if !header.FileInfo().IsDir() {
			content, err := ioutil.ReadAll(tarReader)
			c.Assert(err, check.IsNil)
			contents = append(contents, string(content))
		}
	}
	expected := []string{
		".", "directory", "directory/dir2",
	}
	sort.Strings(expected)
	sort.Strings(headers)
	c.Assert(headers, check.DeepEquals, expected)
	sort.Strings(contents)
	c.Assert(contents, check.IsNil)
}

func (s *S) TestIgnoreFolder(c *check.C) {
	wd, _ := os.Getwd()
	var buf bytes.Buffer
	var ignore []string
	ignorePatterns := []string{"directory"}
	for _, pattern := range ignorePatterns {
		ignPats, _ := processTsuruIgnore(pattern, filepath.Join(wd, "testdata/deploy2"))
		ignore = append(ignore, ignPats...)
	}
	ctx := cmd.Context{Stderr: &buf}
	var gzipBuf, tarBuf bytes.Buffer
	err := targz(&ctx, &gzipBuf, ignore, "testdata/deploy2")
	c.Assert(err, check.IsNil)
	gzipReader, err := gzip.NewReader(&gzipBuf)
	c.Assert(err, check.IsNil)
	_, err = io.Copy(&tarBuf, gzipReader)
	c.Assert(err, check.IsNil)
	tarReader := tar.NewReader(&tarBuf)
	var headers []string
	var contents []string
	for header, err := tarReader.Next(); err == nil; header, err = tarReader.Next() {
		headers = append(headers, header.Name)
		if !header.FileInfo().IsDir() {
			content, err := ioutil.ReadAll(tarReader)
			c.Assert(err, check.IsNil)
			contents = append(contents, string(content))
		}
	}
	expected := []string{
		".", "file1.txt", "file2.txt",
	}
	sort.Strings(expected)
	sort.Strings(headers)
	c.Assert(headers, check.DeepEquals, expected)
}

func (s *S) TestIgnoreRelativeFolder(c *check.C) {
	wd, _ := os.Getwd()
	var buf bytes.Buffer
	var ignore []string
	ignorePatterns := []string{"*/dir2"}
	for _, pattern := range ignorePatterns {
		ignPats, _ := processTsuruIgnore(pattern, filepath.Join(wd, "testdata/deploy2"))
		ignore = append(ignore, ignPats...)
	}
	ctx := cmd.Context{Stderr: &buf}
	var gzipBuf, tarBuf bytes.Buffer
	err := targz(&ctx, &gzipBuf, ignore, "testdata/deploy2")
	c.Assert(err, check.IsNil)
	gzipReader, err := gzip.NewReader(&gzipBuf)
	c.Assert(err, check.IsNil)
	_, err = io.Copy(&tarBuf, gzipReader)
	c.Assert(err, check.IsNil)
	tarReader := tar.NewReader(&tarBuf)
	var headers []string
	var contents []string
	for header, err := tarReader.Next(); err == nil; header, err = tarReader.Next() {
		headers = append(headers, header.Name)
		if !header.FileInfo().IsDir() {
			content, err := ioutil.ReadAll(tarReader)
			c.Assert(err, check.IsNil)
			contents = append(contents, string(content))
		}
	}
	expected := []string{".", "directory", "directory/file.txt", "file1.txt", "file2.txt"}
	sort.Strings(expected)
	sort.Strings(headers)
	c.Assert(headers, check.DeepEquals, expected)
}

func (s *S) TestIgnoreRelativeFile(c *check.C) {
	wd, _ := os.Getwd()
	var buf bytes.Buffer
	var ignore []string
	ignorePatterns := []string{"directory/dir2/*"}
	for _, pattern := range ignorePatterns {
		ignPats, _ := processTsuruIgnore(pattern, filepath.Join(wd, "testdata/deploy2"))
		ignore = append(ignore, ignPats...)
	}
	ctx := cmd.Context{Stderr: &buf}
	var gzipBuf, tarBuf bytes.Buffer
	err := targz(&ctx, &gzipBuf, ignore, "testdata/deploy2")
	c.Assert(err, check.IsNil)
	gzipReader, err := gzip.NewReader(&gzipBuf)
	c.Assert(err, check.IsNil)
	_, err = io.Copy(&tarBuf, gzipReader)
	c.Assert(err, check.IsNil)
	tarReader := tar.NewReader(&tarBuf)
	var headers []string
	var contents []string
	for header, err := tarReader.Next(); err == nil; header, err = tarReader.Next() {
		headers = append(headers, header.Name)
		if !header.FileInfo().IsDir() {
			content, err := ioutil.ReadAll(tarReader)
			c.Assert(err, check.IsNil)
			contents = append(contents, string(content))
		}
	}
	expected := []string{".", "directory", "directory/dir2", "directory/file.txt", "file1.txt", "file2.txt"}
	sort.Strings(expected)
	sort.Strings(headers)
	c.Assert(headers, check.DeepEquals, expected)
}
