// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/tsuru/tsuru/cmd"
	etesting "github.com/tsuru/tsuru/exec/testing"
	"launchpad.net/gocheck"
)

func (s *S) TestCommandsFromBaseManagerAreRegistered(c *gocheck.C) {
	baseManager := cmd.BuildBaseManager("tsuru", version, header, nil)
	manager := buildManager("tsuru")
	for name, instance := range baseManager.Commands {
		command, ok := manager.Commands[name]
		c.Assert(ok, gocheck.Equals, true)
		c.Assert(command, gocheck.FitsTypeOf, instance)
	}
}

func (s *S) TestAppCreateIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	create, ok := manager.Commands["app-create"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(create, gocheck.FitsTypeOf, &AppCreate{})
}

func (s *S) TestAppRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	remove, ok := manager.Commands["app-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(remove, gocheck.FitsTypeOf, &AppRemove{})
}

func (s *S) TestAppListIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	list, ok := manager.Commands["app-list"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(list, gocheck.FitsTypeOf, AppList{})
}

func (s *S) TestAppGrantIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	grant, ok := manager.Commands["app-grant"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(grant, gocheck.FitsTypeOf, &AppGrant{})
}

func (s *S) TestAppRevokeIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	grant, ok := manager.Commands["app-revoke"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(grant, gocheck.FitsTypeOf, &AppRevoke{})
}

func (s *S) TestAppLogIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	log, ok := manager.Commands["log"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(log, gocheck.FitsTypeOf, &AppLog{})
}

func (s *S) TestAppRunIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	run, ok := manager.Commands["run"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(run, gocheck.FitsTypeOf, &AppRun{})
}

func (s *S) TestAppRestartIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	restart, ok := manager.Commands["restart"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(restart, gocheck.FitsTypeOf, &AppRestart{})
}

func (s *S) TestEnvGetIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	get, ok := manager.Commands["env-get"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(get, gocheck.FitsTypeOf, &EnvGet{})
}

func (s *S) TestEnvSetIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	set, ok := manager.Commands["env-set"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(set, gocheck.FitsTypeOf, &EnvSet{})
}

func (s *S) TestEnvUnsetIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	unset, ok := manager.Commands["env-unset"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(unset, gocheck.FitsTypeOf, &EnvUnset{})
}

func (s *S) TestKeyAddIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	add, ok := manager.Commands["key-add"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(add, gocheck.FitsTypeOf, &KeyAdd{})
}

func (s *S) TestKeyRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	remove, ok := manager.Commands["key-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(remove, gocheck.FitsTypeOf, &KeyRemove{})
}

func (s *S) TestServiceListIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	list, ok := manager.Commands["service-list"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(list, gocheck.FitsTypeOf, ServiceList{})
}

func (s *S) TestServiceAddIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	add, ok := manager.Commands["service-add"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(add, gocheck.FitsTypeOf, &ServiceAdd{})
}

func (s *S) TestServiceRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	remove, ok := manager.Commands["service-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(remove, gocheck.FitsTypeOf, &ServiceRemove{})
}

func (s *S) TestServiceBindIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	bind, ok := manager.Commands["bind"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(bind, gocheck.FitsTypeOf, &ServiceBind{})
}

func (s *S) TestServiceUnbindIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	unbind, ok := manager.Commands["unbind"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(unbind, gocheck.FitsTypeOf, &ServiceUnbind{})
}

func (s *S) TestServiceDocIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	doc, ok := manager.Commands["service-doc"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(doc, gocheck.FitsTypeOf, ServiceDoc{})
}

func (s *S) TestServiceInfoIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	info, ok := manager.Commands["service-info"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(info, gocheck.FitsTypeOf, ServiceInfo{})
}

func (s *S) TestServiceInstanceStatusIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	status, ok := manager.Commands["service-status"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(status, gocheck.FitsTypeOf, ServiceInstanceStatus{})
}

func (s *S) TestAppInfoIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	list, ok := manager.Commands["app-info"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(list, gocheck.FitsTypeOf, &AppInfo{})
}

func (s *S) TestUnitAddIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	addunit, ok := manager.Commands["unit-add"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(addunit, gocheck.FitsTypeOf, &UnitAdd{})
}

func (s *S) TestUnitRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	rmunit, ok := manager.Commands["unit-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(rmunit, gocheck.FitsTypeOf, &UnitRemove{})
}

func (s *S) TestAddCNameIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	cname, ok := manager.Commands["add-cname"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(cname, gocheck.FitsTypeOf, &AddCName{})
}

func (s *S) TestRemoveCNameIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	cname, ok := manager.Commands["remove-cname"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(cname, gocheck.FitsTypeOf, &RemoveCName{})
}

func (s *S) TestPlatformListIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	plat, ok := manager.Commands["platform-list"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(plat, gocheck.FitsTypeOf, platformList{})
}

func (s *S) TestSwapIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	cmd, ok := manager.Commands["swap"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(cmd, gocheck.FitsTypeOf, &Swap{})
}

func (s *S) TestAppStartIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	start, ok := manager.Commands["start"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(start, gocheck.FitsTypeOf, &AppStart{})
}

func (s *S) TestPluginInstallIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	command, ok := manager.Commands["plugin-install"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(command, gocheck.FitsTypeOf, &pluginInstall{})
}

func (s *S) TestPluginRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	command, ok := manager.Commands["plugin-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(command, gocheck.FitsTypeOf, &pluginRemove{})
}

func (s *S) TestPluginListIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	command, ok := manager.Commands["plugin-list"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(command, gocheck.FitsTypeOf, &pluginList{})
}

func (s *S) TestPluginLookup(c *gocheck.C) {
	fexec := etesting.FakeExecutor{}
	execut = &fexec
	defer func() {
		execut = nil
	}()
	manager := buildManager("tsuru")
	manager.Run([]string{"myplugin"})
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{}), gocheck.Equals, true)
}

func (s *S) TestAppStopIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	stop, ok := manager.Commands["stop"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(stop, gocheck.FitsTypeOf, &AppStop{})
}

func (s *S) TestAppDeployIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	deployCmd, ok := manager.Commands["app-deploy"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(deployCmd, gocheck.FitsTypeOf, &deploy{})
}

func (s *S) TestDeployIsDeprecated(c *gocheck.C) {
	manager := buildManager("tsuru")
	original := manager.Commands["app-deploy"]
	deprecated, ok := manager.Commands["deploy"]
	c.Assert(ok, gocheck.Equals, true)
	command, ok := deprecated.(*cmd.DeprecatedCommand)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(command.Command, gocheck.Equals, original)
}
