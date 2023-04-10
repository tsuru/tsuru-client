// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"os"

	"github.com/ajg/form"
	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
	"github.com/tsuru/tsuru-client/tsuru/admin"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru-client/tsuru/config"
	"github.com/tsuru/tsuru-client/tsuru/config/selfupdater"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/iaas/dockermachine"
	_ "github.com/tsuru/tsuru/provision/docker/cmds"
)

var (
	version = "dev" // overridden at build time
)

const (
	header = "Supported-Tsuru"
)

func buildManager(name string) *cmd.Manager {
	form.DefaultEncoder = form.DefaultEncoder.UseJSONTags(false)
	form.DefaultDecoder = form.DefaultDecoder.UseJSONTags(false)

	lookup := func(context *cmd.Context) error {
		return client.RunPlugin(context)
	}
	m := cmd.BuildBaseManagerPanicExiter(name, version, header, lookup)
	m.RegisterTopic("app", `App is a program source code running on Tsuru`)
	m.Register(&client.AppRun{})
	m.Register(&client.AppInfo{})
	m.Register(&client.AppCreate{})
	m.Register(&client.AppRemove{})
	m.Register(&client.AppUpdate{})
	m.Register(&client.UnitAdd{})
	m.Register(&client.UnitRemove{})
	m.Register(&client.UnitKill{})
	m.Register(&client.UnitSet{})
	m.Register(&client.AppList{})
	m.Register(&client.AppLog{})
	m.Register(&client.AppGrant{})
	m.Register(&client.AppRevoke{})
	m.Register(&client.AppRestart{})
	m.Register(&client.AppStart{})
	m.Register(&client.AppStop{})
	m.Register(&client.Init{})
	m.Register(&client.CertificateSet{})
	m.Register(&client.CertificateUnset{})
	m.Register(&client.CertificateList{})
	m.Register(&client.CnameAdd{})
	m.Register(&client.CnameRemove{})
	m.Register(&client.EnvGet{})
	m.Register(&client.EnvSet{})
	m.Register(&client.EnvUnset{})
	m.RegisterTopic("service", `A service is a well-defined API that tsuru communicates with to provide extra functionality for applications.
Examples of services are MySQL, Redis, MongoDB, etc. tsuru has built-in services, but it is easy to create and add new services to tsuru.
Services aren’t managed by tsuru, but by their creators.`)
	m.Register(&client.ServiceList{})
	m.Register(&client.ServiceInstanceAdd{})
	m.Register(&client.ServiceInstanceUpdate{})
	m.Register(&client.ServiceInstanceRemove{})
	m.Register(&client.ServiceInfo{})
	m.Register(&client.ServicePlanList{})
	m.Register(&client.ServiceInstanceGrant{})
	m.Register(&client.ServiceInstanceRevoke{})
	m.Register(&client.ServiceInstanceBind{})
	m.Register(&client.ServiceInstanceUnbind{})

	m.RegisterTopic("platform", `A platform is a well-defined pack with installed dependencies for a language or framework that a group of applications will need. A platform might be a container template (Docker image).`)
	m.Register(&admin.PlatformList{})
	m.Register(&admin.PlatformAdd{})
	m.Register(&admin.PlatformUpdate{})
	m.Register(&admin.PlatformRemove{})
	m.Register(&admin.PlatformInfo{})

	m.RegisterTopic("job", `Job is a program that runs following a schedule`)
	m.Register(&client.JobCreate{})
	m.Register(&client.JobInfo{})
	m.Register(&client.JobList{})
	m.Register(&client.JobDelete{})
	m.Register(&client.JobTrigger{})

	m.Register(&client.PluginInstall{})
	m.Register(&client.PluginRemove{})
	m.Register(&client.PluginList{})
	m.Register(&client.PluginBundle{})
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
	m.Register(&admin.TeamQuotaView{})
	m.Register(&admin.TeamChangeQuota{})
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
	m.Register(&admin.AddPoolToSchedulerCmd{})
	m.Register(&client.EventList{})
	m.Register(&client.EventInfo{})
	m.Register(&client.EventCancel{})
	m.Register(&client.RoutersList{})
	m.Register(&client.RouterAdd{})
	m.Register(&client.RouterUpdate{})
	m.Register(&client.RouterRemove{})
	m.Register(&client.RouterInfo{})
	m.Register(&admin.TemplateList{})
	m.Register(&admin.TemplateAdd{})
	m.Register(&admin.TemplateRemove{})
	m.Register(&admin.MachineList{})
	m.Register(&admin.MachineDestroy{})
	m.Register(&admin.TemplateUpdate{})
	m.Register(&admin.TemplateCopy{})
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

	m.RegisterTopic("volume", "Volumes allow applications running on tsuru to use external storage volumes mounted on their filesystem.")
	m.Register(&client.VolumeCreate{})
	m.Register(&client.VolumeUpdate{})
	m.Register(&client.VolumeList{})
	m.Register(&client.VolumePlansList{})
	m.Register(&client.VolumeDelete{})
	m.Register(&client.VolumeInfo{})
	m.Register(&client.VolumeBind{})
	m.Register(&client.VolumeUnbind{})
	m.Register(&client.AppRoutersList{})
	m.Register(&client.AppRoutersAdd{})
	m.Register(&client.AppRoutersRemove{})
	m.Register(&client.AppRoutersUpdate{})
	m.Register(&admin.InfoNodeCmd{})
	m.Register(&client.TokenCreateCmd{})
	m.Register(&client.TokenUpdateCmd{})
	m.Register(&client.TokenListCmd{})
	m.Register(&client.TokenDeleteCmd{})
	m.Register(&client.TokenInfoCmd{})
	m.Register(&client.WebhookList{})
	m.Register(&client.WebhookCreate{})
	m.Register(&client.WebhookUpdate{})
	m.Register(&client.WebhookDelete{})
	m.Register(&admin.BrokerList{})
	m.Register(&admin.BrokerAdd{})
	m.Register(&admin.BrokerUpdate{})
	m.Register(&admin.BrokerDelete{})
	m.Register(&admin.ProvisionerList{})
	m.Register(&admin.ProvisionerInfo{})
	m.Register(&client.AppVersionRouterAdd{})
	m.Register(&client.AppVersionRouterRemove{})
	m.Register(client.UserInfo{})
	m.Register(&client.AutoScaleSet{})
	m.Register(&client.AutoScaleUnset{})
	m.Register(&client.MetadataSet{})
	m.Register(&client.MetadataUnset{})
	m.Register(&client.MetadataGet{})
	m.Register(&admin.AddNodeCmd{})
	m.Register(&admin.RemoveNodeCmd{})
	m.Register(&admin.UpdateNodeCmd{})
	m.Register(&admin.ListNodesCmd{})
	m.Register(&admin.GetNodeHealingConfigCmd{})
	m.Register(&admin.SetNodeHealingConfigCmd{})
	m.Register(&admin.DeleteNodeHealingConfigCmd{})
	m.Register(&admin.RebalanceNodeCmd{})
	m.Register(&admin.AutoScaleRunCmd{})
	m.Register(&admin.ListAutoScaleHistoryCmd{})
	m.Register(&admin.AutoScaleInfoCmd{})
	m.Register(&admin.AutoScaleSetRuleCmd{})
	m.Register(&admin.AutoScaleDeleteRuleCmd{})
	m.Register(&admin.ListHealingHistoryCmd{})
	m.Register(&client.ServiceInstanceInfo{})
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

func recoverCmdPanicExitError() {
	if r := recover(); r != nil {
		if e, ok := r.(*cmd.PanicExitError); ok {
			os.Exit(e.Code)
		}
		panic(r)
	}
}

func main() {
	defer recoverCmdPanicExitError()

	if inDockerMachineDriverMode() {
		err := dockermachine.RunDriver(os.Getenv(localbinary.PluginEnvDriverName))
		if err != nil {
			log.Fatalf("Error running driver: %s", err)
		}
	} else {
		defer config.SaveChangesWithTimeout()

		checkVerResult := selfupdater.CheckLatestVersionBackground(version)
		defer selfupdater.VerifyLatestVersion(checkVerResult)

		localbinary.CurrentBinaryIsDockerMachine = true
		name := cmd.ExtractProgramName(os.Args[0])
		m := buildManager(name)
		m.Run(os.Args[1:])
	}
}
