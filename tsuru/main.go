// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/tsuru/tsuru/cmd"
)

const (
	version = "0.13.0"
	header  = "Supported-Tsuru"
)

func buildManager(name string) *cmd.Manager {
	lookup := func(context *cmd.Context) error {
		command := plugin{}
		return command.Run(context, nil)
	}
	m := cmd.BuildBaseManager(name, version, header, lookup)
	m.RegisterDeprecated(&AppRun{}, "run")
	m.Register(&AppInfo{})
	m.Register(&AppCreate{})
	m.Register(&AppRemove{})
	m.Register(&UnitAdd{})
	m.Register(&UnitRemove{})
	m.Register(AppList{})
	m.RegisterDeprecated(&AppLog{}, "log")
	m.Register(&AppGrant{})
	m.Register(&AppRevoke{})
	m.RegisterDeprecated(&AppRestart{}, "restart")
	m.RegisterDeprecated(&AppStart{}, "start")
	m.RegisterDeprecated(&AppStop{}, "stop")
	m.RegisterDeprecated(&AddCName{}, "add-cname")
	m.RegisterDeprecated(&RemoveCName{}, "remove-cname")
	m.Register(&EnvGet{})
	m.Register(&EnvSet{})
	m.Register(&EnvUnset{})
	m.Register(&KeyAdd{})
	m.Register(&KeyRemove{})
	m.Register(ServiceList{})
	m.Register(&ServiceAdd{})
	m.Register(&ServiceRemove{})
	m.Register(ServiceDoc{})
	m.Register(ServiceInfo{})
	m.Register(ServiceInstanceStatus{})
	m.RegisterDeprecated(&ServiceBind{}, "bind")
	m.RegisterDeprecated(&ServiceUnbind{}, "unbind")
	m.Register(platformList{})
	m.Register(&pluginInstall{})
	m.Register(&pluginRemove{})
	m.Register(&pluginList{})
	m.RegisterDeprecated(&Swap{}, "swap")
	m.RegisterDeprecated(&deploy{}, "deploy")
	m.Register(&PlanList{})
	return m
}

func main() {
	name := cmd.ExtractProgramName(os.Args[0])
	manager := buildManager(name)
	manager.Run(os.Args[1:])
}
