// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/docker/machine/libmachine/engine"
	"github.com/docker/machine/libmachine/host"
	docker "github.com/fsouza/go-dockerclient"
)

type Machine struct {
	*host.Host
	IP         string
	Address    string
	CAPath     string
	DriverOpts DriverOpts
}

type SSHTarget interface {
	RunSSHCommand(string) (string, error)
	GetIP() string
	GetSSHUsername() string
	GetSSHKeyPath() string
}

func (m *Machine) DockerClient() (*docker.Client, error) {
	return docker.NewTLSClient(
		m.Address,
		filepath.Join(m.CAPath, "cert.pem"),
		filepath.Join(m.CAPath, "key.pem"),
		filepath.Join(m.CAPath, "ca.pem"),
	)
}

func (m *Machine) GetIP() string {
	return m.IP
}

func (m *Machine) GetSSHUsername() string {
	return m.Driver.GetSSHUsername()
}

func (m *Machine) GetSSHKeyPath() string {
	return m.Driver.GetSSHKeyPath()
}

// GetPrivateIP returns the instance private IP; if not available,
// will fallback to the public IP.
func (m *Machine) GetPrivateIP() string {
	iface, err := GetPrivateIPInterface(m.DriverName)
	if err == ErrNoPrivateIPInterface || iface == "" {
		return m.GetIP()
	}
	return getIp(iface, m)
}

func (m *Machine) GetPrivateAddress() string {
	return fmt.Sprintf("https://%s:%d", m.GetPrivateIP(), engine.DefaultPort)
}

func getIp(iface string, remote SSHTarget) string {
	output, err := remote.RunSSHCommand(fmt.Sprintf("ip addr show dev %s", iface))
	if err != nil {
		return remote.GetIP()
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		vals := strings.Split(strings.TrimSpace(line), " ")
		if len(vals) >= 2 && vals[0] == "inet" {
			return vals[1][:strings.Index(vals[1], "/")]
		}
	}
	return remote.GetIP()
}

func writeRemoteFile(host SSHTarget, filePath string, remotePath string) error {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %s", filePath, err)
	}
	remoteWriteCmdFmt := "printf '%%s' '%s' | sudo tee %s"
	_, err = host.RunSSHCommand(fmt.Sprintf(remoteWriteCmdFmt, string(file), remotePath))
	if err != nil {
		return fmt.Errorf("failed to write remote file: %s", err)
	}
	return nil
}
