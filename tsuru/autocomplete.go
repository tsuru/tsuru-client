package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/tsuru/tsuru/cmd"
)

func getAllTopics(m *cmd.Manager) []string {
	var result []string
	for key := range m.Commands {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func getSuggestions(m *cmd.Manager, compLine []string) []string {
	compLine = compLine[1:] // remove first "tsuru"
	spacedCompLine := strings.Join(compLine, " ")

	counter := map[string]int{}
	for cmdName, comm := range m.Commands {
		if _, isDeprecated := comm.(*cmd.DeprecatedCommand); isDeprecated {
			continue
		}
		spacedCommand := strings.ReplaceAll(cmdName, "-", " ")
		splitCommand := strings.Split(cmdName, "-")

		if strings.HasPrefix(spacedCommand, spacedCompLine) &&
			len(splitCommand) >= len(compLine) {
			counter[splitCommand[len(compLine)-1]] += 1
		}
	}

	var result []string
	for topic := range counter {
		result = append(result, topic)
	}
	sort.Strings(result)
	return result
}

func handleAutocomplete() bool {
	compLine := os.Getenv("COMP_LINE")
	if compLine == "" {
		return false
	}
	compLineSlice := strings.Split(compLine, " ")

	m := buildManager("tsuru")
	fmt.Println(strings.Join(getSuggestions(m, compLineSlice), "\n"))
	return true
}
