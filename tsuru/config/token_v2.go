// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"

	"golang.org/x/oauth2"
)

type TokenV2 struct {
	Scheme      string        `json:"scheme"`
	OAuth2Token *oauth2.Token `json:"oauth2_token"`
}

func WriteTokenV2(token TokenV2) error {
	tokenPaths := []string{
		JoinWithUserDir(".tsuru", "token-v2.json"),
	}
	targetLabel, err := GetTargetLabel()
	if err == nil {
		err := Filesystem().MkdirAll(JoinWithUserDir(".tsuru", "token-v2.d"), 0700)
		if err != nil {
			return err
		}
		tokenPaths = append(tokenPaths, JoinWithUserDir(".tsuru", "token-v2.d", targetLabel+".json"))
	}

	for _, tokenPath := range tokenPaths {
		file, err := Filesystem().Create(tokenPath)
		if err != nil {
			return err
		}
		defer file.Close()

		enc := json.NewEncoder(file)
		enc.SetIndent("  ", "  ")
		err = enc.Encode(&token)

		if err != nil {
			return err
		}

	}
	return nil
}
