// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/check.v1"

	"github.com/tsuru/tsuru-client/tsuru/admin"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru-client/tsuru/installer"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/exec/exectest"
)

type S struct{}

func (s *S) SetUpSuite(c *check.C) {
	os.Setenv("TSURU_TARGET", "http://localhost:8080")
	os.Setenv("TSURU_TOKEN", "sometoken")
}

func (s *S) TearDownSuite(c *check.C) {
	os.Unsetenv("TSURU_TARGET")
	os.Unsetenv("TSURU_TOKEN")
}

var _ = check.Suite(&S{})
var manager *cmd.Manager

func Test(t *testing.T) { check.TestingT(t) }

func (s *S) SetUpTest(c *check.C) {
	var stdout, stderr bytes.Buffer
	manager = cmd.NewManager("glb", "1.0.0", "Supported-Tsuru", &stdout, &stderr, os.Stdin, nil)
}

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
	c.Assert(create, check.FitsTypeOf, &client.AppCreate{})
}

func (s *S) TestAppRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	remove, ok := manager.Commands["app-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(remove, check.FitsTypeOf, &client.AppRemove{})
}

func (s *S) TestAppListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["app-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &client.AppList{})
}

func (s *S) TestAppGrantIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	grant, ok := manager.Commands["app-grant"]
	c.Assert(ok, check.Equals, true)
	c.Assert(grant, check.FitsTypeOf, &client.AppGrant{})
}

func (s *S) TestAppRevokeIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	grant, ok := manager.Commands["app-revoke"]
	c.Assert(ok, check.Equals, true)
	c.Assert(grant, check.FitsTypeOf, &client.AppRevoke{})
}

func (s *S) TestAppLogIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	log, ok := manager.Commands["app-log"]
	c.Assert(ok, check.Equals, true)
	c.Assert(log, check.FitsTypeOf, &client.AppLog{})
}

func (s *S) TestAppRunIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	run, ok := manager.Commands["app-run"]
	c.Assert(ok, check.Equals, true)
	c.Assert(run, check.FitsTypeOf, &client.AppRun{})
}

func (s *S) TestAppRestartIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	restart, ok := manager.Commands["app-restart"]
	c.Assert(ok, check.Equals, true)
	c.Assert(restart, check.FitsTypeOf, &client.AppRestart{})
}

func (s *S) TestEnvGetIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	get, ok := manager.Commands["env-get"]
	c.Assert(ok, check.Equals, true)
	c.Assert(get, check.FitsTypeOf, &client.EnvGet{})
}

func (s *S) TestEnvSetIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	set, ok := manager.Commands["env-set"]
	c.Assert(ok, check.Equals, true)
	c.Assert(set, check.FitsTypeOf, &client.EnvSet{})
}

func (s *S) TestEnvUnsetIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	unset, ok := manager.Commands["env-unset"]
	c.Assert(ok, check.Equals, true)
	c.Assert(unset, check.FitsTypeOf, &client.EnvUnset{})
}

func (s *S) TestKeyAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	add, ok := manager.Commands["key-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(add, check.FitsTypeOf, &client.KeyAdd{})
}

func (s *S) TestKeyRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	remove, ok := manager.Commands["key-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(remove, check.FitsTypeOf, &client.KeyRemove{})
}

func (s *S) TestKeyListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["key-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &client.KeyList{})
}

func (s *S) TestServiceListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["service-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, client.ServiceList{})
}

func (s *S) TestServiceUpdateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	add, ok := manager.Commands["service-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(add, check.FitsTypeOf, &admin.ServiceUpdate{})
}

func (s *S) TestServiceInstanceUpdateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	add, ok := manager.Commands["service-instance-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(add, check.FitsTypeOf, &client.ServiceInstanceUpdate{})
}

func (s *S) TestServiceInstanceAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	add, ok := manager.Commands["service-instance-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(add, check.FitsTypeOf, &client.ServiceInstanceAdd{})
}

