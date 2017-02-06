package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ajg/form"
	"github.com/pkg/errors"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/autoscale"
	"github.com/tsuru/tsuru/cmd"
	tsuruIo "github.com/tsuru/tsuru/io"
)

type ListAutoScaleHistoryCmd struct {
	fs   *gnuflag.FlagSet
	page int
}

func (c *ListAutoScaleHistoryCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-autoscale-list",
		Usage: "node-autoscale-list [--page/-p 1]",
		Desc:  "List node auto scale history.",
	}
}

func (c *ListAutoScaleHistoryCmd) Run(ctx *cmd.Context, client *cmd.Client) error {
	if c.page < 1 {
		c.page = 1
	}
	limit := 20
	skip := (c.page - 1) * limit
	u, err := cmd.GetURLVersion("1.3", fmt.Sprintf("/node/autoscale?skip=%d&limit=%d", skip, limit))
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var history []autoscale.Event
	if resp.StatusCode == 204 {
		ctx.Stdout.Write([]byte("There is no auto scales yet.\n"))
		return nil
	}
	err = json.NewDecoder(resp.Body).Decode(&history)
	if err != nil {
		return err
	}
	headers := cmd.Row([]string{"Start", "Finish", "Success", "Metadata", "Action", "Reason", "Error"})
	t := cmd.Table{Headers: headers}
	for i := range history {
		event := &history[i]
		t.AddRow(cmd.Row([]string{
			event.StartTime.Local().Format(time.Stamp),
			checkEndOfEvent(event),
			fmt.Sprintf("%t", event.Successful),
			event.MetadataValue,
			event.Action,
			event.Reason,
			event.Error,
		}))
	}
	t.LineSeparator = true
	ctx.Stdout.Write(t.Bytes())
	return nil
}

func checkEndOfEvent(event *autoscale.Event) string {
	if event.EndTime.IsZero() {
		return "in progress"
	}
	return event.EndTime.Local().Format(time.Stamp)
}

func (c *ListAutoScaleHistoryCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("with-flags", gnuflag.ContinueOnError)
		c.fs.IntVar(&c.page, "page", 1, "Current page")
		c.fs.IntVar(&c.page, "p", 1, "Current page")
	}
	return c.fs
}

type AutoScaleRunCmd struct {
	cmd.ConfirmationCommand
}

func (c *AutoScaleRunCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-autoscale-run",
		Usage: "node-autoscale-run [-y/--assume-yes]",
		Desc: `Run node auto scale checks once. This command will work even if [[docker:auto-
scale:enabled]] config entry is set to false. Auto scaling checks may trigger
the addition, removal or rebalancing of nodes, as long as these nodes were
created using an IaaS provider registered in tsuru.`,
	}
}

func (c *AutoScaleRunCmd) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	if !c.Confirm(context, "Are you sure you want to run auto scaling checks?") {
		return nil
	}
	u, err := cmd.GetURLVersion("1.3", "/node/autoscale/run")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", u, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	w := tsuruIo.NewStreamWriter(context.Stdout, nil)
	for n := int64(1); n > 0 && err == nil; n, err = io.Copy(w, response.Body) {
	}
	if err != nil {
		return err
	}
	unparsed := w.Remaining()
	if len(unparsed) > 0 {
		return errors.Errorf("unparsed message error: %s", string(unparsed))
	}
	return nil
}

type AutoScaleInfoCmd struct{}

func (c *AutoScaleInfoCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-autoscale-info",
		Usage: "node-autoscale-info",
		Desc: `Display the current configuration for tsuru autoscale,
including the set of rules and the current metadata filter.

The metadata filter is the value that defines which node metadata will be used
to group autoscale rules. A common approach is to use the "pool" as the
filter. Then autoscale can be configured for each matching rule value.`,
	}
}

func (c *AutoScaleInfoCmd) Run(context *cmd.Context, client *cmd.Client) error {
	config, err := c.getAutoScaleConfig(client)
	if err != nil {
		return err
	}
	if !config.Enabled {
		fmt.Fprintln(context.Stdout, "auto-scale is disabled")
		return nil
	}
	rules, err := c.getAutoScaleRules(client)
	if err != nil {
		return err
	}
	return c.render(context, config, rules)
}

func (c *AutoScaleInfoCmd) getAutoScaleConfig(client *cmd.Client) (*autoscale.Config, error) {
	u, err := cmd.GetURLVersion("1.3", "/node/autoscale/config")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var config autoscale.Config
	err = json.NewDecoder(resp.Body).Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *AutoScaleInfoCmd) getAutoScaleRules(client *cmd.Client) ([]autoscale.Rule, error) {
	u, err := cmd.GetURLVersion("1.3", "/node/autoscale/rules")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rules []autoscale.Rule
	err = json.NewDecoder(resp.Body).Decode(&rules)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *AutoScaleInfoCmd) render(context *cmd.Context, config *autoscale.Config, rules []autoscale.Rule) error {
	var table cmd.Table
	tableHeader := []string{
		"Pool",
		"Max container count",
		"Max memory ratio",
		"Scale down ratio",
		"Rebalance on scale",
		"Enabled",
	}
	table.Headers = tableHeader
	for _, rule := range rules {
		table.AddRow([]string{
			rule.MetadataFilter,
			strconv.Itoa(rule.MaxContainerCount),
			strconv.FormatFloat(float64(rule.MaxMemoryRatio), 'f', 4, 32),
			strconv.FormatFloat(float64(rule.ScaleDownRatio), 'f', 4, 32),
			strconv.FormatBool(!rule.PreventRebalance),
			strconv.FormatBool(rule.Enabled),
		})
	}
	fmt.Fprintf(context.Stdout, "Rules:\n%s", table.String())
	return nil
}

