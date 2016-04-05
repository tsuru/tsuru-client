// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"

	"gopkg.in/check.v1"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec/exectest"
)

var Deprecates check.Checker = deprecationChecker{}

func (s *S) TestCommandsFromBaseManagerAreRegistered(c *check.C) {
	baseManager := cmd.BuildBaseManager("tsuru", version, header, nil)
	manager = buildManager("tsuru")
	for name, instance := range baseManager.Commands {
		command, ok := manager.Commands[name]
		c.Assert(ok, check.Equals, true)
		c.Assert(command, check.FitsTypeOf, instance)
	}
}

func (s *S) TestAppCreateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	create, ok := manager.Commands["app-create"]
	c.Assert(ok, check.Equals, true)
	c.Assert(create, check.FitsTypeOf, &appCreate{})
}

func (s *S) TestAppRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	remove, ok := manager.Commands["app-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(remove, check.FitsTypeOf, &appRemove{})
}

func (s *S) TestAppListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["app-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &appList{})
}

func (s *S) TestAppGrantIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	grant, ok := manager.Commands["app-grant"]
	c.Assert(ok, check.Equals, true)
	c.Assert(grant, check.FitsTypeOf, &appGrant{})
}

func (s *S) TestAppRevokeIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	grant, ok := manager.Commands["app-revoke"]
	c.Assert(ok, check.Equals, true)
	c.Assert(grant, check.FitsTypeOf, &appRevoke{})
}

func (s *S) TestAppLogIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	log, ok := manager.Commands["app-log"]
	c.Assert(ok, check.Equals, true)
	c.Assert(log, check.FitsTypeOf, &appLog{})
}

func (s *S) TestAppRunIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	run, ok := manager.Commands["app-run"]
	c.Assert(ok, check.Equals, true)
	c.Assert(run, check.FitsTypeOf, &appRun{})
}

func (s *S) TestAppRestartIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	restart, ok := manager.Commands["app-restart"]
	c.Assert(ok, check.Equals, true)
	c.Assert(restart, check.FitsTypeOf, &appRestart{})
}

func (s *S) TestEnvGetIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	get, ok := manager.Commands["env-get"]
	c.Assert(ok, check.Equals, true)
	c.Assert(get, check.FitsTypeOf, &envGet{})
}

func (s *S) TestEnvSetIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	set, ok := manager.Commands["env-set"]
	c.Assert(ok, check.Equals, true)
	c.Assert(set, check.FitsTypeOf, &envSet{})
}

func (s *S) TestEnvUnsetIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	unset, ok := manager.Commands["env-unset"]
	c.Assert(ok, check.Equals, true)
	c.Assert(unset, check.FitsTypeOf, &envUnset{})
}

func (s *S) TestKeyAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	add, ok := manager.Commands["key-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(add, check.FitsTypeOf, &keyAdd{})
}

func (s *S) TestKeyRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	remove, ok := manager.Commands["key-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(remove, check.FitsTypeOf, &keyRemove{})
}

func (s *S) TestKeyListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["key-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &keyList{})
}

func (s *S) TestServiceListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["service-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, serviceList{})
}

func (s *S) TestServiceUpdateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	add, ok := manager.Commands["service-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(add, check.FitsTypeOf, &cmd.RemovedCommand{})
}

func (s *S) TestServiceInstanceUpdateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	add, ok := manager.Commands["service-instance-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(add, check.FitsTypeOf, &serviceInstanceUpdate{})
}

func (s *S) TestServiceAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	add, ok := manager.Commands["service-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(add, check.FitsTypeOf, &cmd.RemovedCommand{})
}

func (s *S) TestServiceInstanceAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	add, ok := manager.Commands["service-instance-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(add, check.FitsTypeOf, &serviceInstanceAdd{})
}

func (s *S) TestServiceRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	remove, ok := manager.Commands["service-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(remove, check.FitsTypeOf, &cmd.RemovedCommand{})
}

func (s *S) TestServiceInstanceRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	remove, ok := manager.Commands["service-instance-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(remove, check.FitsTypeOf, &serviceInstanceRemove{})
}

func (s *S) TestServiceBindIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	bind, ok := manager.Commands["service-bind"]
	c.Assert(ok, check.Equals, true)
	c.Assert(bind, check.FitsTypeOf, &cmd.RemovedCommand{})
}

func (s *S) TestServiceInstanceBindIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	bind, ok := manager.Commands["service-instance-bind"]
	c.Assert(ok, check.Equals, true)
	c.Assert(bind, check.FitsTypeOf, &serviceInstanceBind{})
}

func (s *S) TestServiceUnbindIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	unbind, ok := manager.Commands["service-unbind"]
	c.Assert(ok, check.Equals, true)
	c.Assert(unbind, check.FitsTypeOf, &cmd.RemovedCommand{})
}

func (s *S) TestServiceInstanceUnbindIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	unbind, ok := manager.Commands["service-instance-unbind"]
	c.Assert(ok, check.Equals, true)
	c.Assert(unbind, check.FitsTypeOf, &serviceInstanceUnbind{})
}

func (s *S) TestServiceInstanceInfoIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	info, ok := manager.Commands["service-instance-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(info, check.FitsTypeOf, serviceInstanceInfo{})
}

func (s *S) TestServiceInfoIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	info, ok := manager.Commands["service-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(info, check.FitsTypeOf, serviceInfo{})
}

