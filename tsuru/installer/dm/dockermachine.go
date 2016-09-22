// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dm

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/cert"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/drivers/rpc"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/tsuru/tsuru/cmd"
)

var (
	dockerHTTPSPort            = 2376
	storeBasePath              = cmd.JoinWithUserDir(".tsuru", "installs")
	DefaultDockerMachineConfig = &DockerMachineConfig{
		DriverName: "virtualbox",
		Name:       "tsuru",
		DriverOpts: make(DriverOpts),
	}
)

type DockerMachine struct {
	io.Closer
	driverName       string
	storePath        string
	certsPath        string
	Name             string
	client           libmachine.API
	machinesCount    uint64
	globalDriverOpts DriverOpts
}

type DockerMachineConfig struct {
	DriverName string
	CAPath     string
	Name       string
	DriverOpts DriverOpts
}

type MachineProvisioner interface {
	ProvisionMachine(map[string]interface{}) (*Machine, error)
}

func NewDockerMachine(config *DockerMachineConfig) (*DockerMachine, error) {
	storePath := filepath.Join(storeBasePath, config.Name)
	certsPath := filepath.Join(storePath, "certs")
	err := os.MkdirAll(certsPath, 0700)
	if err != nil {
		return nil, fmt.Errorf("failed to create certs dir: %s", err)
	}
	if config.CAPath != "" {
		fmt.Printf("Copying CA file from %s to %s", filepath.Join(config.CAPath, "ca.pem"), filepath.Join(certsPath, "ca.pem"))
		err = copy(filepath.Join(config.CAPath, "ca.pem"), filepath.Join(certsPath, "ca.pem"))
		if err != nil {
			return nil, fmt.Errorf("failed to copy ca file: %s", err)
		}
		fmt.Printf("Copying CA key from %s to %s", filepath.Join(config.CAPath, "ca-key.pem"), filepath.Join(certsPath, "ca-key.pem"))
		err = copy(filepath.Join(config.CAPath, "ca-key.pem"), filepath.Join(certsPath, "ca-key.pem"))
		if err != nil {
			return nil, fmt.Errorf("failed to copy ca key file: %s", err)
		}
	}
	return &DockerMachine{
		driverName:       config.DriverName,
		storePath:        storePath,
		certsPath:        certsPath,
		Name:             config.Name,
		client:           libmachine.NewClient(storePath, certsPath),
		globalDriverOpts: config.DriverOpts,
	}, nil
}

func copy(src, dst string) error {
	fileSrc, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dst, fileSrc, 0644)
	if err != nil {
		return err
	}
	return nil
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
	rawDriver, err := json.Marshal(&drivers.BaseDriver{
		MachineName: d.generateMachineName(),
		StorePath:   d.storePath,
	})
	if err != nil {
		return nil, fmt.Errorf("Error creating docker-machine driver: %s", err)
	}
	host, err := d.client.NewHost(d.driverName, rawDriver)
	if err != nil {
		return nil, err
	}
	d.configureServerCertPaths(host)
	d.configureDriver(host.Driver, driverOpts)
	err = d.client.Create(host)
	if err != nil {
		fmt.Printf("Ignoring error on machine creation: %s", err)
	}
	ip, err := host.Driver.GetIP()
	if err != nil {
		return nil, err
	}
	m := &Machine{
		IP:         ip,
		CAPath:     d.certsPath,
		Host:       host,
		Address:    fmt.Sprintf("https://%s:%d", ip, dockerHTTPSPort),
		DriverOpts: DriverOpts(driverOpts),
	}
	if host.AuthOptions() != nil {
		host.AuthOptions().ServerCertSANs = append(host.AuthOptions().ServerCertSANs, m.GetPrivateIP())
		err = host.ConfigureAuth()
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (d *DockerMachine) configureServerCertPaths(h *host.Host) {
	if h.AuthOptions() != nil {
		h.AuthOptions().ServerCertPath = filepath.Join(d.client.GetMachinesDir(), h.Name, "server.pem")
		h.AuthOptions().ServerKeyPath = filepath.Join(d.client.GetMachinesDir(), h.Name, "server-key.pem")
	}
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

func (d *DockerMachine) configureDriver(driver drivers.Driver, driverOpts DriverOpts) error {
	mergedOpts := make(DriverOpts)
	for k, v := range d.globalDriverOpts {
		mergedOpts[k] = v
	}
	for k, v := range driverOpts {
		mergedOpts[k] = v
	}
	opts := &rpcdriver.RPCFlags{Values: mergedOpts}
	for _, c := range driver.GetCreateFlags() {
		_, ok := opts.Values[c.String()]
		if !ok {
			opts.Values[c.String()] = c.Default()
			if c.Default() == nil {
				opts.Values[c.String()] = false
			}
		}
	}
	opts.Values["swarm-master"] = false
	opts.Values["swarm-host"] = ""
	opts.Values["engine-install-url"] = ""
	opts.Values["swarm-discovery"] = ""
	if err := driver.SetConfigFromFlags(opts); err != nil {
		return fmt.Errorf("Error setting driver configurations: %s", err)
	}
	return nil
}

func (d *DockerMachine) DeleteAll() error {
	hosts, err := d.client.List()
	if err != nil {
		return err
	}
	for _, h := range hosts {
		errDel := d.DeleteMachine(h)
		if errDel != nil {
			return errDel
		}
	}
	err = os.RemoveAll(d.storePath)
	if err != nil {
		return err
	}
	return nil
}

func (d *DockerMachine) DeleteMachine(name string) error {
	h, err := d.client.Load(name)
	if err != nil {
		return err
	}
	err = h.Driver.Remove()
	if err != nil {
		return err
	}
	return d.client.Remove(name)
}

func (d *DockerMachine) Close() error {
	return d.client.Close()
}
