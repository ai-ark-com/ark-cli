package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ai-ark-com/ark-cli/internal/client"
	"github.com/ai-ark-com/ark-cli/internal/output"
)

const (
	// maxPageSize is the API's per-page cap for search and job results.
	maxPageSize = 100
	// maxExportSize is the API's cap on one Export People with Email job.
	maxExportSize = 10_000
	// maxInputFile bounds --body and --from-file so a wrong path cannot
	// swallow memory.
	maxInputFile = 8 << 20
)

// errNoFilters stops a search without filters before it is sent: it would
// match everything and bill for nothing useful.
var errNoFilters = usageErrorf("no filters given: add at least one filter (see --help), or pass --body")

// outputFlags are the rendering options shared by commands that print API data.
type outputFlags struct {
	format  string
	columns []string
}

func (o *outputFlags) register(fs *pflag.FlagSet, defaultFormat output.Format) {
	fs.StringVarP(&o.format, "output", "o", string(defaultFormat), "output format: table, json, or csv")
	fs.StringSliceVar(&o.columns, "columns", nil, "columns for table/csv output as dotted paths, e.g. id,profile.title")
}

func (o *outputFlags) options() (output.Options, error) {
	format, err := output.ParseFormat(o.format)
	if err != nil {
		return output.Options{}, usageError(err)
	}
	return output.Options{Format: format, Columns: o.columns}, nil
}

// pageFlags are the pagination options shared by paged endpoints.
type pageFlags struct {
	page int
	size int
}

func (p *pageFlags) register(fs *pflag.FlagSet, defaultSize int) {
	fs.IntVar(&p.page, "page", 0, "result page (0-based)")
	fs.IntVar(&p.size, "size", defaultSize, fmt.Sprintf("results per page (1-%d)", maxPageSize))
}

func (p *pageFlags) validate() error {
	if p.page < 0 {
		return usageErrorf("--page must be 0 or greater")
	}
	if p.size < 1 || p.size > maxPageSize {
		return usageErrorf("--size must be between 1 and %d", maxPageSize)
	}
	return nil
}

// call runs one API request and renders its result. Argument validation
// belongs before call, so bad input never touches the keychain or network.
func call(cmd *cobra.Command, build BuildInfo, out *outputFlags, fn func(*client.Client) (*client.Result, error)) error {
	opts, err := out.options()
	if err != nil {
		return err
	}
	cl, err := newClient(build)
	if err != nil {
		return err
	}
	res, err := fn(cl)
	if err != nil {
		return err
	}
	return present(cmd, res, opts)
}

// present renders a result. For the human-readable formats a summary line
// (total matches, credits used) goes to stderr, so stdout stays pipeable.
func present(cmd *cobra.Command, res *client.Result, opts output.Options) error {
	total, err := output.Render(cmd.OutOrStdout(), res.Body, opts)
	if err != nil || opts.Format == output.JSON {
		return err
	}
	var summary []string
	if total >= 0 {
		summary = append(summary, fmt.Sprintf("%d total matches", total))
	}
	if res.Credit != "" {
		summary = append(summary, strings.TrimPrefix(res.Credit, "-")+" credits used")
	}
	if len(summary) > 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "\n"+strings.Join(summary, ", "))
	}
	return nil
}

// printJSON writes v as indented JSON to the command's stdout.
func printJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// readInput reads a file, or stdin when path is "-", rejecting anything
// over maxInputFile.
func readInput(cmd *cobra.Command, flag, path string) ([]byte, error) {
	r := cmd.InOrStdin()
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("reading --%s: %w", flag, err)
		}
		defer f.Close()
		r = f
	}
	raw, err := io.ReadAll(io.LimitReader(r, maxInputFile+1))
	if err != nil {
		return nil, fmt.Errorf("reading --%s: %w", flag, err)
	}
	if len(raw) > maxInputFile {
		return nil, usageErrorf("--%s is larger than %d bytes", flag, maxInputFile)
	}
	return raw, nil
}

// readBody loads a JSON object from --body (a file, or "-" for stdin).
func readBody(cmd *cobra.Command, path string) (json.RawMessage, error) {
	raw, err := readInput(cmd, "body", path)
	if err != nil {
		return nil, err
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' || !json.Valid(raw) {
		return nil, usageErrorf("--body must be a JSON object")
	}
	return raw, nil
}

// splitCSV splits comma-separated values that may also be given as repeated
// flags, trimming whitespace and dropping empties.
func splitCSV(values []string) []string {
	var out []string
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// setJSONField sets key in a decoded JSON object when force is set or the
// object lacks the key.
func setJSONField(body map[string]json.RawMessage, key string, value any, force bool) error {
	if _, present := body[key]; present && !force {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	body[key] = encoded
	return nil
}
