// Copyright 2026 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package completions

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsuru/tsuru-client/tsuru/cmd/cmdtest"
)

func TestAppNameCompletionFunc(t *testing.T) {
	setupTest(t)
	result := `[{"name":"app1","pool":"pool1"},{"name":"app2","pool":"pool2"},{"name":"myapp","pool":"pool1"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := AppNameCompletionFunc("app")
	require.NoError(t, err)
	assert.Equal(t, []string{"app1", "app2"}, completions)
}

func TestAppNameCompletionFuncEmptyPrefix(t *testing.T) {
	setupTest(t)
	result := `[{"name":"app1","pool":"pool1"},{"name":"app2","pool":"pool2"},{"name":"myapp","pool":"pool1"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := AppNameCompletionFunc("")
	require.NoError(t, err)
	assert.Equal(t, []string{"app1", "app2", "myapp"}, completions)
}

func TestAppNameCompletionFuncNoContent(t *testing.T) {
	setupTest(t)
	setupFakeTransport(&cmdtest.Transport{Status: http.StatusNoContent})

	completions, err := AppNameCompletionFunc("app")
	require.NoError(t, err)
	assert.Empty(t, completions)
}

func TestAppNameCompletionFuncNoMatch(t *testing.T) {
	setupTest(t)
	result := `[{"name":"myapp","pool":"pool1"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := AppNameCompletionFunc("xyz")
	require.NoError(t, err)
	assert.Empty(t, completions)
}

func TestTeamNameCompletionFunc(t *testing.T) {
	setupTest(t)
	result := `[{"name":"team1"},{"name":"team2"},{"name":"myteam"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := TeamNameCompletionFunc("team")
	require.NoError(t, err)
	assert.Equal(t, []string{"team1", "team2"}, completions)
}

func TestTeamNameCompletionFuncEmptyPrefix(t *testing.T) {
	setupTest(t)
	result := `[{"name":"team1"},{"name":"team2"},{"name":"myteam"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := TeamNameCompletionFunc("")
	require.NoError(t, err)
	assert.Equal(t, []string{"team1", "team2", "myteam"}, completions)
}

func TestTeamNameCompletionFuncNoContent(t *testing.T) {
	setupTest(t)
	setupFakeTransport(&cmdtest.Transport{Status: http.StatusNoContent})

	completions, err := TeamNameCompletionFunc("team")
	require.NoError(t, err)
	assert.Empty(t, completions)
}

func TestJobNameCompletionFunc(t *testing.T) {
	setupTest(t)
	result := `[{"name":"job1"},{"name":"job2"},{"name":"myjob"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := JobNameCompletionFunc("job")
	require.NoError(t, err)
	assert.Equal(t, []string{"job1", "job2"}, completions)
}

func TestJobNameCompletionFuncEmptyPrefix(t *testing.T) {
	setupTest(t)
	result := `[{"name":"job1"},{"name":"job2"},{"name":"myjob"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := JobNameCompletionFunc("")
	require.NoError(t, err)
	assert.Equal(t, []string{"job1", "job2", "myjob"}, completions)
}

func TestJobNameCompletionFuncNoContent(t *testing.T) {
	setupTest(t)
	setupFakeTransport(&cmdtest.Transport{Status: http.StatusNoContent})

	completions, err := JobNameCompletionFunc("job")
	require.NoError(t, err)
	assert.Empty(t, completions)
}

func TestPoolNameCompletionFunc(t *testing.T) {
	setupTest(t)
	result := `[{"name":"pool1"},{"name":"pool2"},{"name":"mypool"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := PoolNameCompletionFunc("pool")
	require.NoError(t, err)
	assert.Equal(t, []string{"pool1", "pool2"}, completions)
}

func TestPoolNameCompletionFuncEmptyPrefix(t *testing.T) {
	setupTest(t)
	result := `[{"name":"pool1"},{"name":"pool2"},{"name":"mypool"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := PoolNameCompletionFunc("")
	require.NoError(t, err)
	assert.Equal(t, []string{"pool1", "pool2", "mypool"}, completions)
}

func TestPoolNameCompletionFuncNoContent(t *testing.T) {
	setupTest(t)
	setupFakeTransport(&cmdtest.Transport{Status: http.StatusNoContent})

	completions, err := PoolNameCompletionFunc("pool")
	require.NoError(t, err)
	assert.Empty(t, completions)
}

func TestPlanNameCompletionFunc(t *testing.T) {
	setupTest(t)
	result := `[{"name":"plan1"},{"name":"plan2"},{"name":"myplan"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := PlanNameCompletionFunc("plan")
	require.NoError(t, err)
	assert.Equal(t, []string{"plan1", "plan2"}, completions)
}

func TestPlanNameCompletionFuncEmptyPrefix(t *testing.T) {
	setupTest(t)
	result := `[{"name":"plan1"},{"name":"plan2"},{"name":"myplan"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := PlanNameCompletionFunc("")
	require.NoError(t, err)
	assert.Equal(t, []string{"plan1", "plan2", "myplan"}, completions)
}

func TestPlanNameCompletionFuncNoContent(t *testing.T) {
	setupTest(t)
	setupFakeTransport(&cmdtest.Transport{Status: http.StatusNoContent})

	completions, err := PlanNameCompletionFunc("plan")
	require.NoError(t, err)
	assert.Empty(t, completions)
}

func TestPlatformNameCompletionFunc(t *testing.T) {
	setupTest(t)
	result := `[{"Name":"python"},{"Name":"nodejs"},{"Name":"go"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := PlatformNameCompletionFunc("py")
	require.NoError(t, err)
	assert.Equal(t, []string{"python"}, completions)
}

func TestPlatformNameCompletionFuncEmptyPrefix(t *testing.T) {
	setupTest(t)
	result := `[{"Name":"python"},{"Name":"nodejs"},{"Name":"go"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := PlatformNameCompletionFunc("")
	require.NoError(t, err)
	assert.Equal(t, []string{"python", "nodejs", "go"}, completions)
}

func TestPlatformNameCompletionFuncNoContent(t *testing.T) {
	setupTest(t)
	setupFakeTransport(&cmdtest.Transport{Status: http.StatusNoContent})

	completions, err := PlatformNameCompletionFunc("python")
	require.NoError(t, err)
	assert.Empty(t, completions)
}

func TestRouterNameCompletionFunc(t *testing.T) {
	setupTest(t)
	result := `[{"name":"router1"},{"name":"router2"},{"name":"myrouter"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := RouterNameCompletionFunc("router")
	require.NoError(t, err)
	assert.Equal(t, []string{"router1", "router2"}, completions)
}

func TestRouterNameCompletionFuncEmptyPrefix(t *testing.T) {
	setupTest(t)
	result := `[{"name":"router1"},{"name":"router2"},{"name":"myrouter"}]`
	setupFakeTransport(&cmdtest.Transport{Message: result, Status: http.StatusOK})

	completions, err := RouterNameCompletionFunc("")
	require.NoError(t, err)
	assert.Equal(t, []string{"router1", "router2", "myrouter"}, completions)
}
