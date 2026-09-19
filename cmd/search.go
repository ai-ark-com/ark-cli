package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/output"
)

// searchSpec describes one search command so the people and company
// variants share a constructor.
type searchSpec struct {
	use, short, long string
	scope            filterScope
	path             string
	defaultSize      int
}

func newSearchCmd(build BuildInfo, spec searchSpec) *cobra.Command {
	var (
		filters searchFlags
		page    pageFlags
		out     outputFlags
	)
	c := &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		Long:  spec.long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := out.options()
			if err != nil {
				return err
			}
			if err := page.validate(); err != nil {
				return err
			}
			plan, err := filters.plan(cmd, page.page, page.size, maxPageSize)
			if err != nil {
				return err
			}
			return runSearch(cmd, build, plan, spec.path, filters.dryRun, opts)
		},
	}
	f := c.Flags()
	filters.register(f, spec.scope)
	page.register(f, spec.defaultSize)
	out.register(f, output.Table)
	f.SortFlags = false
	return c
}

// runSearch sends a plan to path (or prints it with dry-run) and renders
// the response.
func runSearch(cmd *cobra.Command, build BuildInfo, plan *searchPlan, path string, dryRun bool, opts output.Options) error {
	body, err := plan.body()
	if err != nil {
		return err
	}
	if dryRun {
		return printJSON(cmd, body)
	}
	cl, err := newClient(build)
	if err != nil {
		return err
	}
	res, err := cl.Search(cmd.Context(), path, body)
	if err != nil {
		return err
	}
	return present(cmd, res, opts)
}

// body returns the JSON to send: the raw --body, or the typed request.
func (p *searchPlan) body() (json.RawMessage, error) {
	if p.raw != nil {
		return p.raw, nil
	}
	return json.Marshal(p.req)
}

const filterHelp = `Flags are ANDed; comma separated values in one flag are ORed. Each list
flag also has --exclude-x, --all-x (all values must match) and
--all-exclude-x; the all-* forms are hidden from this help. Text filters
take --match smart (default, matches related terms too), word or strict.
Allowed values: "ark catalog". Full list of filters: docs/FILTERS.md.

--dry-run prints the request body without calling the API. --body sends a
hand written JSON body instead (format: docs.ai-ark.com/reference).`

func peopleFilterExamples(verb string) string {
	return fmt.Sprintf(`Examples:
  ark people %[1]s --location Germany --seniority c_suite,vp
  ark people %[1]s --title "marketing manager" --industry "software development" --employees 51-200
  ark people %[1]s --company-domain stripe.com --department engineering_technical -o csv
  ark people %[1]s --all-skill kubernetes,terraform --match word --tenure-company 2y+
  ark people %[1]s --body request.json`, verb)
}
