// Copyright 2015 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/tsuru/tsuru/cmd"
)

const (
	version = "0.18.1"
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
	m.Register(&appUpdate{})
	m.Register(&unitAdd{})
	m.Register(&unitRemove{})
	m.Register(&appList{})
	m.Register(&appLog{})
	m.Register(&appGrant{})
	m.Register(&appRevoke{})
	m.Register(&appRestart{})
	m.Register(&appStart{})
	m.Register(&appStop{})
	m.RegisterRemoved("app-pool-change", "You should use `tsuru app-update` instead.")
	m.RegisterRemoved("app-plan-change", "You should use `tsuru app-update` instead.")
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
	m.Register(&serviceUpdate{})
	m.Register(&serviceRemove{})
	m.Register(serviceInfo{})
	m.Register(serviceInstanceInfo{})
	m.Register(serviceInstanceStatus{})
	m.Register(&serviceInstanceGrant{})
	m.Register(&serviceInstanceRevoke{})
	m.Register(&serviceBind{})
	m.Register(&serviceUnbind{})
	m.Register(platformList{})
	m.Register(&pluginInstall{})
	m.Register(&pluginRemove{})
	m.Register(&pluginList{})
	m.Register(&appSwap{})
	m.Register(&appDeploy{})
	m.Register(&planList{})
	m.RegisterRemoved("app-team-owner-set", "You should use `tsuru service-info` instead.")
	m.Register(&userCreate{})
	m.Register(&resetPassword{})
	m.Register(&userRemove{})
	m.Register(&listUsers{})
	m.Register(&teamCreate{})
	m.Register(&teamRemove{})
	m.Register(&teamList{})
	m.RegisterRemoved("service-doc", "You should use `tsuru service-info` instead.")
	m.RegisterRemoved("team-user-add", "You should use `tsuru role-assign` instead.")
	m.RegisterRemoved("team-user-remove", "You should use `tsuru role-dissociate` instead.")
	m.RegisterRemoved("team-user-list", "You should use `tsuru user-list` instead.")
	m.Register(&changePassword{})
	m.Register(&showAPIToken{})
	m.Register(&regenerateAPIToken{})
	m.Register(&appDeployList{})
	m.Register(&appDeployRollback{})
	m.Register(&cmd.ShellToContainerCmd{})
	m.Register(&poolList{})
	m.Register(&permissionList{})
	m.Register(&roleAdd{})
	m.Register(&roleRemove{})
	m.Register(&roleList{})
	m.Register(&roleInfo{})
	m.Register(&rolePermissionAdd{})
	m.Register(&rolePermissionRemove{})
	m.Register(&roleAssign{})
	m.Register(&roleDissociate{})
	m.Register(&roleDefaultAdd{})
	m.Register(&roleDefaultList{})
	m.Register(&roleDefaultRemove{})
	return m
}

func main() {
	name := cmd.ExtractProgramName(os.Args[0])
	manager := buildManager(name)
	manager.Run(os.Args[1:])
}
