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
	"github.com/tsuru/tsuru/provision"
	_ "github.com/tsuru/tsuru/provision/docker"
)

const (
	version = "1.2.0-dev"
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
	m.Register(&client.PlanList{})
	m.Register(&client.UserCreate{})
	m.Register(&client.ResetPassword{})
	m.Register(&client.UserRemove{})
	m.Register(&client.ListUsers{})
	m.Register(&client.TeamCreate{})
	m.Register(&client.TeamRemove{})
	m.Register(&client.TeamList{})
	m.Register(&client.ChangePassword{})
	m.Register(&client.ShowAPIToken{})
	m.Register(&client.RegenerateAPIToken{})
	m.Register(&client.AppDeployList{})
	m.Register(&client.AppDeployRollback{})
	m.Register(&cmd.ShellToContainerCmd{})
	m.Register(&client.PoolList{})
	m.Register(&client.PermissionList{})
	m.Register(&client.RoleAdd{})
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
	m.RegisterRemoved("pool-teams-add", "Use pool-constraint-set <pool> team <team_to_add> --append")
	m.RegisterRemoved("pool-teams-remove", "Use pool-constraint-set <pool> team <teams_remaining>")
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
	registerProvisionersCommands(m)
	return m
}

func registerProvisionersCommands(m *cmd.Manager) {
	provisioners, err := provision.Registry()
	if err != nil {
		log.Fatalf("Unable to list provisioners: %s", err)
	}
	for _, p := range provisioners {
		if c, ok := p.(cmd.AdminCommandable); ok {
			commands := c.AdminCommands()
			for _, cmd := range commands {
				m.Register(cmd)
			}
		}
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
