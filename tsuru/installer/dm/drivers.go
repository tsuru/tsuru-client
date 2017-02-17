// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

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
	case "google":
		return "eth0", nil
	default:
		return "", ErrNoPrivateIPInterface
	}
}

func GetPrivateIP(m *dockermachine.Machine) string {
	iface, err := GetPrivateIPInterface(m.Host.DriverName)
	if err != nil || iface == "" {
		return m.Base.Address
	}
	ip, err := getIP(m.Host, iface)
	if err == nil {
		return ip
	}
	return m.Base.Address
}

func GetPrivateAddress(m *dockermachine.Machine) string {
	return fmt.Sprintf("%s://%s:%d", m.Base.Protocol, GetPrivateIP(m), m.Base.Port)
}

type sshTarget interface {
	RunSSHCommand(string) (string, error)
}

func getIP(target sshTarget, iface string) (string, error) {
	output, err := target.RunSSHCommand(fmt.Sprintf("ip addr show dev %s", iface))
	if err != nil {
		return "", err
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		vals := strings.Split(strings.TrimSpace(line), " ")
		if len(vals) >= 2 && vals[0] == "inet" {
			return vals[1][:strings.Index(vals[1], "/")], nil
		}
	}
	return "", errors.New("failed to parse private ip from interface")
}

func writeRemoteFile(target sshTarget, filePath string, remotePath string) error {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %s", filePath, err)
	}
	remoteWriteCmdFmt := "printf '%%s' '%s' | sudo tee %s"
	_, err = target.RunSSHCommand(fmt.Sprintf(remoteWriteCmdFmt, string(file), remotePath))
	if err != nil {
		return fmt.Errorf("failed to write remote file: %s", err)
	}
	return nil
}

func DefaultDriverOpts(driverName string) map[string]interface{} {
	opts := make(map[string]interface{})
	switch driverName {
	case "virtualbox":
		opts["virtualbox-memory"] = 2048
		opts["virtualbox-nat-nictype"] = "virtio"
	}
	return opts
}
