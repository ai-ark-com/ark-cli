package cmd

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/catalog"
	"github.com/ai-ark-com/ark-cli/internal/output"
)

func newCatalogCmd(build BuildInfo) *cobra.Command {
	var (
		search string
		limit  int
		out    outputFlags
	)
	c := &cobra.Command{
		Use:   "catalog [name]",
		Short: "List the values a filter accepts (free)",
		Long: `Allowed values of the search filters. Without a name, lists the catalogs.
Industries, technologies and departments are downloaded from ai-ark.com;
the rest are fixed by the API.

Examples:
  ark catalog
  ark catalog seniority
  ark catalog industries --search health
  ark catalog technologies --search crm --limit 20
  ark catalog departments -o json`,
		Args: cobra.MaximumNArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			names := make([]string, 0, len(catalog.All()))
			for _, c := range catalog.All() {
				names = append(names, c.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := out.options()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return listCatalogs(cmd, opts.Format)
			}
			cat, ok := catalog.Lookup(args[0])
			if !ok {
				return usageErrorf("unknown catalog %q; run 'ark catalog' to list them", args[0])
			}
			entries, err := catalog.NewFetcher(build.Version).Entries(cmd.Context(), cat)
			if err != nil {
				return err
			}
			entries = catalog.Filter(entries, search)
			if limit > 0 && len(entries) > limit {
				entries = entries[:limit]
			}
			return printEntries(cmd, entries, opts.Format)
		},
	}
	f := c.Flags()
	f.StringVar(&search, "search", "", "keep only values containing this text (case-insensitive)")
	f.IntVar(&limit, "limit", 0, "print at most this many values (0 = all)")
	out.register(f, output.Table)
	return c
}

func listCatalogs(cmd *cobra.Command, format output.Format) error {
	all := catalog.All()
	if format == output.JSON {
		type item struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Source      string `json:"source,omitempty"`
		}
		items := make([]item, 0, len(all))
		for _, c := range all {
			items = append(items, item{Name: c.Name, Description: c.Description, Source: c.URL})
		}
		return printJSON(cmd, items)
	}
	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tDESCRIPTION")
	for _, c := range all {
		fmt.Fprintf(tw, "%s\t%s\n", c.Name, c.Description)
	}
	return tw.Flush()
}

func printEntries(cmd *cobra.Command, entries []catalog.Entry, format output.Format) error {
	hasCounts := len(entries) > 0 && entries[0].Count >= 0
	switch format {
	case output.JSON:
		type item struct {
			Value string `json:"value"`
			Count *int64 `json:"companies,omitempty"`
		}
		items := make([]item, 0, len(entries))
		for _, e := range entries {
			it := item{Value: e.Value}
			if e.Count >= 0 {
				n := e.Count
				it.Count = &n
			}
			items = append(items, it)
		}
		return printJSON(cmd, items)
	case output.CSV:
		w := csv.NewWriter(cmd.OutOrStdout())
		header := []string{"value"}
		if hasCounts {
			header = append(header, "companies")
		}
		if err := w.Write(header); err != nil {
			return err
		}
		for _, e := range entries {
			row := []string{e.Value}
			if hasCounts {
				row = append(row, strconv.FormatInt(e.Count, 10))
			}
			if err := w.Write(row); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	default:
		tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
		for _, e := range entries {
			if hasCounts {
				fmt.Fprintf(tw, "%s\t%d\n", e.Value, e.Count)
			} else {
				fmt.Fprintln(tw, e.Value)
			}
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "\n%d values\n", len(entries))
		return nil
	}
}
