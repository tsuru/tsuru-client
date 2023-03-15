package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/google/shlex"
	"github.com/tsuru/tsuru/cmd"
)

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
	currLineSlice, err := shlex.Split(currLine)
	if err != nil {
		// incomplete quote, ignore autocomplete
		return true
	}

	if currLine[len(currLine)-1] == ' ' {
		currLineSlice = append(currLineSlice, "") // shlex trims.Split() the last empty space, so we need to add it back
	}

	m := buildManager("tsuru")
	fmt.Println(strings.Join(getSuggestions(m, currLineSlice), "\n"))
	return true
}
