package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tsuru/tsuru/cmd"
)

type poolList struct{}

type PoolsByTeam struct {
	Team  string
	Pools []string
}

type Pool struct {
	Name string
}

type ListPoolResponse struct {
	PoolsByTeam  []PoolsByTeam `json:"pools_by_team"`
	PublicPools  []Pool        `json:"public_pools"`
	DefaultPools []Pool        `json:"default_pool"`
}

func (poolList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/pools")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var pools ListPoolResponse
	err = json.NewDecoder(resp.Body).Decode(&pools)
	if err != nil {
		return err
	}
	t := cmd.Table{Headers: cmd.Row([]string{"Team", "Pools"})}
	for _, pool := range pools.PoolsByTeam {
		t.AddRow(cmd.Row([]string{pool.Team, strings.Join(pool.Pools, ", ")}))
	}
	t.Sort()
	context.Stdout.Write(t.Bytes())
	tp := cmd.Table{Headers: cmd.Row([]string{"Public Pools"})}
	for _, pool := range pools.PublicPools {
		tp.AddRow(cmd.Row([]string{pool.Name}))
	}
	tp.Sort()
	context.Stdout.Write([]byte("\n"))
	context.Stdout.Write(tp.Bytes())
	if len(pools.DefaultPools) > 0 {
		td := cmd.Table{Headers: cmd.Row([]string{"Default Pool"})}
		td.AddRow(cmd.Row([]string{pools.DefaultPools[0].Name}))
		context.Stdout.Write([]byte("\n"))
		context.Stdout.Write(td.Bytes())
	}
	return nil
}

func (poolList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "pool-list",
		Usage:   "pool-list",
		Desc:    "List all pools available for deploy.",
		MinArgs: 0,
	}
}
