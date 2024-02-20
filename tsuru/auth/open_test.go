package auth

import (
	"runtime"
	"testing"

	"github.com/tsuru/tsuru/exec/exectest"
	"gopkg.in/check.v1"
)

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

type S struct{}

func (s *S) TestOpen(c *check.C) {
	fexec := exectest.FakeExecutor{}
	execut = &fexec
	defer func() {
		execut = nil
	}()
	url := "http://someurl"
	err := open(url)
	c.Assert(err, check.IsNil)
	if runtime.GOOS == "linux" {
		c.Assert(fexec.ExecutedCmd("xdg-open", []string{url}), check.Equals, true)
	} else {
		c.Assert(fexec.ExecutedCmd("open", []string{url}), check.Equals, true)
	}
}
