// Package cmd wires the ark CLI command tree.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/client"
	"github.com/ai-ark-com/ark-cli/internal/config"
)

// BuildInfo carries version metadata injected at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// Exit codes, so scripts can react without parsing text.
const (
	exitOK        = 0
	exitError     = 1
	exitUsage     = 2
	exitNotFound  = 3 // HTTP 404: nothing matched
	exitNoCredits = 4 // HTTP 402: out of credits
	exitRateLimit = 5 // HTTP 429 after retries
	exitAuth      = 6 // HTTP 401/403: token rejected
)

// errUsage marks errors caused by invalid flags or arguments.
var errUsage = errors.New("invalid usage")

func usageErrorf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errUsage, fmt.Sprintf(format, args...))
}

func usageError(err error) error {
	return fmt.Errorf("%w: %w", errUsage, err)
}

func newRootCmd(build BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:   "ark",
		Short: "Command line client for the AI-Ark API",
		Long: `Command line client for the AI-Ark API. Calls are made with your own
token and billed to your account; the credits used are printed after
each command.

  ark auth login                       store your token (from app.ai-ark.com)
  ark people preview --location Germany --seniority c_suite
  ark people search  --location Germany --seniority c_suite --size 25
  ark company search --industry "software development" --employees 51-200
  ark catalog seniority                allowed values of a filter
  ark credits                          balance`,
		SilenceUsage:  true,
		SilenceErrors: true,
		// A fixed message for stray arguments: cobra's default would echo
		// them, and a token pasted in the wrong place must never reach the
		// terminal.
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageErrorf("unknown command; run 'ark --help'")
			}
			return cmd.Help()
		},
		CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError(err) })
	root.AddCommand(
		newAuthCmd(build),
		newPeopleCmd(build),
		newCompanyCmd(build),
		newListsCmd(build),
		newCatalogCmd(build),
		newCreditsCmd(build),
		newVersionCmd(build),
	)
	return root
}

// Execute runs the CLI and returns the process exit code.
func Execute(build BuildInfo) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := newRootCmd(build).ExecuteContext(ctx)
	if err == nil {
		return exitOK
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	return exitCode(err)
}

func exitCode(err error) int {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		if errors.Is(err, errUsage) {
			return exitUsage
		}
		return exitError
	}
	switch apiErr.Status {
	case 404:
		return exitNotFound
	case 402:
		return exitNoCredits
	case 429:
		return exitRateLimit
	case 401, 403:
		return exitAuth
	default:
		return exitError
	}
}

// newClient builds an API client from the resolved token and base URL.
func newClient(build BuildInfo) (*client.Client, error) {
	token, _, err := config.Token()
	if err != nil {
		return nil, err
	}
	return clientWith(build, token)
}

// clientWith builds an API client for an explicit token (used by `auth login`
// to validate a token before it is saved).
func clientWith(build BuildInfo, token string) (*client.Client, error) {
	base, err := config.BaseURL()
	if err != nil {
		return nil, err
	}
	if base != config.DefaultBaseURL {
		// Make an overridden host visible: the token is about to be sent there.
		fmt.Fprintf(os.Stderr, "notice: using API host %s (%s)\n", base, config.EnvBaseURL)
	}
	return client.New(base, token, build.Version), nil
}
