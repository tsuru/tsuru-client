package client

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	type ret struct {
		Name 	string 	`json:"jobName"`
		Status 	string 	`json:"status"`
	}
	expected := "Job \"loucoAbreu\" has been created!\nUse job info to check the status of the job.\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `{"jobName":"loucoAbreu","status":"success"}`, Status: http.StatusCreated},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.13/jobs")
			c.Assert(r.Method, check.Equals, "POST")
			var rr tsuru.InputJob
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			fmt.Printf("debug: 	%s\n", string(data))
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