package client

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/tsuru/tsuru/cmd"
)

type Init struct{}

func (i *Init) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "init",
		Usage: "init",
		Desc:  `Creates a standard example of .tsuruignore , tsuru.yaml and Procfile on the current project directory.`,
	}
}

func (i *Init) Run(context *cmd.Context, client *cmd.Client) error {
	err := createInitFiles()
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("Initialized Tsuru sample files: `Procfile`, `.tsuruignore` and `tsuru.yaml`, for more info please refer to the docs: docs.tsuru.io\n")
	context.Stdout.Write([]byte(msg))
	return copyGitIgnore()
}

func createInitFiles() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	fi, err := os.Open(wd)
	if err != nil {
		return err
	}
	defer fi.Close()
	dirFiles, err := fi.Readdir(0)
	if err != nil {
		return err
	}
	initFiles := map[string]string{
		".tsuruignore": "",
		"Procfile":     "",
		"tsuru.yaml":   "",
	}
	for _, f := range dirFiles {
		if _, ok := initFiles[f.Name()]; ok {
			delete(initFiles, f.Name())
			continue
		}
	}
	for f := range initFiles {
		_, errC := os.Create(f)
		if errC != nil {
			return errC
		}
	}
	return nil
}

func copyGitIgnore() error {
	in, err := os.Open(".gitignore")
	if err != nil {
		dotGit := []byte(".git\n.gitignore\n")
		return ioutil.WriteFile(".tsuruignore", dotGit, 0644)
	}
	defer func() {
		if errC := in.Close(); errC != nil {
			err = errC
		}
	}()
	out, err := os.OpenFile(".tsuruignore", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer func() {
		if errC := out.Close(); errC != nil {
			err = errC
		}
	}()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	_, err = out.WriteString("\n.git\n.gitignore\n")
	return err
}
