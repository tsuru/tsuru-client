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

func getSuggestions(m *cmd.Manager, currLine []string) []string {
	currLine = currLine[1:] // remove first "tsuru"
	spacedcurrLine := strings.Join(currLine, " ")

	counter := map[string]int{}
	for cmdName, comm := range m.Commands {
		if _, isDeprecated := comm.(*cmd.DeprecatedCommand); isDeprecated {
			continue
		}
		spacedCommand := strings.ReplaceAll(cmdName, "-", " ")
		splitCommand := strings.Split(cmdName, "-")

		if strings.HasPrefix(spacedCommand, spacedcurrLine) &&
			len(splitCommand) >= len(currLine) {
			counter[splitCommand[len(currLine)-1]] += 1
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
	currLine := os.Getenv("AUTOCOMPLETE_CURRENT_LINE")
	if currLine == "" {
		return false
	}
	currLineSlice := strings.Split(currLine, " ") // TODO: handle quotes

	m := buildManager("tsuru")
	fmt.Println(strings.Join(getSuggestions(m, currLineSlice), "\n"))
	return true
}
