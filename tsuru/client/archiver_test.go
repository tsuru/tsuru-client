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
