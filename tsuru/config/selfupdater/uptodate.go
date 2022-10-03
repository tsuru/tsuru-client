// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package selfupdater

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/tsuru/tsuru-client/tsuru/config"
)

const (
	defaultSnoozeByDuration  time.Duration = 0 * time.Hour
	DefaultLatestManifestURL string        = "https://github.com/tsuru/tsuru-client/releases/latest/download/metadata.json"
)

var (
	stderr                  io.ReadWriter    = os.Stderr
	nowUTC                  func() time.Time = func() time.Time { return time.Now().UTC() } // so we can test time-dependent features
	snoozeDuration          time.Duration
	forceCheckAfterDuration time.Duration
	overrideForceCheck      *bool = nil
	zeroTime                time.Time
)

func init() {
	if snoozeDurationStr := os.Getenv("TSURU_CLIENT_SELF_UPDATE_SNOOZE_DURATION"); snoozeDurationStr != "" {
		if duration, err := time.ParseDuration(snoozeDurationStr); err == nil {
			snoozeDuration = duration
		} else {
			fmt.Fprintln(stderr, "WARN: when setting TSURU_CLIENT_SELF_UPDATE_SNOOZE_DURATION, it must be a parsable duration (eg: 10m, 72h, etc...)")
		}
	}

	if forceCheckStr := os.Getenv("TSURU_CLIENT_FORCE_CHECK_UPDATES"); forceCheckStr != "" {
		if isForceCheck, err := strconv.ParseBool(forceCheckStr); err == nil {
			overrideForceCheck = &isForceCheck
		} else {
			fmt.Fprintln(stderr, "WARN: when setting TSURU_CLIENT_FORCE_CHECK_UPDATES, it must be either true or false")
		}
	}
}

type latestVersionCheckResult struct {
	isFinished    bool
	isOutdated    bool
	latestVersion string
	err           error
}

type latestVersionCheck struct {
	currentVersion         string
	forceCheckBeforeFinish bool
	result                 chan latestVersionCheckResult
}

type releaseMetadata struct {
	Version string    `json:"version"`
	Date    time.Time `json:"date"`
}

// This function "returns" its results over the r.result channel
func getRemoteVersionAndReportsToChanGoroutine(r *latestVersionCheck) {
	conf := config.GetConfig()
	checkResult := latestVersionCheckResult{
		isFinished:    true,
		latestVersion: r.currentVersion,
	}

	if r.currentVersion == "dev" || conf.ClientSelfUpdater.LastCheck.Add(snoozeDuration).After(nowUTC()) {
		r.result <- checkResult
		return
	}

	response, err := http.Get(conf.ClientSelfUpdater.LatestManifestURL)
	if err != nil {
		checkResult.err = fmt.Errorf("Could not GET endpoint %q: %w", conf.ClientSelfUpdater.LatestManifestURL, err)
		r.result <- checkResult
		return
	}
	defer response.Body.Close()
	if response.StatusCode > 300 {
		checkResult.err = fmt.Errorf("Could not GET endpoint %q: %v", conf.ClientSelfUpdater.LatestManifestURL, response.Status)
		r.result <- checkResult
		return
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		checkResult.err = fmt.Errorf("Could not read response body: %w", err)
		r.result <- checkResult
		return
	}

	var metadata releaseMetadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		checkResult.err = fmt.Errorf("Could not parse metadata.json. Unexpected format: %w", err)
		r.result <- checkResult
		return
	}

	current, err := semver.NewVersion(r.currentVersion)
	if err != nil {
		current, _ = semver.NewVersion("0.0.0")
	}
	latest, err := semver.NewVersion(metadata.Version)
	if err != nil {
		checkResult.err = fmt.Errorf("metadata.version is not a SemVersion: %w\nmetadata: %v (parsed from %q)", err, metadata, string(data))
		r.result <- checkResult
		return
	}

	conf.ClientSelfUpdater.LastCheck = nowUTC()
	if current.Compare(latest) < 0 {
		checkResult.latestVersion = latest.String()
		checkResult.isOutdated = true
	}
	r.result <- checkResult
}

func CheckLatestVersionBackground(currentVersion string) *latestVersionCheck {
	conf := config.GetConfig()

	forceCheckBeforeFinish := false
	if conf.ClientSelfUpdater.LastCheck != zeroTime || overrideForceCheck != nil { // do not force on empty config.ClientSelfUpdater
		forceCheckBeforeFinish = conf.ClientSelfUpdater.LastCheck.Add(forceCheckAfterDuration).Before(nowUTC())
		if overrideForceCheck != nil {
			forceCheckBeforeFinish = *overrideForceCheck
		}
	}

	r := &latestVersionCheck{
		currentVersion:         currentVersion,
		forceCheckBeforeFinish: forceCheckBeforeFinish,
	}
	r.result = make(chan latestVersionCheckResult, 1)
	go getRemoteVersionAndReportsToChanGoroutine(r)
	return r
}

func VerifyLatestVersion(lvCheck *latestVersionCheck) {
	checkResult := latestVersionCheckResult{}
	if lvCheck.forceCheckBeforeFinish {
		// blocking
		timeout := 2 * time.Second
		for !checkResult.isFinished {
			select {
			case <-time.After(timeout):
				fmt.Fprintln(stderr, "WARN: Taking too long to check for latest version. CTRL+C to force exit.")
			case checkResult = <-lvCheck.result:
				break
			}
			timeout += 2 * time.Second
		}

	} else {
		// non-blocking
		select {
		case checkResult = <-lvCheck.result:
		default:
		}
	}

	if checkResult.err != nil {
		fmt.Fprintf(stderr, "\n\nERROR: Could not query for latest version: %v\n", checkResult.err)
	}
	if checkResult.isFinished && checkResult.isOutdated {
		fmt.Fprintf(stderr, "\n\nINFO: A new version is available. Please update to the newer version %q (current: %q)\n", checkResult.latestVersion, lvCheck.currentVersion)
		if err := CheckPackageCloudRepo(); err != nil {
			fmt.Fprintf(stderr, "Got error after detecting an outdated package manager configuration: %v\n", err)
		}
	}
}
