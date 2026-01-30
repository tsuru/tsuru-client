// Copyright © 2026 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

type UsageError struct {
	Err error
}

func (e *UsageError) Error() string {
	return e.Err.Error()
}

func (e *UsageError) Unwrap() error {
	return e.Err
}

func isUsageError(err error) bool {
	var usageErr *UsageError
	return errors.As(err, &usageErr)
}

func catchUsageError(args cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, a []string) error {
		err := args(cmd, a)
		if err != nil {
			return &UsageError{Err: err}
		}
		return nil
	}
}
