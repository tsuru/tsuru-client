// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os"

	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
	"github.com/tsuru/tsuru-client/tsuru/admin"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru-client/tsuru/installer"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	_ "github.com/tsuru/tsuru/provision/docker/cmds"
)

const (
	version = "1.5.0-rc4"
	header  = "Supported-Tsuru"
)

func buildManager(name string) *cmd.Manager {
	lookup := func(context *cmd.Context) error {
		return client.RunPlugin(context)
	}
	m := cmd.BuildBaseManager(name, version, header, lookup)
	m.Register(&client.AppRun{})
	m.Register(&client.AppInfo{})
	m.Register(&client.AppCreate{})
	m.Register(&client.AppRemove{})
	m.Register(&client.AppUpdate{})
	m.Register(&client.UnitAdd{})
	m.Register(&client.UnitRemove{})
	m.Register(&client.AppList{})
	m.Register(&client.AppLog{})
	m.Register(&client.AppGrant{})
	m.Register(&client.AppRevoke{})
	m.Register(&client.AppRestart{})
	m.Register(&client.AppStart{})
	m.Register(&client.AppStop{})
	m.Register(&admin.AppLockDelete{})
	m.Register(&client.CertificateSet{})
	m.Register(&client.CertificateUnset{})
	m.Register(&client.CertificateList{})
	m.Register(&client.CnameAdd{})
	m.Register(&client.CnameRemove{})
	m.Register(&client.EnvGet{})
	m.Register(&client.EnvSet{})
	m.Register(&client.EnvUnset{})
	m.Register(&client.KeyAdd{})
	m.Register(&client.KeyRemove{})
	m.Register(&client.KeyList{})
	m.Register(client.ServiceList{})
	m.Register(&client.ServiceInstanceAdd{})
	m.Register(&client.ServiceInstanceUpdate{})
	m.Register(&client.ServiceInstanceRemove{})
	m.Register(client.ServiceInfo{})
	m.Register(client.ServiceInstanceInfo{})
	m.Register(client.ServiceInstanceStatus{})
	m.Register(&client.ServiceInstanceGrant{})
	m.Register(&client.ServiceInstanceRevoke{})
	m.Register(&client.ServiceInstanceBind{})
	m.Register(&client.ServiceInstanceUnbind{})
	m.Register(&admin.PlatformList{})
	m.Register(&admin.PlatformAdd{})
	m.Register(&admin.PlatformUpdate{})
	m.Register(&admin.PlatformRemove{})
	m.Register(&client.PluginInstall{})
	m.Register(&client.PluginRemove{})
	m.Register(&client.PluginList{})
	m.Register(&client.AppSwap{})
	m.Register(&client.AppDeploy{})
	m.Register(&client.AppBuild{})
	m.Register(&client.PlanList{})
	m.Register(&client.UserCreate{})
	m.Register(&client.ResetPassword{})
	m.Register(&client.UserRemove{})
	m.Register(&client.ListUsers{})
	m.Register(&client.TeamCreate{})
	m.Register(&client.TeamUpdate{})
	m.Register(&client.TeamRemove{})
	m.Register(&client.TeamList{})
	m.Register(&client.TeamInfo{})
	m.Register(&client.ChangePassword{})
	m.Register(&client.ShowAPIToken{})
	m.Register(&client.RegenerateAPIToken{})
	m.Register(&client.AppDeployList{})
	m.Register(&client.AppDeployRollback{})
	m.Register(&client.AppDeployRollbackUpdate{})
	m.Register(&client.AppDeployRebuild{})
	m.Register(&cmd.ShellToContainerCmd{})
	m.Register(&client.PoolList{})
	m.Register(&client.PermissionList{})
	m.Register(&client.RoleAdd{})
	m.Register(&client.RoleUpdate{})
	m.Register(&client.RoleRemove{})
	m.Register(&client.RoleList{})
	m.Register(&client.RoleInfo{})
	m.Register(&client.RolePermissionAdd{})
	m.Register(&client.RolePermissionRemove{})
	m.Register(&client.RoleAssign{})
	m.Register(&client.RoleDissociate{})
	m.Register(&client.RoleDefaultAdd{})
	m.Register(&client.RoleDefaultList{})
	m.Register(&client.RoleDefaultRemove{})
	m.Register(&installer.Install{})
	m.Register(&installer.Uninstall{})
	m.Register(&installer.InstallHostList{})
	m.Register(&installer.InstallSSH{})
	m.Register(&installer.InstallConfigInit{})
	m.Register(&admin.AddPoolToSchedulerCmd{})
	m.Register(&client.EventList{})
	m.Register(&client.EventInfo{})
	m.Register(&client.EventCancel{})
	m.Register(&client.RoutersList{})
	m.Register(&admin.TemplateList{})
	m.Register(&admin.TemplateAdd{})
	m.Register(&admin.TemplateRemove{})
	m.Register(&admin.MachineList{})
	m.Register(&admin.MachineDestroy{})
	m.Register(&admin.TemplateUpdate{})
	m.Register(&admin.PlanCreate{})
	m.Register(&admin.PlanRemove{})
	m.Register(&admin.UpdatePoolToSchedulerCmd{})
	m.Register(&admin.RemovePoolFromSchedulerCmd{})
	m.Register(&admin.ServiceCreate{})
	m.Register(&admin.ServiceDestroy{})
	m.Register(&admin.ServiceUpdate{})
	m.Register(&admin.ServiceDocGet{})
	m.Register(&admin.ServiceDocAdd{})
	m.Register(&admin.ServiceTemplate{})
	m.Register(&admin.UserQuotaView{})
	m.Register(&admin.UserChangeQuota{})
	m.Register(&admin.AppQuotaView{})
	m.Register(&admin.AppQuotaChange{})
	m.Register(&admin.AppRoutesRebuild{})
	m.Register(&admin.PoolConstraintList{})
	m.Register(&admin.PoolConstraintSet{})
	m.Register(&admin.EventBlockList{})
	m.Register(&admin.EventBlockAdd{})
	m.Register(&admin.EventBlockRemove{})
	m.Register(&client.TagList{})
	m.Register(&admin.NodeContainerList{})
	m.Register(&admin.NodeContainerAdd{})
	m.Register(&admin.NodeContainerInfo{})
	m.Register(&admin.NodeContainerUpdate{})
	m.Register(&admin.NodeContainerDelete{})
	m.Register(&admin.NodeContainerUpgrade{})
	m.Register(&admin.ClusterAdd{})
	m.Register(&admin.ClusterUpdate{})
	m.Register(&admin.ClusterRemove{})
	m.Register(&admin.ClusterList{})
	m.Register(&client.VolumeCreate{})
	m.Register(&client.VolumeUpdate{})
	m.Register(&client.VolumeList{})
	m.Register(&client.VolumePlansList{})
	m.Register(&client.VolumeDelete{})
	m.Register(&client.VolumeBind{})
	m.Register(&client.VolumeUnbind{})
	m.Register(&client.AppRoutersList{})
	m.Register(&client.AppRoutersAdd{})
	m.Register(&client.AppRoutersRemove{})
	m.Register(&client.AppRoutersUpdate{})
	m.Register(&admin.InfoNodeCmd{})
	m.RegisterRemoved("bs-env-set", "You should use `tsuru node-container-update big-sibling` instead.")
	m.RegisterRemoved("bs-info", "You should use `tsuru node-container-info big-sibling` instead.")
	m.RegisterRemoved("bs-upgrade", "You should use `tsuru node-container-upgrade big-sibling` instead.")
	m.RegisterDeprecated(&admin.AddTeamsToPoolCmd{}, "pool-teams-add")
	m.RegisterDeprecated(&admin.RemoveTeamsFromPoolCmd{}, "pool-teams-remove")
	m.RegisterDeprecated(&admin.AddNodeCmd{}, "docker-node-add")
	m.RegisterDeprecated(&admin.RemoveNodeCmd{}, "docker-node-remove")
	m.RegisterDeprecated(&admin.UpdateNodeCmd{}, "docker-node-update")
	m.RegisterDeprecated(&admin.ListNodesCmd{}, "docker-node-list")
	m.RegisterDeprecated(&admin.GetNodeHealingConfigCmd{}, "docker-healing-info")
	m.RegisterDeprecated(&admin.SetNodeHealingConfigCmd{}, "docker-healing-update")
	m.RegisterDeprecated(&admin.DeleteNodeHealingConfigCmd{}, "docker-healing-delete")
	m.RegisterDeprecated(&admin.RebalanceNodeCmd{}, "containers-rebalance")
	m.RegisterDeprecated(&admin.AutoScaleRunCmd{}, "docker-autoscale-run")
	m.RegisterDeprecated(&admin.ListAutoScaleHistoryCmd{}, "docker-autoscale-list")
	m.RegisterDeprecated(&admin.AutoScaleInfoCmd{}, "docker-autoscale-info")
	m.RegisterDeprecated(&admin.AutoScaleSetRuleCmd{}, "docker-autoscale-rule-set")
	m.RegisterDeprecated(&admin.AutoScaleDeleteRuleCmd{}, "docker-autoscale-rule-remove")
	m.RegisterDeprecated(&admin.ListHealingHistoryCmd{}, "docker-healing-list")
	registerExtraCommands(m)
	return m
}

func registerExtraCommands(m *cmd.Manager) {
	for _, c := range cmd.ExtraCmds() {
		m.Register(c)
	}
}

func inDockerMachineDriverMode() bool {
	return os.Getenv(localbinary.PluginEnvKey) == localbinary.PluginEnvVal
}

func main() {
	if inDockerMachineDriverMode() {
		err := dockermachine.RunDriver(os.Getenv(localbinary.PluginEnvDriverName))
		if err != nil {
			log.Fatalf("Error running driver: %s", err)
		}
	} else {
		localbinary.CurrentBinaryIsDockerMachine = true
		name := cmd.ExtractProgramName(os.Args[0])
		m := buildManager(name)
		m.Run(os.Args[1:])
	}
}
