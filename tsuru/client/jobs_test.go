// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestJobCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"loucoAbreu", "ubuntu:latest", "/bin/sh", "-c", "echo Botafogo is in my heart"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job creation with image is being deprecated. You should use 'tsuru job deploy' to set a job`s image\n" +
		"Job created\nUse \"tsuru job info loucoAbreu\" to check the status of the job\n"
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
				Container: tsuru.JobSpecContainer{
					Image:   "ubuntu:latest",
					Command: []string{"/bin/sh", "-c", "echo Botafogo is in my heart"},
				},
				Schedule:              "* * * * *",
				ActiveDeadlineSeconds: func() *int64 { r := int64(300); return &r }(),
			})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobCreate{}
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *", "-m", "300"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobCreateWithEmptyCommand(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"job-using-entrypoint", "ubuntu:latest"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job creation with image is being deprecated. You should use 'tsuru job deploy' to set a job`s image\n" +
		"Job created\nUse \"tsuru job info job-using-entrypoint\" to check the status of the job\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `{"jobName":"job-using-entrypoint","status":"success"}`, Status: http.StatusCreated},
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
				Name:      "job-using-entrypoint",
				Pool:      "somepool",
				TeamOwner: "admin",
				Container: tsuru.JobSpecContainer{
					Image: "ubuntu:latest",
				},
				Schedule: "* * * * *",
			})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobCreate{}
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobCreateManual(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"daft-punk", "ubuntu:latest", "/bin/sh", "-c", "echo digital love"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job creation with image is being deprecated. You should use 'tsuru job deploy' to set a job`s image\n" +
		"Job created\nUse \"tsuru job info daft-punk\" to check the status of the job\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `{"jobName":"daft-punk","status":"success"}`, Status: http.StatusCreated},
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
				Name:      "daft-punk",
				Pool:      "somepool",
				TeamOwner: "admin",
				Manual:    true,
				Container: tsuru.JobSpecContainer{
					Image:   "ubuntu:latest",
					Command: []string{"/bin/sh", "-c", "echo digital love"},
				},
				Schedule:              "",
				ActiveDeadlineSeconds: func() *int64 { r := int64(0); return &r }(),
			})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobCreate{}
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "--manual", "-m", "0"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobCreateParseMultipleCommands(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"NiltonSantos", "ubuntu:latest", "/bin/sh", "-c", "echo Botafogo is in my heart; sleep 600"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job creation with image is being deprecated. You should use 'tsuru job deploy' to set a job`s image\n" +
		"Job created\nUse \"tsuru job info NiltonSantos\" to check the status of the job\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `{"jobName":"loucoAbreu","status":"success"}`, Status: http.StatusCreated},
		CondFunc: func(r *http.Request) bool {
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			var rr tsuru.InputJob
			err = json.Unmarshal(data, &rr)
			c.Assert(err, check.IsNil)
			c.Assert(rr.Container.Command, check.DeepEquals, []string{"/bin/sh", "-c", "echo Botafogo is in my heart; sleep 600"})
			c.Assert(*rr.ActiveDeadlineSeconds, check.Equals, int64(0))
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobCreate{}
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *", "--max-running-time", "0"})
	err := command.Run(&context)
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
	expected := "Job creation with image is being deprecated. You should use 'tsuru job deploy' to set a job`s image\n" +
		"Job created\nUse \"tsuru job info NiltonSantos\" to check the status of the job\n"
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
	s.setupFakeTransport(&trans)
	command := JobCreate{}
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *"})
	err := command.Run(&context)
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
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: `some error occcured`, Status: http.StatusInternalServerError},
		CondFunc:  func(r *http.Request) bool { return true },
	}
	s.setupFakeTransport(&trans)
	command := JobCreate{}
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *"})
	err := command.Run(&context)
	c.Assert(err, check.ErrorMatches, ".* some error occcured")
}

func (s *S) TestJobCreateMutualScheduleAndManualError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"failjob", "ubuntu:latest", "\"echo \"fail\"\""},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Status: 200})
	command := JobCreate{}
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *", "--manual"})
	err := command.Run(&context)
	c.Assert(err.Error(), check.Equals, "cannot set both manual job and schedule options")
}

func (s *S) TestJobCreateNoJobTypeError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"failjob", "ubuntu:latest", "\"echo \"fail\"\""},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Status: 200})
	command := JobCreate{}
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool"})
	err := command.Run(&context)
	c.Assert(err.Error(), check.Equals, "schedule or manual option must be set")
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
Teams: botafogo (owner)
Created by: botafogo@glorioso.com
Cluster: my-cluster
Pool: kubepool
Plan: c0.1m0.1
Schedule: * * * * *
Image: putfire:v10
Command: [/bin/sh -c sleep 600;]
Concurrency Policy: Allow
Units: 1
+--------------------------+---------+----------+-----+
| Name                     | Status  | Restarts | Age |
+--------------------------+---------+----------+-----+
| garrincha-28072468-kp4jv | running | 0        |     |
+--------------------------+---------+----------+-----+

