// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/machine/drivers/amazonec2"
	"github.com/docker/machine/drivers/azure"
	"github.com/docker/machine/drivers/digitalocean"
	"github.com/docker/machine/drivers/exoscale"
	"github.com/docker/machine/drivers/generic"
	"github.com/docker/machine/drivers/google"
	"github.com/docker/machine/drivers/hyperv"
	"github.com/docker/machine/drivers/none"
	"github.com/docker/machine/drivers/openstack"
	"github.com/docker/machine/drivers/rackspace"
	"github.com/docker/machine/drivers/softlayer"
	"github.com/docker/machine/drivers/virtualbox"
	"github.com/docker/machine/drivers/vmwarefusion"
	"github.com/docker/machine/drivers/vmwarevcloudair"
	"github.com/docker/machine/drivers/vmwarevsphere"
	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/cert"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/docker/machine/libmachine/drivers/rpc"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/fsouza/go-dockerclient"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec"
)

var (
	dockerHTTPSPort            = 2376
	storeBasePath              = cmd.JoinWithUserDir(".tsuru", "installs")
	defaultDockerMachineConfig = &DockerMachineConfig{
		DriverName: "virtualbox",
		Name:       "tsuru",
		DriverOpts: make(map[string]interface{}),
	}
)

type Machine struct {
	*host.Host
	IP      string
	Config  map[string]string
	Address string
	CAPath  string
}

func (m *Machine) dockerClient() (*docker.Client, error) {
	return docker.NewTLSClient(
		m.Address,
		filepath.Join(m.CAPath, "cert.pem"),
		filepath.Join(m.CAPath, "key.pem"),
		filepath.Join(m.CAPath, "ca.pem"),
	)
}

func (m Machine) Driver() drivers.Driver {
	return m.Host.Driver
}

type DockerMachine struct {
	driverOpts map[string]interface{}
	rawDriver  []byte
	driverName string
	storePath  string
	certsPath  string
	Name       string
}