func (s *S) TestServiceStatusIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	bind, ok := manager.Commands["service-status"]
	c.Assert(ok, check.Equals, true)
	c.Assert(bind, check.FitsTypeOf, &cmd.RemovedCommand{})
}

func (s *S) TestServiceInstanceStatusIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	status, ok := manager.Commands["service-instance-status"]
	c.Assert(ok, check.Equals, true)
	c.Assert(status, check.FitsTypeOf, serviceInstanceStatus{})
}

func (s *S) TestAppInfoIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["app-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &appInfo{})
}

func (s *S) TestUnitAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	addunit, ok := manager.Commands["unit-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(addunit, check.FitsTypeOf, &unitAdd{})
}

func (s *S) TestUnitRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	rmunit, ok := manager.Commands["unit-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(rmunit, check.FitsTypeOf, &unitRemove{})
}

func (s *S) TestCNameAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	cname, ok := manager.Commands["cname-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(cname, check.FitsTypeOf, &cnameAdd{})
}

func (s *S) TestCNameRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	cname, ok := manager.Commands["cname-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(cname, check.FitsTypeOf, &cnameRemove{})
}

func (s *S) TestPlatformListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	plat, ok := manager.Commands["platform-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(plat, check.FitsTypeOf, platformList{})
}

func (s *S) TestAppSwapIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	cmd, ok := manager.Commands["app-swap"]
	c.Assert(ok, check.Equals, true)
	c.Assert(cmd, check.FitsTypeOf, &appSwap{})
}

func (s *S) TestAppStartIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	start, ok := manager.Commands["app-start"]
	c.Assert(ok, check.Equals, true)
	c.Assert(start, check.FitsTypeOf, &appStart{})
}

func (s *S) TestPluginInstallIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	command, ok := manager.Commands["plugin-install"]
	c.Assert(ok, check.Equals, true)
	c.Assert(command, check.FitsTypeOf, &pluginInstall{})
}

func (s *S) TestPluginRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	command, ok := manager.Commands["plugin-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(command, check.FitsTypeOf, &pluginRemove{})
}

func (s *S) TestPluginListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	command, ok := manager.Commands["plugin-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(command, check.FitsTypeOf, &pluginList{})
}

func (s *S) TestPluginLookup(c *check.C) {
	// Kids, do not try this at $HOME
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempHome, _ := filepath.Abs("testdata")
	os.Setenv("HOME", tempHome)

	fexec := exectest.FakeExecutor{}
	execut = &fexec
	defer func() {
		execut = nil
	}()
	manager = buildManager("tsuru")
	manager.Run([]string{"myplugin"})
	pluginPath := cmd.JoinWithUserDir(".tsuru", "plugins", "myplugin")
	c.Assert(fexec.ExecutedCmd(pluginPath, []string{}), check.Equals, true)
}

func (s *S) TestAppStopIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	stop, ok := manager.Commands["app-stop"]
	c.Assert(ok, check.Equals, true)
	c.Assert(stop, check.FitsTypeOf, &appStop{})
}

func (s *S) TestAppDeployIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	deployCmd, ok := manager.Commands["app-deploy"]
	c.Assert(ok, check.Equals, true)
	c.Assert(deployCmd, check.FitsTypeOf, &appDeploy{})
}

func (s *S) TestPlanListRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["plan-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &planList{})
}

func (s *S) TestChangePasswordIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	chpass, ok := manager.Commands["change-password"]
	c.Assert(ok, check.Equals, true)
	c.Assert(chpass, check.FitsTypeOf, &changePassword{})
}

func (s *S) TestResetPasswordIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	reset, ok := manager.Commands["reset-password"]
	c.Assert(ok, check.Equals, true)
	c.Assert(reset, check.FitsTypeOf, &resetPassword{})
}

func (s *S) TestUserRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	rmUser, ok := manager.Commands["user-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(rmUser, check.FitsTypeOf, &userRemove{})
}

func (s *S) TestUserListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	rmUser, ok := manager.Commands["user-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(rmUser, check.FitsTypeOf, &listUsers{})
}

func (s *S) TestTeamRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	rmTeam, ok := manager.Commands["team-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(rmTeam, check.FitsTypeOf, &teamRemove{})
}

func (s *S) TestTeamAddUserIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	adduser, ok := manager.Commands["team-user-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(adduser, check.FitsTypeOf, &cmd.RemovedCommand{})
}

func (s *S) TestTeamRemoveUserIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	removeuser, ok := manager.Commands["team-user-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(removeuser, check.FitsTypeOf, &cmd.RemovedCommand{})
}

func (s *S) TestTeamUserListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	listuser, ok := manager.Commands["team-user-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(listuser, check.FitsTypeOf, &cmd.RemovedCommand{})
}

func (s *S) TestUserCreateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	user, ok := manager.Commands["user-create"]
	c.Assert(ok, check.Equals, true)
	c.Assert(user, check.FitsTypeOf, &userCreate{})
}

func (s *S) TestTeamCreateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	create, ok := manager.Commands["team-create"]
	c.Assert(ok, check.Equals, true)
	c.Assert(create, check.FitsTypeOf, &teamCreate{})
}

func (s *S) TestTeamListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["team-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &teamList{})
}

func (s *S) TestPoolListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["pool-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &poolList{})
}

func (s *S) TestAppUpdateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["app-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &appUpdate{})
}
