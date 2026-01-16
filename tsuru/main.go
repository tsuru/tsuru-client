// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cezarsa/form"
	"github.com/pkg/errors"
	goTsuruClient "github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/config"

	"github.com/tsuru/tsuru-client/tsuru/admin"
	"github.com/tsuru/tsuru-client/tsuru/auth"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru-client/tsuru/cmd"
	"github.com/tsuru/tsuru-client/tsuru/cmd/completions"
	"github.com/tsuru/tsuru-client/tsuru/cmd/standards"
	"github.com/tsuru/tsuru-client/tsuru/config/selfupdater"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	tsuruErrors "github.com/tsuru/tsuru/errors"
	"golang.org/x/oauth2"
)

var version = "dev" // overridden at build time

const targetTopic = `Target is used to manage the address of the remote tsuru server.

Each target is identified by a label and a HTTP/HTTPS address. The client
requires at least one target to connect to, there's no default target. A user
may have multiple targets, but only one will be used at a time.`

func buildManager(name string) *cmd.Manager {
	form.DefaultEncoder = form.DefaultEncoder.UseJSONTags(false)
	form.DefaultDecoder = form.DefaultDecoder.UseJSONTags(false)

	return buildManagerCustom(name, os.Stdout, os.Stderr)
}

func buildManagerCustom(name string, stdout, stderr io.Writer) *cmd.Manager {
	retryHook := func(err error) (retry bool) {
		if teamToken := config.ReadTeamToken(); teamToken != "" {
			return false
		}

		mustLogin := false

		err = tsuruHTTP.UnwrapErr(err)

		if oauth2Err, ok := err.(*oauth2.RetrieveError); ok {
			fmt.Fprintf(os.Stderr, "oauth2 error: %s, %s\n", oauth2Err.ErrorCode, oauth2Err.ErrorDescription)
			mustLogin = true
		} else if httpErr, ok := err.(*tsuruErrors.HTTP); ok && httpErr.StatusCode() == http.StatusUnauthorized {
			fmt.Fprintf(os.Stderr, "http error: %d\n", httpErr.StatusCode())
			mustLogin = true
		}

		if !mustLogin {
			return false
		}

		fmt.Fprintln(os.Stderr, "trying to login again")
		c := &auth.Login{}
		loginErr := c.Run(&cmd.Context{
			Stdin:  os.Stdin,
			Stderr: stderr,
			Stdout: stdout,
		})

		if loginErr != nil {
			fmt.Fprintf(os.Stderr, "Could not login: %s\n", loginErr.Error())
			return false
		}

		initAuthorization() // re-init updated token provider
		return true
	}

	lookup := func(context *cmd.Context) error {
		err := client.RunPlugin(context)
		if err != nil {
			if retryHook(err) {
				return client.RunPlugin(context)
			}

			return err
		}

		return nil
	}

	m := cmd.NewManagerPanicExiter(name, stdout, stderr, os.Stdin, lookup)

	m.SetFlagCompletions(map[string]cmd.CompletionFunc{
		standards.FlagApp:      completions.AppNameCompletionFunc,
		standards.FlagTeam:     completions.TeamNameCompletionFunc,
		standards.FlagJob:      completions.JobNameCompletionFunc,
		standards.FlagPool:     completions.PoolNameCompletionFunc,
		standards.FlagPlan:     completions.PlanNameCompletionFunc,
		standards.FlagPlatform: completions.PlatformNameCompletionFunc,
		standards.FlagRouter:   completions.RouterNameCompletionFunc,
	})

	m.RegisterTopic("app", `App is a program source code running on Tsuru.`)

	m.Register(&auth.Login{})
	m.Register(&auth.Logout{})
	m.Register(&versionCmd{})

	m.RegisterTopic("target", targetTopic)
	m.Register(&client.TargetList{})
	m.Register(&client.TargetAdd{})
	m.Register(&client.TargetRemove{})
	m.Register(&client.TargetSet{})

	m.Register(&client.AppRun{})
	m.Register(&client.AppInfo{})
	m.Register(&client.AppCreate{})
	m.Register(&client.AppRemove{})
	m.Register(&client.AppUpdate{})

	m.RegisterTopic("app-process", `An application process represents a command that runs as part of the application.`)
	m.Register(&client.AppProcessUpdate{})

	m.RegisterTopic("unit", "A unit is a container.")
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

	m.RegisterTopic("certificate", `Certificate is used to manage SSL/TLS certificates for secure communication.`)
	m.RegisterTopic("certificate-issuer", `Issuer is used to automate the issuance of SSL/TLS certificates.`)

	m.Register(&client.CertificateSet{})
	m.Register(&client.CertificateUnset{})
	m.Register(&client.CertificateList{})

	m.Register(&client.CertificateIssuerSet{})
	m.Register(&client.CertificateIssuerUnset{})

	m.RegisterTopic("cname", `CNAME (Canonical Name) is a custom domain you assign to your application, allowing users to access it via a friendly URL (e.g., myapp.mydomain.com)`)
	m.Register(&client.CnameAdd{})
	m.Register(&client.CnameRemove{})

	m.RegisterTopic("env", `Manage environment variables from an app or job.`)
	m.Register(&client.EnvGet{})
	m.Register(&client.EnvSet{})
	m.Register(&client.EnvUnset{})

	m.RegisterTopic("service", `A service is a well-defined API that tsuru communicates with to provide extra functionality for applications.
Examples of services are MySQL, Redis, MongoDB, etc. tsuru has built-in services, but it is easy to create and add new services to tsuru.
Services aren’t managed by tsuru, but by their creators.`)
	m.Register(&client.ServiceList{})

	m.RegisterTopic("service-instance", `Manage provisioned resources (like a database or proxy) that can be linked to an application.`)
	m.Register(&client.ServiceInstanceAdd{})
	m.Register(&client.ServiceInstanceUpdate{})
	m.Register(&client.ServiceInstanceRemove{})

	m.Register(&client.ServiceInfo{})

	m.RegisterTopic("service-plan", `Manage service plans of a given service.`)
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

	m.RegisterTopic("job", `Job is a program that runs on a schedule or is executed manually`)
	m.Register(&client.JobCreate{})
	m.Register(&client.JobUpdate{})
	m.Register(&client.JobInfo{})
	m.Register(&client.JobList{})
	m.Register(&client.JobDelete{})
	m.Register(&client.JobTrigger{})
	m.Register(&client.JobLog{})
	m.Register(&client.JobDeploy{})

	m.RegisterTopic("plugin", `Plugins are used to extend tsuru client functionality.`)
	m.Register(&client.PluginInstall{})
	m.Register(&client.PluginRemove{})
	m.Register(&client.PluginList{})
	m.Register(&client.PluginBundle{})

	m.Register(&client.AppDeploy{})
	m.Register(&client.AppBuild{})

	m.RegisterTopic("plan", `Plan specifies how computational resources are allocated to your application.`)
	m.Register(&client.PlanList{})

	m.RegisterTopic("user", `A user is an individual who has access to the tsuru platform.`)
	m.Register(&client.UserCreate{})
	m.Register(&client.ResetPassword{})
	m.Register(&client.UserRemove{})
	m.Register(&client.ListUsers{})

	m.RegisterTopic("team", "Manage teams.")
	m.Register(&client.TeamCreate{})
	m.Register(&client.TeamUpdate{})
	m.Register(&client.TeamRemove{})
	m.Register(&client.TeamList{})
	m.Register(&client.TeamInfo{})

	m.RegisterTopic("team-quota", "Quotas are used to safeguard the cluster against undesired or excessive scaling.")
	m.Register(&admin.TeamQuotaView{})
	m.Register(&admin.TeamChangeQuota{})

	m.Register(&client.ChangePassword{})
	m.Register(&client.AppDeployList{})
	m.Register(&client.AppDeployRollback{})
	m.Register(&client.AppDeployRollbackUpdate{})
	m.Register(&client.AppDeployRebuild{})
	m.Register(&client.ShellToContainerCmd{})

	m.RegisterTopic("pool", "A pool is used by provisioners to allocate space within a cluster for running applications.")
	m.Register(&client.PoolList{})

	m.RegisterTopic("permission", `Manage permissions.`)
	m.Register(&client.PermissionList{})

	m.RegisterTopic("role", "Manage roles.")
	m.Register(&client.RoleAdd{})
	m.Register(&client.RoleUpdate{})
	m.Register(&client.RoleRemove{})
	m.Register(&client.RoleList{})
	m.Register(&client.RoleInfo{})

	m.RegisterTopic("role-permission", "Manage permissions associated with each role.")
	m.Register(&client.RolePermissionAdd{})
	m.Register(&client.RolePermissionRemove{})

	m.Register(&client.RoleAssign{})
	m.Register(&client.RoleDissociate{})

	m.RegisterTopic("role-default", "Manage default roles assigned to users upon creation.")
	m.Register(&client.RoleDefaultList{})
	m.Register(&client.RoleDefaultAdd{})
	m.Register(&client.RoleDefaultRemove{})

	m.Register(&admin.AddPoolToSchedulerCmd{})

	m.RegisterTopic("event", `Events are used to audit all actions performed on Tsuru resources.`)
	m.Register(&client.EventList{})
	m.Register(&client.EventInfo{})
	m.Register(&client.EventCancel{})

	m.RegisterTopic("router", `A router is a component responsible for routing user traffic to applications.`)
	m.Register(&client.RoutersList{})
	m.Register(&client.RouterAdd{})
	m.Register(&client.RouterUpdate{})
	m.Register(&client.RouterRemove{})
	m.Register(&client.RouterInfo{})
	m.Register(&admin.PlanCreate{})
	m.Register(&admin.PlanRemove{})
	m.Register(&admin.UpdatePoolToSchedulerCmd{})
	m.Register(&admin.RemovePoolFromSchedulerCmd{})
	m.Register(&admin.ServiceCreate{})
	m.Register(&admin.ServiceDestroy{})
	m.Register(&admin.ServiceUpdate{})
	m.Register(&admin.ServiceTemplate{})

	m.RegisterTopic("service-doc", "Manage Service Docs")
	m.Register(&admin.ServiceDocGet{})
	m.Register(&admin.ServiceDocAdd{})

	m.RegisterTopic("user-quota", `Quotas are used to safeguard the cluster against undesired or excessive scaling.`)
	m.Register(&admin.UserQuotaView{})
	m.Register(&admin.UserChangeQuota{})

	m.RegisterTopic("app-quota", `Quotas are used to safeguard the cluster against undesired or excessive scaling.`)
	m.Register(&admin.AppQuotaView{})
	m.Register(&admin.AppQuotaChange{})

	m.RegisterTopic("app-routes", `Manage routes within the application router.`)
	m.Register(&admin.AppRoutesRebuild{})

	m.RegisterTopic("pool-constraint", "Pool constraints are rules that define which kind of resources can be associated with a pool.")
	m.Register(&admin.PoolConstraintList{})
	m.Register(&admin.PoolConstraintSet{})

	m.RegisterTopic("event-block", "Event blocks prevent specific system actions during maintenance.")
	m.Register(&admin.EventBlockList{})
	m.Register(&admin.EventBlockAdd{})
	m.Register(&admin.EventBlockRemove{})

	m.RegisterTopic("tag", `Tags are labels that can be assigned to applications and service instances to help organize and manage them effectively.`)
	m.Register(&client.TagList{})

	m.RegisterTopic("cluster", "Manage kubernetes clusters.")
	m.Register(&admin.ClusterAdd{})
	m.Register(&admin.ClusterUpdate{})
	m.Register(&admin.ClusterRemove{})
	m.Register(&admin.ClusterList{})

	m.RegisterTopic("volume", "Volumes allow applications running on tsuru to use external storage volumes mounted on their filesystem.")
	m.Register(&client.VolumeCreate{})
	m.Register(&client.VolumeUpdate{})
	m.Register(&client.VolumeList{})
	m.Register(&client.VolumeDelete{})
	m.Register(&client.VolumeInfo{})
	m.Register(&client.VolumeBind{})
	m.Register(&client.VolumeUnbind{})

	m.RegisterTopic("volume-plan", `Manage volume plans`)
	m.Register(&client.VolumePlansList{})

	m.RegisterTopic("app-router", "Router is a component responsible for routing user traffic to applications.")
	m.Register(&client.AppRoutersList{})
	m.Register(&client.AppRoutersAdd{})
	m.Register(&client.AppRoutersRemove{})
	m.Register(&client.AppRoutersUpdate{})

	m.RegisterTopic("token", "Manage team tokens used to authenticate automations through Tsuru.")
	m.Register(&client.TokenCreateCmd{})
	m.Register(&client.TokenUpdateCmd{})
	m.Register(&client.TokenListCmd{})
	m.Register(&client.TokenDeleteCmd{})
	m.Register(&client.TokenInfoCmd{})
	m.Register(&client.RegenerateAPIToken{})

	m.RegisterTopic("event-webhook", "Event webhooks allow integrating tsuru events with external systems.")
	m.Register(&client.ShowAPIToken{})
	m.Register(&client.WebhookList{})
	m.Register(&client.WebhookCreate{})
	m.Register(&client.WebhookUpdate{})
	m.Register(&client.WebhookDelete{})

	m.RegisterTopic("provisioner", "Manage provisioners.")
	m.Register(&admin.ProvisionerList{})
	m.Register(&admin.ProvisionerInfo{})

	m.RegisterTopic("app-router-version", "Manage multiple application versions behind the application router.")
	m.Register(&client.AppVersionRouterAdd{})
	m.Register(&client.AppVersionRouterRemove{})

	m.Register(client.UserInfo{})

	m.RegisterTopic("autoscale", "Manage autoscaling of application units.")
	m.RegisterTopic("unit-autoscale", "Manage autoscaling of application units.")
	m.RegisterDeprecated(&client.AutoScaleSet{}, "unit-autoscale-set")
	m.RegisterDeprecated(&client.AutoScaleUnset{}, "unit-autoscale-unset")
	m.RegisterDeprecated(&client.AutoScaleSwap{}, "unit-autoscale-swap")

	m.RegisterTopic("metadata", "Metadata is a modern way to define labels and annotations to apps and jobs.")
	m.RegisterTopic("app-metadata", "Metadata is a modern way to define labels and annotations to apps.")
	m.RegisterDeprecated(&client.MetadataSet{}, "app-metadata-set")
	m.RegisterDeprecated(&client.MetadataUnset{}, "app-metadata-unset")
	m.RegisterDeprecated(&client.MetadataGet{}, "app-metadata-get")

	// Shorthands is a frequent command with a short name for convenience
	// To decide which commands should have shorthands, consider:
	// - Frequency of use
	// - Length of the original command name
	// May be no more than 5 commands IMHO
	m.RegisterShorthand(&client.AppDeploy{}, "deploy")
	m.RegisterShorthand(&client.AppDeployRollback{}, "rollback")
	m.RegisterShorthand(&client.AppLog{}, "log")
	m.RegisterShorthand(&client.AppInfo{}, "info")
	m.RegisterShorthand(&client.ShellToContainerCmd{}, "shell")

	m.Register(&client.ServiceInstanceInfo{})

	plugins := client.FindPlugins()
	for _, plugin := range plugins {
		m.RegisterPlugin(&client.ExecutePlugin{PluginName: plugin})
	}

	m.RetryHook = retryHook
	m.AfterFlagParseHook = initAuthorization
	return m
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
	var err error
	defer func() {
		if err != nil {
			os.Exit(1) // this will work only on V2 implementation
		}
	}()
	defer recoverCmdPanicExitError() // TODO: remove on migration completion
	defer config.SaveChangesWithTimeout()

	checkVerResult := selfupdater.CheckLatestVersionBackground(version)
	defer selfupdater.VerifyLatestVersion(checkVerResult)

	name := cmd.ExtractProgramName(os.Args[0])

	m := buildManager(name)
	err = m.Run(os.Args[1:])
}

