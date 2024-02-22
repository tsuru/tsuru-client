// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestBoostrapConfigNoConfig(c *check.C) {
	SetFileSystem(&fstest.RecordingFs{})
	defer func() {
		ResetFileSystem()
	}()
	now := nowUTC()
	nowUTC = func() time.Time { return now } // mocking nowUTC

	stat, err := Filesystem().Stat(configPath)
	errorMsg := err.Error()
	c.Assert(stat, check.IsNil)
	c.Assert(
		(errorMsg == "The system cannot find the file specified." ||
			errorMsg == "no such file or directory"),
		check.Equals,
		true,
		check.Commentf("Got error: %v", err))

	conf := bootstrapConfig()
	c.Assert(conf, check.NotNil)
	expected := newDefaultConf()
	c.Assert(conf, check.DeepEquals, expected)
}

func (s *S) TestBoostrapConfigFromFile(c *check.C) {
	now := nowUTC()
	nowUTC = func() time.Time { return now }
	SetFileSystem(&fstest.RecordingFs{})
	defer func() {
		ResetFileSystem()
	}()
	f, err := Filesystem().OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	c.Assert(err, check.IsNil)
	fmt.Fprintf(f, `{
  "SchemaVersion": "6.6.6",
  "LastUpdate": "2020-12-25T16:00:59Z"
}`)
	f.Close()

	conf := bootstrapConfig()
	conf.originalContent = ""
	c.Assert(conf, check.NotNil)
	expected := newDefaultConf()
	expected.SchemaVersion = "6.6.6"
	expected.LastUpdate = time.Date(2020, 12, 25, 16, 00, 59, 0, time.UTC)
	expected.originalContent = ""

	c.Assert(conf, check.DeepEquals, expected)
}

func (s *S) TestBoostrapConfigWrongFormatBackupFile(c *check.C) {
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	SetFileSystem(&fstest.RecordingFs{})
	defer func() {
		ResetFileSystem()
	}()
	now := nowUTC()
	nowUTC = func() time.Time { return now } // mocking nowUTC

	f, err := Filesystem().OpenFile(configPath, os.O_WRONLY|os.O_CREATE, 0755)
	c.Assert(err, check.IsNil)
	f.WriteString("wrong format")
	f.Close()
	backupConfigPath := configPath + "." + nowUTC().Format("2006-01-02_15:04:05") + ".bak"

	conf := bootstrapConfig()
	c.Assert(conf, check.NotNil)
	expected := newDefaultConf()
	c.Assert(conf, check.DeepEquals, expected)

	stdoutBytes, err := io.ReadAll(stdout)
	c.Assert(err, check.IsNil)
	c.Assert(string(stdoutBytes), check.DeepEquals, "")
	stderrBytes, err := io.ReadAll(stderr)
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(string(stderrBytes), "Error parsing "), check.Equals, true, check.Commentf("Got: %s", string(stderrBytes)))
	c.Assert(strings.Contains(string(stderrBytes), "Backing up current file to "), check.Equals, true, check.Commentf("Got: %s", string(stderrBytes)))
	c.Assert(strings.Contains(string(stderrBytes), "A new configuration will be saved"), check.Equals, true, check.Commentf("Got: %s", string(stderrBytes)))

	stat, err := Filesystem().Stat(backupConfigPath)
	c.Assert(err, check.IsNil)
	c.Assert(stat, check.NotNil)
}

func (s *S) TestConfig(c *check.C) {
	conf1 := GetConfig()
	c.Assert(conf1, check.NotNil)
	conf2 := GetConfig()
	c.Assert(conf1, check.Equals, conf2)
}

func (s *S) TesthasChanges(c *check.C) {
	conf := newDefaultConf()
	hasChanges, err := conf.hasChanges()
	c.Assert(err, check.IsNil)
	c.Assert(hasChanges, check.Equals, true)

	originalContent, err := json.Marshal(conf)
	c.Assert(err, check.IsNil)
	conf.originalContent = string(originalContent)
	hasChanges, err = conf.hasChanges()
	c.Assert(err, check.IsNil)
	c.Assert(hasChanges, check.Equals, false)

	conf.LastUpdate = nowUTC()
	hasChanges, err = conf.hasChanges()
	c.Assert(err, check.IsNil)
	c.Assert(hasChanges, check.Equals, true)

	conf = nil
	hasChanges, err = conf.hasChanges()
	c.Assert(err, check.IsNil)
	c.Assert(hasChanges, check.Equals, false)
}

func (s *S) TestSaveChanges(c *check.C) {
	SetFileSystem(&fstest.RecordingFs{})
	defer func() {
		ResetFileSystem()
	}()
	f, err := Filesystem().OpenFile(configPath, os.O_WRONLY|os.O_CREATE, 0755)
	c.Assert(err, check.IsNil)
	originalContent := `{
  "SchemaVersion": "6.6.6",
  "LastUpdate": "2020-12-25T16:00:59Z"
}`
	fmt.Fprint(f, originalContent)
	f.Close()

	conf := GetConfig()
	c.Assert(conf.SchemaVersion, check.Equals, "6.6.6")

	// change something
	conf.SchemaVersion = "6.6.7"
	now := nowUTC()
	nowUTC = func() time.Time { return now } // stub now
	SaveChangesWithTimeout()
	f, err = Filesystem().Open(configPath)
	c.Assert(err, check.IsNil)
	bytesRead, err := io.ReadAll(f)
	f.Close()
	c.Assert(err, check.IsNil)

	var newConf ConfigType
	err = json.Unmarshal(bytesRead, &newConf)
	c.Assert(err, check.IsNil)
	c.Assert(newConf.SchemaVersion, check.Equals, "6.6.7")
	c.Assert(newConf.LastUpdate, check.Equals, now)
}
