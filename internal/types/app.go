// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"k8s.io/apimachinery/pkg/api/resource"
)

type MiniApp struct {
	Application string `priority:"99"`
	Units       int
	Owner       string
	Errors      string `priority:"-2"`
}

func AppList(apps []tsuru.MiniApp) []MiniApp {
	appsForPrint := make([]MiniApp, len(apps))
	for i, app := range apps {
		appsForPrint[i] = MiniApp{
			Application: app.Name,
			Units:       len(app.Units),
			Owner:       app.TeamOwner,
			Errors: func() string {
				if app.Error == "" {
					unitMap := make(map[string]int)
					unitsWithError := 0
					for _, unit := range app.Units {
						if unit.Id == "" {
							continue
						}
						if unit.Ready != nil && *unit.Ready {
							unitMap["ready"]++
						} else {
							unitsWithError++
							unitMap[unit.Status]++
						}
					}
					var units []string
					for status, count := range unitMap {
						units = append(units, fmt.Sprintf("%d %s", count, status))
					}
					if unitsWithError > 0 {
						return "units: " + strings.Join(units, ", ")
					}
				}
				return app.Error
			}(),
		}
	}
	return appsForPrint
}

type SimpleApp struct {
	Application string `priority:"99"`
	Cluster     string
	CreatedBy   string
	Description string   `priority:"98"`
	Errors      []string `priority:"100"`
	Plan        string
	Platform    string
	Pool        string
	Quota       string
	Router      string
	RouterOpts  map[string]string
	Routers     []string
	Tags        []string
	Teams       []string
	UnitsCount  int `name:"Units"`
	Units       []Unit

	ClusterInternalAddresses []InternalAddress
	ClusterExternalAddresses []ExternalAddress
	ServiceInstances         []ServiceInstance
}

type ServiceInstance struct {
	Service  string `priority:"3"`
	Instance string `priority:"2"`
	Plan     string `priority:"1"`
}

type Unit struct {
	Process   string `priority:"99"`
	Ready     string
	Restarts  uint
	AvgCPU    string `priority:"-1"`
	AvgMemory string `priority:"-1"`
}

type InternalAddress struct {
	Domain  string `priority:"4"`
	Port    string `priority:"3"`
	Process string `priority:"2"`
	Version string `priority:"1"`
}

type ExternalAddress struct {
	Router    string `priority:"4"`
	Addresses string `priority:"3"`
	Status    string `priority:"2"`
	Opts      string `priority:"1"`
}

func AppInfoSimple(app tsuru.App) SimpleApp {
	a := SimpleApp{
		Application: app.Name,
		Cluster:     app.Cluster,
		CreatedBy:   app.Owner,
		Description: app.Description,
		Plan:        app.Plan.Name,
		Platform:    app.Platform,
		Pool:        app.Pool,
		Quota:       fmt.Sprintf("%d/%d", app.Quota.Inuse, app.Quota.Limit),
		Router:      app.Router,
		RouterOpts:  app.Routeropts,
		Tags:        app.Tags,
		Teams:       app.Teams,
	}

	if app.Error != "" {
		a.Errors = []string{app.Error}
	}

	a.Routers = []string{}
	for _, r := range app.Routers {
		a.Routers = append(a.Routers, r.Addresses...)
	}

	a.UnitsCount, a.Units = renderUnitsSummary(app.Units, app.UnitsMetrics)

	a.ServiceInstances = make([]ServiceInstance, len(app.ServiceInstanceBinds))
	for i, si := range app.ServiceInstanceBinds {
		a.ServiceInstances[i] = ServiceInstance{
			Service:  si.Service,
			Instance: si.Instance,
			Plan:     si.Plan,
		}
	}

	a.ClusterInternalAddresses = make([]InternalAddress, len(app.InternalAddresses))
	for i, ia := range app.InternalAddresses {
		a.ClusterInternalAddresses[i] = InternalAddress{
			Domain:  ia.Domain,
			Port:    fmt.Sprintf("%d:%s", ia.Port, ia.Protocol),
			Process: ia.Process,
			Version: ia.Version,
		}
	}

	return a
}

func renderUnitsSummary(units []tsuru.Unit, metrics []tsuru.UnitMetrics) (unitCount int, parsedUnits []Unit) {
	type unitsKey struct {
		process  string
		version  int
		routable bool
	}
	groupedUnits := map[unitsKey][]tsuru.Unit{}
	for _, u := range units {
		routable := false
		if u.Routable != nil {
			routable = *u.Routable
		}
		key := unitsKey{process: u.Processname, version: int(u.Version), routable: routable}
		groupedUnits[key] = append(groupedUnits[key], u)
	}
	keys := make([]unitsKey, 0, len(groupedUnits))
	for key := range groupedUnits {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].version == keys[j].version {
			return keys[i].process < keys[j].process
		}
		return keys[i].version < keys[j].version
	})

	// titles := []string{"Process", "Ready", "Restarts", "Avg CPU (abs)", "Avg Memory"}
	// unitsTable := tablecli.NewTable()
	// tablecli.TableConfig.ForceWrap = false
	// unitsTable.Headers = tablecli.Row(titles)

	unitCount = len(units)
	parsedUnits = []Unit{}

	if len(units) == 0 {
		return
	}
	mapUnitMetrics := map[string]tsuru.UnitMetrics{}
	for _, unitMetric := range metrics {
		mapUnitMetrics[unitMetric.Id] = unitMetric
	}

	for _, key := range keys {
		summaryTitle := key.process
		if key.version > 0 {
			summaryTitle = fmt.Sprintf("%s (v%d)", key.process, key.version)
		}

		summaryUnits := groupedUnits[key]

		if !key.routable {
			summaryTitle = summaryTitle + " (unroutable)"
		}

		readyUnits := 0
		restarts := 0
		cpuTotal := resource.NewQuantity(0, resource.DecimalSI)
		memoryTotal := resource.NewQuantity(0, resource.BinarySI)

		for _, unit := range summaryUnits {
			if unit.Ready != nil && *unit.Ready {
				readyUnits += 1
			}

			if unit.Restarts != nil {
				restarts += *unit.Restarts
			}

			unitMetric := mapUnitMetrics[unit.Id]
			qt, err := resource.ParseQuantity(unitMetric.Cpu)
			if err == nil {
				cpuTotal.Add(qt)
			}
			qt, err = resource.ParseQuantity(unitMetric.Memory)
			if err == nil {
				memoryTotal.Add(qt)
			}
		}

		parsedUnits = append(parsedUnits, Unit{
			Process:   summaryTitle,
			Ready:     fmt.Sprintf("%d/%d", readyUnits, len(summaryUnits)),
			Restarts:  uint(restarts),
			AvgCPU:    fmt.Sprintf("%d%%", cpuTotal.MilliValue()/int64(10)/int64(len(summaryUnits))),
			AvgMemory: fmt.Sprintf("%vMi", memoryTotal.Value()/int64(1024*1024)/int64(len(summaryUnits))),
		})
	}
	return
}
