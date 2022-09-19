package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/Masterminds/semver"
)

var (
	latestManifestURL string        = "https://github.com/tsuru/tsuru-client/releases/latest/download/manifest.json"
	stdout            io.ReadWriter = os.Stdout
	stderr            io.ReadWriter = os.Stderr
)

type latestVersionCheckResult struct {
	isFinished    bool
	isOutdated    bool
	latestVersion string
	err           error
}
type latestVersionCheck struct {
	lastChecked            time.Time
	currentVersion         string
	forceCheckBeforeFinish bool
	result                 chan latestVersionCheckResult
}

type releaseMetadata struct {
	Version string    `json:"version"`
	Date    time.Time `json:"date"`
}

// This function "returns" its results over the r.result channel
func getRemoteVersionAndReportsToChan(r *latestVersionCheck) {
	checkResult := latestVersionCheckResult{
		isFinished: true,
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

	metadata := releaseMetadata{}
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		checkResult.err = fmt.Errorf("Could not parse metadata.json. Unexpected format: %w", err)
		r.result <- checkResult
		return
	}

	checkResult.latestVersion = metadata.Version
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
	go getRemoteVersionAndReportsToChan(r)
	return r
}
func verifyLatestVersion(checkVerResult *latestVersionCheck) {
	checkResult := latestVersionCheckResult{}
	if checkVerResult.forceCheckBeforeFinish {
		checkResult = <-checkVerResult.result // blocking
	} else {
		select { // non-blocking
		case checkResult = <-checkVerResult.result:
		default:
		}
	}

	if checkResult.isFinished && checkResult.isOutdated {
		fmt.Fprintf(stderr, "A new version is available. Please update to the newer version %q (current: %q)\n", checkResult.latestVersion, checkVerResult.currentVersion)
	}
}
