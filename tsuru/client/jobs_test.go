package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestJobCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"loucoAbreu", "ubuntu:latest", "echo \"vivo essa paixão\""},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job \"loucoAbreu\" has been created!\nUse job info to check the status of the job.\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `{"jobName":"loucoAbreu","status":"success"}`, Status: http.StatusCreated},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.13/jobs")
			c.Assert(r.Method, check.Equals, "POST")
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			var rr tsuru.InputJob
			err = json.Unmarshal(data, &rr)
			c.Assert(err, check.IsNil)
			c.Assert(rr, check.DeepEquals, tsuru.InputJob{
				Name: "loucoAbreu",
				Pool: "somepool",
				TeamOwner: "admin",
				Container: tsuru.InputJobContainer{
					Image: "ubuntu:latest",
					Command: []string{"echo \"vivo essa paixão\""},
				},
				Schedule: "* * * * *",
			})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobCreate{}
	command.Info()
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobCreateApiError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"loucoAbreu", "ubuntu:latest", "echo \"vivo essa paixão\""},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "500 Internal Server Error: some error occcured"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `some error occcured`, Status: http.StatusInternalServerError},
		CondFunc: func(r *http.Request) bool { return true },
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobCreate{}
	command.Info()
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *"})
	err := command.Run(&context, client)
	c.Assert(err.Error(), check.Equals, expected)
}