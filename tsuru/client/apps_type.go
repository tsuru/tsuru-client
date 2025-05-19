// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"fmt"

	provTypes "github.com/tsuru/tsuru/types/provision"
)

func getParamsScaleDownLines(behavior provTypes.BehaviorAutoScaleSpec) []string {
	lines := []string{}

	if behavior.ScaleDown.UnitsPolicyValue != nil {
		lines = append(lines, fmt.Sprintf("Units: %d", *behavior.ScaleDown.UnitsPolicyValue))
	}
	if behavior.ScaleDown.PercentagePolicyValue != nil {
		lines = append(lines, fmt.Sprintf("Percentage: %d%%", *behavior.ScaleDown.PercentagePolicyValue))
	}
	if behavior.ScaleDown.StabilizationWindow != nil {
		lines = append(lines, fmt.Sprintf("Stabilization window: %ds", *behavior.ScaleDown.StabilizationWindow))
	}
	return lines
}
