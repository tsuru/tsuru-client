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

func (s *S) TestBrokerAddInfo(c *check.C) {
	c.Assert((&BrokerAdd{}).Info(), check.NotNil)
}

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
				Config: tsuru.ServiceBrokerConfig{
					AuthConfig: tsuru.ServiceBrokerConfigAuthConfig{
						BasicAuthConfig: tsuru.ServiceBrokerConfigAuthConfigBasicAuthConfig{
							Password: "password",
							Username: "username",
						},
						BearerConfig: tsuru.ServiceBrokerConfigAuthConfigBearerConfig{
							Token: "ABCDE",
						},
					},
					Context: map[string]string{
						"p1": "v1",
						"p2": "v2",
					},
					CacheExpirationSeconds: 15 * 60,
				},
			})
			return true
		},
	}
	manager := cmd.NewManagerPanicExiter("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerAdd{}
	command.Flags().Parse(true, []string{"-t", "ABCDE", "-p", "password", "-u", "username", "-c", "p1=v1", "-c", "p2=v2", "--cache", "15m"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestBrokerAddEmptyAuth(c *check.C) {
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
				Config: tsuru.ServiceBrokerConfig{
					Context: map[string]string{
						"p1": "v1",
						"p2": "v2",
					},
				},
			})
			return true
		},
	}
	manager := cmd.NewManagerPanicExiter("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerAdd{}
	command.Flags().Parse(true, []string{"-c", "p1=v1", "-c", "p2=v2"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestBrokerAddDefaultCacheExpiration(c *check.C) {
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
			var ret tsuru.ServiceBroker
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret.Config.CacheExpirationSeconds, check.Equals, int32(0))
			return true
		},
	}
	manager := cmd.NewManagerPanicExiter("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerAdd{}
	command.Flags().Parse(true, nil)
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestBrokerUpdateInfo(c *check.C) {
	c.Assert((&BrokerUpdate{}).Info(), check.NotNil)
}
func (s *S) TestBrokerUpdate(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Service broker successfully updated.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"br1", "http://x.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.7/brokers/br1")
			c.Assert(r.Method, check.Equals, "PUT")
			var ret tsuru.ServiceBroker
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.ServiceBroker{
				Name: "br1",
				URL:  "http://x.com",
				Config: tsuru.ServiceBrokerConfig{
					AuthConfig: tsuru.ServiceBrokerConfigAuthConfig{
						BasicAuthConfig: tsuru.ServiceBrokerConfigAuthConfigBasicAuthConfig{
							Password: "password",
							Username: "username",
						},
						BearerConfig: tsuru.ServiceBrokerConfigAuthConfigBearerConfig{
							Token: "ABCDE",
						},
					},
					Context: map[string]string{
						"p1": "v1",
						"p2": "v2",
					},
					CacheExpirationSeconds: 2 * 60 * 60,
				},
			})
			return true
		},
	}
	manager := cmd.NewManagerPanicExiter("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerUpdate{}
	command.Flags().Parse(true, []string{"-t", "ABCDE", "-p", "password", "-u", "username", "-c", "p1=v1", "-c", "p2=v2", "--cache", "2h"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestBrokerUpdateEmptyAuth(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Service broker successfully updated.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"br1", "http://x.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.7/brokers/br1")
			c.Assert(r.Method, check.Equals, "PUT")
			var ret tsuru.ServiceBroker
			c.Assert(r.Header.Get("Content-Type"), check.Equals, "application/json")
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret, check.DeepEquals, tsuru.ServiceBroker{
				Name: "br1",
				URL:  "http://x.com",
				Config: tsuru.ServiceBrokerConfig{
					Context: map[string]string{
						"p1": "v1",
						"p2": "v2",
					},
				},
			})
			return true
		},
	}
	manager := cmd.NewManagerPanicExiter("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerUpdate{}
	command.Flags().Parse(true, []string{"-c", "p1=v1", "-c", "p2=v2"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestBrokerUpdateNoCache(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Service broker successfully updated.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"br1", "http://x.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			var ret tsuru.ServiceBroker
			data, err := ioutil.ReadAll(r.Body)
			c.Assert(err, check.IsNil)
			err = json.Unmarshal(data, &ret)
			c.Assert(err, check.IsNil)
			c.Assert(ret.Config.CacheExpirationSeconds, check.Equals, int32(-1))
			return true
		},
	}
	manager := cmd.NewManagerPanicExiter("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerUpdate{}
	command.Flags().Parse(true, []string{"--no-cache"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestBrokerUpdateErrorWithCacheAndNoCache(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"br1", "http://x.com"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Error("should not make the request")
			return false
		},
	}
	manager := cmd.NewManagerPanicExiter("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerUpdate{}
	command.Flags().Parse(true, []string{"--cache", "30m", "--no-cache"})
	err := command.Run(&context, client)
	c.Assert(err, check.ErrorMatches, "Can't set --cache and --no-cache flags together.")
}

func (s *S) TestBrokerDeleteInfo(c *check.C) {
	c.Assert((&BrokerDelete{}).Info(), check.NotNil)
}
func (s *S) TestBrokerDelete(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := "Service broker successfully deleted.\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"br1"},
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.7/brokers/br1")
			c.Assert(r.Method, check.Equals, "DELETE")
			return true
		},
	}
	manager := cmd.NewManagerPanicExiter("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerDelete{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestBrokerListInfo(c *check.C) {
	c.Assert((&BrokerList{}).Info(), check.NotNil)
}
func (s *S) TestBrokerList(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `+-------+-------------------------------------------------+----------+--------+-------------------------------+
| Name  | URL                                             | Insecure | Auth   | Context                       |
+-------+-------------------------------------------------+----------+--------+-------------------------------+
| aws   | https://192.168.99.100:31767/aws-service-broker | true     | Bearer | Namespace: aws-service-broker |
|       |                                                 |          |        | Platform: tsuru               |
+-------+-------------------------------------------------+----------+--------+-------------------------------+
| azure | https://localhost:9090                          | false    | Basic  |                               |
|       |                                                 |          |        |                               |
+-------+-------------------------------------------------+----------+--------+-------------------------------+
`
	brokers := []tsuru.ServiceBroker{
		{
			Name: "aws",
			URL:  "https://192.168.99.100:31767/aws-service-broker",
			Config: tsuru.ServiceBrokerConfig{
				Insecure: true,
				AuthConfig: tsuru.ServiceBrokerConfigAuthConfig{
					BearerConfig: tsuru.ServiceBrokerConfigAuthConfigBearerConfig{
						Token: "xpto",
					},
				},
				Context: map[string]string{
					"Namespace": "aws-service-broker",
					"Platform":  "tsuru",
				},
			},
		},
		{
			Name: "azure",
			URL:  "https://localhost:9090",
			Config: tsuru.ServiceBrokerConfig{
				AuthConfig: tsuru.ServiceBrokerConfigAuthConfig{
					BasicAuthConfig: tsuru.ServiceBrokerConfigAuthConfigBasicAuthConfig{
						Password: "pass",
						Username: "user",
					},
				},
			},
		},
	}
	body, err := json.Marshal(map[string]interface{}{"brokers": brokers})
	c.Assert(err, check.IsNil)
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	trans := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(body), Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			c.Assert(r.URL.Path, check.Equals, "/1.7/brokers")
			c.Assert(r.Method, check.Equals, "GET")
			return true
		},
	}
	manager := cmd.NewManagerPanicExiter("admin", "0.1", "admin-ver", &stdout, &stderr, nil, nil)
	client := cmd.NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := BrokerList{}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}
