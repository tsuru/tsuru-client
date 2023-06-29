// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

func Open(ex Executor, url string) error {
	opts := ExecuteOptions{
		Cmd:  "open",
		Args: []string{url},
	}
	return ex.Command(opts)
}
