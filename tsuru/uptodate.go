package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
)

var (
	latestManifestURL string        = "https://github.com/tsuru/tsuru-client/releases/latest/download/manifest.json"
	stdout            io.ReadWriter = os.Stdout
	stderr            io.ReadWriter = os.Stderr
)

type latestVersionCheckResult struct {
	isFinished bool
	err        error

	isOutdated    bool
	latestVersion string
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
	checkResult := latestVersionCheckResult{
		isFinished:    true,
		latestVersion: r.currentVersion,
	}

	if r.currentVersion == "dev" {
		r.result <- checkResult
		return
	}

	response, err := http.Get(latestManifestURL)
	if err != nil {
		checkResult.err = fmt.Errorf("Could not get %q endpoint: %w", latestManifestURL, err)
		r.result <- checkResult
		return
	}
	defer response.Body.Close()

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
		checkResult.err = fmt.Errorf("metadata.version is not a SemVersion: %w", err)
		r.result <- checkResult
		return
	}

	if current.Compare(latest) < 0 {
		checkResult.latestVersion = latest.String()
		checkResult.isOutdated = true
	}
	r.result <- checkResult
}

func checkLatestVersionBackground() *latestVersionCheck {
	r := &latestVersionCheck{
		currentVersion:         version,
		forceCheckBeforeFinish: false,
	}
	r.result = make(chan latestVersionCheckResult)
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
		fmt.Fprintf(stderr, "A new version is available. Please update to the newer version %q (current: %q)\n", checkResult.latestVersion, lvCheck.currentVersion)
	}
}
