// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"strings"
)

func Open(ex Executor, url string) error {
	url = strings.Replace(url, "&", "^&", -1)
	opts := ExecuteOptions{
		Cmd:  "cmd",
		Args: []string{"/c", "start", "", url},
	}
	return ex.Command(opts)
}
