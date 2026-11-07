package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/client"
)

func newJobPostsCmd(build BuildInfo) *cobra.Command {
	c := &cobra.Command{
		Use:   "job-posts",
		Short: "Search job posts to find companies that are hiring",
	}
	c.AddCommand(newJobPostSearchCmd(build))
	return c
}

func newJobPostSearchCmd(build BuildInfo) *cobra.Command {
	return newSearchCmd(build, searchSpec{
		use:   "search",
		short: "Search job posts (0.1 credits per result)",
		long: `Search job posts, newest first. A post tells you which company is hiring
for a role or asks for a tool right now. Each result has the id of the
hiring company (company.id), which "ark people search --current-company-id"
accepts to find the people to contact there.

` + jobFilterHelp + `

Examples:
  ark job-posts search --title "sales manager" --location Germany,Austria --posted 2026-10-01..
  ark job-posts search --text salesforce --hiring-model direct --seniority mid_senior,director
  ark job-posts search --technology Salesforce --salary 80k-150k --exclude-company-domain sap.com
  ark job-posts search --company-domain sap.com --expires 2026-11-01.. -o json | jq -r '.content[].url'
  ark job-posts search --title "account executive" -o json | jq -r '.content[].company.id' | sort -u`,
		scope:       jobScope,
		path:        client.PathJobPostSearch,
		defaultSize: 10,
	})
}

const jobFilterHelp = `Flags are ANDed; comma separated values in one flag are ORed. Each list
flag also has --exclude-x, --all-x (all values must match) and
--all-exclude-x; the all-* forms are hidden from this help. Text filters
take --match word (default, the exact phrase), smart (every word, in any
order) or strict (the whole value); --description and --text take word or
smart only. Job post seniority, industries and technologies have their own
lists: "ark catalog". Full list of filters: docs/FILTERS.md.

--dry-run prints the request body without calling the API. --body sends a
hand written JSON body instead (format: docs.ai-ark.com/reference).`
