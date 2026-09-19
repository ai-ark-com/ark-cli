package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/client"
)

func newCompanyCmd(build BuildInfo) *cobra.Command {
	c := &cobra.Command{
		Use:   "company",
		Short: "Search companies",
	}
	c.AddCommand(newCompanySearchCmd(build))
	return c
}

func newCompanySearchCmd(build BuildInfo) *cobra.Command {
	return newSearchCmd(build, searchSpec{
		use:   "search",
		short: "Search companies (0.1 credits per result)",
		long: `Search companies. Each result has a company id that "ark people search
--company-id" accepts. --lookalike takes up to 5 domains or LinkedIn company
URLs and returns similar companies, narrowed by any other filters.

` + filterHelp + `

Examples:
  ark company search --industry "software development" --location Germany --employees 51-200
  ark company search --domain stripe.com,adyen.com -o json | jq '.content[].id'
  ark company search --lookalike raisin.com --location Germany
  ark company search --technology hubspot --founded 2015+ --funding-type SERIES_A,SERIES_B
  ark company search --employee-title "head of sales" --employee-seniority head,vp
  ark company search --near 52.52,13.405 --radius 25 --revenue 10m+`,
		scope:       companyScope,
		path:        client.PathCompanySearch,
		defaultSize: 10,
	})
}
