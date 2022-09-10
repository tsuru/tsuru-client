// Copyright 2018 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

func (s *S) TestWebhookCreateInfo(c *check.C) {
	c.Assert((&WebhookCreate{}).Info(), check.NotNil)
}

func (s *S) TestWebhookCreate(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Webhook successfully created.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"wh1", "http://x.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/events/webhooks")
			c.Assert(r.Method, check.Equals, "POST")
			var ret tsuru.Webhook
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.Webhook{
				Name:        "wh1",
				Url:         "http://x.com",
				EventFilter: tsuru.WebhookEventFilter{},
			})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := WebhookCreate{}
	command.Flags().Parse(true, []string{})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestWebhookCreateFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Webhook successfully created.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"wh1", "http://x.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/events/webhooks")
			c.Assert(r.Method, check.Equals, "POST")
			var ret tsuru.Webhook
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.Webhook{
				Name:        "wh1",
				Url:         "http://x.com",
				Method:      "GET",
				Body:        "xyz",
				TeamOwner:   "t1",
				Description: "desc1",
				Headers: map[string][]string{
					"a": {"b", "c"},
					"b": {"d"},
				},
				Insecure: true,
				EventFilter: tsuru.WebhookEventFilter{
					TargetTypes:  []string{"a1", "b1"},
					TargetValues: []string{"a2", "b2"},
					KindTypes:    []string{"k1"},
					KindNames:    []string{"k2"},
					SuccessOnly:  true,
				},
			})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := WebhookCreate{}
	command.Flags().Parse(true, []string{
		"--description", "desc1",
		"--team", "t1",
		"--method", "GET",
		"--body", "xyz",
		"--header", "a=b",
		"--header", "a=c",
		"--header", "b=d",
		"--target-type", "a1",
		"--target-type", "b1",
		"--target-value", "a2",
		"--target-value", "b2",
		"--kind-type", "k1",
		"--kind-name", "k2",
		"--success-only",
		"--insecure",
	})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestWebhookListInfo(c *check.C) {
	c.Assert((&WebhookList{}).Info(), check.NotNil)
}

func (s *S) TestWebhookList(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `+------+-------------+------+------------------+---------+---------+----------+--------------------+
| Name | Description | Team | URL              | Headers | Body    | Insecure | Filters            |
+------+-------------+------+------------------+---------+---------+----------+--------------------+
| wh0  |             |      | http://all       |         | <event> | false    |                    |
+------+-------------+------+------------------+---------+---------+----------+--------------------+
| wh1  | desc1       | t1   | GET http://x.com | a: b    | xyz     | true     | kind-type == k1    |
|      |             |      |                  | a: c    |         |          | kind-name == k2    |
|      |             |      |                  | b: d    |         |          | target-type == a1  |
|      |             |      |                  |         |         |          | target-type == b1  |
|      |             |      |                  |         |         |          | target-value == a2 |
|      |             |      |                  |         |         |          | target-value == b2 |
|      |             |      |                  |         |         |          | success-only       |
+------+-------------+------+------------------+---------+---------+----------+--------------------+
`
	wh := tsuru.Webhook{
		Name:        "wh1",
		Url:         "http://x.com",
		Method:      "GET",
		Body:        "xyz",
		TeamOwner:   "t1",
		Description: "desc1",
		Headers: map[string][]string{
			"a": {"b", "c"},
			"b": {"d"},
		},
		Insecure: true,
		EventFilter: tsuru.WebhookEventFilter{
			TargetTypes:  []string{"a1", "b1"},
			TargetValues: []string{"a2", "b2"},
			KindTypes:    []string{"k1"},
			KindNames:    []string{"k2"},
			SuccessOnly:  true,
		},
	}
	body, err := json.Marshal([]tsuru.Webhook{{Name: "wh0", Url: "http://all"}, wh})
	c.Assert(err, check.IsNil)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"wh1", "http://x.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(body), Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/events/webhooks")
			c.Assert(r.Method, check.Equals, "GET")
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := WebhookList{}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestWebhookDeleteInfo(c *check.C) {
	c.Assert((&WebhookDelete{}).Info(), check.NotNil)
}

func (s *S) TestWebhookDelete(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Webhook successfully deleted.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"wh1"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.6/events/webhooks/wh1")
			c.Assert(r.Method, check.Equals, "DELETE")
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := WebhookDelete{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestWebhookUpdateInfo(c *check.C) {
	c.Assert((&WebhookUpdate{}).Info(), check.NotNil)
}

func (s *S) TestWebhookUpdate(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Webhook successfully updated.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"wh1", "http://x.com"},
	}
	wh := tsuru.Webhook{
		Name:    "wh1",
		Url:     "http://x.com",
		Method:  "GET",
		Headers: map[string][]string{"a": {"b", "c"}},
	}
	body, err := json.Marshal(wh)
	c.Assert(err, check.IsNil)
	callCount := 0
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(body), Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			callCount++
			c.Assert(r.URL.Path, check.Equals, "/1.6/events/webhooks/wh1")
			if callCount == 1 {
				c.Assert(r.Method, check.Equals, "GET")
				return true
			}
			c.Assert(r.Method, check.Equals, "PUT")
			var ret tsuru.Webhook
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			var data []byte
			data, err = ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.Webhook{
				Name:        "wh1",
				Url:         "http://x.com",
				Method:      "GET",
				EventFilter: tsuru.WebhookEventFilter{},
				Headers:     map[string][]string{"a": {"b", "c"}},
			})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := WebhookUpdate{}
	command.Flags().Parse(true, []string{})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(callCount, check.Equals, 2)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestWebhookUpdateWithFlags(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Webhook successfully updated.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"wh1", "http://x.com"},
	}
	wh := tsuru.Webhook{
		Name:    "wh1",
		Url:     "http://x.com",
		Method:  "GET",
		Body:    "xyz",
		Headers: map[string][]string{"a": {"b", "c"}},
	}
	body, err := json.Marshal(wh)
	c.Assert(err, check.IsNil)
	callCount := 0
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(body), Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			callCount++
			c.Assert(r.URL.Path, check.Equals, "/1.6/events/webhooks/wh1")
			if callCount == 1 {
				c.Assert(r.Method, check.Equals, "GET")
				return true
			}
			c.Assert(r.Method, check.Equals, "PUT")
			var ret tsuru.Webhook
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			var data []byte
			data, err = ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.Webhook{
				Name:   "wh1",
				Url:    "http://y.com",
				Method: "GET",
				EventFilter: tsuru.WebhookEventFilter{
					KindNames: []string{"app.deploy"},
				},
			})
			return true
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := WebhookUpdate{}
	command.Flags().Parse(true, []string{
		"--url", "http://y.com",
		"--no-header",
		"--no-body",
		"--kind-name", "app.deploy",
	})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(callCount, check.Equals, 2)
	c.Assert(stdout.String(), check.Equals, expected)
}
