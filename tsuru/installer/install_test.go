// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"io/ioutil"
	"os"

	"gopkg.in/check.v1"
)

func (s *S) TestParseConfigFileNoFile(c *check.C) {
	dmConfig, err := parseConfigFile("")
	c.Assert(err, check.IsNil)
	c.Assert(dmConfig, check.DeepEquals, defaultDockerMachineConfig)
}

func (s *S) TestParseConfigFile(c *check.C) {
	conf := `
name: tsuru-test
ca-path: /tmp/certs
driver:
    name: amazonec2
    options:
        opt1: option1-value
`
	err := ioutil.WriteFile("/tmp/config.yml", []byte(conf), 0644)
	if err != nil {
		c.Fatal("Failed to write config file for test")
	}
	defer os.Remove("/tmp/config.yml")
	expected := &DockerMachineConfig{
		DriverName: "amazonec2",
		DriverOpts: map[string]interface{}{
			"opt1": "option1-value",
		},
		CAPath: "/tmp/certs",
		Name:   "tsuru-test",
	}
	dmConfig, err := parseConfigFile("/tmp/config.yml")
	c.Assert(err, check.IsNil)
	c.Assert(dmConfig, check.DeepEquals, expected)
}
