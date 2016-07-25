// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"encoding/json"
	"fmt"
	"os/exec"

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
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/docker/machine/libmachine/drivers/rpc"
	"github.com/docker/machine/libmachine/host"
	"github.com/fsouza/go-dockerclient"
)

var (
	dockerHTTPPort  = 2375
	dockerHTTPSPort = 2376
)

type Machine struct {
	IP      string
	Config  map[string]string
	TLS     bool
	Address string
}

func (m *Machine) dockerClient() (*docker.Client, error) {
	if m.TLS {
		return docker.NewTLSClient(m.Address, "cert.pem", "key.pem", "ca.pem")
	}
	return docker.NewClient(m.Address)
}

type DockerMachine struct {
	driverOpts map[string]interface{}
	rawDriver  []byte
	driverName string
	storePath  string
	certsPath  string
	tlsSupport bool
	Name       string
}

type DockerMachineConfig struct {
	DriverName string
	DriverOpts map[string]interface{}
}

func NewDockerMachine(config *DockerMachineConfig) (*DockerMachine, error) {
	storePath := "/tmp/automatic"
	certsPath := "/tmp/automatic/certs"
	rawDriver, err := json.Marshal(&drivers.BaseDriver{
		MachineName: "tsuru",
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
		tlsSupport: config.DriverName != "virtualbox" && config.DriverName != "none",
		Name:       "tsuru",
	}, nil
}

func (d *DockerMachine) CreateMachine() (*Machine, error) {
	client := libmachine.NewClient(d.storePath, d.certsPath)
	defer client.Close()
	host, err := client.NewHost(d.driverName, d.rawDriver)
	if err != nil {
		return nil, err
	}
	configureDriver(host.Driver, d.driverOpts)
	d.configureHost(host)
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
		TLS:    d.tlsSupport,
	}
	if m.TLS {
		m.Address = fmt.Sprintf("https://%s:%d", m.IP, dockerHTTPSPort)
	} else {
		m.Address = fmt.Sprintf("http://%s:%d", m.IP, dockerHTTPPort)
	}
	return &m, nil
}

func (d *DockerMachine) CreateRegistryCertificate() error {
	client := libmachine.NewClient(d.storePath, d.certsPath)
	defer client.Close()
	host, err := client.Load(d.Name)
	if err != nil {
		return err
	}
	ip, err := host.Driver.GetIP()
	if err != nil {
		return err
	}

	args := []string{
		"-o StrictHostKeyChecking=no",
		"-i",
		fmt.Sprintf("%s/machines/%s/id_rsa", d.storePath, d.Name),
		"-r",
		fmt.Sprintf("%s/", d.certsPath),
		fmt.Sprintf("%s@%s:/home/%s/", host.Driver.GetSSHUsername(), ip, host.Driver.GetSSHUsername()),
	}
	comm := exec.Command("scp", args...)
	stdout, err := comm.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Command: %s. Error:%s", string(stdout), err.Error())
	}
	registryCAConf := fmt.Sprintf(`
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
DNS.1 = %s.localdomain
DNS.2 = %s
IP.1 = %s
	`, d.Name, d.Name, ip)

	certsBasePath := fmt.Sprintf("/home/%s/certs/%s:5000", host.Driver.GetSSHUsername(), ip)
	_, err = host.RunSSHCommand(fmt.Sprintf("mkdir -p %s", certsBasePath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("cp /home/%s/certs/*.pem %s/", host.Driver.GetSSHUsername(), certsBasePath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("openssl genrsa -out %s/registry-key.pem 2048", certsBasePath))
	if err != nil {
		return err
	}
	registryCAConfPath := fmt.Sprintf("/home/%s/certs/registry_ca.conf", host.Driver.GetSSHUsername())
	_, err = host.RunSSHCommand(fmt.Sprintf("echo -e '%s' >> %s", registryCAConf, registryCAConfPath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("openssl req -new -key %s/registry-key.pem -out %s/registry.csr -subj '/CN=%s.localdomain' -config %s", certsBasePath, certsBasePath, d.Name, registryCAConfPath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("openssl x509 -req -in %s/registry.csr -CA '%s/ca.pem' -CAkey '%s/ca-key.pem' -CAcreateserial -out '%s/registry.pem' -days 365 -extensions v3_req -extfile %s", certsBasePath, certsBasePath, certsBasePath, certsBasePath, registryCAConfPath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("sudo mkdir /etc/docker/certs.d && sudo cp -r %s /etc/docker/certs.d/", certsBasePath))
	if err != nil {
		return err
	}
	_, err = host.RunSSHCommand(fmt.Sprintf("cat %s/ca.pem | sudo tee -a /etc/ssl/certs/ca-certificates.crt", certsBasePath))
	if err != nil {
		return err
	}
	return err
}

func configureDriver(driver drivers.Driver, driverOpts map[string]interface{}) error {
	opts := &rpcdriver.RPCFlags{Values: driverOpts}
	for _, c := range driver.GetCreateFlags() {
		_, ok := opts.Values[c.String()]
		if ok == false {
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

func (d *DockerMachine) configureHost(host *host.Host) {
	if d.tlsSupport == false {
		host.HostOptions.EngineOptions.Env = []string{"DOCKER_TLS=no"}
		host.HostOptions.EngineOptions.ArbitraryFlags = []string{fmt.Sprintf("host=tcp://0.0.0.0:%d", dockerHTTPPort)}
	}
}

func (d *DockerMachine) DeleteMachine(m *Machine) error {
	client := libmachine.NewClient(d.storePath, d.certsPath)
	defer client.Close()
	h, err := client.Load("tsuru")
	if err != nil {
		return err
	}
	err = h.Driver.Remove()
	if err != nil {
		return err
	}
	return client.Remove("tsuru")
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