type DockerMachineConfig struct {
	DriverName string
	DriverOpts map[string]interface{}
	CAPath     string
	Name       string
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
	rawDriver, err := json.Marshal(&drivers.BaseDriver{
		MachineName: config.Name,
		StorePath:   storePath,
	})
	if err != nil {
		return nil, fmt.Errorf("Error creating docker-machine driver: %s", err)
	}
	return &DockerMachine{
		driverOpts: config.DriverOpts,
		rawDriver:  rawDriver,
		driverName: config.DriverName,
		storePath:  storePath,
		certsPath:  certsPath,
		Name:       config.Name,
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

func (d *DockerMachine) CreateMachine() (*Machine, error) {
	client := libmachine.NewClient(d.storePath, d.certsPath)
	defer client.Close()
	host, err := client.NewHost(d.driverName, d.rawDriver)
	if err != nil {
		return nil, err
	}
	configureDriver(host.Driver, d.driverOpts)
	err = client.Create(host)
	if err != nil {
		fmt.Printf("Ignoring error on machine creation: %s", err)
	}
	ip, err := host.Driver.GetIP()
	if err != nil {
		return nil, err
	}
	config := map[string]string{}
	m := Machine{
		IP:     ip,
		Config: config,
		CAPath: d.certsPath,
		Host:   host,
	}
	m.Address = fmt.Sprintf("https://%s:%d", m.IP, dockerHTTPSPort)
	err = d.uploadRegistryCertificate(m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type sshTarget interface {
	RunSSHCommand(string) (string, error)
	Driver() drivers.Driver
}

func (d *DockerMachine) uploadRegistryCertificate(host sshTarget) error {
	driver := host.Driver()
	ip, err := driver.GetIP()
	if err != nil {
		return err
	}
	if _, err = os.Stat(filepath.Join(d.certsPath, "registry-cert.pem")); os.IsNotExist(err) {
		errCreate := d.createRegistryCertificate(ip)
		if errCreate != nil {
			return errCreate
		}
	}
	fmt.Printf("Uploading registry certificate...\n")
	args := []string{
		"-o StrictHostKeyChecking=no",
		"-i",
		fmt.Sprintf("%s/machines/%s/id_rsa", d.storePath, d.Name),
		"-r",
		fmt.Sprintf("%s/", d.certsPath),
		fmt.Sprintf("%s@%s:/home/%s/", driver.GetSSHUsername(), ip, driver.GetSSHUsername()),
	}
	stdout := bytes.NewBufferString("")
	opts := exec.ExecuteOptions{
		Cmd:    "scp",
		Args:   args,
		Stdout: stdout,
	}
	err = client.Executor().Execute(opts)
	if err != nil {
		return fmt.Errorf("Command: %s. Error:%s", stdout.String(), err.Error())
	}
	certsBasePath := fmt.Sprintf("/home/%s/certs/%s:5000", driver.GetSSHUsername(), ip)
	_, err = host.RunSSHCommand(fmt.Sprintf("mkdir -p %s", certsBasePath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("cp /home/%s/certs/*.pem %s/", driver.GetSSHUsername(), certsBasePath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("sudo mkdir /etc/docker/certs.d && sudo cp -r /home/%s/certs/* /etc/docker/certs.d/", driver.GetSSHUsername()))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("cat %s/ca.pem | sudo tee -a /etc/ssl/certs/ca-certificates.crt", certsBasePath))
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

func configureDriver(driver drivers.Driver, driverOpts map[string]interface{}) error {
	opts := &rpcdriver.RPCFlags{Values: driverOpts}
	for _, c := range driver.GetCreateFlags() {
		_, ok := opts.Values[c.String()]
		if !ok {
			opts.Values[c.String()] = c.Default()
			if c.Default() == nil {
				opts.Values[c.String()] = false
			}
		}
	}
	if err := driver.SetConfigFromFlags(opts); err != nil {
		return fmt.Errorf("Error setting driver configurations: %s", err)
	}
	return nil
}

func (d *DockerMachine) DeleteMachine(m *Machine) error {
	client := libmachine.NewClient(d.storePath, d.certsPath)
	defer client.Close()
	h, err := client.Load(d.Name)
	if err != nil {
		return err
	}
	err = h.Driver.Remove()
	if err != nil {
		return err
	}
	return client.Remove(d.Name)
}

func RunDriver(driverName string) error {
	switch driverName {
	case "amazonec2":
		plugin.RegisterDriver(amazonec2.NewDriver("", ""))
	case "azure":
		plugin.RegisterDriver(azure.NewDriver("", ""))
	case "digitalocean":
		plugin.RegisterDriver(digitalocean.NewDriver("", ""))
	case "exoscale":
		plugin.RegisterDriver(exoscale.NewDriver("", ""))
	case "generic":
		plugin.RegisterDriver(generic.NewDriver("", ""))
	case "google":
		plugin.RegisterDriver(google.NewDriver("", ""))
	case "hyperv":
		plugin.RegisterDriver(hyperv.NewDriver("", ""))
	case "none":
		plugin.RegisterDriver(none.NewDriver("", ""))
	case "openstack":
		plugin.RegisterDriver(openstack.NewDriver("", ""))
	case "rackspace":
		plugin.RegisterDriver(rackspace.NewDriver("", ""))
	case "softlayer":
		plugin.RegisterDriver(softlayer.NewDriver("", ""))
	case "virtualbox":
		plugin.RegisterDriver(virtualbox.NewDriver("", ""))
	case "vmwarefusion":
		plugin.RegisterDriver(vmwarefusion.NewDriver("", ""))
	case "vmwarevcloudair":
		plugin.RegisterDriver(vmwarevcloudair.NewDriver("", ""))
	case "vmwarevsphere":
		plugin.RegisterDriver(vmwarevsphere.NewDriver("", ""))
	default:
		return fmt.Errorf("Unsupported driver: %s\n", driverName)
	}
	return nil
}