func (s *S) TestServiceInstanceRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	remove, ok := manager.Commands["service-instance-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(remove, check.FitsTypeOf, &client.ServiceInstanceRemove{})
}

func (s *S) TestServiceInstanceBindIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	bind, ok := manager.Commands["service-instance-bind"]
	c.Assert(ok, check.Equals, true)
	c.Assert(bind, check.FitsTypeOf, &client.ServiceInstanceBind{})
}

func (s *S) TestServiceInstanceUnbindIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	unbind, ok := manager.Commands["service-instance-unbind"]
	c.Assert(ok, check.Equals, true)
	c.Assert(unbind, check.FitsTypeOf, &client.ServiceInstanceUnbind{})
}

func (s *S) TestServiceInstanceInfoIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	info, ok := manager.Commands["service-instance-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(info, check.FitsTypeOf, client.ServiceInstanceInfo{})
}

func (s *S) TestServiceInfoIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	info, ok := manager.Commands["service-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(info, check.FitsTypeOf, client.ServiceInfo{})
}

func (s *S) TestServiceInstanceStatusIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	status, ok := manager.Commands["service-instance-status"]
	c.Assert(ok, check.Equals, true)
	c.Assert(status, check.FitsTypeOf, client.ServiceInstanceStatus{})
}

func (s *S) TestAppInfoIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["app-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &client.AppInfo{})
}

func (s *S) TestUnitAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	addunit, ok := manager.Commands["unit-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(addunit, check.FitsTypeOf, &client.UnitAdd{})
}

func (s *S) TestUnitRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	rmunit, ok := manager.Commands["unit-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(rmunit, check.FitsTypeOf, &client.UnitRemove{})
}

func (s *S) TestCNameAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	cname, ok := manager.Commands["cname-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(cname, check.FitsTypeOf, &client.CnameAdd{})
}

func (s *S) TestCNameRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	cname, ok := manager.Commands["cname-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(cname, check.FitsTypeOf, &client.CnameRemove{})
}

func (s *S) TestPlatformListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	plat, ok := manager.Commands["platform-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(plat, check.FitsTypeOf, &admin.PlatformList{})
}
func (s *S) TestPlatformAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	plat, ok := manager.Commands["platform-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(plat, check.FitsTypeOf, &admin.PlatformAdd{})
}

func (s *S) TestAppSwapIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	cmd, ok := manager.Commands["app-swap"]
	c.Assert(ok, check.Equals, true)
	c.Assert(cmd, check.FitsTypeOf, &client.AppSwap{})
}

func (s *S) TestAppStartIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	start, ok := manager.Commands["app-start"]
	c.Assert(ok, check.Equals, true)
	c.Assert(start, check.FitsTypeOf, &client.AppStart{})
}

func (s *S) TestPluginInstallIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	command, ok := manager.Commands["plugin-install"]
	c.Assert(ok, check.Equals, true)
	c.Assert(command, check.FitsTypeOf, &client.PluginInstall{})
}

func (s *S) TestPluginRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	command, ok := manager.Commands["plugin-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(command, check.FitsTypeOf, &client.PluginRemove{})
}

func (s *S) TestPluginListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	command, ok := manager.Commands["plugin-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(command, check.FitsTypeOf, &client.PluginList{})
}

func (s *S) TestPluginLookup(c *check.C) {
	// Kids, do not try this at $HOME
	defer os.Setenv("HOME", os.Getenv("HOME"))
	tempHome, _ := filepath.Abs("client/testdata")
	os.Setenv("HOME", tempHome)

	fexec := exectest.FakeExecutor{}
	client.Execut = &fexec
	defer func() {
		client.Execut = nil
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
	c.Assert(stop, check.FitsTypeOf, &client.AppStop{})
}

func (s *S) TestAppDeployIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	deployCmd, ok := manager.Commands["app-deploy"]
	c.Assert(ok, check.Equals, true)
	c.Assert(deployCmd, check.FitsTypeOf, &client.AppDeploy{})
}

func (s *S) TestPlanListRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["plan-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &client.PlanList{})
}

