package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestBoostrapConfigNoConfig(c *check.C) {
	fsystem = &fstest.RecordingFs{}
	stat, err := fsystem.Stat(configPath)
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
	expected.lastChanges = conf.lastChanges
	c.Assert(conf, check.DeepEquals, expected)
}

func (s *S) TestBoostrapConfigFromFile(c *check.C) {
	fsystem = &fstest.RecordingFs{}
	f, _ := fsystem.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	fmt.Fprintf(f, `{
  "SchemaVersion": "6.6.6",
  "LastUpdate": "2020-12-25T16:00:59Z"
}`)
	f.Close()

	conf := bootstrapConfig()
	c.Assert(conf, check.NotNil)
	expected := &ConfigType{
		SchemaVersion: "6.6.6",
		LastUpdate:    time.Date(2020, 12, 25, 16, 00, 59, 0, time.UTC),
		lastChanges:   time.Date(2020, 12, 25, 16, 00, 59, 0, time.UTC),
	}
	c.Assert(conf, check.DeepEquals, expected)
}

func (s *S) TestBoostrapConfigWrongFormatBackupFile(c *check.C) {
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	fsystem = &fstest.RecordingFs{}

	f, _ := fsystem.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	f.WriteString("wrong format")
	f.Close()
	nowTimeStr := time.Now().UTC().Format("2006-01-02_15:04:05")
	backupConfigPath := configPath + "." + nowTimeStr + ".bak"

	conf := bootstrapConfig()
	c.Assert(conf, check.NotNil)
	expected := newDefaultConf()
	expected.lastChanges = conf.lastChanges
	c.Assert(conf, check.DeepEquals, expected)

	stdoutBytes, err := io.ReadAll(stdout)
	c.Assert(err, check.IsNil)
	c.Assert(string(stdoutBytes), check.DeepEquals, "")
	stderrBytes, err := io.ReadAll(stderr)
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(string(stderrBytes), "Error parsing "), check.Equals, true, check.Commentf("Got: %s", string(stderrBytes)))
	c.Assert(strings.Contains(string(stderrBytes), "Backing up current file to "), check.Equals, true, check.Commentf("Got: %s", string(stderrBytes)))
	c.Assert(strings.Contains(string(stderrBytes), "A new configuration will be saved"), check.Equals, true, check.Commentf("Got: %s", string(stderrBytes)))

	stat, err := fsystem.Stat(backupConfigPath)
	c.Assert(err, check.IsNil)
	c.Assert(stat, check.NotNil)
}

func (s *S) TestConfig(c *check.C) {
	config = nil
	conf1 := getConfig()
	c.Assert(conf1, check.NotNil)
	conf2 := getConfig()
	c.Assert(conf1, check.Equals, conf2)
}

func (s *S) TesthasChanges(c *check.C) {
	conf := &ConfigType{
		LastUpdate:  time.Date(2020, 12, 25, 16, 00, 59, 0, time.UTC),
		lastChanges: time.Date(2020, 12, 25, 16, 00, 59, 0, time.UTC),
	}
	c.Assert(conf.hasChanges(), check.Equals, false)

	conf = &ConfigType{
		LastUpdate:  time.Date(2020, 12, 25, 16, 00, 59, 1, time.UTC),
		lastChanges: time.Date(2020, 12, 25, 16, 00, 59, 0, time.UTC),
	}
	c.Assert(conf.hasChanges(), check.Equals, false)

	conf = &ConfigType{
		LastUpdate:  time.Date(2020, 12, 25, 16, 00, 59, 0, time.UTC),
		lastChanges: time.Date(2020, 12, 25, 16, 00, 59, 1, time.UTC),
	}
	c.Assert(conf.hasChanges(), check.Equals, true)

	conf = nil
	c.Assert(conf.hasChanges(), check.Equals, false)

	conf = newDefaultConf()
	c.Assert(conf.hasChanges(), check.Equals, true)
}

func (s *S) TestSaveChanges(c *check.C) {
	fsystem = &fstest.RecordingFs{}
	f, _ := fsystem.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	originalContent := `{
  "SchemaVersion": "6.6.6",
  "LastUpdate": "2020-12-25T16:00:59Z"
}`
	fmt.Fprint(f, originalContent)
	f.Close()

	config = bootstrapConfig()
	c.Assert(config, check.NotNil)

	now := time.Now().UTC()
	config.SchemaVersion = "6.6.7"
	config.LastUpdate = now
	SaveChanges() // no changes

	f, _ = fsystem.Open(configPath)
	bytesRead, err := io.ReadAll(f)
	f.Close()
	c.Assert(err, check.IsNil)
	c.Assert(string(bytesRead), check.Equals, originalContent)

	now = config.LastUpdate.Add(10 * time.Millisecond)
	config.lastChanges = now
	SaveChanges()
	f, _ = fsystem.Open(configPath)
	bytesRead, err = io.ReadAll(f)
	f.Close()
	c.Assert(err, check.IsNil)
	c.Assert(string(bytesRead), check.Equals, fmt.Sprintf(`{
  "SchemaVersion": "6.6.7",
  "LastUpdate": "%s"
}`, now.Format(time.RFC3339Nano)),
	)
	config = nil
}
