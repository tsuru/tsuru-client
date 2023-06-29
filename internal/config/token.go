// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// GetTokenFromFs returns the token for the current target.
func GetTokenFromFs(fsys afero.Fs) (string, error) {
	tokenPaths := []string{filepath.Join(ConfigPath, "token")}
	if targetLabel, err := getTargetLabel(fsys); err == nil {
		tokenPaths = append([]string{filepath.Join(ConfigPath, "token.d", targetLabel)}, tokenPaths...)
	}

	var err error
	for _, tokenPath := range tokenPaths {
		var tkFile afero.File
		if tkFile, err = fsys.Open(tokenPath); err == nil {
			defer tkFile.Close()
			token, err1 := io.ReadAll(tkFile)
			if err1 != nil {
				return "", err1
			}
			tokenStr := strings.TrimSpace(string(token))
			return tokenStr, nil
		}
	}
	if os.IsNotExist(err) {
		return "", nil
	}
	return "", err
}

// SaveToken saves the token on the filesystem for future use.
func SaveToken(fsys afero.Fs, token string) error {
	tokenPaths := []string{filepath.Join(ConfigPath, "token")}
	targetLabel, err := getTargetLabel(fsys)
	if err == nil {
		err := fsys.MkdirAll(filepath.Join(ConfigPath, "token.d"), 0700)
		if err != nil {
			return err
		}
		tokenPaths = append(tokenPaths, filepath.Join(ConfigPath, "token.d", targetLabel))
	}
	for _, tokenPath := range tokenPaths {
		file, err := fsys.Create(tokenPath)
		if err != nil {
			return err
		}
		defer file.Close()
		n, err := file.WriteString(token)
		if err != nil {
			return err
		}
		if n != len(token) {
			return fmt.Errorf("failed to write token file")
		}
	}
	return nil
}

// RemoveCurrentTokensFromFs removes the token for the current target and alias.
func RemoveCurrentTokensFromFs(fsys afero.Fs) error {
	tokenPaths := []string{filepath.Join(ConfigPath, "token")}
	if targetLabel, err := getTargetLabel(fsys); err == nil {
		tokenPaths = append([]string{filepath.Join(ConfigPath, "token.d", targetLabel)}, tokenPaths...)
	}

	errs := []error{}
	for _, tokenPath := range tokenPaths {
		if err := fsys.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
