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
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	fi, err := os.Open(wd)
	if err != nil {
		return err
	}
	dirFiles, err := fi.Readdir(0)
	if err != nil {
		return err
	}
	stdExpFiles := map[string]string{".tsuruignore": "", "Procfile": "", "tsuru.yaml": ""}
	for _, f := range dirFiles {
		if _, ok := stdExpFiles[f.Name()]; ok {
			delete(stdExpFiles, f.Name())
			continue
		}
	}
	for f := range stdExpFiles {
		_, errC := os.Create(f)
		if errC != nil {
			return errC
		}
	}
	msg := fmt.Sprintf("Initialized Tsuru sample files: `Procfile`, `.tsuruignore` and `tsuru.yaml`, for more info please refer to the docs: docs.tsuru.io\n")
	context.Stdout.Write([]byte(msg))
	return copyGitIgnore()
}

func copyGitIgnore() error {
	in, err := os.Open(".gitignore")
	if err != nil {
		dotGit := []byte(".git\n")
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
	_, err = out.WriteString("\n.git\n")
	return err
}
