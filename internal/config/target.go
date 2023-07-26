// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/afero"
)

var (
	errUndefinedTarget = fmt.Errorf(`no target defined. Please use target-add/target-set to define a target.

For more details, please run "tsuru help target"`)
)

// getSavedTargets returns a map of label->target
func getSavedTargets(fsys afero.Fs) (map[string]string, error) {
	var targets = map[string]string{} // label->target

	// legacyTargetsPath := JoinWithUserDir(".tsuru_targets") // XXX: remove legacy file
	targetsPath := filepath.Join(ConfigPath, "targets")
	f, err := fsys.Open(targetsPath)
	if os.IsNotExist(err) {
		return targets, nil
	}
	if err != nil {
		return nil, err
	}

	defer f.Close()
	if b, err := io.ReadAll(f); err == nil {
		var targetLines = strings.Split(strings.TrimSpace(string(b)), "\n")
		for i := range targetLines {
			var targetSplit = strings.Fields(targetLines[i])

			if len(targetSplit) == 2 {
				targets[targetSplit[0]] = targetSplit[1]
			}
		}
	}
	return targets, nil
}

// getTargetLabel finds the saved label of a target (os self if already a label).
// If target is unknown, the original target is returned with an error.
func getTargetLabel(fsys afero.Fs, target string) (string, error) {
	targets, err := getSavedTargets(fsys)
	if err != nil {
		return "", err
	}
	targetKeys := make([]string, len(targets))
	for k := range targets {
		if k == target {
			return k, nil
		}
		targetKeys = append(targetKeys, k)
	}
	sort.Strings(targetKeys)
	for _, k := range targetKeys {
		if targets[k] == target {
			return k, nil
		}
	}
	return target, fmt.Errorf("label for target %q not found ", target)

}

// GetCurrentTargetFromFs returns the current target (from filesystem .tsuru/target)
func GetCurrentTargetFromFs(fsys afero.Fs) (target string, err error) {
	targetPath := filepath.Join(ConfigPath, "target")
	if f, err := fsys.Open(targetPath); err == nil {
		defer f.Close()
		if b, err := io.ReadAll(f); err == nil {
			target = strings.TrimSpace(string(b))
		}
	}

	if target == "" {
		return "", errUndefinedTarget
	}

	return target, nil
}

// GetTargetURL returns the target URL from a given alias. If the alias is not
// found, it returns the alias itself as a  NormalizedTargetURL.
func GetTargetURL(fsys afero.Fs, alias string) (string, error) {
	targets, err := getSavedTargets(fsys)
	if err != nil {
		return "", err
	}

	targetURL := NormalizeTargetURL(alias)
	if val, ok := targets[alias]; ok {
		targetURL = val
	}

	return targetURL, nil
}

// IsCurrentTarget checks if the target in the same from ~/.tsuru/target
func IsCurrentTarget(fsys afero.Fs, target string) bool {
	target, _ = GetTargetURL(fsys, target)
	if file, err := fsys.Open(filepath.Join(ConfigPath, "target")); err == nil {
		defer file.Close()
		defaultTarget, _ := io.ReadAll(file)
		if target == string(defaultTarget) {
			return true
		}
	}
	return false
}

// SaveTarget saves the label->target in ~/.tsuru/targets list
func SaveTarget(fsys afero.Fs, label, target string) error {
	allTargets, err := getSavedTargets(fsys)
	if err != nil {
		return err
	}
	allTargets[label] = NormalizeTargetURL(target)

	// sorting by label
	labels := make([]string, 0, len(allTargets))
	for l := range allTargets {
		labels = append(labels, l)
	}
	sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })

	// writing all targets to temp file for atomicy
	file, err := fsys.Create(filepath.Join(ConfigPath, "targets.tmp"))
	if err != nil {
		return err
	}
	for _, l := range labels {
		_, err1 := fmt.Fprintf(file, "%s\t%s\n", l, allTargets[l])
		if err1 != nil {
			return fmt.Errorf("something went wrong when writing to targets.tmp: %w", err1)
		}
	}

	// replace targets file
	return fsys.Rename(filepath.Join(ConfigPath, "targets.tmp"), filepath.Join(ConfigPath, "targets"))
}

// SaveTargetAsCurrent saves the target in ~/.tsuru/target
func SaveTargetAsCurrent(fsys afero.Fs, target string) error {
	target = NormalizeTargetURL(target)
	file, err := fsys.Create(filepath.Join(ConfigPath, "target"))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "%s\n", target)
	if err != nil {
		return err
	}

	return nil
}

// NormalizeTargetURL adds an https:// if it has no protocol
func NormalizeTargetURL(target string) string {
	if m, _ := regexp.MatchString("^https?://", target); !m {
		target = "https://" + target
	}
	return target
}
