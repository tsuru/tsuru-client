// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"errors"

	"github.com/tsuru/config"
)

var ErrNoPrivateIPInterface = errors.New("no private IP interface")

type DriverOpts map[string]interface{}

// GetPrivateIPInterface returns the interface name which contains
// the private IP address of a machine provisioned with the given
// driver. This is a workaround while the Driver interface does not
// provide a way to access the instance private IP.
func GetPrivateIPInterface(driverName string) (string, error) {
	iface, err := config.GetString("driver:private-ip-interface")
	if err == nil {
		return iface, nil
	}
	switch driverName {
	case "amazonec2":
		return "eth0", nil
	default:
		return "", ErrNoPrivateIPInterface
	}
}
