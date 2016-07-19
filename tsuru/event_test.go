package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

var okEvt = `
{
  "ID": {
    "Target": {
      "Name": "",
      "Value": ""
    },
    "ObjId": "578e3908413daf5fd9891aac"
  },
  "UniqueID": "578e3908413daf5fd9891aac",
  "StartTime": "2016-07-19T11:28:24.686-03:00",
  "EndTime": "2016-07-19T11:29:22.01-03:00",
  "Target": {
    "Name": "app",
    "Value": "myapp"
  },
  "StartCustomData": {
    "Kind": 3,
    "Data": "JQQAAANhcHAAXwMAAANlbnYATAEAAANUU1VSVV9BUFBfVE9LRU4AbwAAAAJuYW1lABAAAABUU1VSVV9BUFBfVE9LRU4AAnZhbHVlACkAAAA5MDBkMDcyYzAwN2E3ZjgwMjlmZjdiMmNlNzkzNTFiMjRlYTY0YjNmAAhwdWJsaWMAAAJpbnN0YW5jZW5hbWUAAQAAAAAAA1RTVVJVX0FQUE5BTUUATQAAAAJuYW1lAA4AAABUU1VSVV9BUFBOQU1FAAJ2YWx1ZQAJAAAAb3RoZXJhcHAACHB1YmxpYwAAAmluc3RhbmNlbmFtZQABAAAAAAADVFNVUlVfQVBQRElSAF0AAAACbmFtZQANAAAAVFNVUlVfQVBQRElSAAJ2YWx1ZQAaAAAAL2hvbWUvYXBwbGljYXRpb24vY3VycmVudAAIcHVibGljAAACaW5zdGFuY2VuYW1lAAEAAAAAAAACZnJhbWV3b3JrAAcAAABweXRob24AAm5hbWUACQAAAG90aGVyYXBwAAJpcAAYAAAAb3RoZXJhcHAuZmFrZXJvdXRlci5jb20ABGNuYW1lAAUAAAAABHRlYW1zABYAAAACMAAKAAAAdHN1cnV0ZWFtAAACdGVhbW93bmVyAAoAAAB0c3VydXRlYW0AAm93bmVyABsAAABtYWpvcnRvbUBncm91bmRjb250cm9sLmNvbQAQZGVwbG95cwAAAAAACHVwZGF0ZXBsYXRmb3JtAAADbG9jawB6AAAACGxvY2tlZAABAnJlYXNvbgAlAAAAUE9TVCAvYXBwcy9vdGhlcmFwcC9yZXBvc2l0b3J5L2Nsb25lAAJvd25lcgAbAAAAbWFqb3J0b21AZ3JvdW5kY29udHJvbC5jb20ACWFjcXVpcmVkYXRlAPZkwQRWAQAAAANwbGFuAF8AAAACX2lkAA4AAABhdXRvZ2VuZXJhdGVkABJtZW1vcnkAAAAAAAAAAAASc3dhcAAAAAAAAAAAABBjcHVzaGFyZQBkAAAACGRlZmF1bHQAAAJyb3V0ZXIAAQAAAAAAAnBvb2wABgAAAHBvb2wxAAJkZXNjcmlwdGlvbgABAAAAAANyb3V0ZXJvcHRzAAUAAAAAA3F1b3RhABsAAAAQbGltaXQA/////xBpbnVzZQAAAAAAAAACY29tbWl0AAEAAAAAAmFyY2hpdmV1cmwAGAAAAGh0dHA6Ly9zb21ldGhpbmcudGFyLmd6ABJmaWxlc2l6ZQAAAAAAAAAAAAJ1c2VyABsAAABtYWpvcnRvbUBncm91bmRjb250cm9sLmNvbQACaW1hZ2UAAQAAAAACb3JpZ2luAAEAAAAACHJvbGxiYWNrAAAIYnVpbGQAAAJraW5kAAwAAABhcmNoaXZlLXVybAACbWVzc2FnZQABAAAAAAA="
  },
  "EndCustomData": {
    "Kind": 3,
    "Data": "GgAAAAJpbWFnZQAKAAAAYXBwLWltYWdlAAA="
  },
  "OtherCustomData": {
    "Kind": 0,
    "Data": null
  },
  "Kind": {
    "Type": "permission",
    "Name": "app.deploy"
  },
  "Owner": {
    "Type": "user",
    "Name": "someone@removed.com"
  },
  "LockUpdateTime": "2016-07-19T11:28:24.686-03:00",
  "Error": "",
  "Log": "Obtaining file:///home/application/current (from -r /home/application/current/requirements.txt (line 1))\n  appxxxx 0.1.0 does not provide the extra 'tests'\nRequirement already satisfied (use --upgrade to upgrade): Flask in /home/application/.app_env/lib/python2.7/site-packages (from appxxxx==0.1.0-\u003e-r /home/application/current/requirements.txt (line 1))\nRequirement already satisfied (use --upgrade to upgrade): redis in /home/application/.app_env/lib/python2.7/site-packages (from appxxxx==0.1.0-\u003e-r /home/application/current/requirements.txt (line 1))\nRequirement already satisfied (use --upgrade to upgrade): click\u003e=2.0 in /home/application/.app_env/lib/python2.7/site-packages (from Flask-\u003eappxxxx==0.1.0-\u003e-r /home/application/current/requirements.txt (line 1))\nRequirement already satisfied (use --upgrade to upgrade): itsdangerous\u003e=0.21 in /home/application/.app_env/lib/python2.7/site-packages (from Flask-\u003eappxxxx==0.1.0-\u003e-r /home/application/current/requirements.txt (line 1))\nRequirement already satisfied (use --upgrade to upgrade): Werkzeug\u003e=0.7 in /home/application/.app_env/lib/python2.7/site-packages (from Flask-\u003eappxxxx==0.1.0-\u003e-r /home/application/current/requirements.txt (line 1))\nRequirement already satisfied (use --upgrade to upgrade): Jinja2\u003e=2.4 in /home/application/.app_env/lib/python2.7/site-packages (from Flask-\u003eappxxxx==0.1.0-\u003e-r /home/application/current/requirements.txt (line 1))\nRequirement already satisfied (use --upgrade to upgrade): MarkupSafe in /home/application/.app_env/lib/python2.7/site-packages (from Jinja2\u003e=2.4-\u003eFlask-\u003eappxxxx==0.1.0-\u003e-r /home/application/current/requirements.txt (line 1))\nInstalling collected packages: appxxxx\n  Running setup.py develop for appxxxx\nSuccessfully installed appxxxx-0.1.0\n\n---- Building application image ----\n ---\u003e Sending image to repository (0.06MB)\n ---\u003e Cleaning up\n\n---- Starting 1 new unit [web: 1] ----\n ---\u003e Started unit 447fc77ff3 [web]\n\n---- Binding and checking 1 new unit ----\n ---\u003e healthcheck fail(447fc77ff3): Get http://10.0.0.1:32774/: dial tcp 10.0.0.1:32774: getsockopt: connection refused. Trying again in 3s\n ---\u003e healthcheck successful(447fc77ff3)\n ---\u003e Bound and checked unit 447fc77ff3 [web]\n\n---- Adding routes to new units ----\n ---\u003e Added route to unit 447fc77ff3 [web]\n\n---- Removing routes from old units ----\n ---\u003e Removed route from unit d535216e4b [web]\n\n---- Removing 1 old unit ----\n ---\u003e Removed old unit d535216e4b [web]\n\n---- Unbinding 1 old unit ----\n ---\u003e Removed bind for old unit d535216e4b [web]\n",
  "RemoveDate": "0001-01-01T00:00:00Z",
  "CancelInfo": {
    "Owner": "",
    "StartTime": "0001-01-01T00:00:00Z",
    "AckTime": "0001-01-01T00:00:00Z",
    "Reason": "",
    "Asked": false,
    "Canceled": false
  },
  "Cancelable": true,
  "Running": false
}
`

