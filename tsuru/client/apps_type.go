// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"encoding/json"
	"fmt"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
)

type behaviorScaleDownJson struct {
	ScaleDown scaleDownJson `json:"scaleDown"`
}

type scaleDownJson struct {
	UnitsPolicyValue      *int32 `json:"unitsPolicyValue,omitempty"`
	PercentagePolicyValue *int32 `json:"percentagePolicyValue,omitempty"`
	StabilizationWindow   *int32 `json:"stabilizationWindow,omitempty"`
}

func getParamsScaleDownLines(behavior tsuru.AutoScaleSpecBehavior) []string {
	b, err := json.Marshal(behavior)
	if err != nil {
		return nil
	}
	var behaviorJson behaviorScaleDownJson
	err = json.Unmarshal(b, &behaviorJson)
	if err != nil {
		return nil
	}

	lines := []string{}

	if behaviorJson.ScaleDown.UnitsPolicyValue != nil {
		lines = append(lines, fmt.Sprintf("Units: %d", *behaviorJson.ScaleDown.UnitsPolicyValue))
	}
	if behaviorJson.ScaleDown.PercentagePolicyValue != nil {
		lines = append(lines, fmt.Sprintf("Percentage: %d%%", *behaviorJson.ScaleDown.PercentagePolicyValue))
	}
	if behaviorJson.ScaleDown.StabilizationWindow != nil {
		lines = append(lines, fmt.Sprintf("Stabilization window: %ds", *behaviorJson.ScaleDown.StabilizationWindow))
	}
	return lines
}