type AutoScaleSetRuleCmd struct {
	fs                 *gnuflag.FlagSet
	filterValue        string
	maxContainerCount  int
	maxMemoryRatio     float64
	scaleDownRatio     float64
	noRebalanceOnScale bool
	enable             bool
	disable            bool
}

func (c *AutoScaleSetRuleCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-autoscale-rule-set",
		Usage: "node-autoscale-rule-set [-f/--filter-value <pool name>] [-c/--max-container-count 0] [-m/--max-memory-ratio 0.9] [-d/--scale-down-ratio 1.33] [--no-rebalance-on-scale] [--enable] [--disable]",
		Desc:  "Creates or update an auto-scale rule. Using resources limitation (amount of container or memory usage).",
	}
}

func (c *AutoScaleSetRuleCmd) Run(context *cmd.Context, client *cmd.Client) error {
	if (c.enable && c.disable) || (!c.enable && !c.disable) {
		return errors.New("either --disable or --enable must be set")
	}
	rule := autoscale.Rule{
		MetadataFilter:    c.filterValue,
		MaxContainerCount: c.maxContainerCount,
		MaxMemoryRatio:    float32(c.maxMemoryRatio),
		ScaleDownRatio:    float32(c.scaleDownRatio),
		PreventRebalance:  c.noRebalanceOnScale,
		Enabled:           c.enable,
	}
	val, err := form.EncodeToValues(rule)
	if err != nil {
		return err
	}
	body := strings.NewReader(val.Encode())
	u, err := cmd.GetURLVersion("1.3", "/node/autoscale/rules")
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Rule successfully defined.")
	return nil
}

func (c *AutoScaleSetRuleCmd) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("node-autoscale-rule-set", gnuflag.ExitOnError)
		msg := "The pool name matching the rule. This is the unique identifier of the rule."
		c.fs.StringVar(&c.filterValue, "filter-value", "", msg)
		c.fs.StringVar(&c.filterValue, "f", "", msg)
		msg = "The maximum amount of containers on every node. Might be zero, which means no maximum value. Whenever this value is reached, tsuru will trigger a new auto scale event."
		c.fs.IntVar(&c.maxContainerCount, "max-container-count", 0, msg)
		c.fs.IntVar(&c.maxContainerCount, "c", 0, msg)
		msg = "The maximum memory usage per node. 0 means no limit, 1 means 100%. It is fine to use values greater than 1, which means that tsuru will overcommit memory in nodes. Keep in mind that container count has higher precedence than memory ratio, so if --max-container-count is defined, the value of --max-memory-ratio will be ignored."
		c.fs.Float64Var(&c.maxMemoryRatio, "max-memory-ratio", .0, msg)
		c.fs.Float64Var(&c.maxMemoryRatio, "m", .0, msg)
		msg = "The ratio for triggering an scale down event. The default value is 1.33, which mean that whenever it gets one third of the resource utilization (memory ratio or container count)."
		c.fs.Float64Var(&c.scaleDownRatio, "scale-down-ratio", 1.33, msg)
		c.fs.Float64Var(&c.scaleDownRatio, "d", 1.33, msg)
		msg = "A boolean flag indicating whether containers should NOT be rebalanced after running an scale. The default behavior is to always rebalance the containers."
		c.fs.BoolVar(&c.noRebalanceOnScale, "no-rebalance-on-scale", false, msg)
		msg = "A boolean flag indicating whether the rule should be enabled"
		c.fs.BoolVar(&c.enable, "enable", false, msg)
		msg = "A boolean flag indicating whether the rule should be disabled"
		c.fs.BoolVar(&c.disable, "disable", false, msg)
	}
	return c.fs
}

type AutoScaleDeleteRuleCmd struct {
	cmd.ConfirmationCommand
}

func (c *AutoScaleDeleteRuleCmd) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "node-autoscale-rule-remove",
		Usage: "node-autoscale-rule-remove [rule-name] [-y/--assume-yes]",
		Desc:  `Removes an auto-scale rule. The name of the rule may be omitted, which means "remove the default rule".`,
	}
}

func (c *AutoScaleDeleteRuleCmd) Run(context *cmd.Context, client *cmd.Client) error {
	var rule string
	confirmMsg := "Are you sure you want to remove the default rule?"
	if len(context.Args) > 0 {
		rule = context.Args[0]
		confirmMsg = fmt.Sprintf("Are you sure you want to remove the rule %q?", rule)
	}
	if !c.Confirm(context, confirmMsg) {
		return nil
	}
	u, err := cmd.GetURLVersion("1.3", "/node/autoscale/rules/"+rule)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Rule successfully removed.")
	return nil
}
