package main

import (
	"encoding/json"
	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/provision"
	"net/http"
	"strings"
)

type poolList struct{}

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
	var pools []provision.Pool
	err = json.NewDecoder(resp.Body).Decode(&pools)
	if err != nil {
		return err
	}
	t := cmd.Table{Headers: cmd.Row([]string{"Pools", "Teams"})}
	for _, p := range pools {
		t.AddRow(cmd.Row([]string{p.Name, strings.Join(p.Teams, ", ")}))
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
