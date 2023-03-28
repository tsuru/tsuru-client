// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux
// +build linux

package client

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

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
		{Name: "directory", Type: tar.TypeDir},
		{Name: "directory/file.txt", Type: tar.TypeReg, Data: []byte("wat\n")},
		{Name: "file1.txt", Type: tar.TypeReg, Data: []byte("something happened\n")},
		{Name: "file2.txt", Type: tar.TypeReg, Data: []byte("twice\n")},
		{Name: ".tsuruignore", Type: tar.TypeReg, Data: []byte("*.txt")},
		{Name: "directory/dir2", Type: tar.TypeDir},
		{Name: "directory/dir2/file.txt", Type: tar.TypeReg, Data: []byte("")},
	}
	c.Assert(got, check.DeepEquals, expected)

	c.Assert(stderr.String(), check.Matches, `(?s)(.*)Skipping file "directory" as it already exists in the current directory.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)Skipping file "directory/file.txt" as it already exists in the current directory.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)Skipping file "file1.txt" as it already exists in the current directory.(.*)`)
	c.Assert(stderr.String(), check.Matches, `(?s)(.*)Skipping file "file2.txt" as it already exists in the current directory.(.*)`)
}

func (s *S) TestArchive_SingleDirectory_NoFilesOnly(c *check.C) {
	var b bytes.Buffer
	err := Archive(&b, false, []string{"./testdata/deploy"}, ArchiveOptions{Stderr: io.Discard})
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

func (s *S) TestArchive_(c *check.C) {
	workingDir, err := os.Getwd()
	c.Assert(err, check.IsNil)

	defer func() { os.Chdir(workingDir) }()

	tests := []struct {
		files     []string
		absPath   bool
		filesOnly bool
		ignored   []string
		paths     []string
		expected  []string
	}{
		{
			files:    []string{"f1", "f2", "d1/f3", "d1/d2/f4"},
			paths:    []string{"."},
			expected: []string{"d1", "d1/d2", "d1/d2/f4", "d1/f3", "f1", "f2"},
		},

		{
			files:    []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
			paths:    []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
			expected: []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
		},

		{
			files:     []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
			filesOnly: true,
			paths:     []string{"testdata/deploy/file1.txt", "testdata/deploy2/file2.txt"},
			expected:  []string{"file1.txt", "file2.txt"},
		},

		{
			files:    []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2/file3.txt", "testdata/deploy2/directory/file4.txt"},
			paths:    []string{"testdata/deploy", "testdata/deploy2"},
			expected: []string{"testdata/deploy", "testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2", "testdata/deploy2/directory", "testdata/deploy2/directory/file4.txt", "testdata/deploy2/file3.txt"},
		},

		{
			files:     []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2/file3.txt", "testdata/deploy2/directory/file4.txt"},
			filesOnly: true,
			paths:     []string{"testdata/deploy", "testdata/deploy2"},
			expected:  []string{"file1.txt", "file2.txt", "directory", "directory/file4.txt", "file3.txt"},
		},

		{
			files:    []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy/directory/file.txt"},
			paths:    []string{"testdata/deploy", ".."},
			expected: []string{"testdata/deploy", "testdata/deploy/directory", "testdata/deploy/directory/file.txt", "testdata/deploy/file1.txt", "testdata/deploy/file2.txt"},
		},

		{
			files:    []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy/directory/file.txt"},
			paths:    []string{"testdata/deploy"},
			expected: []string{"directory", "directory/file.txt", "file1.txt", "file2.txt"},
		},

		{
			files:    []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:  []string{"*.txt"},
			paths:    []string{"testdata/deploy2"},
			expected: []string{"directory", "directory/dir2"},
		},

		{
			files:    []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2/file3.txt", "testdata/deploy2/directory/file4.txt"},
			ignored:  []string{"*.txt"},
			paths:    []string{"testdata/deploy", "testdata/deploy2"},
			expected: []string{"testdata/deploy", "testdata/deploy2", "testdata/deploy2/directory"},
		},

		{
			files:     []string{"testdata/deploy/file1.txt", "testdata/deploy/file2.txt", "testdata/deploy2/file3.txt", "testdata/deploy2/directory/file4.txt"},
			filesOnly: true,
			ignored:   []string{"*.txt"},
			paths:     []string{"testdata/deploy", "testdata/deploy2"},
			expected:  []string{"directory"},
		},

		{
			files:    []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:  []string{"*.txt"},
			paths:    []string{"testdata/deploy2"},
			expected: []string{"directory", "directory/dir2"},
			absPath:  true,
		},

		{
			files:    []string{"file1.txt", "file2.txt", "directory/file.txt", "directory/dir2/file.txt"},
			ignored:  []string{"*.txt"},
			paths:    []string{"."},
			expected: []string{".tsuruignore", "directory", "directory/dir2"},
		},

		{
			files:    []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:  []string{"directory"},
			paths:    []string{"testdata/deploy2"},
			expected: []string{"file1.txt", "file2.txt"},
		},

		{
			files:    []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:  []string{"*/dir2"},
			paths:    []string{"testdata/deploy2"},
			expected: []string{"directory", "directory/file.txt", "file1.txt", "file2.txt"},
		},

		{
			files:    []string{"testdata/deploy2/file1.txt", "testdata/deploy2/file2.txt", "testdata/deploy2/directory/file.txt", "testdata/deploy2/directory/dir2/file.txt"},
			ignored:  []string{"directory/dir2/*"},
			paths:    []string{"testdata/deploy2"},
			expected: []string{"directory", "directory/dir2", "directory/file.txt", "file1.txt", "file2.txt"},
		},
	}

	for _, tt := range tests {
		root := c.MkDir()

		err := os.Chdir(root)
		c.Assert(err, check.IsNil)

		for _, file := range tt.files {
			err = os.MkdirAll(filepath.Join(root, filepath.Dir(file)), 0700)
			c.Assert(err, check.IsNil)

			_, err = os.Create(filepath.Join(root, file))
			c.Assert(err, check.IsNil)
		}

		if len(tt.ignored) > 0 {
			var f *os.File
			f, err = os.Create(filepath.Join(root, ".tsuruignore"))
			c.Assert(err, check.IsNil)

			for _, l := range tt.ignored {
				fmt.Fprintln(f, l)
			}

			err = f.Close()
			c.Assert(err, check.IsNil)
		}

		if tt.absPath {
			for i := range tt.paths {
				tt.paths[i] = filepath.Join(root, tt.paths[i])
			}
		}

		var b bytes.Buffer
		err = Archive(&b, tt.filesOnly, tt.paths, DefaultArchiveOptions(io.Discard))
		c.Assert(err, check.IsNil)

		files := extractFiles(s.t, c, &b)

		var got []string
		for _, f := range files {
			got = append(got, f.Name)
		}

		c.Assert(got, check.DeepEquals, tt.expected)
	}
}
