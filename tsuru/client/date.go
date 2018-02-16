// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"fmt"
	"time"
)

func formatDateAndDuration(date time.Time, duration *time.Duration) string {
	timestamp := date.Local().Format(time.Stamp)
	durationStr := "…"
	if duration != nil {
		seconds := *duration / time.Second
		minutes := seconds / 60
		seconds = seconds % 60
		durationStr = fmt.Sprintf("%02d:%02d", minutes, seconds)
	}
	return fmt.Sprintf("%s (%s)", timestamp, durationStr)
}
