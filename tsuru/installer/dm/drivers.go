// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/tsuru/config"
	"github.com/tsuru/tsuru/iaas/dockermachine"
)

// getPrivateIPInterface returns the interface name which contains
// the private IP address of a machine provisioned with the given
// driver. This is a workaround while the Driver interface does not
// provide a way to access the instance private IP.
func getPrivateIPInterface() string {
	iface, err := config.GetString("driver:private-ip-interface")
	if err == nil {
		return iface
	}
	return "eth0"
}

func GetPrivateIP(m *dockermachine.Machine) string {
	iface := getPrivateIPInterface()
	if iface == "" {
		return m.Base.Address
	}
	if m.Host != nil && m.Host.Driver != nil {
		ip, err := getIP(m.Host, iface)
		if err == nil {
			return ip
		}
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

func WriterRemoteData(target sshTarget, remotePath string, remoteData []byte) error {
	base64Data := base64.StdEncoding.EncodeToString(remoteData)
	remoteWriteCmdFmt := `set -o pipefail
if type base64; then
	base64dec="base64 --decode"
elif type openssl; then
	base64dec="openssl enc -base64 -d -A"
elif type python; then
	base64dec="python -m base64 -d"
fi
if [ -z "$base64dec" ]; then
	echo "no base64 decoder available"
	exit 1
fi
echo '%s' | $base64dec | sudo tee %s`
	_, err := target.RunSSHCommand(fmt.Sprintf(remoteWriteCmdFmt, base64Data, remotePath))
	if err != nil {
		return fmt.Errorf("failed to write remote file: %s", err)
	}
	return nil
}

func writeRemoteFile(target sshTarget, filePath string, remotePath string) error {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %s", filePath, err)
	}
	return WriterRemoteData(target, remotePath, file)
}

func DefaultDriverConfig(driverName string) map[string]interface{} {
	opts := make(map[string]interface{})
	switch driverName {
	case "virtualbox":
		opts["driver:options:virtualbox-memory"] = 2048
		opts["driver:options:virtualbox-nat-nictype"] = "virtio"
	case "google":
		opts["driver:private-ip-interface"] = "ens4"
	}
	return opts
}

func IaaSCompatibleDriver(driverName string) bool {
	return driverName != "virtualbox" && driverName != "generic"
}
