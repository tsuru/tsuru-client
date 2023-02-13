// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux
// +build linux

package client

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	check "gopkg.in/check.v1"
)

func (s *S) TestArchive_NoDestination(c *check.C) {
	err := Archive(nil, false, nil, ArchiveOptions{})
	c.Assert(err, check.ErrorMatches, "destination cannot be nil")
}

func (s *S) TestArchive_NoPaths(c *check.C) {
	err := Archive(io.Discard, false, nil, ArchiveOptions{})
	c.Assert(err, check.ErrorMatches, "paths cannot be empty")

	err = Archive(io.Discard, false, []string{}, ArchiveOptions{})
	c.Assert(err, check.ErrorMatches, "paths cannot be empty")
}

func (s *S) TestArchive_FileOutsideOfCurrentDir(c *check.C) {
	var stderr bytes.Buffer
	err := Archive(io.Discard, false, []string{"../../../../var/www/html"}, ArchiveOptions{Stderr: &stderr})
	c.Assert(err, check.ErrorMatches, "missing files to archive")
	c.Assert(stderr.String(), check.Matches, `(?s).*WARNING: skipping file "\.\.\/\.\.\/\.\.\/\.\.\/var\/www\/html" since you cannot add files outside the current directory.*`)
}

func (s *S) TestArchive_PassingWholeDir(c *check.C) {
	workingDir, err := os.Getwd()
	c.Assert(err, check.IsNil)

	defer func() { os.Chdir(workingDir) }()

	err = os.Chdir(filepath.Join(workingDir, "./testdata/deploy/"))
	c.Assert(err, check.IsNil)

	var b bytes.Buffer

	err = Archive(&b, false, []string{"."}, ArchiveOptions{})
	c.Assert(err, check.IsNil)

	got := extractFiles(s.t, c, &b)
	expected := []miniFile{
		{Name: "directory", Type: tar.TypeDir},
		{Name: "directory/file.txt", Type: tar.TypeReg, Data: []byte("wat\n")},
		{Name: "file1.txt", Type: tar.TypeReg, Data: []byte("something happened\n")},
		{Name: "file2.txt", Type: tar.TypeReg, Data: []byte("twice\n")},
	}
	c.Assert(got, check.DeepEquals, expected)
}

func (s *S) TestArchive_PassingWholeDir_WithTsuruIgnore(c *check.C) {
	workingDir, err := os.Getwd()
	c.Assert(err, check.IsNil)

	defer func() { os.Chdir(workingDir) }()

	err = os.Chdir(filepath.Join(workingDir, "./testdata/deploy2/"))
	c.Assert(err, check.IsNil)

	var b, stderr bytes.Buffer

	err = Archive(&b, false, []string{"."}, ArchiveOptions{IgnoreFiles: []string{".tsuruignore"}, Stderr: &stderr})
	c.Assert(err, check.IsNil)

	got := extractFiles(s.t, c, &b)
	expected := []miniFile{
		{Name: ".tsuruignore", Type: tar.TypeReg, Data: []byte("*.txt")},
		{Name: "directory", Type: tar.TypeDir},
		{Name: "directory/dir2", Type: tar.TypeDir},
	}
	c.Assert(got, check.DeepEquals, expected)

	c.Assert(stderr.String(), check.Matches, `(?s)(.*)Using pattern\(s\) from "\.tsuruignore" to include/exclude files\.\.\.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)File "directory/dir2/file\.txt" matches with some pattern provided in the ignore file\.\.\. skipping it\.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)File "directory/file\.txt" matches with some pattern provided in the ignore file\.\.\. skipping it\.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)File "file1\.txt" matches with some pattern provided in the ignore file\.\.\. skipping it\.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)File "file2\.txt" matches with some pattern provided in the ignore file\.\.\. skipping it\.(.*)`)
}

func (s *S) TestArchive_FilesOnly(c *check.C) {
	var b bytes.Buffer
	err := Archive(&b, true, []string{"./testdata/deploy/directory/file.txt", "./testdata/deploy2/file1.txt"}, ArchiveOptions{})
	c.Assert(err, check.IsNil)

	got := extractFiles(s.t, c, &b)
	expected := []miniFile{
		{Name: "file.txt", Type: tar.TypeReg, Data: []byte("wat\n")},
		{Name: "file1.txt", Type: tar.TypeReg, Data: []byte("something happened\n")},
	}
	c.Assert(got, check.DeepEquals, expected)
}

func (s *S) TestArchive_WithSymlink(c *check.C) {
	workingDir, err := os.Getwd()
	c.Assert(err, check.IsNil)

	defer func() { os.Chdir(workingDir) }()

	err = os.Chdir(filepath.Join(workingDir, "./testdata-symlink/"))
	c.Assert(err, check.IsNil)

	var b bytes.Buffer
	err = Archive(&b, false, []string{"."}, ArchiveOptions{})
	c.Assert(err, check.IsNil)

	got := extractFiles(s.t, c, &b)
	expected := []miniFile{
		{Name: "link", Linkname: "test", Type: tar.TypeSymlink},
		{Name: "test", Type: tar.TypeDir},
		{Name: "test/index.html", Type: tar.TypeReg, Data: []byte{}},
	}
	c.Assert(got, check.DeepEquals, expected)
}

func (s *S) TestArchive_UnsupportedFileType(c *check.C) {
	workingDir, err := os.Getwd()
	c.Assert(err, check.IsNil)

	defer func() { os.Chdir(workingDir) }()

	err = os.Chdir(c.MkDir())
	c.Assert(err, check.IsNil)

	l, err := net.Listen("unix", "./server.sock")
	c.Assert(err, check.IsNil)
	defer l.Close()

	var stderr bytes.Buffer
	err = Archive(io.Discard, false, []string{"."}, ArchiveOptions{Stderr: &stderr})
	c.Assert(err, check.ErrorMatches, "missing files to archive")
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)WARNING: Skipping file "server.sock" due to unsupported file type.(.*)`)
}

func (s *S) TestArchive_FilesOnly_MultipleDirs(c *check.C) {
	var b, stderr bytes.Buffer
	err := Archive(&b, true, []string{"./testdata/deploy", "./testdata/deploy2"}, ArchiveOptions{Stderr: &stderr})
	c.Assert(err, check.IsNil)

	got := extractFiles(s.t, c, &b)
	expected := []miniFile{
		{Name: "file.txt", Type: tar.TypeReg, Data: []byte("wat\n")},
		{Name: "file1.txt", Type: tar.TypeReg, Data: []byte("something happened\n")},
		{Name: "file2.txt", Type: tar.TypeReg, Data: []byte("twice\n")},
		{Name: ".tsuruignore", Type: tar.TypeReg, Data: []byte("*.txt")},
	}
	c.Assert(got, check.DeepEquals, expected)

	c.Assert(stderr.String(), check.Matches, `(?s)(.*)Skipping file "testdata/deploy2/directory/dir2/file.txt" as it already exists in the current directory.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)Skipping file "testdata/deploy2/directory/file.txt" as it already exists in the current directory.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)Skipping file "testdata/deploy2/file1.txt" as it already exists in the current directory.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)Skipping file "testdata/deploy2/file2.txt" as it already exists in the current directory.(.*)`)
}

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

		if h.Typeflag == tar.TypeReg || h.Typeflag == tar.TypeRegA {
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
