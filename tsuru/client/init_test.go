package client

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tsuru/tsuru/cmd"

	check "gopkg.in/check.v1"
)

func (s *S) TestInitCreatesFiles(c *check.C) {
	wd, err := os.Getwd()
	c.Assert(err, check.IsNil)
	testPath := "testdata/deploy"
	err = os.Chdir(filepath.Join(wd, testPath))
	c.Assert(err, check.IsNil)
	defer os.Chdir(wd)
	var stdout bytes.Buffer
	context := cmd.Context{Stdout: &stdout}
	client := cmd.NewClient(&http.Client{}, nil, manager)
	cmd := Init{}
	err = cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	tpath, err := os.Open(filepath.Join(wd, testPath))
	c.Assert(err, check.IsNil)
	defer tpath.Close()
	content, err := tpath.Readdir(0)
	c.Assert(err, check.IsNil)
	var addedFiles []string
	for _, c := range content {
		if (c.Name() == ".tsuruignore") || (c.Name() == "Procfile") || (c.Name() == "tsuru.yaml") {
			addedFiles = append(addedFiles, c.Name())
		}
	}
	for _, c := range addedFiles {
		err = os.Remove(c)
	}
	c.Assert(err, check.IsNil)
	if len(addedFiles) != 3 {
		err = errors.New("Tsuru init failed to create a file")
	}
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.NotNil)
}

func (s *S) TestInitInfo(c *check.C) {
	c.Assert((&Init{}).Info(), check.NotNil)
}
