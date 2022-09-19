package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func (s *S) TestGetRemoteVersionAndReportsToChan(c *check.C) {
	eInvalid := "metadata.version is not a SemVersion: Invalid Semantic Version"

	for _, testCase := range []struct {
		currentVer         string
		latestVer          string
		expectedlatestVer  string
		expectedIsOutdated bool
		expectedMatchError string
	}{
		{"1.1.1", "1.2.2", "1.2.2", true, ""},              // has newer version
		{"invalid", "0.0.1", "0.0.1", true, ""},            // current invalid, always gives latest
		{"1.2.3", "1.2.3", "1.2.3", false, ""},             // is already latest
		{"1.1.2", "1.1.1", "1.1.1", false, ""},             // somehow, current is greater than latest
		{"dev", "1.2.3", "", false, ""},                    // dev version is a special case, early return
		{"1.1.1", "invalid", "invalid", false, eInvalid},   // latest invalid, gives error
		{"invalid", "invalid", "invalid", false, eInvalid}, // current and latest invalid, gives error
	} {

		tsMetadata := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Header().Add("Content-Type", "application/octet-stream")
			data := []byte(fmt.Sprintf(
				`{"project_name":"tsuru","tag":"%s","previous_tag":"1.0.0","version":"%s","commit":"1234567890abcdef","date":"2020-12-25T23:58:00.123456789Z","runtime":{"goos":"linux","goarch":"amd64"}}`,
				testCase.latestVer, testCase.latestVer,
			))
			w.Write(data)
		}))
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, tsMetadata.URL, 302) // github behavior: /releases/latest -> /releases/1.2.3
		}))
		latestManifestURL = ts.URL

		r := &latestVersionCheck{currentVersion: testCase.currentVer}
		r.result = make(chan latestVersionCheckResult)
		go getRemoteVersionAndReportsToChan(r)

		result := <-r.result

		c.Assert(result.isFinished, check.Equals, true)
		c.Assert(result.isOutdated, check.Equals, testCase.expectedIsOutdated)
		c.Assert(result.latestVersion, check.Equals, testCase.expectedlatestVer)
		if testCase.expectedMatchError == "" {
			c.Assert(result.err, check.IsNil)
		} else {
			c.Assert(result.err, check.NotNil)
			c.Assert(result.err, check.ErrorMatches, testCase.expectedMatchError)
		}
	}
}

func (s *S) TestGetRemoteVersionAndReportsToChanInvalidJSON(c *check.C) {
	tsMetadata := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/octet-stream")
		data := []byte("wrong format")
		w.Write(data)
	}))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, tsMetadata.URL, 302) // github behavior: /releases/latest -> /releases/1.2.3
	}))
	latestManifestURL = ts.URL

	r := &latestVersionCheck{currentVersion: "1.2.3"}
	r.result = make(chan latestVersionCheckResult)
	go getRemoteVersionAndReportsToChan(r)

	result := <-r.result

	c.Assert(result.isFinished, check.Equals, true)
	c.Assert(result.isOutdated, check.Equals, false)
	c.Assert(result.latestVersion, check.Equals, "")
	c.Assert(result.err, check.ErrorMatches, "Could not parse metadata.json. Unexpected format: invalid character.*")
}
