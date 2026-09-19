package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/catalog"
	"github.com/ai-ark-com/ark-cli/internal/client"
	"github.com/ai-ark-com/ark-cli/internal/output"
)

// newPeopleExportCmd is `ark people export`: submitting a job is the command
// itself; tracking it is done through the subcommands.
func newPeopleExportCmd(build BuildInfo) *cobra.Command {
	var (
		filters searchFlags
		size    int
		webhook string
		out     outputFlags
	)
	c := &cobra.Command{
		Use:   "export",
		Short: "Export up to 10,000 people with verified emails (async)",
		Long: `Search plus email finding as one async job. Same filters as
"ark people search"; --size is the total number of people to export.
0.5 credits per person plus 0.5 per found email. The job calls --webhook
when done. Poll "ark people export status" and read the results with
"ark people export results".

` + filterHelp + `

Examples:
  ark people export --industry "software development" --seniority c_suite --size 500 --webhook https://example.com/hook
  ark people export status 719aba5a-876f-4690-bb57-5157153836b4
  ark people export results 719aba5a-876f-4690-bb57-5157153836b4 -o csv > people.csv`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := out.options()
			if err != nil {
				return err
			}
			if size < 1 || size > maxExportSize {
				return usageErrorf("--size must be between 1 and %d", maxExportSize)
			}
			if !filters.dryRun {
				// The API requires a public HTTPS webhook (loopback is rejected).
				if err := client.ValidateWebhook(webhook); err != nil {
					return usageError(err)
				}
			}
			plan, err := filters.plan(cmd, 0, size, maxExportSize)
			if err != nil {
				return err
			}
			if err := plan.setWebhook(webhook, cmd.Flags().Changed("webhook")); err != nil {
				return err
			}
			return runSearch(cmd, build, plan, client.PathPeopleExport, filters.dryRun, opts)
		},
	}
	f := c.Flags()
	f.StringVar(&webhook, "webhook", "", "public HTTPS URL to POST to when the job completes (required)")
	f.IntVar(&size, "size", 100, fmt.Sprintf("total number of people to export (1-%d)", maxExportSize))
	filters.register(f, peopleScope)
	out.register(f, output.JSON)
	f.SortFlags = false

	c.AddCommand(
		newJobStatusCmd(build, client.JobExport, "an export"),
		newJobResultsCmd(build, client.JobExport, "an export"),
		newJobListCmd(build, client.JobExport, "export"),
		newJobNotifyCmd(build, client.JobExport, "an export"),
	)
	return c
}

// setWebhook stores the webhook on a typed plan, or overlays it on a raw
// --body (only when given explicitly or absent from the body).
func (p *searchPlan) setWebhook(webhook string, explicit bool) error {
	if p.req != nil {
		p.req.Webhook = webhook
		return nil
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(p.raw, &body); err != nil {
		return usageError(err)
	}
	if err := setJSONField(body, "webhook", webhook, explicit); err != nil {
		return err
	}
	raw, err := json.Marshal(body)
	p.raw = raw
	return err
}

// newPeopleEmailsCmd is `ark people emails`: email finding for a previous
// search's result set, identified by its trackId.
func newPeopleEmailsCmd(build BuildInfo) *cobra.Command {
	c := &cobra.Command{
		Use:   "emails",
		Short: "Find verified emails for a previous search (by track id)",
		Long: `Email finding for a previous search. Run "ark people search -o json",
take the trackId from the response and submit it here within 6 hours. A
trackId works once. 1 credit per found email, nothing for misses.`,
	}
	c.AddCommand(
		newEmailsFindCmd(build),
		newJobStatusCmd(build, client.JobEmailFinder, "an email-finder job"),
		newJobResultsCmd(build, client.JobEmailFinder, "an email-finder job"),
		newJobListCmd(build, client.JobEmailFinder, "email-finder"),
		newJobNotifyCmd(build, client.JobEmailFinder, "an email-finder job"),
	)
	return c
}

func newEmailsFindCmd(build BuildInfo) *cobra.Command {
	var (
		webhook string
		out     outputFlags
	)
	c := &cobra.Command{
		Use:     "find <track-id>",
		Short:   "Start email finding for a search result set",
		Example: `  ark people emails find 719aba5a-876f-4690-bb57-5157153836b4 --webhook https://example.com/hook`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateJobArgs(args[0], webhook); err != nil {
				return err
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.FindEmails(cmd.Context(), args[0], webhook)
			})
		},
	}
	webhookFlag(c, &webhook)
	out.register(c.Flags(), output.JSON)
	return c
}

