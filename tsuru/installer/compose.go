package installer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
)

var defaultCompose = `
version: "3"

services:
  redis:
    image: redis:latest
    networks:
      - tsuru
  
  mongo:
    image: mongo:latest
    networks:
      - tsuru

  planb:
    image: tsuru/planb:latest
    command: --listen :8080 --read-redis-host redis --write-redis-host redis
    ports:
      - 80:8080
    networks:
      - tsuru

  registry:
    image: registry:2
    environment:
      - "REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=/var/lib/registry"
      - "REGISTRY_HTTP_TLS_CERTIFICATE=/certs/{{MANAGER_ADDR}}:5000/registry-cert.pem"
      - "REGISTRY_HTTP_TLS_KEY=/certs/{{MANAGER_ADDR}}:5000/registry-key.pem"
    volumes:
      - "/var/lib/registry:/var/lib/registry"
      - "/etc/docker/certs.d:/certs:ro"
    ports:
      - 5000:5000
    networks:
      - tsuru

  tsuru:
    image: tsuru/api:v1
    volumes:
      - "/etc/docker/certs.d:/certs:ro"
    ports:
      - 8080:8080
    networks:
      - tsuru
    environment:
      - MONGODB_ADDR=mongo
      - MONGODB_PORT=27017
      - REDIS_ADDR=redis
      - REDIS_PORT=6379
      - HIPACHE_DOMAIN={{MANAGER_ADDR}}.nip.io
      - REGISTRY_ADDR={{MANAGER_ADDR}}
      - REGISTRY_PORT=5000
      - TSURU_ADDR=http://{{MANAGER_ADDR}}
      - TSURU_PORT=8080
      - IAAS_CONF={{IAAS_CONF}}

networks:
  tsuru:
    driver: overlay
    ipam:
      driver: default
      config:
        - subnet: 10.0.9.0/24
`

func resolveConfig(baseConfig string, customConfigs map[string]string) (string, error) {
	if baseConfig == "" {
		baseConfig = defaultCompose
	} else {
		b, err := ioutil.ReadFile(baseConfig)
		if err != nil {
			return "", err
		}
		baseConfig = string(b)
	}
	for k, v := range customConfigs {
		if v != "" {
			baseConfig = strings.Replace(baseConfig, fmt.Sprintf("{{%s}}", k), v, -1)
		}
	}
	return baseConfig, nil
}

func composeDeploy(c ServiceCluster, installConfig *InstallOpts) error {
	componentsConfig := installConfig.ComponentsConfig
	manager := c.GetManager()
	componentsConfig.IaaSConfig.Dockermachine.InsecureRegistry = fmt.Sprintf("%s:5000", dm.GetPrivateIP(manager))
	iaasConfig, err := json.Marshal(componentsConfig.IaaSConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal iaas config: %s", err)
	}
	configs := map[string]string{
		"MANAGER_ADDR": manager.Base.Address,
		"IAAS_CONF":    string(iaasConfig),
	}
	config, err := resolveConfig(installConfig.ComposeFile, configs)
	if err != nil {
		return err
	}
	remoteWriteCmdFmt := "printf '%%s' '%s' | sudo tee %s"
	_, err = manager.Host.RunSSHCommand(fmt.Sprintf(remoteWriteCmdFmt, config, "/tmp/compose.yml"))
	if err != nil {
		return fmt.Errorf("failed to write remote file: %s", err)
	}
	fmt.Printf("Deploying compose file in cluster manager....\n")
	output, err := manager.Host.RunSSHCommand("docker deploy -c /tmp/compose.yml tsuru")
	if err != nil {
		return err
	}
	fmt.Printf(output)
	return nil
}
