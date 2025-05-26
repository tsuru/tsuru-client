// Copyright 2017 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/check.v1"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	"github.com/tsuru/tsuru-client/tsuru/admin"
	"github.com/tsuru/tsuru-client/tsuru/client"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
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
	manager = cmd.NewManagerPanicExiter("glb", &stdout, &stderr, os.Stdin, nil)
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

func (s *S) TestServiceListIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["service-list"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &client.ServiceList{})
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
	c.Assert(info, check.FitsTypeOf, &client.ServiceInstanceInfo{})
}

func (s *S) TestServiceInfoIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	info, ok := manager.Commands["service-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(info, check.FitsTypeOf, &client.ServiceInfo{})
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

func (s *S) TestUnitSetIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	rmunit, ok := manager.Commands["unit-set"]
	c.Assert(ok, check.Equals, true)
	c.Assert(rmunit, check.FitsTypeOf, &client.UnitSet{})
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

func (s *S) TestPluginBundleIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	command, ok := manager.Commands["plugin-bundle"]
	c.Assert(ok, check.Equals, true)
	c.Assert(command, check.FitsTypeOf, &client.PluginBundle{})
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
	pluginPath := config.JoinWithUserDir(".tsuru", "plugins", "myplugin")
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

func (s *S) TestAppDeployRollbackIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	deployRollbackCmd, ok := manager.Commands["app-deploy-rollback"]
	c.Assert(ok, check.Equals, true)
	c.Assert(deployRollbackCmd, check.FitsTypeOf, &client.AppDeployRollback{})
}

func (s *S) TestAppDeployRollbackUpdateIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	deployRollbackUpdateCmd, ok := manager.Commands["app-deploy-rollback-update"]
	c.Assert(ok, check.Equals, true)
	c.Assert(deployRollbackUpdateCmd, check.FitsTypeOf, &client.AppDeployRollbackUpdate{})
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

