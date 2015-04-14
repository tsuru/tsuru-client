package main

import (
	"encoding/json"
	"github.com/tsuru/tsuru/cmd"
	"net/http"
	"strings"
)

type poolList struct{}

type PoolsByTeam struct {
	Team  string
	Pools []string
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
	var poolsByTeam []PoolsByTeam
	err = json.NewDecoder(resp.Body).Decode(&poolsByTeam)
	if err != nil {
		return err
	}
	t := cmd.Table{Headers: cmd.Row([]string{"Team", "Pools"})}
	for _, p := range poolsByTeam {
		t.AddRow(cmd.Row([]string{p.Team, strings.Join(p.Pools, ", ")}))
	}
	t.Sort()
	context.Stdout.Write(t.Bytes())
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
