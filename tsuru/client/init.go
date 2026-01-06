package client

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	tsuruErrors "github.com/tsuru/tsuru/errors"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
)

type Init struct{}

func (i *Init) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "init",
		Usage: "init",
		Desc: `Creates a standard example of .tsuruignore, tsuru.yaml and Procfile on the current project directory.

"Procfile" describes the components required to run an application. 
	It is the way to tell tsuru how to run your applications;

".tsuruignore" describes to tsuru what it should not add into your 
	deploy process, via "tsuru app deploy" command. You can use 
	".tsuruignore" to avoid sending files that were committed, 
	but aren't necessary for running the app, like tests and 
	documentation;

"tsuru.yaml" describes certain aspects of your app, like information
	about deployment hooks and deployment time health checks.`,
	}
}

func (i *Init) Run(context *cmd.Context) (err error) {
	err = createInitFiles()
	if err != nil {
		return
	}
	const msg = `
Initialized Tsuru sample files: "Procfile", ".tsuruignore" and "tsuru.yaml", 
for more info please refer to "tsuru init -h" or the docs at docs.tsuru.io`
	_, err = context.Stdout.Write([]byte(msg))
	if err != nil {
		return
	}
	err = writeTsuruYaml()
	if err != nil {
		return
	}
	err = writeProcfile()
	if err != nil {
		return
	}
	return copyGitIgnore()
}

func createInitFiles() (err error) {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	fi, err := os.Open(wd)
	if err != nil {
		return
	}
	defer func() {
		if errClose := fi.Close(); errClose != nil {
			err = tsuruErrors.NewMultiError(err, errClose)
		}
	}()
	dirFiles, err := fi.Readdir(0)
	if err != nil {
		return
	}
	initFiles := map[string]string{
		".tsuruignore": "",
		"Procfile":     "",
		"tsuru.yaml":   "",
	}
	for _, f := range dirFiles {
		delete(initFiles, f.Name())
	}
	for f := range initFiles {
		_, err = os.Create(f)
		if err != nil {
			return
		}
	}
	return
}

func copyGitIgnore() (err error) {
	in, err := os.Open(".gitignore")
	if err != nil {
		dotGit := []byte(".git\n.gitignore\n")
		return os.WriteFile(".tsuruignore", dotGit, 0644)
	}
	defer func() {
		if errClose := in.Close(); errClose != nil {
			err = tsuruErrors.NewMultiError(err, errClose)
		}
	}()
	out, err := os.OpenFile(".tsuruignore", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return
	}
	defer func() {
		if errClose := out.Close(); errClose != nil {
			err = tsuruErrors.NewMultiError(err, errClose)
		}
	}()
	_, err = io.Copy(out, in)
	if err != nil {
		return
	}
	_, err = out.WriteString("\n.git\n.gitignore\n")
	return
}

func writeTsuruYaml() error {
	return os.WriteFile("tsuru.yaml", nil, 0644)
}

func writeProcfile() (err error) {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	projectName := filepath.Base(wd)
	procfile := fmt.Sprintf("web: %s", projectName)
	return os.WriteFile("Procfile", []byte(procfile), 0644)
}
