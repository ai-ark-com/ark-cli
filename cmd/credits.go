package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/client"
	"github.com/ai-ark-com/ark-cli/internal/output"
)

func newCreditsCmd(build BuildInfo) *cobra.Command {
	var out outputFlags
	c := &cobra.Command{
		Use:   "credits",
		Short: "Show your remaining credit balance (free)",
		Long: `Credits left on the account. Free.

Prices:
  people preview          1 per page          company search       0.1 per result
  people search           0.5 per result      people emails find   1 per found email
  people export           0.5 per person + 0.5 per found email
  people enrich           1 if an email is found, else 0
  people phone            5 per found number  people reverse-lookup 0.5 per request
  people personality      4 per request       lists                free`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if out.format != string(output.JSON) {
				cl, err := newClient(build)
				if err != nil {
					return err
				}
				res, err := cl.Credits(cmd.Context())
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s credits\n", formatCredits(res.Body))
				return nil
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.Credits(cmd.Context())
			})
		},
	}
	out.register(c.Flags(), output.Table)
	return c
}