func initAuthorization() {
	name := cmd.ExtractProgramName(os.Args[0])
	roundTripper, tokenProvider, err := goTsuruClient.RoundTripperAndTokenProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read token V2: %q\n", err.Error())
		os.Exit(1)
	}

	tsuruHTTP.AuthenticatedClient = tsuruHTTP.NewTerminalClient(tsuruHTTP.TerminalClientOptions{
		RoundTripper:  roundTripper,
		ClientName:    name,
		ClientVersion: version,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	})
	config.DefaultTokenProvider = tokenProvider
}

type versionCmd struct{}

func (c *versionCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name: "version",

		Usage: "version",
		Desc:  "display the current version",
	}
}

func (c *versionCmd) Run(context *cmd.Context) error {
	fmt.Fprint(context.Stdout, versionString())

	apiVersion, err := apiVersionString()
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, apiVersion)

	return nil
}

func versionString() string {
	return fmt.Sprintf("Client version: %s.\n", version)
}

func apiVersionString() (string, error) {
	url, err := config.GetURL("/info")
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := tsuruHTTP.AuthenticatedClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "Unable to retrieve server version")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var version map[string]string
	err = json.Unmarshal(body, &version)
	if err != nil {
		return "", err
	}

	resp.Body.Close()
	return fmt.Sprintf("Server version: %s.\n", version["version"]), nil
}
