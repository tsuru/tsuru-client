// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/fs"
)

func WriteTokenV1(token string) error {
	tokenPaths := []string{
		JoinWithUserDir(".tsuru", "token"),
	}
	targetLabel, err := GetTargetLabel()
	if err == nil {
		err := Filesystem().MkdirAll(JoinWithUserDir(".tsuru", "token.d"), 0700)
		if err != nil {
			return err
		}
		tokenPaths = append(tokenPaths, JoinWithUserDir(".tsuru", "token.d", targetLabel))
	}
	for _, tokenPath := range tokenPaths {
		file, err := Filesystem().Create(tokenPath)
		if err != nil {
			return err
		}
		defer file.Close()
		n, err := file.WriteString(token)
		if err != nil {
			return err
		}
		if n != len(token) {
			return errors.New("Failed to write token file.")
		}
	}
	return nil
}

func ReadTokenV1() (string, error) {
	var token []byte
	if token := ReadTeamToken(); token != "" {
		return token, nil
	}
	tokenPaths := []string{
		JoinWithUserDir(".tsuru", "token"),
	}
	targetLabel, err := GetTargetLabel()
	if err == nil {
		tokenPaths = append([]string{JoinWithUserDir(".tsuru", "token.d", targetLabel)}, tokenPaths...)
	}
	for _, tokenPath := range tokenPaths {
		var tkFile fs.File
		tkFile, err = Filesystem().Open(tokenPath)
		if err == nil {
			defer tkFile.Close()
			token, err = io.ReadAll(tkFile)
			if err != nil {
				return "", err
			}
			return string(token), nil
		}
	}
	if os.IsNotExist(err) {
		return "", nil
	}
	return "", err
}

func RemoveTokenV1() error {
	tokenPaths := []string{
		JoinWithUserDir(".tsuru", "token"),
	}
	targetLabel, err := GetTargetLabel()
	if err == nil {
		tokenPaths = append([]string{JoinWithUserDir(".tsuru", "token.d", targetLabel)}, tokenPaths...)
	}
	for _, tokenPath := range tokenPaths {
		err := Filesystem().Remove(tokenPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func ReadTeamToken() string {
	return os.Getenv("TSURU_TOKEN")
}

func MergeFlagSet(fs1, fs2 *gnuflag.FlagSet) *gnuflag.FlagSet {
	fs2.VisitAll(func(flag *gnuflag.Flag) {
		fs1.Var(flag.Value, flag.Name, flag.Usage)
	})
	return fs1
}
