// Copyright 2026 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package formatter

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tsuru/tsuru/streamfmt"
)

func TestColoredEncoderWriter_Write_RegularLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	_, err := w.Write([]byte("regular line\n"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "regular line")
	assert.Contains(t, output, "[")  // timestamp
	assert.Contains(t, output, "s]") // timestamp
}

func TestColoredEncoderWriter_Write_SectionLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	sectionLine := streamfmt.Section("Build phase")
	_, err := w.Write([]byte(sectionLine + "\n"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Build phase")
	assert.Contains(t, output, sectionIndicator)
}

func TestColoredEncoderWriter_Write_ActionLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	actionLine := streamfmt.Action("Running tests")
	_, err := w.Write([]byte(actionLine + "\n"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Running tests")
	assert.Contains(t, output, actionArrow)
}

func TestColoredEncoderWriter_Write_ActionLineWithIndentation(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	actionLine := "    " + streamfmt.Action("Nested action")
	_, err := w.Write([]byte(actionLine + "\n"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Nested action")
	assert.Contains(t, output, actionArrow)
}

func TestColoredEncoderWriter_Write_ErrorLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	errorLine := streamfmt.Error("Something went wrong")
	_, err := w.Write([]byte(errorLine + "\n"))
	assert.NoError(t, err)

	output := buf.String()
	// streamfmt.Error converts to uppercase
	assert.Contains(t, output, "SOMETHING WENT WRONG")
	assert.Contains(t, output, errorCross)
}

func TestColoredEncoderWriter_Write_MultipleLines(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	lines := "line1\nline2\nline3\n"
	n, err := w.Write([]byte(lines))
	assert.NoError(t, err)
	assert.Equal(t, len(lines), n)

	output := buf.String()
	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "line2")
	assert.Contains(t, output, "line3")
}

func TestColoredEncoderWriter_Write_EmptyLinesAreSkipped(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	lines := "line1\n\n\nline2\n"
	_, err := w.Write([]byte(lines))
	assert.NoError(t, err)

	output := buf.String()
	// Should only have 2 timestamp prefixes (one per non-empty line)
	timestampCount := strings.Count(output, "s] ")
	assert.Equal(t, 2, timestampCount)
}

func TestColoredEncoderWriter_Write_TimestampFormat(t *testing.T) {
	var buf bytes.Buffer
	started := time.Now().Add(-10 * time.Second)
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: started,
	}

	_, err := w.Write([]byte("test\n"))
	assert.NoError(t, err)

	output := buf.String()
	// Timestamp should be roughly 10 seconds
	assert.True(t, strings.Contains(output, "10s]") || strings.Contains(output, "11s]"))
}

func TestNewColoredStreamWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewColoredStreamWriter(&buf)
	assert.NotNil(t, w)

	// Check it's the right type by writing to it
	n, err := w.Write([]byte("test\n"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Greater(t, buf.Len(), 0)
}

func TestColoredEncoderWriter_Write_MixedContent(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	mixedContent := streamfmt.Section("Build") + "\n" +
		streamfmt.Action("Installing deps") + "\n" +
		"npm install completed\n" +
		streamfmt.Error("Build failed") + "\n"

	_, err := w.Write([]byte(mixedContent))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, sectionIndicator)
	assert.Contains(t, output, actionArrow)
	assert.Contains(t, output, errorCross)
	assert.Contains(t, output, "npm install completed")
}

func TestColoredEncoderWriter_WriteTimestamp(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	w.writeTimestamp(42.5)
	output := buf.String()
	assert.Contains(t, output, "42s]")
}

func TestColoredEncoderWriter_WriteSectionLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	sectionLine := streamfmt.Section("Deploying")
	w.writeSectionLine(sectionLine)

	output := buf.String()
	assert.Contains(t, output, sectionIndicator)
	assert.Contains(t, output, "Deploying")
	// Should not contain the raw prefix/suffix
	assert.NotContains(t, output, streamfmt.SectionPrefix)
	assert.NotContains(t, output, streamfmt.SectionSuffix)
}

func TestColoredEncoderWriter_WriteActionLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	actionLine := streamfmt.Action("Compiling code")
	w.writeActionLine(actionLine)

	output := buf.String()
	assert.Contains(t, output, actionArrow)
	assert.Contains(t, output, "Compiling code")
	// Should not contain the raw prefix
	assert.NotContains(t, output, streamfmt.ActionPrefix)
}

func TestColoredEncoderWriter_WriteErrorLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	errorLine := streamfmt.Error("Fatal error")
	w.writeErrorLine(errorLine)

	output := buf.String()
	assert.Contains(t, output, errorCross)
	// streamfmt.Error converts to uppercase
	assert.Contains(t, output, "FATAL ERROR")
	// Should not contain the raw prefix/suffix
	assert.NotContains(t, output, streamfmt.ErrorPrefix)
	assert.NotContains(t, output, streamfmt.ErrorSuffix)
}

