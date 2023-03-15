package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/google/shlex"
	"github.com/tsuru/tsuru/cmd"
)

func getSuggestions(m *cmd.Manager, fullCurrLine []string) []string {
	if len(fullCurrLine) <= 1 {
		return []string{} // first argument is expected to be the binary name
	}

	currLine := fullCurrLine[1:] // remove first "tsuru"
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

// examples for AUTOCOMPLETE_CURRENT_LINE: "tsuru ", "tsuru app s"
func handleAutocomplete() bool {
	currLine := os.Getenv("AUTOCOMPLETE_CURRENT_LINE")
	if currLine == "" {
		return false
	}
	currLineSlice, err := shlex.Split(currLine)
	if err != nil || len(currLineSlice) == 0 {
		return true
	}

	if currLine[len(currLine)-1] == ' ' {
		currLineSlice = append(currLineSlice, "") // shlex.Split() trims the last empty space, so we need to add it back
	}

	m := buildManager("tsuru")
	fmt.Println(strings.Join(getSuggestions(m, currLineSlice), "\n"))
	return true
}
