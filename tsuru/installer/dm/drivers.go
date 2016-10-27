// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/tsuru/config"
	"github.com/tsuru/tsuru/iaas/dockermachine"
)

var ErrNoPrivateIPInterface = errors.New("no private IP interface")

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

func GetPrivateIP(m *dockermachine.Machine) string {
	ip, err := getPrivateIP(m.Host.Driver)
	if err == nil {
		return ip
	}
	return m.Base.Address
}

func GetPrivateAddress(m *dockermachine.Machine) string {
	ip, err := getPrivateIP(m.Host.Driver)
	if err != nil {
		ip = m.Base.Address
	}
	return fmt.Sprintf("%s://%s:%d", m.Base.Protocol, ip, m.Base.Port)
}

func getPrivateIP(driver drivers.Driver) (string, error) {
	if driver == nil {
		return "", errors.New("driver must not be nil")
	}
	iface, err := GetPrivateIPInterface(driver.DriverName())
	if err == ErrNoPrivateIPInterface || iface == "" {
		return driver.GetIP()
	}
	output, err := drivers.RunSSHCommandFromDriver(driver, fmt.Sprintf("ip addr show dev %s", iface))
	if err != nil {
		return driver.GetIP()
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		vals := strings.Split(strings.TrimSpace(line), " ")
		if len(vals) >= 2 && vals[0] == "inet" {
			return vals[1][:strings.Index(vals[1], "/")], nil
		}
	}
	return driver.GetIP()
}

func writeRemoteFile(driver drivers.Driver, filePath string, remotePath string) error {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %s", filePath, err)
	}
	remoteWriteCmdFmt := "printf '%%s' '%s' | sudo tee %s"
	_, err = drivers.RunSSHCommandFromDriver(driver, fmt.Sprintf(remoteWriteCmdFmt, string(file), remotePath))
	if err != nil {
		return fmt.Errorf("failed to write remote file: %s", err)
	}
	return nil
}
