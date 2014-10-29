// Copyright 2014 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec/testing"
	"launchpad.net/gocheck"
)

var deprecates gocheck.Checker = deprecationChecker{}

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
	c.Assert(create, gocheck.FitsTypeOf, &appCreate{})
}

func (s *S) TestAppRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	remove, ok := manager.Commands["app-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(remove, gocheck.FitsTypeOf, &appRemove{})
}

func (s *S) TestAppListIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	list, ok := manager.Commands["app-list"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(list, gocheck.FitsTypeOf, appList{})
}

func (s *S) TestAppGrantIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	grant, ok := manager.Commands["app-grant"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(grant, gocheck.FitsTypeOf, &appGrant{})
}

func (s *S) TestAppRevokeIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	grant, ok := manager.Commands["app-revoke"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(grant, gocheck.FitsTypeOf, &appRevoke{})
}

func (s *S) TestAppLogIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	log, ok := manager.Commands["app-log"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(log, gocheck.FitsTypeOf, &appLog{})
}

func (s *S) TestLogIsDeprecated(c *gocheck.C) {
	c.Assert("app-log", deprecates, "log")
}

func (s *S) TestAppRunIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	run, ok := manager.Commands["app-run"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(run, gocheck.FitsTypeOf, &appRun{})
}

func (s *S) TestRunIsDeprecated(c *gocheck.C) {
	c.Assert("app-run", deprecates, "run")
}

func (s *S) TestAppRestartIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	restart, ok := manager.Commands["app-restart"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(restart, gocheck.FitsTypeOf, &appRestart{})
}

func (s *S) TestRestartIsDeprecated(c *gocheck.C) {
	c.Assert("app-restart", deprecates, "restart")
}

func (s *S) TestEnvGetIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	get, ok := manager.Commands["env-get"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(get, gocheck.FitsTypeOf, &envGet{})
}

func (s *S) TestEnvSetIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	set, ok := manager.Commands["env-set"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(set, gocheck.FitsTypeOf, &envSet{})
}

func (s *S) TestEnvUnsetIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	unset, ok := manager.Commands["env-unset"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(unset, gocheck.FitsTypeOf, &envUnset{})
}

func (s *S) TestKeyAddIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	add, ok := manager.Commands["key-add"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(add, gocheck.FitsTypeOf, &keyAdd{})
}

func (s *S) TestKeyRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	remove, ok := manager.Commands["key-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(remove, gocheck.FitsTypeOf, &keyRemove{})
}

func (s *S) TestKeyListIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	list, ok := manager.Commands["key-list"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(list, gocheck.FitsTypeOf, &keyList{})
}

func (s *S) TestServiceListIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	list, ok := manager.Commands["service-list"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(list, gocheck.FitsTypeOf, serviceList{})
}

func (s *S) TestServiceAddIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	add, ok := manager.Commands["service-add"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(add, gocheck.FitsTypeOf, &serviceAdd{})
}

func (s *S) TestServiceRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	remove, ok := manager.Commands["service-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(remove, gocheck.FitsTypeOf, &serviceRemove{})
}

func (s *S) TestServiceBindIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	bind, ok := manager.Commands["service-bind"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(bind, gocheck.FitsTypeOf, &serviceBind{})
}

func (s *S) TestBindIsDeprecated(c *gocheck.C) {
	c.Assert("service-bind", deprecates, "bind")
}

func (s *S) TestServiceUnbindIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	unbind, ok := manager.Commands["service-unbind"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(unbind, gocheck.FitsTypeOf, &serviceUnbind{})
}

func (s *S) TestUnbindIsDeprecated(c *gocheck.C) {
	c.Assert("service-unbind", deprecates, "unbind")
}

func (s *S) TestServiceDocIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	doc, ok := manager.Commands["service-doc"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(doc, gocheck.FitsTypeOf, serviceDoc{})
}

func (s *S) TestServiceInfoIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	info, ok := manager.Commands["service-info"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(info, gocheck.FitsTypeOf, serviceInfo{})
}

func (s *S) TestServiceInstanceStatusIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	status, ok := manager.Commands["service-status"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(status, gocheck.FitsTypeOf, serviceInstanceStatus{})
}

func (s *S) TestAppInfoIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	list, ok := manager.Commands["app-info"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(list, gocheck.FitsTypeOf, &appInfo{})
}

func (s *S) TestUnitAddIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	addunit, ok := manager.Commands["unit-add"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(addunit, gocheck.FitsTypeOf, &unitAdd{})
}

func (s *S) TestUnitRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	rmunit, ok := manager.Commands["unit-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(rmunit, gocheck.FitsTypeOf, &unitRemove{})
}

func (s *S) TestCNameAddIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	cname, ok := manager.Commands["cname-add"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(cname, gocheck.FitsTypeOf, &cnameAdd{})
}

func (s *S) TestAddCnameIsDeprecated(c *gocheck.C) {
	c.Assert("cname-add", deprecates, "add-cname")
}

func (s *S) TestCNameRemoveIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	cname, ok := manager.Commands["cname-remove"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(cname, gocheck.FitsTypeOf, &cnameRemove{})
}

func (s *S) TestRemoveCNameIsDeprecated(c *gocheck.C) {
	c.Assert("cname-remove", deprecates, "remove-cname")
}

func (s *S) TestPlatformListIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	plat, ok := manager.Commands["platform-list"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(plat, gocheck.FitsTypeOf, platformList{})
}

func (s *S) TestAppSwapIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	cmd, ok := manager.Commands["app-swap"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(cmd, gocheck.FitsTypeOf, &appSwap{})
}

func (s *S) TestSwapIsDeprecated(c *gocheck.C) {
	c.Assert("app-swap", deprecates, "swap")
}

func (s *S) TestAppStartIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	start, ok := manager.Commands["app-start"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(start, gocheck.FitsTypeOf, &appStart{})
}

func (s *S) TestStartIsDeprecated(c *gocheck.C) {
	c.Assert("app-start", deprecates, "start")
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
	fexec := testing.FakeExecutor{}
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
	stop, ok := manager.Commands["app-stop"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(stop, gocheck.FitsTypeOf, &appStop{})
}

func (s *S) TestStopIsDeprecated(c *gocheck.C) {
	c.Assert("app-stop", deprecates, "stop")
}

func (s *S) TestAppDeployIsRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	deployCmd, ok := manager.Commands["app-deploy"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(deployCmd, gocheck.FitsTypeOf, &appDeploy{})
}

func (s *S) TestDeployIsDeprecated(c *gocheck.C) {
	c.Assert("app-deploy", deprecates, "deploy")
}

func (s *S) TestPlanListRegistered(c *gocheck.C) {
	manager := buildManager("tsuru")
	list, ok := manager.Commands["plan-list"]
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(list, gocheck.FitsTypeOf, &planList{})
}
