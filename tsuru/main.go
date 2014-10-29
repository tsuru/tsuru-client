// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/tsuru/tsuru/cmd"
)

const (
	version = "0.13.2"
	header  = "Supported-Tsuru"
)

func buildManager(name string) *cmd.Manager {
	lookup := func(context *cmd.Context) error {
		command := plugin{}
		return command.Run(context, nil)
	}
	m := cmd.BuildBaseManager(name, version, header, lookup)
	m.RegisterDeprecated(&AppRun{}, "run")
	m.Register(&appInfo{})
	m.Register(&appCreate{})
	m.Register(&appRemove{})
	m.Register(&unitAdd{})
	m.Register(&unitRemove{})
	m.Register(appList{})
	m.RegisterDeprecated(&appLog{}, "log")
	m.Register(&appGrant{})
	m.Register(&appRevoke{})
	m.RegisterDeprecated(&appRestart{}, "restart")
	m.RegisterDeprecated(&AppStart{}, "start")
	m.RegisterDeprecated(&AppStop{}, "stop")
	m.RegisterDeprecated(&AddCName{}, "add-cname")
	m.RegisterDeprecated(&RemoveCName{}, "remove-cname")
	m.Register(&EnvGet{})
	m.Register(&EnvSet{})
	m.Register(&EnvUnset{})
	m.Register(&keyAdd{})
	m.Register(&keyRemove{})
	m.Register(&keyList{})
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