func (s *S) TestChangePasswordIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	chpass, ok := manager.Commands["change-password"]
	c.Assert(ok, check.Equals, true)
	c.Assert(chpass, check.FitsTypeOf, &client.ChangePassword{})
}

func (s *S) TestResetPasswordIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	reset, ok := manager.Commands["reset-password"]
	c.Assert(ok, check.Equals, true)
	c.Assert(reset, check.FitsTypeOf, &client.ResetPassword{})
}

func (s *S) TestUserRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	rmUser, ok := manager.Commands["user-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(rmUser, check.FitsTypeOf, &client.UserRemove{})
}

func (s *S) TestUserListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	rmUser, ok := manager.Commands["user-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(rmUser, check.FitsTypeOf, &client.ListUsers{})
}

func (s *S) TestTeamRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	rmTeam, ok := manager.Commands["team-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(rmTeam, check.FitsTypeOf, &client.TeamRemove{})
}

func (s *S) TestUserCreateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	user, ok := manager.Commands["user-create"]
	c.Assert(ok, check.Equals, true)
	c.Assert(user, check.FitsTypeOf, &client.UserCreate{})
}

func (s *S) TestTeamCreateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	create, ok := manager.Commands["team-create"]
	c.Assert(ok, check.Equals, true)
	c.Assert(create, check.FitsTypeOf, &client.TeamCreate{})
}

func (s *S) TestTeamListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["team-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &client.TeamList{})
}

func (s *S) TestPoolListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["pool-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &client.PoolList{})
}

func (s *S) TestAppUpdateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["app-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &client.AppUpdate{})
}

func (s *S) TestInstallIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["install"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &installer.Install{})
}

func (s *S) TestUninstallIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["uninstall"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &installer.Uninstall{})
}

func (s *S) TestNodeAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["node-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &admin.AddNodeCmd{})
	change, ok = manager.Commands["docker-node-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &cmd.DeprecatedCommand{})
}

func (s *S) TestNodeRemoveIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["node-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &admin.RemoveNodeCmd{})
	change, ok = manager.Commands["docker-node-remove"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &cmd.DeprecatedCommand{})
}

func (s *S) TestNodeUpdateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["node-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &admin.UpdateNodeCmd{})
	change, ok = manager.Commands["docker-node-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &cmd.DeprecatedCommand{})
}

func (s *S) TestNodeListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["node-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &admin.ListNodesCmd{})
	change, ok = manager.Commands["docker-node-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &cmd.DeprecatedCommand{})
}

func (s *S) TestNodeHealingInfoIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["node-healing-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &admin.GetNodeHealingConfigCmd{})
	change, ok = manager.Commands["docker-healing-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &cmd.DeprecatedCommand{})
}

func (s *S) TestNodeHealingUpdateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["node-healing-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &admin.SetNodeHealingConfigCmd{})
	change, ok = manager.Commands["docker-healing-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &cmd.DeprecatedCommand{})
}

func (s *S) TestNodeHealingDeleteIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["node-healing-delete"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &admin.DeleteNodeHealingConfigCmd{})
	change, ok = manager.Commands["docker-healing-delete"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &cmd.DeprecatedCommand{})
}

func (s *S) TestNodeRebalanceIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	change, ok := manager.Commands["node-rebalance"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &admin.RebalanceNodeCmd{})
	change, ok = manager.Commands["containers-rebalance"]
	c.Assert(ok, check.Equals, true)
	c.Assert(change, check.FitsTypeOf, &cmd.DeprecatedCommand{})
}

func (s *S) TestServiceCreateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["service-create"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &admin.ServiceCreate{})
}

func (s *S) TestServiceDestroyIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["service-destroy"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &admin.ServiceDestroy{})
}

func (s *S) TestServiceDocGetIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["service-doc-get"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &admin.ServiceDocGet{})
}

func (s *S) TestServiceDocAddIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["service-doc-add"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &admin.ServiceDocAdd{})
}

func (s *S) TestServiceTemplateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["service-template"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &admin.ServiceTemplate{})
}
