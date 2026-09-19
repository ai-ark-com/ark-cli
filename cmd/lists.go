package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/client"
	"github.com/ai-ark-com/ark-cli/internal/output"
)

// maxListValues is the API's cap on one list.
const maxListValues = 10_000

func newListsCmd(build BuildInfo) *cobra.Command {
	c := &cobra.Command{
		Use:   "lists",
		Short: "Create suppression lists of people or company ids (free)",
		Long: `A list holds up to 10,000 person or company ids. Pass its id to a search
with --exclude-list to leave those out, e.g. people already contacted.
Lists are free and expire 24 hours after creation.`,
	}
	c.AddCommand(newListsCreateCmd(build), newListsUpdateCmd(build))
	return c
}

// listValueFlags collect ids from --values and/or --from-file.
type listValueFlags struct {
	values []string
	file   string
}

func (l *listValueFlags) register(c *cobra.Command) {
	c.Flags().StringSliceVar(&l.values, "values", nil, "ids to store (comma-separated, repeatable)")
	c.Flags().StringVar(&l.file, "from-file", "", "read ids from this file, one per line (\"-\" for stdin)")
}

func (l *listValueFlags) collect(cmd *cobra.Command) ([]string, error) {
	ids := splitCSV(l.values)
	for _, id := range ids {
		if err := client.ValidateID("--values", id); err != nil {
			return nil, usageError(err)
		}
	}
	if l.file != "" {
		fromFile, err := readIDs(cmd, l.file)
		if err != nil {
			return nil, err
		}
		ids = append(ids, fromFile...)
	}
	if len(ids) == 0 {
		return nil, usageErrorf("no ids given: use --values or --from-file")
	}
	if len(ids) > maxListValues {
		return nil, usageErrorf("a list holds at most %d ids, got %d", maxListValues, len(ids))
	}
	return ids, nil
}

func newListsCreateCmd(build BuildInfo) *cobra.Command {
	var (
		kind   string
		values listValueFlags
		out    outputFlags
	)
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new list",
		Example: `  ark lists create --type people --values 294ce93e-...,8c83763c-...
  ark people search --location Germany -o json | jq -r '.content[].id' | ark lists create --type people --from-file -`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			listType, err := parseListType(kind)
			if err != nil {
				return err
			}
			ids, err := values.collect(cmd)
			if err != nil {
				return err
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.SaveList(cmd.Context(), &client.ListRequest{Type: listType, Values: ids})
			})
		},
	}
	c.Flags().StringVar(&kind, "type", "", "list type: people or companies (required)")
	_ = c.MarkFlagRequired("type")
	values.register(c)
	out.register(c.Flags(), output.JSON)
	return c
}

func newListsUpdateCmd(build BuildInfo) *cobra.Command {
	var (
		replace bool
		values  listValueFlags
		out     outputFlags
	)
	c := &cobra.Command{
		Use:     "update <list-id>",
		Short:   "Append ids to a list, or replace its contents",
		Example: `  ark lists update 6f52adbf-... --values 0d089797-... --replace`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.ValidateID("list id", args[0]); err != nil {
				return usageError(err)
			}
			ids, err := values.collect(cmd)
			if err != nil {
				return err
			}
			mode := client.ListAppend
			if replace {
				mode = client.ListReplace
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.SaveList(cmd.Context(), &client.ListRequest{ID: args[0], Values: ids, Mode: mode})
			})
		},
	}
	c.Flags().BoolVar(&replace, "replace", false, "overwrite the list instead of appending")
	values.register(c)
	out.register(c.Flags(), output.JSON)
	return c
}

func parseListType(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "people", "person", client.ListPeople:
		return client.ListPeople, nil
	case "companies", "company", client.ListCompanies:
		return client.ListCompanies, nil
	default:
		return "", usageErrorf("--type must be people or companies, got %q", s)
	}
}

// readIDs reads one id per line from --from-file, ignoring blank lines and
// #-comments, and validates each id.
func readIDs(cmd *cobra.Command, path string) ([]string, error) {
	raw, err := readInput(cmd, "from-file", path)
	if err != nil {
		return nil, err
	}
	var ids []string
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := client.ValidateID("--from-file", line); err != nil {
			return nil, usageError(err)
		}
		ids = append(ids, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading --from-file: %w", err)
	}
	return ids, nil
}
