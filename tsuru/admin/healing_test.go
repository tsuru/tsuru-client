package admin

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	check "gopkg.in/check.v1"
)

func (s *S) TestListHealingHistoryCmdInfo(c *check.C) {
	expected := cmd.Info{
		Name:  "healing-list",
		Usage: "healing-list [--node] [--container]",
		Desc:  "List healing history for nodes or containers.",
	}
	historyCmd := ListHealingHistoryCmd{}
	c.Assert(historyCmd.Info(), check.DeepEquals, &expected)
}

var healingJsonData = `[{
	"StartTime": "2014-10-23T08:00:00.000Z",
	"EndTime": "2014-10-23T08:30:00.000Z",
	"Successful": true,
	"Action": "node-healing",
	"FailingNode": {"Address": "addr1"},
	"CreatedNode": {"Address": "addr2"},
	"Error": ""
},
{
	"StartTime": "2014-10-23T10:00:00.000Z",
	"EndTime": "2014-10-23T10:30:00.000Z",
	"Successful": false,
	"Action": "node-healing",
	"FailingNode": {"Address": "addr1"},
	"CreatedNode": {"Address": "addr2"},
	"Error": ""
},
{
	"StartTime": "2014-10-23T06:00:00.000Z",
	"EndTime": "2014-10-23T06:30:00.000Z",
	"Successful": true,
	"Action": "container-healing",
	"FailingContainer": {"ID": "123456789012"},
	"CreatedContainer": {"ID": "923456789012"},
	"Error": ""
},
{
	"StartTime": "2014-10-23T08:00:00.000Z",
	"EndTime": "2014-10-23T08:30:00.000Z",
	"Successful": false,
	"Action": "container-healing",
	"FailingContainer": {"ID": "123456789012"},
	"Error": "err1"
},
{
	"StartTime": "2014-10-23T02:00:00.000Z",
	"EndTime": "2014-10-23T02:30:00.000Z",
	"Successful": false,
	"Action": "container-healing",
	"FailingContainer": {"ID": "123456789012"},
	"Error": "err1"
}]`

func (s *S) TestListHealingHistoryCmdRun(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: healingJsonData, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.3/healing"
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	healing := &ListHealingHistoryCmd{}
	err := healing.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `Node:
+-----------------+-----------------+---------+---------+---------+-------+
| Start           | Finish          | Success | Failing | Created | Error |
+-----------------+-----------------+---------+---------+---------+-------+
| Oct 23 03:00:00 | Oct 23 03:30:00 | true    | addr1   | addr2   |       |
+-----------------+-----------------+---------+---------+---------+-------+
| Oct 23 05:00:00 | Oct 23 05:30:00 | false   | addr1   | addr2   |       |
+-----------------+-----------------+---------+---------+---------+-------+
Container:
+-----------------+-----------------+---------+------------+------------+-------+
| Start           | Finish          | Success | Failing    | Created    | Error |
+-----------------+-----------------+---------+------------+------------+-------+
| Oct 23 01:00:00 | Oct 23 01:30:00 | true    | 1234567890 | 9234567890 |       |
+-----------------+-----------------+---------+------------+------------+-------+
| Oct 23 03:00:00 | Oct 23 03:30:00 | false   | 1234567890 |            | err1  |
+-----------------+-----------------+---------+------------+------------+-------+
| Oct 22 21:00:00 | Oct 22 21:30:00 | false   | 1234567890 |            | err1  |
+-----------------+-----------------+---------+------------+------------+-------+
`
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestListHealingHistoryCmdRunEmpty(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusNoContent},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.3/healing"
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	healing := &ListHealingHistoryCmd{}
	err := healing.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `Node:
+-------+--------+---------+---------+---------+-------+
| Start | Finish | Success | Failing | Created | Error |
+-------+--------+---------+---------+---------+-------+
Container:
+-------+--------+---------+---------+---------+-------+
| Start | Finish | Success | Failing | Created | Error |
+-------+--------+---------+---------+---------+-------+
`
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestListHealingHistoryCmdRunFilterNode(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: healingJsonData, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.3/healing" && req.URL.RawQuery == "filter=node"
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := &ListHealingHistoryCmd{}
	cmd.Flags().Parse(true, []string{"--node"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `Node:
+-----------------+-----------------+---------+---------+---------+-------+
| Start           | Finish          | Success | Failing | Created | Error |
+-----------------+-----------------+---------+---------+---------+-------+
| Oct 23 03:00:00 | Oct 23 03:30:00 | true    | addr1   | addr2   |       |
+-----------------+-----------------+---------+---------+---------+-------+
| Oct 23 05:00:00 | Oct 23 05:30:00 | false   | addr1   | addr2   |       |
+-----------------+-----------------+---------+---------+---------+-------+
`
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestListHealingHistoryCmdRunFilterContainer(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Stdout: &buf}
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: healingJsonData, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.3/healing" && req.URL.RawQuery == "filter=container"
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := &ListHealingHistoryCmd{}
	cmd.Flags().Parse(true, []string{"--container"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `Container:
+-----------------+-----------------+---------+------------+------------+-------+
| Start           | Finish          | Success | Failing    | Created    | Error |
+-----------------+-----------------+---------+------------+------------+-------+
| Oct 23 01:00:00 | Oct 23 01:30:00 | true    | 1234567890 | 9234567890 |       |
+-----------------+-----------------+---------+------------+------------+-------+
| Oct 23 03:00:00 | Oct 23 03:30:00 | false   | 1234567890 |            | err1  |
+-----------------+-----------------+---------+------------+------------+-------+
| Oct 22 21:00:00 | Oct 22 21:30:00 | false   | 1234567890 |            | err1  |
+-----------------+-----------------+---------+------------+------------+-------+
`
	c.Assert(buf.String(), check.Equals, expected)
}

func (s *S) TestListHealingHistoryInProgressCmdRun(c *check.C) {
	var buf bytes.Buffer
	context := cmd.Context{Stdout: &buf}
	msg := fmt.Sprintf(`[{
  	"StartTime": "2014-10-23T08:00:00.000Z",
  	"EndTime": "%s",
  	"Successful": true,
  	"Action": "container-healing",
    "FailingContainer": {"ID": "123456789012"},
    "CreatedContainer": {"ID": "923456789012"},
  	"Error": ""
  }]`, time.Time{}.Format(time.RFC3339))
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: msg, Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/1.3/healing" && req.URL.RawQuery == "filter=container"
		},
	}
	manager := cmd.Manager{}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, &manager)
	cmd := &ListHealingHistoryCmd{}
	cmd.Flags().Parse(true, []string{"--container"})
	err := cmd.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `Container:
+-----------------+-------------+---------+------------+------------+-------+
| Start           | Finish      | Success | Failing    | Created    | Error |
+-----------------+-------------+---------+------------+------------+-------+
| Oct 23 03:00:00 | in progress | true    | 1234567890 | 9234567890 |       |
+-----------------+-------------+---------+------------+------------+-------+
`
	c.Assert(buf.String(), check.Equals, expected)
}
