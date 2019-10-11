// Copyright 2018 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package formatter

import (
	"fmt"
	"time"
)

var LocalTZ = time.Local

func Local(date time.Time) time.Time {
	return date.In(LocalTZ)
}

func FormatStamp(date time.Time) string {
	return date.In(LocalTZ).Format(time.Stamp)
}

func FormatDate(date time.Time) string {
	if date.IsZero() {
		return "-"
	}
	return date.In(LocalTZ).Format(time.RFC822)
}

func FormatDuration(duration *time.Duration) string {
	if duration == nil {
		return "…"
	}

	seconds := *duration / time.Second
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func FormatDateAndDuration(date time.Time, duration *time.Duration) string {
	return fmt.Sprintf("%s (%s)", FormatDate(date), FormatDuration(duration))
}
