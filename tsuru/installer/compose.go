package installer

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/installer/defaultconfig"
	"github.com/tsuru/tsuru-client/tsuru/installer/dm"
)

func resolveConfig(baseConfig string, customConfigs map[string]string) (string, error) {
	if baseConfig == "" {
		baseConfig = defaultconfig.Compose
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
	manager := c.GetManager()
	configs := map[string]string{
		"CLUSTER_ADDR":         manager.Base.Address,
		"CLUSTER_PRIVATE_ADDR": dm.GetPrivateIP(manager),
		"TSURU_API_IMAGE":      installConfig.ComponentsConfig.TsuruAPIImage,
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
