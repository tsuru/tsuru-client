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
    volumes:
      - redis-data:/data/db
  
  mongo:
    image: mongo:latest
    networks:
      - tsuru
    volumes:
      - mongo-data:/data

  planb:
    image: tsuru/planb:latest
    command: --listen :8080 --read-redis-host redis --write-redis-host redis
    ports:
      - 80:8080
    networks:
      - tsuru
    depends_on:
      - redis

  registry:
    image: registry:2
    environment:
      - "REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=/var/lib/registry"
      - "REGISTRY_HTTP_TLS_CERTIFICATE=/certs/{{CLUSTER_ADDR}}:5000/registry-cert.pem"
      - "REGISTRY_HTTP_TLS_KEY=/certs/{{CLUSTER_ADDR}}:5000/registry-key.pem"
    volumes:
      - "/var/lib/registry:/var/lib/registry"
      - "/etc/docker/certs.d:/certs:ro"
      - registry-data:/var/lib/registry
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
    depends_on:
      - redis
      - mongo
      - registry
      - planb
    environment:
      - MONGODB_ADDR=mongo
      - MONGODB_PORT=27017
      - REDIS_ADDR=redis
      - REDIS_PORT=6379
      - HIPACHE_DOMAIN={{CLUSTER_ADDR}}.nip.io
      - REGISTRY_ADDR={{CLUSTER_PRIVATE_ADDR}}
      - REGISTRY_PORT=5000
      - TSURU_ADDR=http://{{CLUSTER_ADDR}}
      - TSURU_PORT=8080
      - IAAS_CONF={{IAAS_CONF}}

networks:
  tsuru:
    driver: overlay
    ipam:
      driver: default
      config:
        - subnet: 10.0.9.0/24

volumes:
  mongo-data:
  redis-data:
  registry-data:
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
		"CLUSTER_ADDR":         manager.Base.Address,
		"CLUSTER_PRIVATE_ADDR": dm.GetPrivateIP(manager),
		"IAAS_CONF":            string(iaasConfig),
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
	output, err := manager.Host.RunSSHCommand("sudo docker deploy -c /tmp/compose.yml tsuru")
	if err != nil {
		return err
	}
	fmt.Printf(output)
	return nil
}
