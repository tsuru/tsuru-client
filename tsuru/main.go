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
	m.RegisterDeprecated(&appRun{}, "run")
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
	m.RegisterDeprecated(&appStart{}, "start")
	m.RegisterDeprecated(&appStop{}, "stop")
	m.RegisterDeprecated(&cnameAdd{}, "add-cname")
	m.RegisterDeprecated(&cnameRemove{}, "remove-cname")
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
	m.RegisterDeprecated(&serviceBind{}, "bind")
	m.RegisterDeprecated(&serviceUnbind{}, "unbind")
	m.Register(platformList{})
	m.Register(&pluginInstall{})
	m.Register(&pluginRemove{})
	m.Register(&pluginList{})
	m.RegisterDeprecated(&appSwap{}, "swap")
	m.RegisterDeprecated(&appDeploy{}, "deploy")
	m.Register(&planList{})
	m.Register(&SetTeamOwner{})
	m.Register(&autoScaleEnable{})
	m.Register(&autoScaleDisable{})
	return m
}

func main() {
	name := cmd.ExtractProgramName(os.Args[0])
	manager := buildManager(name)
	manager.Run(os.Args[1:])
}
