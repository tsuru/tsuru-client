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

func TestColoredEncoderWriter_Write_FirstPrintAddsNewline(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder: &buf,
		Started: time.Now(),
	}

	n, err := w.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.True(t, strings.HasPrefix(buf.String(), "\n"))
}

func TestColoredEncoderWriter_Write_RegularLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder:      &buf,
		Started:      time.Now(),
		firstPrinted: true,
	}

	_, err := w.Write([]byte("regular line"))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "regular line")
	assert.Contains(t, output, "[")  // timestamp
	assert.Contains(t, output, "s]") // timestamp
}

func TestColoredEncoderWriter_Write_SectionLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder:      &buf,
		Started:      time.Now(),
		firstPrinted: true,
	}

	sectionLine := streamfmt.Section("Build phase")
	_, err := w.Write([]byte(sectionLine))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Build phase")
	assert.Contains(t, output, sectionIndicator)
}

func TestColoredEncoderWriter_Write_ActionLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder:      &buf,
		Started:      time.Now(),
		firstPrinted: true,
	}

	actionLine := streamfmt.Action("Running tests")
	_, err := w.Write([]byte(actionLine))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Running tests")
	assert.Contains(t, output, actionArrow)
}

func TestColoredEncoderWriter_Write_ActionLineWithIndentation(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder:      &buf,
		Started:      time.Now(),
		firstPrinted: true,
	}

	actionLine := "    " + streamfmt.Action("Nested action")
	_, err := w.Write([]byte(actionLine))
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Nested action")
	assert.Contains(t, output, actionArrow)
}

func TestColoredEncoderWriter_Write_ErrorLine(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder:      &buf,
		Started:      time.Now(),
		firstPrinted: true,
	}

	errorLine := streamfmt.Error("Something went wrong")
	_, err := w.Write([]byte(errorLine))
	assert.NoError(t, err)

	output := buf.String()
	// streamfmt.Error converts to uppercase
	assert.Contains(t, output, "SOMETHING WENT WRONG")
	assert.Contains(t, output, errorCross)
}

func TestColoredEncoderWriter_Write_MultipleLines(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder:      &buf,
		Started:      time.Now(),
		firstPrinted: true,
	}

	lines := "line1\nline2\nline3"
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
		Encoder:      &buf,
		Started:      time.Now(),
		firstPrinted: true,
	}

	lines := "line1\n\n\nline2"
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
		Encoder:      &buf,
		Started:      started,
		firstPrinted: true,
	}

	_, err := w.Write([]byte("test"))
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
	n, err := w.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Greater(t, buf.Len(), 0)
}

func TestColoredEncoderWriter_Write_MixedContent(t *testing.T) {
	var buf bytes.Buffer
	w := &coloredEncoderWriter{
		Encoder:      &buf,
		Started:      time.Now(),
		firstPrinted: true,
	}

	mixedContent := streamfmt.Section("Build") + "\n" +
		streamfmt.Action("Installing deps") + "\n" +
		"npm install completed\n" +
		streamfmt.Error("Build failed")

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
