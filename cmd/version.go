package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd(build BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "ark %s (commit %s, built %s, %s/%s)\n",
				build.Version, build.Commit, build.Date, runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}
}
