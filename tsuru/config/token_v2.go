// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"os"

	"github.com/tsuru/tsuru/fs"
	"golang.org/x/oauth2"
)

type TokenV2 struct {
	Scheme       string         `json:"scheme"`
	OAuth2Token  *oauth2.Token  `json:"oauth2_token,omitempty"`
	OAuth2Config *oauth2.Config `json:"oauth2_config,omitempty"`
}

const (
	tokenV2Filename  = "token-v2.json"
	tokenV2Directory = "token-v2.d"
)

func WriteTokenV2(token TokenV2) error {
	tokenPaths := []string{
		JoinWithUserDir(".tsuru", tokenV2Filename),
	}
	targetLabel, err := GetTargetLabel()
	if err == nil {
		err := Filesystem().MkdirAll(JoinWithUserDir(".tsuru", tokenV2Directory), 0700)
		if err != nil {
			return err
		}
		tokenPaths = append(tokenPaths, JoinWithUserDir(".tsuru", tokenV2Directory, targetLabel+".json"))
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

func ReadTokenV2() (*TokenV2, error) {
	tokenPaths := []string{
		JoinWithUserDir(".tsuru", tokenV2Filename),
	}
	targetLabel, err := GetTargetLabel()
	if err == nil {
		tokenPaths = append([]string{JoinWithUserDir(".tsuru", tokenV2Directory, targetLabel+".json")}, tokenPaths...)
	}
	for _, tokenPath := range tokenPaths {
		var tkFile fs.File
		tkFile, err = Filesystem().Open(tokenPath)
		if err == nil {
			defer tkFile.Close()

			t := TokenV2{}
			err = json.NewDecoder(tkFile).Decode(&t)
			if err != nil {
				return nil, err
			}
			return &t, nil
		}
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}

func RemoveTokenV2() error {
	tokenPaths := []string{
		JoinWithUserDir(".tsuru", tokenV2Filename),
	}
	targetLabel, err := GetTargetLabel()
	if err == nil {
		err := Filesystem().MkdirAll(JoinWithUserDir(".tsuru", tokenV2Directory), 0700)
		if err != nil {
			return err
		}
		tokenPaths = append(tokenPaths, JoinWithUserDir(".tsuru", tokenV2Directory, targetLabel+".json"))
	}

	for _, tokenPath := range tokenPaths {
		err := Filesystem().Remove(tokenPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

	}
	return nil
}
