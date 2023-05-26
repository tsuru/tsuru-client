package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestJobCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"loucoAbreu", "ubuntu:latest", "\"/bin/sh -c \"echo Botafogo is in my heart\"\""},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job created\nUse \"tsuru job info loucoAbreu\" to check the status of the job\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `{"jobName":"loucoAbreu","status":"success"}`, Status: http.StatusCreated},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.13/jobs")
			c.Assert(r.Method, check.Equals, "POST")
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			var rr tsuru.InputJob
			err = json.Unmarshal(data, &rr)
			c.Assert(err, check.IsNil)
			c.Assert(rr, check.DeepEquals, tsuru.InputJob{
				Name:      "loucoAbreu",
				Pool:      "somepool",
				TeamOwner: "admin",
				Container: tsuru.InputJobContainer{
					Image:   "ubuntu:latest",
					Command: []string{"/bin/sh", "-c", "echo Botafogo is in my heart"},
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

func (s *S) TestJobCreateParseMultipleCommands(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"NiltonSantos", "ubuntu:latest", "\"/bin/sh -c \"echo Botafogo is in my heart;\" \"sleep 600\"\""},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job created\nUse \"tsuru job info NiltonSantos\" to check the status of the job\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `{"jobName":"loucoAbreu","status":"success"}`, Status: http.StatusCreated},
		CondFunc: func(r *http.Request) bool {
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			var rr tsuru.InputJob
			err = json.Unmarshal(data, &rr)
			c.Assert(err, check.IsNil)
			c.Assert(rr.Container.Command, check.DeepEquals, []string{"/bin/sh", "-c", "echo Botafogo is in my heart;", "sleep 600"})
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

func (s *S) TestJobCreateParseJSON(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"NiltonSantos", "ubuntu:latest", `["/bin/sh", "-c", "echo Botafogo is in my heart;", "sleep 600"]`},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job created\nUse \"tsuru job info NiltonSantos\" to check the status of the job\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `{"jobName":"loucoAbreu","status":"success"}`, Status: http.StatusCreated},
		CondFunc: func(r *http.Request) bool {
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			var rr tsuru.InputJob
			err = json.Unmarshal(data, &rr)
			c.Assert(err, check.IsNil)
			c.Assert(rr.Container.Command, check.DeepEquals, []string{"/bin/sh", "-c", "echo Botafogo is in my heart;", "sleep 600"})
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
		Args:   []string{"loucoAbreu", "ubuntu:latest", "\"echo \"putfire\"\""},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "500 Internal Server Error: some error occcured"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `some error occcured`, Status: http.StatusInternalServerError},
		CondFunc:  func(r *http.Request) bool { return true },
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobCreate{}
	command.Info()
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *"})
	err := command.Run(&context, client)
	c.Assert(err.Error(), check.Equals, expected)
}

func (s *S) TestJobInfo(c *check.C) {
	var stdout, stderr bytes.Buffer
	jobName := "garrincha"
	context := cmd.Context{
		Args:   []string{jobName},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := `Job: garrincha
Teams: [botafogo]
Created by: botafogo@glorioso.com
Pool: kubepool
Plan: c0.1m0.1
Schedule: * * * * *
Image: putfire:v10
Command: [/bin/sh -c sleep 600;]
Units: 1
+--------------------------+---------+----------+-----+
| Name                     | Status  | Restarts | Age |
+--------------------------+---------+----------+-----+
| garrincha-28072468-kp4jv | running | 0        |     |
+--------------------------+---------+----------+-----+

`
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: `
{
	"job": {
		"name": "garrincha",
		"teams": [
			"botafogo"
		],
		"teamOwner": "botafogo",
		"owner": "botafogo@glorioso.com",
		"plan": {
			"name": "c0.1m0.1",
			"memory": 134217728,
			"cpumilli": 100,
			"override": {
				"memory": null,
				"cpumilli": null
			}
		},
		"metadata": {
			"labels": [],
			"annotations": []
		},
		"pool": "kubepool",
		"description": "",
		"spec": {
			"schedule": "* * * * *",
			"container": {
				"image": "putfire:v10",
				"command": [
					"/bin/sh",
					"-c",
					"sleep 600;"
				]
			},
			"envs": []
		}
	},
	"units": [{
		"ID": "garrincha-28072468-kp4jv",
		"Name": "garrincha-28072468-kp4jv",
		"AppName": "",
		"ProcessName": "",
		"Type": "",
		"InternalIP": "",
		"Status": "running",
		"StatusReason": "",
		"Address": null,
		"Addresses": null,
		"Version": 0,
		"Routable": false,
		"Restarts": 0,
		"Ready": null,
		"HostAddr": "",
		"HostPort": "",
		"IP": "10.92.15.84"
	}]
}
`, Status: http.StatusOK,
		}, CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, fmt.Sprintf("/1.13/jobs/%s", jobName))
			c.Assert(r.Method, check.Equals, "GET")
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobInfo{}
	command.Info()
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobInfoApiError(c *check.C) {
	var stdout, stderr bytes.Buffer
	jobName := "garrincha"
	context := cmd.Context{
		Args:   []string{jobName},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "500 Internal Server Error: some api error"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some api error", Status: http.StatusInternalServerError},
		CondFunc:  func(r *http.Request) bool { return true },
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobInfo{}
	command.Info()
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, expected)
}

func (s *S) TestJobList(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected :=
		`+------------+----------+-----------------+---------------------------------------------------------+
| Name       | Schedule | Image           | Command                                                 |
+------------+----------+-----------------+---------------------------------------------------------+
| august     |          | midnights:v10   | /bin/sh -c date; echo Hello from the Kubernetes cluster |
+------------+----------+-----------------+---------------------------------------------------------+
| tim-mcgraw |          | fearless:latest | sleep 30                                                |
+------------+----------+-----------------+---------------------------------------------------------+
`
	timMcgrawjob := tsuru.Job{
		Name:      "tim-mcgraw",
		TeamOwner: "taylor",
		Pool:      "kubepool",
		Owner:     "taylorswift@evermore.com",
		Plan: tsuru.Plan{
			Name:     "c0.1m0.1",
			Memory:   int64(134217728),
			Cpumilli: int32(100),
		},
		Spec: tsuru.JobSpec{
			Container: tsuru.InputJobContainer{
				Image:   "fearless:latest",
				Command: []string{"sleep", "30"},
			},
		},
	}
	augustJob := tsuru.Job{
		Name:      "august",
		TeamOwner: "ts",
		Pool:      "kubepool",
		Owner:     "folklore@evermore.com",
		Plan: tsuru.Plan{
			Name:     "c0.1m0.1",
			Memory:   int64(134217728),
			Cpumilli: int32(100),
		},
		Spec: tsuru.JobSpec{
			Container: tsuru.InputJobContainer{
				Image:   "midnights:v10",
				Command: []string{"/bin/sh", "-c", "date; echo Hello from the Kubernetes cluster"},
			},
		},
	}

	jobList := []tsuru.Job{
		timMcgrawjob,
		augustJob,
	}

	messageBytes, err := json.Marshal(jobList)
	c.Assert(err, check.IsNil)

	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(messageBytes), Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.13/jobs")
			c.Assert(r.Header.Get("Accept"), check.Equals, "application/json")
			c.Assert(r.Method, check.Equals, "GET")
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobList{}
	command.Info()
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobListApiError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "500 Internal Server Error: some api error"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some api error", Status: http.StatusInternalServerError},
		CondFunc: func(r *http.Request) bool {
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobList{}
	command.Info()
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, expected)
}

func (s *S) TestJobDelete(c *check.C) {
	var stdout, stderr bytes.Buffer
	jobName := "all-time-low"
	context := cmd.Context{
		Args:   []string{jobName},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job successfully deleted\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, fmt.Sprintf("/1.13/jobs/%s", jobName))
			c.Assert(r.Method, check.Equals, "DELETE")
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobDelete{}
	command.Info()
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobDeleteApiError(c *check.C) {
	var stdout, stderr bytes.Buffer
	jobName := "all-time-low"
	context := cmd.Context{
		Args:   []string{jobName},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "500 Internal Server Error: some api error"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some api error", Status: http.StatusInternalServerError},
		CondFunc: func(r *http.Request) bool {
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobDelete{}
	command.Info()
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, expected)
}

func (s *S) TestJobTrigger(c *check.C) {
	var stdout, stderr bytes.Buffer
	jobName := "counter-strike"
	context := cmd.Context{
		Args:   []string{jobName},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job successfully triggered\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `{"status":"success"}`, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, fmt.Sprintf("/1.13/jobs/%s/trigger", jobName))
			c.Assert(r.Method, check.Equals, "POST")
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobTrigger{}
	command.Info()
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobTriggerApiError(c *check.C) {
	var stdout, stderr bytes.Buffer
	jobName := "counter-strike"
	context := cmd.Context{
		Args:   []string{jobName},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "500 Internal Server Error: some error"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some error", Status: http.StatusInternalServerError},
		CondFunc: func(r *http.Request) bool {
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := JobTrigger{}
	command.Info()
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, expected)
}
