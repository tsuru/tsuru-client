package client

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tsuru/tsuru/cmd"

	check "gopkg.in/check.v1"
)

func (s *S) TestInitCreateInitFiles(c *check.C) {
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	deploy3 := filepath.Join("testdata", "deploy3")
	err = os.Chdir(filepath.Join(wd, deploy3))
	c.Assert(err, check.IsNil)
	err = os.Mkdir("fakeDir3", os.ModePerm)
	defer os.Remove(filepath.Join(wd, deploy3, "fakeDir3"))
	c.Assert(err, check.IsNil)
	err = os.Chdir("fakeDir3")
	c.Assert(err, check.IsNil)
	err = createInitFiles()
	c.Assert(err, check.IsNil)
	tpath, err := os.Open(filepath.Join(wd, deploy3, "fakeDir3"))
	c.Assert(err, check.IsNil)
	defer tpath.Close()
	content, err := tpath.Readdir(0)
	c.Assert(err, check.IsNil)
	var createdFiles []string
	for _, c := range content {
		if (c.Name() == ".tsuruignore") || (c.Name() == "Procfile") || (c.Name() == "tsuru.yaml") {
			createdFiles = append(createdFiles, c.Name())
		}
	}
	for _, cf := range createdFiles {
		err = os.Remove(cf)
		c.Assert(err, check.IsNil)
	}
	if len(createdFiles) != 3 {
		err = errors.New("Tsuru init failed to create a file")
	}
	c.Assert(err, check.IsNil)
}

func (s *S) TestInitInfo(c *check.C) {
	c.Assert((&Init{}).Info(), check.NotNil)
}

func (s *S) TestCopyGitIgnoreWithGitIgnore(c *check.C) {
	expected := "vendor/\n.git\n.gitignore\n"
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	testPath := filepath.Join("testdata", "deploy3")
	err = os.Chdir(filepath.Join(wd, testPath))
	c.Assert(err, check.IsNil)
	err = copyGitIgnore()
	c.Assert(err, check.IsNil)
	defer ioutil.WriteFile(".tsuruignore", []byte(""), 0644)
	tsuruignore, err := ioutil.ReadFile(".tsuruignore")
	c.Assert(err, check.IsNil)
	c.Assert(string(tsuruignore), check.Equals, expected)
}

func (s *S) TestCopyGitIgnoreWithoutGitIgnore(c *check.C) {
	expected := ".git\n.gitignore\n"
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	tmpDir := os.TempDir()
	err = os.Chdir(tmpDir)
	c.Assert(err, check.IsNil)
	err = copyGitIgnore()
	c.Assert(err, check.IsNil)
	tsuruignore, err := ioutil.ReadFile(".tsuruignore")
	c.Assert(err, check.IsNil)
	c.Assert(string(tsuruignore), check.Equals, expected)
	err = os.Remove(filepath.Join(tmpDir, ".tsuruignore"))
	c.Assert(err, check.IsNil)
}

func (s *S) TestWriteTsuruYaml(c *check.C) {
	deploy3 := filepath.Join("testdata", "deploy3")
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	tPath := filepath.Join(wd, "testdata", "deploy3", "fakeDir")
	err = os.Chdir(filepath.Join(wd, "testdata", "deploy3"))
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	err = os.Mkdir("fakeDir", os.ModePerm)
	c.Assert(err, check.IsNil)
	defer os.Remove(tPath)
	err = os.Chdir("fakeDir")
	c.Assert(err, check.IsNil)
	_, err = os.Create("tsuru.yaml")
	c.Assert(err, check.IsNil)
	err = writeTsuruYaml()
	c.Assert(err, check.IsNil)
	data, err := ioutil.ReadFile("tsuru.yaml")
	c.Assert(err, check.IsNil)
	c.Assert(data, check.NotNil)
	err = os.Remove("tsuru.yaml")
	c.Assert(err, check.IsNil)
	err = os.Chdir(filepath.Join(wd, deploy3))
	c.Assert(err, check.IsNil)
}

func (s *S) TestWriteProcfile(c *check.C) {
	deploy3 := filepath.Join("testdata", "deploy3")
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	tPath := filepath.Join(wd, deploy3, "fakeDirP")
	err = os.Mkdir(tPath, os.ModePerm)
	c.Assert(err, check.IsNil)
	defer os.Remove(tPath)
	err = os.Chdir(tPath)
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	_, err = os.Create("Procfile")
	c.Assert(err, check.IsNil)
	err = writeProcfile()
	c.Assert(err, check.IsNil)
	data, err := ioutil.ReadFile("Procfile")
	c.Assert(err, check.IsNil)
	c.Assert(data, check.NotNil)
	err = os.Chdir(filepath.Join(wd, "testdata", "deploy3"))
	c.Assert(err, check.IsNil)
	err = os.Remove(filepath.Join(tPath, "Procfile"))
	c.Assert(err, check.IsNil)
	err = os.Remove(tPath)
	c.Assert(err, check.IsNil)
}

func (s *S) TestInitRun(c *check.C) {
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	fakeRunDir := filepath.Join(wd, "testdata", "deploy3", "fakeRun")
	err = os.Mkdir(fakeRunDir, os.ModePerm)
	c.Assert(err, check.IsNil)
	defer os.Remove(fakeRunDir)
	err = os.Chdir(fakeRunDir)
	c.Assert(err, check.IsNil)
	var stdout bytes.Buffer
	context := cmd.Context{Stdout: &stdout}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	cmd := Init{}
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	fkRun, err := os.Open(fakeRunDir)
	c.Assert(err, check.IsNil)
	content, err := fkRun.Readdir(0)
	c.Assert(err, check.IsNil)
	c.Assert(len(content), check.Equals, 3)
	for _, f := range content {
		err = os.Remove(filepath.Join(fakeRunDir, f.Name()))
		c.Assert(err, check.IsNil)
	}
	err = fkRun.Close()
	c.Assert(err, check.IsNil)
	err = os.Chdir(wd)
	c.Assert(err, check.IsNil)
	err = os.Remove(fakeRunDir)
	c.Assert(err, check.IsNil)
}
