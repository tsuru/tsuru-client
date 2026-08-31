// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"strings"
	"testing"

	check "gopkg.in/check.v1"
)

func extractFiles(t *testing.T, c *check.C, r io.Reader) (m []miniFile) {
	t.Helper()

	gzr, err := gzip.NewReader(r)
	c.Assert(err, check.IsNil)

	tr := tar.NewReader(gzr)

	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		c.Assert(err, check.IsNil)

		var data []byte

		if h.Typeflag == tar.TypeReg {
			var b bytes.Buffer
			written, err := io.CopyN(&b, tr, h.Size)
			c.Assert(err, check.IsNil)
			c.Assert(written, check.Equals, h.Size)
			data = b.Bytes()
		}

		m = append(m, miniFile{
			Name:     h.Name,
			Linkname: h.Linkname,
			Type:     h.Typeflag,
			Data:     data,
		})
	}

	return m
}

type miniFile struct {
	Name     string
	Linkname string
	Type     byte
	Data     []byte
}

// TestArchive_ForwardSlashesInTarNames verifies that tar entry names always use
// forward slashes ("/") regardless of the OS path separator.
// This is important because Windows uses backslashes but tar format requires forward slashes.
func (s *S) TestArchive_ForwardSlashesInTarNames(c *check.C) {
	var b bytes.Buffer
	err := Archive(&b, false, []string{"./testdata/deploy"}, ArchiveOptions{Stderr: io.Discard})
	c.Assert(err, check.IsNil)

	files := extractFiles(s.t, c, &b)
	c.Assert(len(files) > 0, check.Equals, true)

	// Verify all tar entry names use forward slashes only
	for _, f := range files {
		c.Assert(strings.Contains(f.Name, "\\"), check.Equals, false,
			check.Commentf("tar entry name %q contains backslash, should use forward slashes only", f.Name))
	}

	// Verify the expected paths are present with forward slashes
	names := make(map[string]bool)
	for _, f := range files {
		names[f.Name] = true
	}
	c.Assert(names["directory"], check.Equals, true)
	c.Assert(names["directory/file.txt"], check.Equals, true)
}

// TestArchive_MultipleDirectoriesForwardSlashes verifies forward slashes when archiving multiple directories
func (s *S) TestArchive_MultipleDirectoriesForwardSlashes(c *check.C) {
	var b bytes.Buffer
	err := Archive(&b, false, []string{"./testdata/deploy", "./testdata/deploy2"}, ArchiveOptions{Stderr: io.Discard})
	c.Assert(err, check.IsNil)

	files := extractFiles(s.t, c, &b)
	c.Assert(len(files) > 0, check.Equals, true)

	// Verify all tar entry names use forward slashes only
	for _, f := range files {
		c.Assert(strings.Contains(f.Name, "\\"), check.Equals, false,
			check.Commentf("tar entry name %q contains backslash, should use forward slashes only", f.Name))
	}

	// Collect names for debugging
	var names []string
	for _, f := range files {
		names = append(names, f.Name)
	}
	// Just check that we have files and all paths use forward slashes
	c.Assert(len(names) > 0, check.Equals, true, check.Commentf("names: %v", names))
}
