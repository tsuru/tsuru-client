package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

var (
	stdout io.ReadWriter = os.Stdout
	stderr io.ReadWriter = os.Stderr
)

type latestVersionCheckResult struct {
	isFinished    bool
	isOutdated    bool
	latestVersion string
}
type latestVersionCheck struct {
	lastChecked            time.Time
	currentVersion         string
	forceCheckBeforeFinish bool
	result                 chan latestVersionCheckResult
}

func checkLatestVersionBackground() *latestVersionCheck {
	// TODO: Implement
	r := &latestVersionCheck{
		currentVersion:         version,
		forceCheckBeforeFinish: false,
	}
	r.result = make(chan latestVersionCheckResult)
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