Service instances: 2
+----------+-----------------+
| Service  | Instance (Plan) |
+----------+-----------------+
| mongodb  | mongoapi        |
| redisapi | myredisapi      |
+----------+-----------------+

`
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: `
{
	"cluster": "my-cluster",
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
			"concurrencyPolicy": "Allow",
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
	}],
	"serviceInstanceBinds": [{"service": "redisapi", "instance": "myredisapi"}, {"service": "mongodb", "instance": "mongoapi"}]
}
`, Status: http.StatusOK,
		}, CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, fmt.Sprintf("/1.13/jobs/%s", jobName))
			c.Assert(r.Method, check.Equals, "GET")
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobInfo{}
	command.Info()
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobInfoManual(c *check.C) {
	var stdout, stderr bytes.Buffer
	jobName := "manualJob"
	context := cmd.Context{
		Args:   []string{jobName},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := `Job: manualjob
Teams: tsuru (owner)
Created by: tsuru@tsuru.io
Cluster: my-cluster
Pool: kubepool
Plan: c0.1m0.1
Image: manualjob:v0
Command: [/bin/sh -c sleep 600;]
`
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: `
{
	"cluster": "my-cluster",
	"job": {
		"name": "manualjob",
		"teams": [
			"tsuru"
		],
		"teamOwner": "tsuru",
		"owner": "tsuru@tsuru.io",
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
			"schedule": "* * 31 2 *",
			"manual": true,
			"container": {
				"image": "manualjob:v0",
				"command": [
					"/bin/sh",
					"-c",
					"sleep 600;"
				]
			},
			"envs": []
		}
	}
}
`, Status: http.StatusOK,
		}, CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, fmt.Sprintf("/1.13/jobs/%s", jobName))
			c.Assert(r.Method, check.Equals, "GET")
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobInfo{}
	command.Info()
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobInfoOptionalFieldsSet(c *check.C) {
	var stdout, stderr bytes.Buffer
	jobName := "manualJob"
	context := cmd.Context{
		Args:   []string{jobName},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := `Job: manualjob
Description: my manualjob
Teams: tsuru (owner), anotherTeam
Created by: tsuru@tsuru.io
Cluster: my-cluster
Pool: kubepool
Plan: c0.1m0.1
Image: manualjob:v0
Command: [/bin/sh -c sleep 600;]
Max Running Time: 300s
`
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{
			Message: `
{
	"cluster": "my-cluster",
	"job": {
		"name": "manualjob",
		"description": "my manualjob",
		"teams": [
			"tsuru"
		],
		"teamOwner": "tsuru",
		"teams": ["anotherTeam"],
		"owner": "tsuru@tsuru.io",
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
		"spec": {
			"schedule": "* * 31 2 *",
			"manual": true,
			"activeDeadlineSeconds": 300,
			"container": {
				"image": "manualjob:v0",
				"command": [
					"/bin/sh",
					"-c",
					"sleep 600;"
				]
			},
			"envs": []
		}
	}
}
`, Status: http.StatusOK,
		}, CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, fmt.Sprintf("/1.13/jobs/%s", jobName))
			c.Assert(r.Method, check.Equals, "GET")
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobInfo{}
	command.Info()
	err := command.Run(&context)
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
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some api error", Status: http.StatusInternalServerError},
		CondFunc:  func(r *http.Request) bool { return true },
	}
	s.setupFakeTransport(&trans)
	command := JobInfo{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, ".* some api error")
}

func (s *S) TestJobList(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected :=
		`+------------+-----------+-----------------+---------------------------------------------------------+
