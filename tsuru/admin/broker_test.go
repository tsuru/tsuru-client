package admin

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

func (s *S) TestBrokerAdd(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Service broker successfully added.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"br1", "http://x.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.7/brokers")
			c.Assert(r.Method, check.Equals, "POST")
			var ret tsuru.ServiceBroker
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.ServiceBroker{
				Name: "br1",
				URL:  "http://x.com",
				Config: &tsuru.ServiceBrokerConfig{
					AuthConfig: &tsuru.ServiceBrokerConfigAuthConfig{
						BasicAuthConfig: &tsuru.ServiceBrokerConfigAuthConfigBasicAuthConfig{
							Password: "password",
							Username: "username",
						},
						BearerConfig: &tsuru.ServiceBrokerConfigAuthConfigBearerConfig{
							Token: "ABCDE",
						},
					},
					Context: map[string]string{
						"p1": "v1",
						"p2": "v2",
					},
				},
			})
			return true
		},
	}
	manager := cmd.NewManager("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerAdd{}
	command.Flags().Parse(true, []string{"-t", "ABCDE", "-p", "password", "-u", "username", "-c", "p1=v1", "-c", "p2=v2"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