var errEvt = `
{
  "ID": {
    "Target": {
      "Name": "",
      "Value": ""
    },
    "ObjId": "5787bcc8413daf2aeb040730"
  },
  "UniqueID": "5787bcc8413daf2aeb040730",
  "StartTime": "2016-07-14T13:24:40.66-03:00",
  "EndTime": "2016-07-14T13:25:00.349-03:00",
  "Target": {
    "Name": "container",
    "Value": "94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f"
  },
  "StartCustomData": {
    "Kind": 3,
    "Data": "kQEAAAdfaWQAV46KeNV3FmPu0YcNAmlkACEAAAAyMjcxN2MzZDdjZDg1MTEzMzllZGJjYzliZjdiOTMxZQACYXBwbmFtZQAGAAAAbXlhcHAAAnByb2Nlc3NuYW1lAAQAAAB3ZWIAAnR5cGUABwAAAHB5dGhvbgACaXAAAQAAAAACaG9zdGFkZHIACgAAADEyNy4wLjAuMQACaG9zdHBvcnQAAQAAAAACcHJpdmF0ZWtleQABAAAAAAJzdGF0dXMACAAAAGNyZWF0ZWQAAnZlcnNpb24AAwAAAHYxAAJpbWFnZQANAAAAdHN1cnUvcHl0aG9uAAJuYW1lAAEAAAAAAnVzZXIABQAAAHJvb3QAAmJ1aWxkaW5naW1hZ2UADQAAAHRzdXJ1L3B5dGhvbgAJbGFzdHN0YXR1c3VwZGF0ZQAAKNPtfMf//wlsYXN0c3VjY2Vzc3N0YXR1c3VwZGF0ZQC5UsgEVgEAAAlsb2NrZWR1bnRpbAAAKNPtfMf//wJleHBvc2VkcG9ydAABAAAAAAA="
  },
  "EndCustomData": {
    "Kind": 3,
    "Data": "JAEAAAJpZAABAAAAAAJhcHBuYW1lAAEAAAAAAnByb2Nlc3NuYW1lAAEAAAAAAnR5cGUAAQAAAAACaXAAAQAAAAACaG9zdGFkZHIAAQAAAAACaG9zdHBvcnQAAQAAAAACcHJpdmF0ZWtleQABAAAAAAJzdGF0dXMAAQAAAAACdmVyc2lvbgABAAAAAAJpbWFnZQABAAAAAAJuYW1lAAEAAAAAAnVzZXIAAQAAAAACYnVpbGRpbmdpbWFnZQABAAAAAAlsYXN0c3RhdHVzdXBkYXRlAAAo0+18x///CWxhc3RzdWNjZXNzc3RhdHVzdXBkYXRlAAAo0+18x///CWxvY2tlZHVudGlsAAAo0+18x///AmV4cG9zZWRwb3J0AAEAAAAAAA=="
  },
  "OtherCustomData": {
    "Kind": 0,
    "Data": null
  },
  "Kind": {
    "Type": "internal",
    "Name": "healer"
  },
  "Owner": {
    "Type": "internal",
    "Name": ""
  },
  "LockUpdateTime": "2016-07-14T13:24:40.66-03:00",
  "Error": "Error healing container \"94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f\": Error trying to heal containers 94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f: couldn't move container: Error moving some containers. - Moving unit 94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f for \"myapp\" from 10.0.0.4...\nError moving container: Error moving unit 94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f Caused by: Post http://10.0.0.5:8000/services/myapp/destinations: dial tcp 10.0.0.5:8000: getsockopt: no route to host\n",
  "Log": "",
  "RemoveDate": "0001-01-01T00:00:00Z",
  "CancelInfo": {
    "Owner": "",
    "StartTime": "0001-01-01T00:00:00Z",
    "AckTime": "0001-01-01T00:00:00Z",
    "Reason": "",
    "Asked": false,
    "Canceled": false
  },
  "Cancelable": false,
  "Running": false
}
`
var evtsData = fmt.Sprintf("[%s, %s]", okEvt, errEvt)

