package standards

import "github.com/tsuru/gnuflag"

// Each ShortFlag must be unique across the tsuru-client commands.
var (
	ShortFlagApp         string = "a"
	ShortFlagJob         string = "j"
	ShortFlagPlan        string = "p"
	ShortFlagTeam        string = "t"
	ShortFlagDescription string = "d"
	ShortFlagName        string = "n"
	ShortFlagOnlyName    string = "q"
	ShortFlagUser        string = "u"
	ShortFlagCNAME       string = "c"
)

// Flag is used to define common flag names across the tsuru-client commands.

var (
	// Resource Flags
	FlagApp      string = "app"
	FlagJob      string = "job"
	FlagPlan     string = "plan"
	FlagRouter   string = "router"
	FlagTeam     string = "team"
	FlagPool     string = "pool"
	FlagPlatform string = "platform"

	// Common properties flags
	FlagName        string = "name"
	FlagDescription string = "description"
	FlagTag         string = "tag"

	// Common action flags
	FlagNoRestart string = "no-restart"

	// Output Flags
	FlagJSON string = "json"
)

var Deprecated = "[DEPRECATED] "

func DeprecatedString(flagset *gnuflag.FlagSet, value *string, name, defaultValue, usage string) {
	flagset.StringVar(value, name, defaultValue, Deprecated+usage)
}

func DeprecatedBool(flagset *gnuflag.FlagSet, value *bool, name string, defaultValue bool, usage string) {
	flagset.BoolVar(value, name, defaultValue, Deprecated+usage)
}

func DeprecatedVar(flagset *gnuflag.FlagSet, value gnuflag.Value, name, usage string) {
	flagset.Var(value, name, Deprecated+usage)
}
