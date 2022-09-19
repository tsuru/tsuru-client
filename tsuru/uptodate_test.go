package main

import (
	"time"

	"gopkg.in/check.v1"
)

func (s *S) TestVerifyLatestVersionSyncTimeout(c *check.C) {
	timeoutChan := make(chan bool)
	go func(ch chan bool) {
		time.Sleep(1 * time.Second)
		ch <- true
	}(timeoutChan)

	resultChan := make(chan bool)
	cv := &latestVersionCheck{forceCheckBeforeFinish: true}
	go func(ch chan bool, cv1 *latestVersionCheck) {
		verifyLatestVersion(cv1)
		ch <- true
	}(resultChan, cv)

	select {
	case <-timeoutChan:
	case <-resultChan:
		c.Assert("Response was received", check.Equals, "verifyLatestVersion should timeout")
	}
}

func (s *S) TestVerifyLatestVersionSyncFinish(c *check.C) {
	// testing sleep(500ms) -> cv.result ||> verifyLatestVersion() -> resultChan
	//    time   | 0----5----10----15
	// premature |   *
	// result    |      O
	// timeout   |           *

	resultChan := make(chan bool, 1)
	timeoutChan := make(chan bool, 1)
	prematureChan := make(chan bool, 1)
	cv := &latestVersionCheck{forceCheckBeforeFinish: true}
	cv.result = make(chan latestVersionCheckResult, 1)

	go func(ch chan bool) {
		time.Sleep(1000 * time.Millisecond)
		ch <- true
	}(timeoutChan)

	go func(ch chan bool) {
		time.Sleep(200 * time.Millisecond)
		ch <- true
	}(prematureChan)

	go func(cv1 *latestVersionCheck) {
		time.Sleep(500 * time.Millisecond)
		cv1.result <- latestVersionCheckResult{
			isFinished:    true,
			isOutdated:    false,
			latestVersion: "latest",
		}
	}(cv)

	go func(ch chan bool, cv1 *latestVersionCheck) {
		verifyLatestVersion(cv1)
		ch <- true
	}(resultChan, cv)

	select {
	case <-prematureChan:
	case <-resultChan:
		c.Assert("Should have finished after prematureChan", check.Equals, "but ended before")
	}

	select {
	case <-timeoutChan:
		c.Assert("Reached final timeout", check.Equals, "resultChan was expected")
	case <-resultChan:
	}
}
