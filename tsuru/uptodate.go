package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/tsuru/tsuru-client/tsuru/config"
)

var (
	defaultSnoozeByDuration time.Duration = 24 * time.Hour
)

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

	if r.currentVersion == "dev" || nowUTC().Before(conf.ClientSelfUpdater.SnoozeUntil) {
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
	conf.ClientSelfUpdater.SnoozeUntil = nowUTC().Add(defaultSnoozeByDuration)
	conf.ClientSelfUpdater.ForceCheckAfter = nowUTC().Add(config.DefaultForceCheckAfterDuration)

	if current.Compare(latest) < 0 {
		checkResult.latestVersion = latest.String()
		checkResult.isOutdated = true
	}
	r.result <- checkResult
}

func checkLatestVersionBackground() *latestVersionCheck {
	conf := config.GetConfig()
	r := &latestVersionCheck{
		currentVersion:         version,
		forceCheckBeforeFinish: nowUTC().After(conf.ClientSelfUpdater.ForceCheckAfter),
	}
	r.result = make(chan latestVersionCheckResult, 1)
	go getRemoteVersionAndReportsToChanGoroutine(r)
	return r
}

func verifyLatestVersion(lvCheck *latestVersionCheck) {
	checkResult := latestVersionCheckResult{}
	if lvCheck.forceCheckBeforeFinish {
		// blocking
		checkResult = <-lvCheck.result

	} else {
		// non-blocking
		select {
		case checkResult = <-lvCheck.result:
		default:
		}
	}

	if checkResult.err != nil {
		fmt.Fprintf(stderr, "Could not query for latest version: %v\n", checkResult.err)
	}
	if checkResult.isFinished && checkResult.isOutdated {
		fmt.Fprintf(stderr, "INFO: A new version is available. Please update to the newer version %q (current: %q)\n", checkResult.latestVersion, lvCheck.currentVersion)
	}
}
