// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/cert"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas/dockermachine"
)

var (
	storeBasePath              = cmd.JoinWithUserDir(".tsuru", "installs")
	DefaultDockerMachineConfig = &DockerMachineConfig{
		DriverName: "virtualbox",
		Name:       "tsuru",
		DriverOpts: make(DriverOpts),
	}
)

type DockerMachine struct {
	driverName       string
	Name             string
	storePath        string
	certsPath        string
	dm               dockermachine.DockerMachineAPI
	machinesCount    uint64
	globalDriverOpts DriverOpts
	dockerHubMirror  string
}

type DockerMachineConfig struct {
	DriverName      string
	CAPath          string
	Name            string
	DriverOpts      DriverOpts
	DockerHubMirror string
}

type MachineProvisioner interface {
	ProvisionMachine(map[string]interface{}) (*Machine, error)
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
		driverName:       config.DriverName,
		Name:             config.Name,
		dm:               dm,
		globalDriverOpts: config.DriverOpts,
		dockerHubMirror:  config.DockerHubMirror,
		certsPath:        certsPath,
		storePath:        storePath,
	}, nil
}

func (d *DockerMachine) ProvisionMachine(driverOpts map[string]interface{}) (*Machine, error) {
	m, err := d.CreateMachine(driverOpts)
	if err != nil {
		return nil, fmt.Errorf("error creating machine %s", err)
	}
	err = d.uploadRegistryCertificate(m)
	if err != nil {
		return nil, fmt.Errorf("error uploading registry certificates to %s: %s", m.IP, err)
	}
	return m, nil
}

func (d *DockerMachine) CreateMachine(driverOpts map[string]interface{}) (*Machine, error) {
	driverOpts["swarm-master"] = false
	driverOpts["swarm-host"] = ""
	driverOpts["engine-install-url"] = ""
	driverOpts["swarm-discovery"] = ""
	mergedOpts := make(DriverOpts)
	for k, v := range d.globalDriverOpts {
		mergedOpts[k] = v
	}
	for k, v := range driverOpts {
		mergedOpts[k] = v
	}
	mIaas, err := d.dm.CreateMachine(dockermachine.CreateMachineOpts{
		Name:           d.generateMachineName(),
		DriverName:     d.driverName,
		Params:         mergedOpts,
		RegistryMirror: d.dockerHubMirror,
	})
	if err != nil {
		return nil, err
	}
	m := &Machine{
		Host:       mIaas.Host,
		IP:         mIaas.Base.Address,
		Address:    mIaas.Base.FormatNodeAddress(),
		DriverOpts: mergedOpts,
	}
	if m.Host.AuthOptions() != nil {
		m.Host.AuthOptions().ServerCertSANs = append(m.Host.AuthOptions().ServerCertSANs, m.GetPrivateIP())
		err = m.Host.ConfigureAuth()
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (d *DockerMachine) generateMachineName() string {
	atomic.AddUint64(&d.machinesCount, 1)
	return fmt.Sprintf("%s-%d", d.Name, atomic.LoadUint64(&d.machinesCount))
}

func (d *DockerMachine) uploadRegistryCertificate(host SSHTarget) error {
	registryCertPath := filepath.Join(d.certsPath, "registry-cert.pem")
	registryKeyPath := filepath.Join(d.certsPath, "registry-key.pem")
	var registryIP string
	if _, err := os.Stat(registryCertPath); os.IsNotExist(err) {
		errCreate := d.createRegistryCertificate(host.GetIP())
		if errCreate != nil {
			return errCreate
		}
		registryIP = host.GetIP()
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
	certsBasePath := fmt.Sprintf("/home/%s/certs/%s:5000", host.GetSSHUsername(), registryIP)
	if _, err := host.RunSSHCommand(fmt.Sprintf("mkdir -p %s", certsBasePath)); err != nil {
		return err
	}
	dockerCertsPath := "/etc/docker/certs.d"
	if _, err := host.RunSSHCommand(fmt.Sprintf("sudo mkdir %s", dockerCertsPath)); err != nil {
		return err
	}
	if err := writeRemoteFile(host, registryCertPath, filepath.Join(certsBasePath, "registry-cert.pem")); err != nil {
		return err
	}
	if err := writeRemoteFile(host, registryKeyPath, filepath.Join(certsBasePath, "registry-key.pem")); err != nil {
		return err
	}
	if err := writeRemoteFile(host, filepath.Join(d.certsPath, "ca-key.pem"), filepath.Join(dockerCertsPath, "ca-key.pem")); err != nil {
		return err
	}
	if err := writeRemoteFile(host, filepath.Join(d.certsPath, "ca.pem"), filepath.Join(dockerCertsPath, "ca.pem")); err != nil {
		return err
	}
	if err := writeRemoteFile(host, filepath.Join(d.certsPath, "cert.pem"), filepath.Join(dockerCertsPath, "cert.pem")); err != nil {
		return err
	}
	if err := writeRemoteFile(host, filepath.Join(d.certsPath, "key.pem"), filepath.Join(dockerCertsPath, "key.pem")); err != nil {
		return err
	}
	if _, err := host.RunSSHCommand(fmt.Sprintf("sudo cp -r /home/%s/certs/* %s/", host.GetSSHUsername(), dockerCertsPath)); err != nil {
		return err
	}
	if _, err := host.RunSSHCommand(fmt.Sprintf("sudo cat %s/ca.pem | sudo tee -a /etc/ssl/certs/ca-certificates.crt", dockerCertsPath)); err != nil {
		return err
	}
	_, err := host.RunSSHCommand("sudo mkdir -p /var/lib/registry/")
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
	return d.dm.DeleteAll()
}

func (d *DockerMachine) Close() error {
	return d.dm.Close()
}

type TempDockerMachine struct {
	client    libmachine.API
	storePath string
	certsPath string
}

func NewTempDockerMachine() (*TempDockerMachine, error) {
	storePath, err := ioutil.TempDir("", "store")
	if err != nil {
		return nil, fmt.Errorf("failed o create store dir: %s", err)
	}
	certsPath := filepath.Join(storePath, "certs")
	err = os.Mkdir(certsPath, 0700)
	if err != nil {
		return nil, fmt.Errorf("failed to create certs dir: %s", err)
	}
	return &TempDockerMachine{
		client:    libmachine.NewClient(storePath, certsPath),
		storePath: storePath,
		certsPath: certsPath,
	}, nil
}

func (d *TempDockerMachine) Close() error {
	defer os.RemoveAll(d.storePath)
	return d.client.Close()
}

func (d *TempDockerMachine) NewHost(driverName, sshKey string, driver map[string]interface{}) (*host.Host, error) {
	sshKeyPath := filepath.Join(d.certsPath, "id_rsa")
	err := ioutil.WriteFile(sshKeyPath, []byte(sshKey), 0700)
	if err != nil {
		return nil, err
	}
	driver["SSHKeyPath"] = sshKeyPath
	driver["StorePath"] = d.storePath
	rawDriver, err := json.Marshal(driver)
	if err != nil {
		return nil, err
	}
	h, err := d.client.NewHost(driverName, rawDriver)
	if err != nil {
		return nil, err
	}
	err = d.client.Save(h)
	if err != nil {
		return nil, err
	}
	return d.client.Load(h.Name)
}