func (s *S) TestEventList(c *check.C) {
	os.Setenv("TSURU_DISABLE_COLORS", "1")
	defer os.Unsetenv("TSURU_DISABLE_COLORS")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: evtsData, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.1/events"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := eventList{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `+--------------------------+---------------------------------+---------+-----------+------------+-------------------------+
| ID                       | Start (duration)                | Success | Owner     | Kind       | Target                  |
+--------------------------+---------------------------------+---------+-----------+------------+-------------------------+
| 578e3908413daf5fd9891aac | 19 Jul 16 11:28 -0300 (57.324s) | true    | someone@… | app.deploy | app: myapp              |
| 5787bcc8413daf2aeb040730 | 14 Jul 16 13:24 -0300 (19.689s) | false   |           | healer     | container: 94d3140395a8 |
+--------------------------+---------------------------------+---------+-----------+------------+-------------------------+
`
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestEventInfo(c *check.C) {
	os.Setenv("TSURU_DISABLE_COLORS", "1")
	defer os.Unsetenv("TSURU_DISABLE_COLORS")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"578e3908413daf5fd9891aac"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: okEvt, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.1/events/578e3908413daf5fd9891aac"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := eventInfo{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `ID:       578e3908413daf5fd9891aac
Start:    19 Jul 16 11:28 -0300
End:      19 Jul 16 11:29 -0300 (57.324s)
Target:   app(myapp)
Kind:     permission(app.deploy)
Owner:    user(someone@removed.com)
Success:  true
Canceled: false
Start Custom Data:
    app:
      cname: []
      deploys: 0
      description: ""
      env:
        TSURU_APPDIR:
          instancename: ""
          name: TSURU_APPDIR
          public: false
          value: /home/application/current
        TSURU_APPNAME:
          instancename: ""
          name: TSURU_APPNAME
          public: false
          value: otherapp
        TSURU_APP_TOKEN:
          instancename: ""
          name: TSURU_APP_TOKEN
          public: false
          value: 900d072c007a7f8029ff7b2ce79351b24ea64b3f
      framework: python
      ip: otherapp.fakerouter.com
      lock:
        acquiredate: "2016-07-19T17:03:18.39-03:00"
        locked: true
        owner: majortom@groundcontrol.com
        reason: POST /apps/otherapp/repository/clone
      name: otherapp
      owner: majortom@groundcontrol.com
      plan:
        _id: autogenerated
        cpushare: 100
        default: false
        memory: 0
        router: ""
        swap: 0
      pool: pool1
      quota:
        inuse: 0
        limit: -1
      routeropts: {}
      teamowner: tsuruteam
      teams:
      - tsuruteam
      updateplatform: false
    archiveurl: http://something.tar.gz
    build: false
    commit: ""
    filesize: 0
    image: ""
    kind: archive-url
    message: ""
    origin: ""
    rollback: false
    user: majortom@groundcontrol.com

End Custom Data:
    image: app-image

Log:
    Obtaining file:///home/application/current (from -r /home/application/current/requirements.txt (line 1))
      appxxxx 0.1.0 does not provide the extra 'tests'
    Requirement already satisfied (use --upgrade to upgrade): Flask in /home/application/.app_env/lib/python2.7/site-packages (from appxxxx==0.1.0->-r /home/application/current/requirements.txt (line 1))
    Requirement already satisfied (use --upgrade to upgrade): redis in /home/application/.app_env/lib/python2.7/site-packages (from appxxxx==0.1.0->-r /home/application/current/requirements.txt (line 1))
    Requirement already satisfied (use --upgrade to upgrade): click>=2.0 in /home/application/.app_env/lib/python2.7/site-packages (from Flask->appxxxx==0.1.0->-r /home/application/current/requirements.txt (line 1))
    Requirement already satisfied (use --upgrade to upgrade): itsdangerous>=0.21 in /home/application/.app_env/lib/python2.7/site-packages (from Flask->appxxxx==0.1.0->-r /home/application/current/requirements.txt (line 1))
    Requirement already satisfied (use --upgrade to upgrade): Werkzeug>=0.7 in /home/application/.app_env/lib/python2.7/site-packages (from Flask->appxxxx==0.1.0->-r /home/application/current/requirements.txt (line 1))
    Requirement already satisfied (use --upgrade to upgrade): Jinja2>=2.4 in /home/application/.app_env/lib/python2.7/site-packages (from Flask->appxxxx==0.1.0->-r /home/application/current/requirements.txt (line 1))
    Requirement already satisfied (use --upgrade to upgrade): MarkupSafe in /home/application/.app_env/lib/python2.7/site-packages (from Jinja2>=2.4->Flask->appxxxx==0.1.0->-r /home/application/current/requirements.txt (line 1))
    Installing collected packages: appxxxx
      Running setup.py develop for appxxxx
    Successfully installed appxxxx-0.1.0

    ---- Building application image ----
     ---> Sending image to repository (0.06MB)
     ---> Cleaning up

    ---- Starting 1 new unit [web: 1] ----
     ---> Started unit 447fc77ff3 [web]

    ---- Binding and checking 1 new unit ----
     ---> healthcheck fail(447fc77ff3): Get http://10.0.0.1:32774/: dial tcp 10.0.0.1:32774: getsockopt: connection refused. Trying again in 3s
     ---> healthcheck successful(447fc77ff3)
     ---> Bound and checked unit 447fc77ff3 [web]

    ---- Adding routes to new units ----
     ---> Added route to unit 447fc77ff3 [web]

    ---- Removing routes from old units ----
     ---> Removed route from unit d535216e4b [web]

    ---- Removing 1 old unit ----
     ---> Removed old unit d535216e4b [web]

    ---- Unbinding 1 old unit ----
     ---> Removed bind for old unit d535216e4b [web]

`
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestEventInfoWithError(c *check.C) {
	os.Setenv("TSURU_DISABLE_COLORS", "1")
	defer os.Unsetenv("TSURU_DISABLE_COLORS")
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"5787bcc8413daf2aeb040730"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: errEvt, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.1/events/5787bcc8413daf2aeb040730"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	command := eventInfo{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `ID:       5787bcc8413daf2aeb040730
Start:    14 Jul 16 13:24 -0300
End:      14 Jul 16 13:25 -0300 \(19\.689s\)
Target:   container\(94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f\)
Kind:     internal\(healer\)
Owner:    internal\(\)
Success:  false
Error:    "Error healing container \\"94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f\\": Error trying to heal containers 94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f: couldn't move container: Error moving some containers\. - Moving unit 94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f for \\"myapp\\" from 10\.0\.0\.4\.\.\.\\nError moving container: Error moving unit 94d3140395a85e4a60b06de26f6a51270d7b762c65cc9478e2c544ae4d7fb82f Caused by: Post http://10\.0\.0\.5:8000/services/myapp/destinations: dial tcp 10\.0\.0\.5:8000: getsockopt: no route to host\\n"
Canceled: false
Start Custom Data:
    _id: 578e8a78d5771663eed1870d
    appname: myapp
    buildingimage: tsuru/python
    exposedport: ""
    hostaddr: 127\.0\.0\.1
    hostport: ""
    id: 22717c3d7cd8511339edbcc9bf7b931e
    image: tsuru/python
    ip: ""
    laststatusupdate: "0001-01-01T00:00:00Z"
    lastsuccessstatusupdate: ".*?"
    lockeduntil: "0001-01-01T00:00:00Z"
    name: ""
    privatekey: ""
    processname: web
    status: created
    type: python
    user: root
    version: v1

End Custom Data:
    appname: ""
    buildingimage: ""
    exposedport: ""
    hostaddr: ""
    hostport: ""
    id: ""
    image: ""
    ip: ""
    laststatusupdate: "0001-01-01T00:00:00Z"
    lastsuccessstatusupdate: "0001-01-01T00:00:00Z"
    lockeduntil: "0001-01-01T00:00:00Z"
    name: ""
    privatekey: ""
    processname: ""
    status: ""
    type: ""
    user: ""
    version: ""

`
	c.Assert(stdout.String(), check.Matches, expected)
}
