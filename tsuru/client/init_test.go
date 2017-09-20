package client

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	check "gopkg.in/check.v1"
)

func (s *S) TestInitCreateInitFiles(c *check.C) {
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	tmpDir := os.TempDir()
	err = os.Chdir(tmpDir)
	c.Assert(err, check.IsNil)
	err = createInitFiles()
	c.Assert(err, check.IsNil)
	tpath, err := os.Open(tmpDir)
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
	testPath := "testdata/deploy3"
	err = os.Chdir(filepath.Join(wd, testPath))
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	err = copyGitIgnore()
	c.Assert(err, check.IsNil)
	tsuruignore, err := ioutil.ReadFile(".tsuruignore")
	c.Assert(err, check.IsNil)
	c.Assert(string(tsuruignore), check.Equals, expected)
	err = ioutil.WriteFile(".tsuruignore", []byte(""), 0644)
	c.Assert(err, check.IsNil)
}

func (s *S) TestCopyGitIgnoreWithoutGitIgnore(c *check.C) {
	expected := ".git\n.gitignore\n"
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	tmpDir := os.TempDir()
	err = os.Chdir(tmpDir)
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	err = copyGitIgnore()
	c.Assert(err, check.IsNil)
	tsuruignore, err := ioutil.ReadFile(".tsuruignore")
	c.Assert(err, check.IsNil)
	c.Assert(string(tsuruignore), check.Equals, expected)
	err = os.Remove(filepath.Join(tmpDir, ".tsuruignore"))
	c.Assert(err, check.IsNil)
}