| Name       | Schedule  | Image           | Command                                                 |
+------------+-----------+-----------------+---------------------------------------------------------+
| august     | manual    | midnights:v10   | /bin/sh -c date; echo Hello from the Kubernetes cluster |
+------------+-----------+-----------------+---------------------------------------------------------+
| tim-mcgraw | * * * * * | fearless:latest | sleep 30                                                |
+------------+-----------+-----------------+---------------------------------------------------------+
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
			Schedule: "* * * * *",
			Manual:   false,
			Container: tsuru.JobSpecContainer{
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
			Schedule: "* * 31 2 *",
			Manual:   true,
			Container: tsuru.JobSpecContainer{
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
	s.setupFakeTransport(&trans)
	command := JobList{}
	err = command.Run(&context)
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
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some api error", Status: http.StatusInternalServerError},
		CondFunc: func(r *http.Request) bool {
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobList{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, ".* some api error")
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
	s.setupFakeTransport(&trans)
	command := JobDelete{}
	err := command.Run(&context)
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
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some api error", Status: http.StatusInternalServerError},
		CondFunc: func(r *http.Request) bool {
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobDelete{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, ".* some api error")
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
	s.setupFakeTransport(&trans)
	command := JobTrigger{}
	err := command.Run(&context)
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
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some error", Status: http.StatusInternalServerError},
		CondFunc: func(r *http.Request) bool {
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobTrigger{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, ".* some error")
}

func (s *S) TestJobUpdate(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"tulioMaravilha", "/bin/sh", "-c", "echo we like you"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job update with image is being deprecated. You should use 'tsuru job deploy' to set a job`s image\nJob updated\nUse \"tsuru job info tulioMaravilha\" to check the status of the job\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.13/jobs/tulioMaravilha")
			c.Assert(r.Method, check.Equals, "PUT")
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			var rr tsuru.InputJob
			err = json.Unmarshal(data, &rr)
			c.Assert(err, check.IsNil)
			c.Assert(rr, check.DeepEquals, tsuru.InputJob{
				Name: "tulioMaravilha",
				Container: tsuru.JobSpecContainer{
					Image:   "tsuru/scratch:latest",
					Command: []string{"/bin/sh", "-c", "echo we like you"},
				},
			})
			c.Assert(rr.ActiveDeadlineSeconds, check.IsNil)
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobUpdate{}
	c.Assert(command.Info().MinArgs, check.Equals, 1)
	unlimitedMaxArgs := 0
	c.Assert(command.Info().MaxArgs, check.Equals, unlimitedMaxArgs)
	command.Flags().Parse(true, []string{"-i", "tsuru/scratch:latest", "-m", "-200"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobUpdateJSONCommands(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"tulioMaravilha", `[ "/bin/sh", "-c", "echo we like you" ]`},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	expected := "Job update with image is being deprecated. You should use 'tsuru job deploy' to set a job`s image\nJob updated\nUse \"tsuru job info tulioMaravilha\" to check the status of the job\n"
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.13/jobs/tulioMaravilha")
			c.Assert(r.Method, check.Equals, "PUT")
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := io.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			var rr tsuru.InputJob
			err = json.Unmarshal(data, &rr)
			c.Assert(err, check.IsNil)
			c.Assert(rr, check.DeepEquals, tsuru.InputJob{
				Name: "tulioMaravilha",
				Container: tsuru.JobSpecContainer{
					Image:   "tsuru/scratch:latest",
					Command: []string{"/bin/sh", "-c", "echo we like you"},
				},
			})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobUpdate{}
	command.Flags().Parse(true, []string{"-i", "tsuru/scratch:latest"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestJobUpdateApiError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"tulioMaravilha"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some error", Status: http.StatusInternalServerError},
		CondFunc: func(r *http.Request) bool {
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobUpdate{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, ".* some error")
}

func (s *S) TestJobUpdateMutualScheduleAndManualError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"failjob", "ubuntu:latest", "\"echo \"fail\"\""},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	s.setupFakeTransport(&cmdtest.Transport{Status: 200})
	command := JobUpdate{}
	command.Flags().Parse(true, []string{"-t", "admin", "-o", "somepool", "-s", "* * * * *", "--manual"})
	err := command.Run(&context)
	c.Assert(err.Error(), check.Equals, "cannot set both manual job and schedule options")
}

func (s *S) TestJobLog(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"cerrone"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	log := `
	[{
		"Date": "2023-06-06T17:45:57.11625803Z",
		"Message": "Hello World!",
		"Source": "",
		"Name": "cerrone",
		"Type": "job",
		"Unit": "cerrone-7k2c8"
	}]
`
	expectedPrefix := "2023-06-06 12:45:57 -0500 [cerrone-7k2c8]:"
	expectedMessage := "Hello World!"
	expected := fmt.Sprintf("%s %s\n", cmd.Colorfy(expectedPrefix, "blue", "", ""), expectedMessage)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: log, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.13/jobs/cerrone/log")
			c.Assert(r.URL.Query(), check.DeepEquals, url.Values{"follow": []string{"false"}})
			c.Assert(r.Method, check.Equals, "GET")
			c.Assert(r.Header.Get("Accept"), check.Equals, "application/x-json-stream")
			c.Assert(r.Body, check.IsNil)
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobLog{}
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.DeepEquals, expected)
}

func (s *S) TestJobLogFollow(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"frank-ocean"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	log := `
	[{
		"Date": "2023-06-06T17:45:57.11625803Z",
		"Message": "Hello World!",
		"Source": "",
		"Name": "frank-ocean",
		"Type": "job",
		"Unit": "frank-ocean-7k2c8"
	}]
`
	expectedPrefix := "2023-06-06 12:45:57 -0500 [frank-ocean-7k2c8]:"
	expectedMessage := "Hello World!"
	expected := fmt.Sprintf("%s %s\n", cmd.Colorfy(expectedPrefix, "blue", "", ""), expectedMessage)
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: log, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Query(), check.DeepEquals, url.Values{"follow": []string{"true"}})
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobLog{}
	command.Flags().Parse(true, []string{"-f"})
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.DeepEquals, expected)
}

func (s *S) TestJobLogApiError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"gorillaz"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "some error", Status: http.StatusInternalServerError},
		CondFunc: func(r *http.Request) bool {
			return true
		},
	}
	s.setupFakeTransport(&trans)
	command := JobLog{}
	err := command.Run(&context)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, ".* some error")
}
