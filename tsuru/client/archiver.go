// Copyright 2023 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

var ErrMissingFilesToArchive = errors.New("missing files to archive")

type ArchiveOptions struct {
	CompressionLevel *int      // defaults to default compression "-1"
	IgnoreFiles      []string  // default to none
	Stderr           io.Writer // defaults to io.Discard
}

func DefaultArchiveOptions(w io.Writer) ArchiveOptions {
	return ArchiveOptions{
		CompressionLevel: func(lvl int) *int { return &lvl }(gzip.BestCompression),
		IgnoreFiles:      []string{".tsuruignore"},
		Stderr:           w,
	}
}

func Archive(dst io.Writer, filesOnly bool, paths []string, opts ArchiveOptions) error {
	if dst == nil {
		return fmt.Errorf("destination cannot be nil")
	}

	if len(paths) == 0 {
		return fmt.Errorf("paths cannot be empty")
	}

	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}

	var ignoreLines []string
	for _, ignoreFile := range opts.IgnoreFiles {
		data, err := os.ReadFile(ignoreFile)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}

		if err != nil {
			return fmt.Errorf("failed to read ignore file %q: %w", ignoreFile, err)
		}

		fmt.Fprintf(opts.Stderr, "Using pattern(s) from %q to include/exclude files...\n", ignoreFile)

		ignoreLines = append(ignoreLines, strings.Split(string(data), "\n")...)
	}

	ignore, err := gitignore.CompileIgnoreLines(ignoreLines...)
	if err != nil {
		return fmt.Errorf("failed to compile all ignore patterns: %w", err)
	}

	if opts.CompressionLevel == nil {
		opts.CompressionLevel = func(n int) *int { return &n }(gzip.DefaultCompression)
	}

	zw, err := gzip.NewWriterLevel(dst, *opts.CompressionLevel)
	if err != nil {
		return err
	}
	defer zw.Close()

	tw := tar.NewWriter(zw)
	defer tw.Close()

	a := &archiver{
		ignore: *ignore,
		stderr: opts.Stderr,
		files:  map[string]struct{}{},
	}

	return a.archive(tw, filesOnly, paths)
}

type archiver struct {
	ignore gitignore.GitIgnore
	stderr io.Writer
	files  map[string]struct{}
}

func (a *archiver) archive(tw *tar.Writer, filesOnly bool, paths []string) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get the current directory: %w", err)
	}

	var added int

	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get the absolute filename of %q: %w", path, err)
		}

		if !strings.HasPrefix((abs + string(os.PathSeparator)), (workingDir + string(os.PathSeparator))) {
			fmt.Fprintf(a.stderr, "WARNING: skipping file %q since you cannot add files outside the current directory\n", path)
			continue
		}

		fi, err := os.Lstat(path)
		if err != nil {
			return err
		}

		var n int

		var changeDir string
		if fi.IsDir() {
			// NOTE(nettoclaudio): when user passes a single directory we should
			// consider it as the root directory for backward-compability.
			if len(paths) == 1 {
				changeDir, path = abs, "."
			}

			n, err = a.addDir(tw, filesOnly, path, changeDir)
			if err != nil {
				return err
			}

			added += n
			continue
		}

		n, err = a.addFile(tw, filesOnly, path, fi)
		if err != nil {
			return err
		}

		added += n
	}

	if added == 0 {
		return ErrMissingFilesToArchive
	}

	return nil
}

func (a *archiver) addFile(tw *tar.Writer, filesOnly bool, filename string, fi os.FileInfo) (int, error) {
	isDir, isRegular, isSymlink := fi.IsDir(), fi.Mode().IsRegular(), fi.Mode()&os.ModeSymlink == os.ModeSymlink

	if !isDir && !isRegular && !isSymlink { // neither dir, regular nor symlink
		fmt.Fprintf(a.stderr, "WARNING: Skipping file %q due to unsupported file type.\n", filename)
		return 0, nil
	}

	if isDir && filesOnly { // there's no need to create dirs in files only
		return 0, nil
	}

	if a.ignore.MatchesPath(filename) {
		fmt.Fprintf(a.stderr, "File %q matches with some pattern provided in the ignore file... skipping it.\n", filename)
		return 0, nil
	}

	var linkname string
	if isSymlink {
		target, err := os.Readlink(filename)
		if err != nil {
			return 0, err
		}

		linkname = target
	}

	h, err := tar.FileInfoHeader(fi, linkname)
	if err != nil {
		return 0, err
	}

	if !filesOnly { // should preserve the directory tree
		h.Name = filename
	}

	if _, found := a.files[h.Name]; found {
		fmt.Fprintf(a.stderr, "Skipping file %q as it already exists in the current directory.\n", filename)
		return 0, nil
	}

	a.files[h.Name] = struct{}{}

	if strings.TrimRight(h.Name, string(os.PathSeparator)) == "." { // skipping root dir
		return 0, nil
	}

	if err = tw.WriteHeader(h); err != nil {
		return 0, err
	}

	if isDir || isSymlink { // there's no data to copy from dir or symlink
		return 1, nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	written, err := io.CopyN(tw, f, h.Size)
	if err != nil {
		return 0, err
	}

	if written < h.Size {
		return 0, io.ErrShortWrite
	}

	return 1, nil
}

func (a *archiver) addDir(tw *tar.Writer, filesOnly bool, path, changeDir string) (int, error) {
	if changeDir != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return 0, err
		}

		defer os.Chdir(cwd)

		if err = os.Chdir(changeDir); err != nil {
			return 0, err
		}
	}

	var added int
	return added, filepath.WalkDir(path, fs.WalkDirFunc(func(path string, dentry fs.DirEntry, err error) error {
		if err != nil { // fail fast
			return err
		}

		fi, err := dentry.Info()
		if err != nil {
			return err
		}

		var n int
		n, err = a.addFile(tw, filesOnly, path, fi)
		if err != nil {
			return err
		}

		added += n

		return nil
	}))
}
