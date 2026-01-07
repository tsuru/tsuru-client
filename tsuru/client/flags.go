// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"github.com/spf13/pflag"
)

func mergeFlagSet(fs1, fs2 *pflag.FlagSet) *pflag.FlagSet {
	fs1.AddFlagSet(fs2)
	return fs1
}
