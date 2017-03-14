// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/docker/machine/libmachine/cert"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas/dockermachine"
)

var (
	storeBasePath              = cmd.JoinWithUserDir(".tsuru", "installs")
	DefaultDockerMachineConfig = &DockerMachineConfig{
		DriverName:  "virtualbox",
		Name:        "tsuru",
		DriverOpts:  make(map[string]interface{}),
		DockerFlags: []string{"experimental"},
	}
)

type DockerMachine struct {
	storePath     string
	certsPath     string
	API           dockermachine.DockerMachineAPI
	machinesCount uint64
	config        DockerMachineConfig
}

type DockerMachineConfig struct {
	DriverName      string
	CAPath          string
	Name            string
	DriverOpts      map[string]interface{}
	DockerHubMirror string
	DockerFlags     []string
}

type MachineProvisioner interface {
	ProvisionMachine(map[string]interface{}) (*dockermachine.Machine, error)
}

func NewDockerMachine(config *DockerMachineConfig) (*DockerMachine, error) {
	storePath := filepath.Join(storeBasePath, config.Name)
	certsPath := filepath.Join(storePath, "certs")
	dm, err := dockermachine.NewDockerMachine(dockermachine.DockerMachineConfig{
		CaPath:    config.CAPath,
		OutWriter: os.Stdout,
		ErrWriter: os.Stderr,
		StorePath: storePath,
	})
	if err != nil {
		return nil, err
	}
	return &DockerMachine{
		API:       dm,
		certsPath: certsPath,
		storePath: storePath,
		config:    *config,
	}, nil
}

func (d *DockerMachine) ProvisionMachine(driverOpts map[string]interface{}) (*dockermachine.Machine, error) {
	m, err := d.CreateMachine(driverOpts)
	if err != nil {
		return nil, fmt.Errorf("error creating machine %s", err)
	}
	err = d.uploadRegistryCertificate(GetPrivateIP(m), m.Host.Driver.GetSSHUsername(), m.Host)
	if err != nil {
		return nil, fmt.Errorf("error uploading registry certificates to %s: %s", m.Base.Address, err)
	}
	return m, nil
}

func (d *DockerMachine) CreateMachine(driverOpts map[string]interface{}) (*dockermachine.Machine, error) {
	driverOpts["swarm-master"] = false
	driverOpts["swarm-host"] = ""
	driverOpts["engine-install-url"] = ""
	driverOpts["swarm-discovery"] = ""
	mergedOpts := make(map[string]interface{})
	for k, v := range d.config.DriverOpts {
		mergedOpts[k] = v
	}
	for k, v := range driverOpts {
		mergedOpts[k] = v
	}
	m, err := d.API.CreateMachine(dockermachine.CreateMachineOpts{
		Name:           d.generateMachineName(),
		DriverName:     d.config.DriverName,
		Params:         mergedOpts,
		RegistryMirror: d.config.DockerHubMirror,
		ArbitraryFlags: d.config.DockerFlags,
	})
	if err != nil {
		return nil, err
	}
	if m.Host.AuthOptions() != nil {
		m.Host.AuthOptions().ServerCertSANs = append(m.Host.AuthOptions().ServerCertSANs, GetPrivateIP(m))
		err = m.Host.ConfigureAuth()
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (d *DockerMachine) generateMachineName() string {
	atomic.AddUint64(&d.machinesCount, 1)
	return fmt.Sprintf("%s-%d", d.config.Name, atomic.LoadUint64(&d.machinesCount))
}

func nixPathJoin(elem ...string) string {
	return strings.Join(elem, "/")
}

func (d *DockerMachine) uploadRegistryCertificate(ip, user string, target sshTarget) error {
	registryCertPath := filepath.Join(d.certsPath, "registry-cert.pem")
	registryKeyPath := filepath.Join(d.certsPath, "registry-key.pem")
	var registryIP string
	if _, errReg := os.Stat(registryCertPath); os.IsNotExist(errReg) {
		errCreate := d.createRegistryCertificate(ip)
		if errCreate != nil {
			return errCreate
		}
		registryIP = ip
	} else {
		certData, errRead := ioutil.ReadFile(registryCertPath)
		if errRead != nil {
			return fmt.Errorf("failed to read registry-cert.pem: %s", errRead)
		}
		block, _ := pem.Decode(certData)
		cert, errRead := x509.ParseCertificate(block.Bytes)
		if errRead != nil {
			return fmt.Errorf("failed to parse registry certificate: %s", errRead)
		}
		registryIP = cert.IPAddresses[0].String()
	}
	fmt.Printf("Uploading registry certificate...\n")
	certsBasePath := fmt.Sprintf("/home/%s/certs/%s:5000", user, registryIP)
	if _, err := target.RunSSHCommand(fmt.Sprintf("mkdir -p %s", certsBasePath)); err != nil {
		return err
	}
	dockerCertsPath := "/etc/docker/certs.d"
	if _, err := target.RunSSHCommand(fmt.Sprintf("sudo mkdir -p %s", dockerCertsPath)); err != nil {
		return err
	}
	fileCopies := map[string]string{
		registryCertPath:                         nixPathJoin(certsBasePath, "registry-cert.pem"),
		registryKeyPath:                          nixPathJoin(certsBasePath, "registry-key.pem"),
		filepath.Join(d.certsPath, "ca-key.pem"): nixPathJoin(dockerCertsPath, "ca-key.pem"),
		filepath.Join(d.certsPath, "ca.pem"):     nixPathJoin(dockerCertsPath, "ca.pem"),
		filepath.Join(d.certsPath, "cert.pem"):   nixPathJoin(dockerCertsPath, "cert.pem"),
		filepath.Join(d.certsPath, "key.pem"):    nixPathJoin(dockerCertsPath, "key.pem"),
	}
	for src, dst := range fileCopies {
		errWrite := writeRemoteFile(target, src, dst)
		if errWrite != nil {
			return errWrite
		}
	}
	if _, err := target.RunSSHCommand(fmt.Sprintf("sudo cp -r /home/%s/certs/* %s/", user, dockerCertsPath)); err != nil {
		return err
	}
	if _, err := target.RunSSHCommand(fmt.Sprintf("sudo cat %s/ca.pem | sudo tee -a /etc/ssl/certs/ca-certificates.crt", dockerCertsPath)); err != nil {
		return err
	}
	_, err := target.RunSSHCommand("sudo mkdir -p /var/lib/registry/")
	return err
}

func (d *DockerMachine) createRegistryCertificate(hosts ...string) error {
	fmt.Printf("Creating registry certificate...\n")
	caOrg := mcnutils.GetUsername()
	org := caOrg + ".<bootstrap>"
	generator := &cert.X509CertGenerator{}
	certOpts := &cert.Options{
		Hosts:       hosts,
		CertFile:    filepath.Join(d.certsPath, "registry-cert.pem"),
		KeyFile:     filepath.Join(d.certsPath, "registry-key.pem"),
		CAFile:      filepath.Join(d.certsPath, "ca.pem"),
		CAKeyFile:   filepath.Join(d.certsPath, "ca-key.pem"),
		Org:         org,
		Bits:        2048,
		SwarmMaster: false,
	}
	return generator.GenerateCert(certOpts)
}

func (d *DockerMachine) DeleteAll() error {
	return d.API.DeleteAll()
}

func (d *DockerMachine) Close() error {
	return d.API.Close()
}
