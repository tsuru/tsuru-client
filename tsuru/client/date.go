// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"fmt"
	"time"
)

func formatDate(date time.Time) string {
	return date.Local().Format(time.Stamp)
}

func formatDuration(duration *time.Duration) string {
	if duration == nil {
		return "…"
	}

	seconds := *duration / time.Second
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func formatDateAndDuration(date time.Time, duration *time.Duration) string {
	return fmt.Sprintf("%s (%s)", formatDate(date), formatDuration(duration))
}
