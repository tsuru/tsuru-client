// Copyright 2026 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package formatter

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru/streamfmt"
)

// Lesson learned from: https://github.com/kubernetes/kubernetes/issues/101695
var terminalEscaper = strings.NewReplacer("\x1b", "^[", "\r", "\\r")

// Unicode characters used for visual formatting
const (
	sectionIndicator = "──" // U+2500 BOX DRAWINGS LIGHT HORIZONTAL
	actionArrow      = "→"  // U+2192 RIGHTWARDS ARROW
	errorCross       = "❌"  // U+274C CROSS MARK
)

// coloredEncoderWriter wraps an io.Writer to produce colorized, timestamped output.
// It buffers incomplete lines across Write calls to handle chunked input correctly.
type coloredEncoderWriter struct {
	Started time.Time
	Encoder io.Writer

	pending []byte // buffer for incomplete line fragments
}

var trimmedActionPrefix = strings.TrimSpace(streamfmt.ActionPrefix)

// Write implements io.Writer. It processes each line of input and writes
// colorized output with timestamps. Lines are categorized as:
//   - Section headers: displayed with blue indicator
//   - Actions: displayed with green arrow
//   - Errors: displayed with red cross
//   - Regular lines: displayed as-is
//
// Write buffers incomplete lines across calls to handle chunked input correctly.
// Only complete lines (terminated by '\n', '\r', or '\r\n') are formatted and written.
func (w *coloredEncoderWriter) Write(p []byte) (int, error) {

	// Prepend any pending data from previous Write call
	data := p
	if len(w.pending) > 0 {
		data = append(w.pending, p...)
		w.pending = nil
	}

	elapsedSeconds := time.Since(w.Started).Seconds()

	for {
		idx := bytes.IndexAny(data, "\r\n")
		if idx == -1 {
			// No newline found; save remainder for next Write call
			if len(data) > 0 {
				w.pending = make([]byte, len(data))
				copy(w.pending, data)
			}
			break
		}

		line := string(data[:idx])
		line = terminalEscaper.Replace(line)
		// Skip past the delimiter, handling \r\n as a single line ending
		if idx+1 < len(data) && data[idx] == '\r' && data[idx+1] == '\n' {
			data = data[idx+2:]
		} else {
			data = data[idx+1:]
		}

		if len(line) == 0 {
			continue
		}

		w.writeTimestamp(elapsedSeconds)
		w.writeFormattedLine(line)
	}

	return len(p), nil
}

// writeTimestamp writes the elapsed time prefix in gray.
func (w *coloredEncoderWriter) writeTimestamp(seconds float64) {
	timestamp := fmt.Sprintf("[%4.0fs] ", seconds)
	fmt.Fprint(w.Encoder, cmd.Colorfy(timestamp, "gray", "", ""))
}

// writeFormattedLine determines the line type and writes it with appropriate formatting.
func (w *coloredEncoderWriter) writeFormattedLine(line string) {
	switch {
	case strings.HasPrefix(line, streamfmt.SectionPrefix):
		w.writeSectionLine(line)

	case strings.HasPrefix(strings.TrimLeft(line, " "), trimmedActionPrefix):
		w.writeActionLine(line)

	case strings.HasPrefix(line, streamfmt.ErrorPrefix):
		w.writeErrorLine(line)

	default:
		fmt.Fprintf(w.Encoder, "%s\n", line)
	}
}

// writeSectionLine writes a section header with blue indicator.
func (w *coloredEncoderWriter) writeSectionLine(line string) {
	// Defense-in-depth: handle malformed lines missing expected suffix
	if !strings.HasSuffix(line, streamfmt.SectionSuffix) {
		fmt.Fprintf(w.Encoder, "%s\n", line)
		return
	}

	content := line[len(streamfmt.SectionPrefix) : len(line)-len(streamfmt.SectionSuffix)]

	fmt.Fprint(w.Encoder, cmd.Colorfy(sectionIndicator, "blue", "", ""))
	fmt.Fprintf(w.Encoder, " %s \n", cmd.Colorfy(content, "", "", "bold"))
}

// writeActionLine writes an action line with green arrow, preserving indentation.
func (w *coloredEncoderWriter) writeActionLine(line string) {
	trimmedLine := strings.TrimLeft(line, " ")
	leadingSpaces := len(line) - len(trimmedLine)
	content := trimmedLine[len(trimmedActionPrefix):]

	fmt.Fprint(w.Encoder, strings.Repeat(" ", leadingSpaces))
	fmt.Fprint(w.Encoder, cmd.Colorfy(actionArrow, "green", "", "bold"))
	fmt.Fprintf(w.Encoder, " %s\n", cmd.Colorfy(content, "", "", "reset"))
}

// writeErrorLine writes an error line with red cross indicator.
func (w *coloredEncoderWriter) writeErrorLine(line string) {
	// Defense-in-depth: handle malformed lines missing expected suffix
	if !strings.HasSuffix(line, streamfmt.ErrorSuffix) {
		fmt.Fprintf(w.Encoder, "%s\n", line)
		return
	}

	content := line[len(streamfmt.ErrorPrefix) : len(line)-len(streamfmt.ErrorSuffix)]

	fmt.Fprint(w.Encoder, cmd.Colorfy(errorCross+errorCross, "red", "", "bold"))
	fmt.Fprintf(w.Encoder, " %s\n", cmd.Colorfy(content, "red", "", "reset"))
}

func NewColoredStreamWriter(encoder io.Writer) io.Writer {
	return &coloredEncoderWriter{Encoder: encoder, Started: time.Now()}
}