func TestColoredEncoderWriter_WriteActionLinePreservesIndentation(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Action line with leading spaces
	actionLine := "   " + streamfmt.Action("Indented action")
	w.writeActionLine(actionLine)

	output := buf.String()
	assert.True(t, strings.HasPrefix(output, "   "))
}

func TestColoredEncoderWriter_WriteSectionLine_MissingSuffix(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Simulate chunked write: prefix present but suffix missing
	incompleteLine := streamfmt.SectionPrefix + "Partial section"
	w.writeSectionLine(incompleteLine)

	output := buf.String()
	// Should fall back to raw output without panic
	assert.Contains(t, output, incompleteLine)
	assert.NotContains(t, output, sectionIndicator)
}

func TestColoredEncoderWriter_WriteErrorLine_MissingSuffix(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Simulate chunked write: prefix present but suffix missing
	incompleteLine := streamfmt.ErrorPrefix + "Partial error"
	w.writeErrorLine(incompleteLine)

	output := buf.String()
	// Should fall back to raw output without panic
	assert.Contains(t, output, incompleteLine)
	assert.NotContains(t, output, errorCross)
}

func TestColoredEncoderWriter_Write_ChunkedLines(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Simulate chunked writes splitting a line across multiple Write calls
	_, err := w.Write([]byte("hello wo"))
	assert.NoError(t, err)
	assert.Empty(t, buf.String()) // No output yet, line incomplete

	_, err = w.Write([]byte("rld\n"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "hello world")
}

func TestColoredEncoderWriter_Write_ChunkedSectionLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Split a section line across multiple Write calls
	sectionLine := streamfmt.Section("Build phase")
	midpoint := len(sectionLine) / 2

	_, err := w.Write([]byte(sectionLine[:midpoint]))
	assert.NoError(t, err)
	assert.Empty(t, buf.String()) // No output yet

	_, err = w.Write([]byte(sectionLine[midpoint:] + "\n"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Build phase")
	assert.Contains(t, output, sectionIndicator)
}

func TestColoredEncoderWriter_Write_ChunkedErrorLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Split an error line across multiple Write calls
	errorLine := streamfmt.Error("Something failed")
	midpoint := len(errorLine) / 2

	_, err := w.Write([]byte(errorLine[:midpoint]))
	assert.NoError(t, err)
	assert.Empty(t, buf.String()) // No output yet

	_, err = w.Write([]byte(errorLine[midpoint:] + "\n"))
	assert.NoError(t, err)

	output := buf.String()
	// streamfmt.Error converts to uppercase
	assert.Contains(t, output, "SOMETHING FAILED")
	assert.Contains(t, output, errorCross)
}

func TestColoredEncoderWriter_Write_MultipleChunkedLines(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Write multiple complete lines plus a partial line
	_, err := w.Write([]byte("line1\nline2\npartial"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "line2")
	assert.NotContains(t, output, "partial") // Not yet written

	// Complete the partial line
	buf.Reset()
	_, err = w.Write([]byte(" line3\n"))
	assert.NoError(t, err)

	output = buf.String()
	assert.Contains(t, output, "partial line3")
}

func TestColoredEncoderWriter_Write_PendingBufferIsCopied(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Write partial data
	data := []byte("hello")
	_, err := w.Write(data)
	assert.NoError(t, err)

	// Modify original slice - should not affect pending buffer
	data[0] = 'X'

	// Complete the line
	_, err = w.Write([]byte(" world\n"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "hello world")
	assert.NotContains(t, output, "Xello")
}

func TestColoredEncoderWriter_Write_EscapesTerminalControlSequences(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Write line containing ANSI escape sequence (e.g., color codes)
	_, err := w.Write([]byte("text with \x1b[31mred\x1b[0m color\n"))
	assert.NoError(t, err)

	output := buf.String()
	// Escape sequences from input should be replaced with ^[
	assert.Contains(t, output, "^[[31m")
	assert.Contains(t, output, "^[[0m")
	assert.Contains(t, output, "text with ^[[31mred^[[0m color")
}

func TestColoredEncoderWriter_Write_EscapesCarriageReturn(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Write line containing embedded carriage return (not as line ending)
	_, err := w.Write([]byte("progress: 50%\roverwritten\n"))
	assert.NoError(t, err)

	output := buf.String()
	// First part ends at \r, second part is separate line
	assert.Contains(t, output, "progress: 50%")
	assert.Contains(t, output, "overwritten")
}

func TestColoredEncoderWriter_Write_HandlesCRLFLineEnding(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Write lines with Windows-style CRLF endings
	_, err := w.Write([]byte("line1\r\nline2\r\n"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "line2")
	// Should have exactly 2 timestamps (one per line)
	timestampCount := strings.Count(output, "s] ")
	assert.Equal(t, 2, timestampCount)
}

func TestColoredEncoderWriter_Write_HandlesMixedLineEndings(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	// Mix of LF, CR, and CRLF endings
	_, err := w.Write([]byte("unix\nwindows\r\nold-mac\rend\n"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "unix")
	assert.Contains(t, output, "windows")
	assert.Contains(t, output, "old-mac")
	assert.Contains(t, output, "end")
}
