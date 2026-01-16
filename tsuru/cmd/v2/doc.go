// Copyright © 2026 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package v2

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func generateDocCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "generate-doc",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return doc.GenMarkdownTree(cmd.Root(), "./")
		},
	}
	return cmd
}