func newJobStatusCmd(build BuildInfo, kind client.JobKind, what string) *cobra.Command {
	var out outputFlags
	c := &cobra.Command{
		Use:   "status <track-id>",
		Short: "Show the state and counters of " + what + " (free)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateJobArgs(args[0], ""); err != nil {
				return err
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.JobStatistics(cmd.Context(), kind, args[0])
			})
		},
	}
	out.register(c.Flags(), output.JSON)
	return c
}

func newJobResultsCmd(build BuildInfo, kind client.JobKind, what string) *cobra.Command {
	var (
		page pageFlags
		out  outputFlags
	)
	c := &cobra.Command{
		Use:   "results <track-id>",
		Short: "Fetch a page of results of " + what + " (free)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := page.validate(); err != nil {
				return err
			}
			if err := validateJobArgs(args[0], ""); err != nil {
				return err
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.JobResults(cmd.Context(), kind, args[0], page.page, page.size)
			})
		},
	}
	page.register(c.Flags(), 10)
	out.register(c.Flags(), output.Table)
	return c
}

func newJobListCmd(build BuildInfo, kind client.JobKind, what string) *cobra.Command {
	var (
		state    string
		refunded string
		sort     []string
		page     pageFlags
		out      outputFlags
	)
	c := &cobra.Command{
		Use:   "list",
		Short: "List your " + what + " submissions with their refund status (free)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := page.validate(); err != nil {
				return err
			}
			q := &client.SubmissionsQuery{Page: page.page, Size: page.size, Sort: splitCSV(sort)}
			if state != "" {
				canon, ok := catalog.Normalize(catalog.SubmissionStates, state)
				if !ok {
					return usageErrorf("--state must be pending or settled")
				}
				q.State = canon
			}
			if refunded != "" {
				v, err := strconv.ParseBool(refunded)
				if err != nil {
					return usageErrorf("--refunded must be true or false")
				}
				q.FullyRefunded = &v
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.JobSubmissions(cmd.Context(), kind, q)
			})
		},
	}
	f := c.Flags()
	f.StringVar(&state, "state", "", "only submissions in this state: pending or settled")
	f.StringVar(&refunded, "refunded", "", "only fully refunded (true) or not refunded (false) submissions")
	f.StringSliceVar(&sort, "sort", nil, "sort as property,asc|desc (repeatable)")
	page.register(f, 25)
	out.register(f, output.Table)
	return c
}

func newJobNotifyCmd(build BuildInfo, kind client.JobKind, what string) *cobra.Command {
	var (
		webhook string
		out     outputFlags
	)
	c := &cobra.Command{
		Use:   "resend-webhook <track-id>",
		Short: "Re-send the completion webhook of " + what + " (free)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateJobArgs(args[0], webhook); err != nil {
				return err
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.ResendWebhook(cmd.Context(), kind, args[0], webhook)
			})
		},
	}
	webhookFlag(c, &webhook)
	out.register(c.Flags(), output.JSON)
	return c
}

func webhookFlag(c *cobra.Command, webhook *string) {
	c.Flags().StringVar(webhook, "webhook", "", "public HTTPS URL to POST to when the job completes (required)")
	_ = c.MarkFlagRequired("webhook")
}

// validateJobArgs rejects a malformed track id or webhook before any network
// or keychain access, so mistakes fail fast and cheaply.
func validateJobArgs(trackID, webhook string) error {
	if err := client.ValidateID("track id", trackID); err != nil {
		return usageError(err)
	}
	if webhook != "" {
		if err := client.ValidateWebhook(webhook); err != nil {
			return usageError(err)
		}
	}
	return nil
}
