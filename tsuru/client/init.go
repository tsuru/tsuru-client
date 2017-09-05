package client

import (
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
	return nil
}
