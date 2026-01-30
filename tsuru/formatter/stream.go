// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package formatter

import (
	"io"
	"net/http"

	"github.com/pkg/errors"
	v2 "github.com/tsuru/tsuru-client/tsuru/cmd/v2"
	tsuruIO "github.com/tsuru/tsuru/io"
)

// StreamJSONResponse supports the JSON streaming format from the tsuru API.
func StreamJSONResponse(w io.Writer, response *http.Response) error {
	if response == nil {
		return errors.New("response cannot be nil")
	}

	var writer io.Writer
	var formatter *tsuruIO.SimpleJsonMessageFormatter
	if v2.ColorStream() {
		writer = NewColoredStreamWriter(w)
		formatter = &tsuruIO.SimpleJsonMessageFormatter{NoTimestamp: true}
	} else {
		writer = w
		formatter = nil
	}

	defer response.Body.Close()
	output := tsuruIO.NewStreamWriter(writer, formatter)
	var err error
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(output, response.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := output.Remaining()
	if len(unparsed) > 0 {
		return errors.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}
