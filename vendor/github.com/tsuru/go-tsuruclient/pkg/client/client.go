package client

import (
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
)

var errUndefinedTarget = errors.New("undefined tsuru target")

func getHome() string {
	envs := []string{"HOME", "HOMEPATH"}
	var home string
	for i := 0; i < len(envs) && home == ""; i++ {
		home = os.Getenv(envs[i])
	}
	if home == "" {
		if u, err := user.Current(); err == nil && u.HomeDir != "" {
			home = u.HomeDir
		}
	}
	return home
}

func joinWithUserDir(p ...string) string {
	paths := []string{getHome()}
	paths = append(paths, p...)
	return filepath.Join(paths...)
}

func getTarget() (string, error) {
	if target := os.Getenv("TSURU_TARGET"); target != "" {
		return target, nil
	}
	targetPath := joinWithUserDir(".tsuru", "target")
	target, err := readTarget(targetPath)
	if err == errUndefinedTarget {
		target, err = readTarget(joinWithUserDir(".tsuru_target"))
	}
	return target, err
}

func readTarget(targetPath string) (string, error) {
	data, err := ioutil.ReadFile(targetPath)
	if err != nil {
		return "", errUndefinedTarget
	}
	return strings.TrimSpace(string(data)), nil
}

// GetTarget returns the current target, as defined in the TSURU_TARGET
// environment variable or in the target file.
func GetTarget() (string, error) {
	var prefix string
	target, err := getTarget()
	if err != nil {
		return "", err
	}
	if m, _ := regexp.MatchString("^https?://", target); !m {
		prefix = "http://"
	}
	return prefix + target, nil
}

func ReadToken() (string, error) {
	if token := os.Getenv("TSURU_TOKEN"); token != "" {
		return token, nil
	}
	tokenPath := joinWithUserDir(".tsuru", "token")
	token, err := ioutil.ReadFile(tokenPath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func ClientFromEnvironment(cfg *tsuru.Configuration) (*tsuru.APIClient, error) {
	if cfg == nil {
		cfg = &tsuru.Configuration{}
	}
	var err error
	if cfg.BasePath == "" {
		cfg.BasePath, err = GetTarget()
		if err != nil {
			return nil, err
		}
	}
	if cfg.DefaultHeader == nil {
		cfg.DefaultHeader = map[string]string{}
	}
	if _, authSet := cfg.DefaultHeader["Authorization"]; !authSet {
		if token, tokenErr := ReadToken(); tokenErr == nil && token != "" {
			cfg.DefaultHeader["Authorization"] = "bearer " + token
		}
	}
	cli := tsuru.NewAPIClient(cfg)
	return cli, nil
}
