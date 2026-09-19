package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/client"
)

func newPeopleCmd(build BuildInfo) *cobra.Command {
	c := &cobra.Command{
		Use:   "people",
		Short: "Search, preview, export, and enrich people",
	}
	c.AddCommand(
		newPeoplePreviewCmd(build),
		newPeopleSearchCmd(build),
		newPeopleExportCmd(build),
		newPeopleEmailsCmd(build),
		newPeopleEnrichCmd(build),
		newPeopleReverseLookupCmd(build),
		newPeoplePhoneCmd(build),
		newPeoplePersonalityCmd(build),
	)
	return c
}

func newPeoplePreviewCmd(build BuildInfo) *cobra.Command {
	return newSearchCmd(build, searchSpec{
		use:   "preview",
		short: "Preview a people search for a flat 1 credit per page",
		long: `Same as "people search" but 1 credit per page regardless of size. Use it
to check a query before paying per result. Last names are masked and there
are no emails or phones, but each result has the real person id (works with
"ark people enrich") and has_* flags that say what data exists.

` + filterHelp + `

` + peopleFilterExamples("preview"),
		scope:       peopleScope,
		path:        client.PathPeoplePreview,
		defaultSize: 25,
	})
}

func newPeopleSearchCmd(build BuildInfo) *cobra.Command {
	return newSearchCmd(build, searchSpec{
		use:   "search",
		short: "Search people (0.5 credits per result)",
		long: `Search people. Each result is the full profile plus the company, without
emails or phones. To get emails, pass the trackId from the JSON response to
"ark people emails find" within 6 hours, or use "ark people enrich" per
person.

` + filterHelp + `

` + peopleFilterExamples("search"),
		scope:       peopleScope,
		path:        client.PathPeopleSearch,
		defaultSize: 10,
	})
}