func (s *S) TestTeamInfoIsRegistered(c *check.C) {
	manager = buildManager("tsuru")
	list, ok := manager.Commands["team-info"]
	c.Assert(ok, check.Equals, true)
	c.Assert(list, check.FitsTypeOf, &client.TeamInfo{})
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

func (s *S) TestInvalidCommandTopicMatch(c *check.C) {
	var stdout, stderr bytes.Buffer
	mngr := buildManagerCustom("tsuru", &stdout, &stderr)

	mngr.Run([]string{"target"})

	expectedOutput := fmt.Sprintf(`%s
	
The following commands are available in the "target" topic:

  target add           Adds a new entry to the list of available targets
  target list          Displays the list of targets, marking the current
  target remove        Remove a target from target-list (tsuru server)
  target set           Change current target (tsuru server)

Use tsuru help <commandname> to get more information about a command.
`, targetTopic)

	obtained := strings.ReplaceAll(stdout.String(), "\t\n", "\n")
	expected := strings.ReplaceAll(expectedOutput, "\t\n", "\n")

	c.Assert(stderr.String(), check.Equals, "")
	c.Assert(obtained, check.Equals, expected)
}

type recordingExiter int

func (e *recordingExiter) Exit(code int) {
	*e = recordingExiter(code)
}

func (s *S) TestInvalidCommandFuzzyMatch02(c *check.C) {
	var exiter recordingExiter
	var stdout, stderr bytes.Buffer
	mngr := buildManagerCustom("tsuru", &stdout, &stderr)
	mngr.SetExiter(&exiter)
	mngr.Run([]string{"target lisr"})
	expectedOutput := `.*: "target lisr" is not a tsuru command. See "tsuru help".	
Did you mean?	
	target list
`
	expectedOutput = strings.ReplaceAll(expectedOutput, "\n", "\\W")
	expectedOutput = strings.ReplaceAll(expectedOutput, "\t", "\\W+")
	c.Assert(stderr.String(), check.Matches, expectedOutput)
	c.Assert(int(exiter), check.Equals, 1)
}

func (s *S) TestInvalidCommandFuzzyMatch03(c *check.C) {
	var exiter recordingExiter
	var stdout, stderr bytes.Buffer
	mngr := buildManagerCustom("tsuru", &stdout, &stderr)
	mngr.SetExiter(&exiter)
	mngr.Run([]string{"list"})

	output := stderr.String()

	c.Assert(strings.Contains(output, `"list" is not a tsuru command. See "tsuru help"`), check.Equals, true)
	c.Assert(strings.Contains(output, `Did you mean?`), check.Equals, true)
	c.Assert(strings.Contains(output, `target list`), check.Equals, true)

	c.Assert(int(exiter), check.Equals, 1)
}

func (s *S) TestInvalidCommandFuzzyMatch04(c *check.C) {
	var exiter recordingExiter
	var stdout, stderr bytes.Buffer
	mngr := buildManagerCustom("tsuru", &stdout, &stderr)
	mngr.SetExiter(&exiter)
	mngr.Run([]string{"not-command"})
	expectedOutput := `.*: "not-command" is not a tsuru command. See "tsuru help".
`
	expectedOutput = strings.ReplaceAll(expectedOutput, "\n", "\\W")
	expectedOutput = strings.ReplaceAll(expectedOutput, "\t", "\\W+")
	c.Assert(stderr.String(), check.Matches, expectedOutput)
	c.Assert(int(exiter), check.Equals, 1)
}

func (s *S) TestInvalidCommandFuzzyMatch05(c *check.C) {
	var exiter recordingExiter
	var stdout, stderr bytes.Buffer
	mngr := buildManagerCustom("tsuru", &stdout, &stderr)
	mngr.SetExiter(&exiter)
	mngr.Run([]string{"target", "sit"})
	expectedOutput := `.*: "target sit" is not a tsuru command. See "tsuru help".

Did you mean?
	target list
	target set
`

	expectedOutput = strings.ReplaceAll(expectedOutput, "\n", "\\W")
	expectedOutput = strings.ReplaceAll(expectedOutput, "\t", "\\W+")
	c.Assert(stderr.String(), check.Matches, expectedOutput)
	c.Assert(int(exiter), check.Equals, 1)
}

func (s *S) TestVersion(c *check.C) {
	command := versionCmd{}
	context := cmd.Context{
		Args:   []string{},
		Stdout: &bytes.Buffer{},
	}
	command.Run(&context)
	c.Assert(context.Stdout.(*bytes.Buffer).String(), check.Matches, "Client version: dev.\n.*")
}

func (s *S) TestVersionInfo(c *check.C) {
	expected := &cmd.Info{
		Name:    "version",
		MinArgs: 0,
		Usage:   "version",
		Desc:    "display the current version",
	}
	c.Assert((&versionCmd{}).Info(), check.DeepEquals, expected)
}

func (s *S) TestVersionWithAPI(c *check.C) {

	command := versionCmd{}
	context := cmd.Context{
		Args:   []string{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{"version":"1.7.4"}`))
	}))
	defer ts.Close()

	os.Setenv("TSURU_TARGET", ts.URL)
	defer os.Unsetenv("TSURU_TARGET")
	err := command.Run(&context)
	c.Assert(err, check.IsNil)
	c.Assert(context.Stdout.(*bytes.Buffer).String(),
		check.Equals, "Client version: dev.\nServer version: 1.7.4.\n")
}

func (s *S) TestVersionAPIInvalidURL(c *check.C) {
	tsuruHTTP.AuthenticatedClient = &http.Client{}
	command := versionCmd{}
	context := cmd.Context{
		Args:   []string{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	URL := "notvalid.test"
	os.Setenv("TSURU_TARGET", URL)
	defer os.Unsetenv("TSURU_TARGET")
	err := command.Run(&context)
	c.Assert(true, check.Equals, strings.Contains(err.Error(), "Unable to retrieve server version"))
	c.Assert(true, check.Equals, strings.Contains(err.Error(), "no such host"))

	stdout := context.Stdout.(*bytes.Buffer).String()

	c.Assert(stdout, check.Matches, "Client version: dev.\n")
}
