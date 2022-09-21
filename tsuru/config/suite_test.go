package config

import (
	"testing"
	"time"

	"gopkg.in/check.v1"
)

type S struct{}

var _ = check.Suite(&S{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) SetUpTest(c *check.C) {
	privConfig = nil
	nowUTC = func() time.Time { return time.Now().UTC() } // so we can test time-dependent sh!t
}