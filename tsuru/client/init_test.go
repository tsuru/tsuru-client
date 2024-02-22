package client

import (
	"bytes"
	"errors"
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
	defer os.RemoveAll(filepath.Join(wd, deploy3, "fakeDir3"))
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
	defer os.WriteFile(".tsuruignore", []byte(""), 0644)
	tsuruignore, err := os.ReadFile(".tsuruignore")
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
	defer os.Remove(filepath.Join(tmpDir, ".tsuruignore"))
	tsuruignore, err := os.ReadFile(".tsuruignore")
	c.Assert(err, check.IsNil)
	c.Assert(string(tsuruignore), check.Equals, expected)
}

func (s *S) TestInitRun(c *check.C) {
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	fakeRunDir := filepath.Join(wd, "testdata", "deploy3", "fakeRun")
	err = os.Mkdir(fakeRunDir, os.ModePerm)
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(fakeRunDir)
	defer os.Chdir(wd)
	err = os.Chdir(fakeRunDir)
	c.Assert(err, check.IsNil)
	var stdout bytes.Buffer
	context := cmd.Context{Stdout: &stdout}
	cmd := Init{}
	err = cmd.Run(&context)
	c.Assert(err, check.IsNil)
	fkRun, err := os.Open(fakeRunDir)
	c.Assert(err, check.IsNil)
	defer fkRun.Close()
	content, err := fkRun.Readdir(0)
	c.Assert(err, check.IsNil)
	c.Assert(len(content), check.Equals, 3)
}
