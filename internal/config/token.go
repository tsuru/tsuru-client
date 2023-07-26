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
	"regexp"
	"strings"

	"github.com/spf13/afero"
)

// GetTokenFromFs returns the token for the target.
func GetTokenFromFs(fsys afero.Fs, target string) (string, error) {
	tokenPaths := []string{}
	if targetLabel, err := getTargetLabel(fsys, target); err == nil {
		tokenPaths = append(tokenPaths, filepath.Join(ConfigPath, "token.d", targetLabel))
	}
	tokenPaths = append(tokenPaths, filepath.Join(ConfigPath, "token")) // always defaults to current token

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

// SaveTokenToFs saves the token on the filesystem for future use.
func SaveTokenToFs(fsys afero.Fs, target, token string) error {
	err := fsys.MkdirAll(filepath.Join(ConfigPath, "token.d"), 0700)
	if err != nil {
		return err
	}

	tokenPaths := []string{}
	if IsCurrentTarget(fsys, target) {
		tokenPaths = append(tokenPaths, filepath.Join(ConfigPath, "token"))
	} else if _, fErr := fsys.Stat(filepath.Join(ConfigPath, "token")); os.IsNotExist(fErr) {
		tokenPaths = append(tokenPaths, filepath.Join(ConfigPath, "token"))
		SaveTargetAsCurrent(fsys, target)
	}

	targetLabel, _ := getTargetLabel(fsys, target) // ignore err, and consider label=host
	tokenPaths = append(tokenPaths, filepath.Join(ConfigPath, "token.d", hostFromURL(targetLabel)))

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

// RemoveTokensFromFs removes the token for target.
func RemoveTokensFromFs(fsys afero.Fs, target string) error {
	tokenPaths := []string{}
	if IsCurrentTarget(fsys, target) {
		tokenPaths = append(tokenPaths, filepath.Join(ConfigPath, "token"))
	}
	if targetLabel, err := getTargetLabel(fsys, target); err == nil {
		tokenPaths = append(tokenPaths, filepath.Join(ConfigPath, "token.d", targetLabel))
	}

	errs := []error{}
	for _, tokenPath := range tokenPaths {
		if err := fsys.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func hostFromURL(url string) string {
	return regexp.MustCompile("^(https?://)?([0-9a-zA-Z_.-]+).*").ReplaceAllString(url, "$2")
}
