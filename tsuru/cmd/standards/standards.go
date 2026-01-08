package standards

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
	ShortFlagPool        string = "o"
	ShortFlagTag         string = "g"
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
	FlagUser     string = "user"

	// Common properties flags
	FlagName        string = "name"
	FlagDescription string = "description"
	FlagTag         string = "tag"

	// Common action flags
	FlagNoRestart string = "no-restart"

	// Output Flags
	FlagOnlyName string = "only-name"
	FlagJSON     string = "json"
)
