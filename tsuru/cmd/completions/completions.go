package completions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tsuru/go-tsuruclient/pkg/config"
	tsuruHTTP "github.com/tsuru/tsuru-client/tsuru/http"
	appTypes "github.com/tsuru/tsuru/types/app"
	provisionTypes "github.com/tsuru/tsuru/types/provision"
)

func AppNameCompletionFunc(toComplete string) ([]string, error) {
	query := make(url.Values)
	query.Set("name", toComplete)
	query.Set("simplified", "true")

	u, err := config.GetURL(fmt.Sprintf("/apps?%s", query.Encode()))
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	response, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusNoContent {
		return []string{}, nil
	}
	defer response.Body.Close()

	var apps []appTypes.AppResume
	err = json.NewDecoder(response.Body).Decode(&apps)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(apps))
	for _, app := range apps {
		if !strings.HasPrefix(app.Name, toComplete) {
			continue
		}
		result = append(result, app.Name)
	}

	return result, nil
}

func TeamNameCompletionFunc(toComplete string) ([]string, error) {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	teams, resp, err := apiClient.TeamApi.TeamsList(context.TODO())
	if resp != nil && resp.StatusCode == http.StatusNoContent {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch teams")
	}

	result := make([]string, 0, len(teams))

	for _, team := range teams {
		if !strings.HasPrefix(team.Name, toComplete) {
			continue
		}
		result = append(result, team.Name)
	}

	return result, nil
}

func JobNameCompletionFunc(toComplete string) ([]string, error) {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return nil, err
	}

	jobs, resp, err := apiClient.JobApi.ListJob(context.Background())

	if resp != nil && resp.StatusCode == http.StatusNoContent {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(jobs))

	for _, job := range jobs {
		if !strings.HasPrefix(job.Name, toComplete) {
			continue
		}
		result = append(result, job.Name)
	}

	return result, nil
}

func PoolNameCompletionFunc(toComplete string) ([]string, error) {
	url, err := config.GetURL("/pools")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return []string{}, nil
	}
	defer resp.Body.Close()
	var pools []provisionTypes.Pool
	err = json.NewDecoder(resp.Body).Decode(&pools)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(pools))

	for _, pool := range pools {
		if !strings.HasPrefix(pool.Name, toComplete) {
			continue
		}
		result = append(result, pool.Name)
	}

	return result, nil
}

func PlanNameCompletionFunc(toComplete string) ([]string, error) {
	url, err := config.GetURL("/plans")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	var plans []appTypes.Plan
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return []string{}, nil
	}
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(plans))

	for _, plan := range plans {
		if !strings.HasPrefix(plan.Name, toComplete) {
			continue
		}
		result = append(result, plan.Name)
	}

	return result, nil
}

func PlatformNameCompletionFunc(toComplete string) ([]string, error) {
	url, err := config.GetURL("/platforms")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	var platforms []appTypes.Platform
	resp, err := tsuruHTTP.AuthenticatedClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return []string{}, nil
	}
	err = json.NewDecoder(resp.Body).Decode(&platforms)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(platforms))

	for _, platform := range platforms {
		if !strings.HasPrefix(platform.Name, toComplete) {
			continue
		}
		result = append(result, platform.Name)
	}
	return result, nil
}

func RouterNameCompletionFunc(toComplete string) ([]string, error) {
	apiClient, err := tsuruHTTP.TsuruClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	routers, _, err := apiClient.RouterApi.RouterList(context.TODO())
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(routers))

	for _, router := range routers {
		if !strings.HasPrefix(router.Name, toComplete) {
			continue
		}
		result = append(result, router.Name)
	}

	return result, nil
}
