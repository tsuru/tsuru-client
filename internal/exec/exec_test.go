// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

var endLine = "\n"

func TestFakeExecCommand(t *testing.T) {
	t.Run("empty fake exec", func(t *testing.T) {
		fakeE := FakeExec{}
		err := fakeE.Command(ExecuteOptions{})
		assert.NoError(t, err)
	})

	t.Run("fake exec with output", func(t *testing.T) {
		fakeE := FakeExec{
			OutStderr: "error output",
			OutStdout: "standard output",
			OutErr:    fmt.Errorf("error"),
		}
		stderr, stdout := bytes.Buffer{}, bytes.Buffer{}
		err := fakeE.Command(ExecuteOptions{
			Stdout: &stdout,
			Stderr: &stderr,
		})
		assert.ErrorIs(t, err, fakeE.OutErr)
		assert.Equal(t, fakeE.OutStdout, stdout.String())
		assert.Equal(t, fakeE.OutStderr, stderr.String())
	})
}

func TestOsExec(t *testing.T) {
	t.Parallel()
	t.Run("err no command", func(t *testing.T) {
		ex := OsExec{}
		err := ex.Command(ExecuteOptions{})
		assert.Error(t, err)
	})

	t.Run("empty echo", func(t *testing.T) {
		ex := OsExec{}
		stdout, stderr := bytes.Buffer{}, bytes.Buffer{}
		err := ex.Command(ExecuteOptions{
			Cmd:    "echo",
			Stdout: &stdout,
			Stderr: &stderr,
		})
		assert.NoError(t, err)
		assert.Equal(t, endLine, stdout.String())
		assert.Equal(t, "", stderr.String())
	})

	t.Run("echo 123 456", func(t *testing.T) {
		ex := OsExec{}
		stdout, stderr := bytes.Buffer{}, bytes.Buffer{}
		err := ex.Command(ExecuteOptions{
			Cmd:    "echo",
			Args:   []string{"123", "456"},
			Stdout: &stdout,
			Stderr: &stderr,
		})
		assert.NoError(t, err)
		assert.Equal(t, "123 456"+endLine, stdout.String())
		assert.Equal(t, "", stderr.String())
	})

	t.Run("echo 123\\n456", func(t *testing.T) {
		ex := OsExec{}
		stdout, stderr := bytes.Buffer{}, bytes.Buffer{}
		err := ex.Command(ExecuteOptions{
			Cmd:    "echo",
			Args:   []string{"123\\n456"},
			Stdout: &stdout,
			Stderr: &stderr,
		})
		assert.NoError(t, err)
		assert.Equal(t, "123\\n456"+endLine, stdout.String())
		assert.Equal(t, "", stderr.String())
	})
}

func init() {
	if runtime.GOOS == "windows" {
		endLine = "\r\n"
	}
}
