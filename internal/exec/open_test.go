// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	fEx := FakeExec{}
	err := Open(&fEx, "http://localhost?a=1&b=2")
	assert.NoError(t, err)
	switch runtime.GOOS {
	case "darwin":
		assert.Equal(t, "open", fEx.CalledOpts.Cmd)
		assert.EqualValues(t, []string{"http://localhost?a=1&b=2"}, fEx.CalledOpts.Args)
	case "windows":
		assert.Equal(t, "cmd", fEx.CalledOpts.Cmd)
		assert.Equal(t, []string{"/c", "start", "", "http://localhost?a=1^&b=2"}, fEx.CalledOpts.Args)
	default:
		assert.Equal(t, "xdg-open", fEx.CalledOpts.Cmd)
		assert.Equal(t, []string{"http://localhost?a=1&b=2"}, fEx.CalledOpts.Args)
	}
}
