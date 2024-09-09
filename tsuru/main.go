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
	"github.com/tsuru/tsuru-client/tsuru/config/selfupdater"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	"github.com/tsuru/tsuru/cmd"
	tsuruErrors "github.com/tsuru/tsuru/errors"
	"golang.org/x/oauth2"
)

var (
	version = "dev" // overridden at build time
)

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

	m.RegisterTopic("app", `App is a program source code running on Tsuru`)

	m.Register(&auth.Login{})
	m.Register(&auth.Logout{})
	m.Register(&versionCmd{})

	m.Register(&config.TargetList{})
	m.Register(&config.TargetAdd{})
	m.Register(&config.TargetRemove{})
	m.Register(&config.TargetSet{})
	m.RegisterTopic("target", targetTopic)

	m.Register(&client.AppRun{})
	m.Register(&client.AppInfo{})
	m.Register(&client.AppCreate{})
	m.Register(&client.AppRemove{})
	m.Register(&client.AppUpdate{})
	m.Register(&client.AppProcessUpdate{})
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

	m.RegisterTopic("job", `Job is a program that runs on a schedule or is executed manually`)
	m.Register(&client.JobCreate{})
	m.Register(&client.JobUpdate{})
	m.Register(&client.JobInfo{})
	m.Register(&client.JobList{})
	m.Register(&client.JobDelete{})
	m.Register(&client.JobTrigger{})
	m.Register(&client.JobLog{})
	m.Register(&client.JobDeploy{})

	m.Register(&client.PluginInstall{})
	m.Register(&client.PluginRemove{})
	m.Register(&client.PluginList{})
	m.Register(&client.PluginBundle{})
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
	m.Register(&client.ShellToContainerCmd{})
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
	m.RegisterDeprecated(&client.MetadataSet{}, "app-metadata-set")
	m.RegisterDeprecated(&client.MetadataUnset{}, "app-metadata-unset")
	m.RegisterDeprecated(&client.MetadataGet{}, "app-metadata-get")
	m.Register(&client.ServiceInstanceInfo{})
	registerExtraCommands(m)
	m.RetryHook = retryHook
	m.AfterFlagParseHook = initAuthorization
	return m
}

func registerExtraCommands(m *cmd.Manager) {
	for _, c := range cmd.ExtraCmds() {
		m.Register(c)
	}
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
	defer config.SaveChangesWithTimeout()

	checkVerResult := selfupdater.CheckLatestVersionBackground(version)
	defer selfupdater.VerifyLatestVersion(checkVerResult)

	name := cmd.ExtractProgramName(os.Args[0])

	m := buildManager(name)
	m.Run(os.Args[1:])
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
		Name:    "version",
		MinArgs: 0,
		Usage:   "version",
		Desc:    "display the current version",
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
