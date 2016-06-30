// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fake

import (
	"github.com/tsuru/tsuru-client/tsuru/installer/iaas"
)

func init() {
	iaas.Register("fake", &fakeIaas{})
}

type fakeIaas struct{}

func (i *fakeIaas) CreateMachine(params map[string]string) (*iaas.Machine, error) {
	return &iaas.Machine{}, nil
}

func (i *fakeIaas) DeleteMachine(m *iaas.Machine) error {
	return nil
}
