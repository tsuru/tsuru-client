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

type scaleDownOutput struct {
	UnitsPolicyValue      string `json:"unitsPolicyValue,omitempty"`
	PercentagePolicyValue string `json:"percentagePolicyValue,omitempty"`
	StabilizationWindow   string `json:"stabilizationWindow,omitempty"`
}

func getParamsScaleDownJson(behavior tsuru.AutoScaleSpecBehavior) scaleDownOutput {
	b, err := json.Marshal(behavior)
	if err != nil {
		return scaleDownOutput{}
	}
	var behaviorJson behaviorScaleDownJson
	err = json.Unmarshal(b, &behaviorJson)
	if err != nil {
		return scaleDownOutput{}
	}
	output := scaleDownOutput{}
	if behaviorJson.ScaleDown.UnitsPolicyValue != nil {
		output.UnitsPolicyValue = fmt.Sprintf("%d", *behaviorJson.ScaleDown.UnitsPolicyValue)
	}
	if behaviorJson.ScaleDown.PercentagePolicyValue != nil {
		output.PercentagePolicyValue = fmt.Sprintf("%d", *behaviorJson.ScaleDown.PercentagePolicyValue)
	}
	if behaviorJson.ScaleDown.StabilizationWindow != nil {
		output.StabilizationWindow = fmt.Sprintf("%d", *behaviorJson.ScaleDown.StabilizationWindow)
	}
	return output
}
