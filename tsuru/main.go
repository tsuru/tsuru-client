// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/tsuru/tsuru/cmd"
)

const (
	version = "0.16.0"
	header  = "Supported-Tsuru"
)

func buildManager(name string) *cmd.Manager {
	lookup := func(context *cmd.Context) error {
		command := plugin{}
		return command.Run(context, nil)
	}
	m := cmd.BuildBaseManager(name, version, header, lookup)
	m.Register(&appRun{})
	m.Register(&appInfo{})
	m.Register(&appCreate{})
	m.Register(&appRemove{})
	m.Register(&unitAdd{})
	m.Register(&unitRemove{})
	m.Register(appList{})
	m.Register(&appLog{})
	m.Register(&appGrant{})
	m.Register(&appRevoke{})
	m.Register(&appRestart{})
	m.Register(&appStart{})
	m.Register(&appStop{})
	m.Register(&appChangePool{})
	m.Register(&cnameAdd{})
	m.Register(&cnameRemove{})
	m.Register(&envGet{})
	m.Register(&envSet{})
	m.Register(&envUnset{})
	m.Register(&keyAdd{})
	m.Register(&keyRemove{})
	m.Register(&keyList{})
	m.Register(serviceList{})
	m.Register(&serviceAdd{})
	m.Register(&serviceRemove{})
	m.Register(serviceDoc{})
	m.Register(serviceInfo{})
	m.Register(serviceInstanceStatus{})
	m.Register(&serviceBind{})
	m.Register(&serviceUnbind{})
	m.Register(platformList{})
	m.Register(&pluginInstall{})
	m.Register(&pluginRemove{})
	m.Register(&pluginList{})
	m.Register(&appSwap{})
	m.Register(&appDeploy{})
	m.Register(&planList{})
	m.Register(&SetTeamOwner{})
	m.Register(&userCreate{})
	m.Register(&resetPassword{})
	m.Register(&userRemove{})
	m.Register(&teamCreate{})
	m.Register(&teamRemove{})
	m.Register(&teamList{})
	m.Register(&teamUserAdd{})
	m.Register(&teamUserRemove{})
	m.Register(teamUserList{})
	m.Register(&changePassword{})
	m.Register(&showAPIToken{})
	m.Register(&regenerateAPIToken{})
	m.Register(&appDeployList{})
	m.Register(&appDeployRollback{})
	m.Register(&cmd.ShellToContainerCmd{})
	m.Register(&poolList{})
	return m
}

func main() {
	name := cmd.ExtractProgramName(os.Args[0])
	manager := buildManager(name)
	manager.Run(os.Args[1:])
}
