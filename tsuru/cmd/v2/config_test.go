// Copyright 2026 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package v2

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestPager(t *testing.T) {
	originalViper := defaultViper
	defer func() { defaultViper = originalViper }()

	tests := []struct {
		name          string
		setup         func(v *viper.Viper)
		expectedPager string
		expectedFound bool
	}{
		{
			name:          "pager key not set",
			setup:         func(v *viper.Viper) {},
			expectedPager: "",
			expectedFound: false,
		},
		{
			name: "pager set to boolean true",
			setup: func(v *viper.Viper) {
				v.Set("pager", true)
			},
			expectedPager: "",
			expectedFound: false,
		},
		{
			name: "pager set to string true",
			setup: func(v *viper.Viper) {
				v.Set("pager", "true")
			},
			expectedPager: "",
			expectedFound: false,
		},
		{
			name: "pager set to boolean false",
			setup: func(v *viper.Viper) {
				v.Set("pager", false)
			},
			expectedPager: "",
			expectedFound: true,
		},
		{
			name: "pager set to string false",
			setup: func(v *viper.Viper) {
				v.Set("pager", "false")
			},
			expectedPager: "",
			expectedFound: true,
		},
		{
			name: "pager set to custom pager command",
			setup: func(v *viper.Viper) {
				v.Set("pager", "less -R")
			},
			expectedPager: "less -R",
			expectedFound: true,
		},
		{
			name: "pager set to another custom pager",
			setup: func(v *viper.Viper) {
				v.Set("pager", "more")
			},
			expectedPager: "more",
			expectedFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			tt.setup(v)
			defaultViper = v

			pager, found := Pager()

			assert.Equal(t, tt.expectedPager, pager)
			assert.Equal(t, tt.expectedFound, found)
		})
	}
}
