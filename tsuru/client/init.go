package client

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/tsuru/tsuru/cmd"
)

type Init struct{}

func (i *Init) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "init",
		Usage: "init",
		Desc: `
Creates a standard example of .tsuruignore , tsuru.yaml and Procfile 
	on the current project directory.

"Procfile" describes the components required to run an application. 
	It is the way to tell tsuru how to run your applications;

".tsuruignore" describes to tsuru what it should not add into your 
	deploy process, via "tsuru app-deploy" command and works just 
	like you're used to use .gitignore;

"tsuru.yaml" describes certain aspects of your app, like information
	about deployment hooks and deployment time health checks.`,
	}
}

func (i *Init) Run(context *cmd.Context, client *cmd.Client) error {
	err := createInitFiles()
	if err != nil {
		return err
	}
	const msg = `
Initialized Tsuru sample files: "Procfile", ".tsuruignore" and "tsuru.yaml", 
for more info please refer to "tsuru init -h" or the docs at docs.tsuru.io`
	context.Stdout.Write([]byte(msg))
	err = writeTsuruYaml()
	if err != nil {
		return err
	}
	err = writeProcfile()
	if err != nil {
		return err
	}
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

func writeTsuruYaml() error {
	const yamlSample = `
hooks:
  restart:
    # 'before' hook lists commands that will run before the unit is restarted. 
    # Commands listed in this hook will run once per unit.
    before:
      - python manage.py generate_local_file
    # 'after' hook is like before-each, but runs after restarting a unit.
    after:
      - python manage.py clear_local_cache
  # 'build' hook lists commands that will be run during deploy, when the image 
  # is being generated.
  build:
    - python manage.py collectstatic --noinput
    - python manage.py compress

# This health check will be called during the deployment process and tsuru will 
# make sure this health check is passing before continuing with the deployment process.
healthcheck:
  # 'path': Which path to call in your application. This path will be called for each unit. 
  # It is the only mandatory field, if it’s not set your health check will be ignored.
  path: /healthcheck
  # 'method': used to make the http request, defaults to GET.
  method: GET
  # 'status': Expected response code for the request, defaults to 200.
  status: 200
  # 'match': A regular expression to be matched against the request body. 
  # If it’s not set the body won’t be read and only the status code will be checked. 
  # This regular expression uses Go syntax and runs with . matching \n (s flag).
  match: .*OKAY.*
  # 'allowed_failures': Number of allowed failures before the health check considers 
  # the application as unhealthy, defaults to 0.
  allowed_failures: 0
  # 'use_in_router': Whether this health check path should also be registered in the router.
  # Please, ensure that the check is consistent to prevent units being disabled by the router.
  # Defaults to false.
  use_in_router: false`
	return ioutil.WriteFile("tsuru.yaml", []byte(yamlSample), 0644)
}

func writeProcfile() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	projectPathSplitted := strings.Split(wd, "/")
	projectName := projectPathSplitted[len(projectPathSplitted)-1]
	procfile := fmt.Sprintf("web: %s", projectName)
	return ioutil.WriteFile("Procfile", []byte(procfile), 0644)
}
