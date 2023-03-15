// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"gopkg.in/check.v1"
)

func (s *S) TestGetSuggestions(c *check.C) {
	manager = buildManager("tsuru")

	tests := []struct {
		AutocompLine  []string
		PositiveMatch []string
		NegativeMatch []string
	}{
		{
			[]string{"tsuru", ""},
			[]string{"app", "login", "logout", "user"},
			[]string{"deploy", "list", ""},
		},
		{
			[]string{"tsuru", "l"},
			[]string{"login", "logout"},
			[]string{"app", "deploy", "list", ""}},
		{
			[]string{"tsuru", "app", ""},
			[]string{"build", "deploy", "info", "start", "stop"},
			[]string{"app", ""}},
		{
			[]string{"tsuru", "app", "s"},
			[]string{"start", "stop"},
			[]string{"app", "", "build", "deploy", "info"}},
		{
			[]string{"tsuru", "app", "st"},
			[]string{"start", "stop"},
			[]string{"app", "", "build", "deploy", "info"}},
		{
			[]string{"tsuru", "app", "sta"},
			[]string{"start"},
			[]string{"app", "", "build", "deploy", "info", "stop"},
		},
	}

	for _, test := range tests {
		suggestions := getSuggestions(manager, test.AutocompLine)
		for _, s := range test.PositiveMatch {
			c.Assert(sliceContains(suggestions, s), check.Equals, true)
		}
		for _, s := range test.NegativeMatch {
			c.Assert(sliceContains(suggestions, s), check.Equals, false)
		}
	}
}

func (s *S) TestGetSuggestionsWrongInput(c *check.C) {
	manager = buildManager("tsuru")

	tests := []struct {
		AutocompLine []string
	}{
		{[]string{"tsuru"}},
		{[]string{"tsuru", "\"app "}},
		{[]string{"tsuru", "'app"}},
		{[]string{"tsuru", "\"'app"}},
		{[]string{"tsuru", "\"'app'"}},
		{[]string{"tsuru", "\"'app\""}}, // valid, but no suggestion should be given
	}

	for _, test := range tests {
		suggestions := getSuggestions(manager, test.AutocompLine)
		c.Assert(len(suggestions), check.Equals, 0, check.Commentf("test command: %v", test.AutocompLine))
	}
}

func sliceContains(sStr []string, s string) bool {
	for _, sStr := range sStr {
		if sStr == s {
			return true
		}
	}
	return false
}
