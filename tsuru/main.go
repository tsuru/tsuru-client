// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/tsuru-base"
)

const (
	version = "0.12-dev"
	header  = "Supported-Tsuru"
)

func buildManager(name string) *cmd.Manager {
	lookup := func(context *cmd.Context) error {
		command := plugin{}
		return command.Run(context, nil)
	}
	m := cmd.BuildBaseManager(name, version, header, lookup)
	m.Register(&tsuru.AppRun{})
	m.Register(&tsuru.AppInfo{})
	m.Register(&AppCreate{})
	m.Register(&AppRemove{})
	m.Register(&UnitAdd{})
	m.Register(&UnitRemove{})
	m.Register(tsuru.AppList{})
	m.Register(&tsuru.AppLog{})
	m.Register(&tsuru.AppGrant{})
	m.Register(&tsuru.AppRevoke{})
	m.Register(&tsuru.AppRestart{})
	m.Register(&tsuru.AppStart{})
	m.Register(&tsuru.AppStop{})
	m.Register(&tsuru.AddCName{})
	m.Register(&tsuru.RemoveCName{})
	m.Register(&tsuru.EnvGet{})
	m.Register(&tsuru.EnvSet{})
	m.Register(&tsuru.EnvUnset{})
	m.Register(&KeyAdd{})
	m.Register(&KeyRemove{})
	m.Register(tsuru.ServiceList{})
	m.Register(&tsuru.ServiceAdd{})
	m.Register(&tsuru.ServiceRemove{})
	m.Register(tsuru.ServiceDoc{})
	m.Register(tsuru.ServiceInfo{})
	m.Register(tsuru.ServiceInstanceStatus{})
	m.Register(&tsuru.ServiceBind{})
	m.Register(&tsuru.ServiceUnbind{})
	m.Register(platformList{})
	m.Register(&pluginInstall{})
	m.Register(&pluginRemove{})
	m.Register(&pluginList{})
	m.Register(&Swap{})
	m.Register(&deploy{})
	return m
}

func main() {
	name := cmd.ExtractProgramName(os.Args[0])
	manager := buildManager(name)
	manager.Run(os.Args[1:])
}
