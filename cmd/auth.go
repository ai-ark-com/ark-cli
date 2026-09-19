package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/ai-ark-com/ark-cli/internal/config"
)

// maxTokenLength bounds a piped token so a wrong stdin cannot be read forever.
const maxTokenLength = 4096

func newAuthCmd(build BuildInfo) *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Manage your API token",
	}
	c.AddCommand(newAuthLoginCmd(build), newAuthStatusCmd(build), newAuthLogoutCmd())
	return c
}

func newAuthLoginCmd(build BuildInfo) *cobra.Command {
	var tokenFlag string
	c := &cobra.Command{
		Use:   "login",
		Short: "Store your API token in the OS keychain",
		Long: `Store the API token in the OS keychain (macOS Keychain, Windows Credential
Manager, libsecret on Linux). Create a token at
https://app.ai-ark.com/settings/api-management/dashboard. It is checked
against the API before it is saved.

The prompt does not echo. Prefer it (or stdin) over --token, which ends up
in shell history.`,
		// A fixed message: cobra's NoArgs would echo the stray argument, and
		// a pasted token must never reach the terminal.
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageErrorf("login takes no arguments; you are prompted for the token (or use --token)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			token := strings.TrimSpace(tokenFlag)
			if token == "" {
				var err error
				if token, err = promptToken(cmd); err != nil {
					return err
				}
			}
			if token == "" {
				return usageErrorf("no token provided")
			}
			// Validate before persisting: a working token can read its balance.
			cl, err := clientWith(build, token)
			if err != nil {
				return err
			}
			if _, err := cl.Credits(cmd.Context()); err != nil {
				return fmt.Errorf("token rejected by API: %w", err)
			}
			if err := config.SaveToken(token); err != nil {
				return fmt.Errorf("saving token to keychain: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Token saved. Try: ark people preview --location Germany --seniority c_suite")
			if _, source, _ := config.Token(); source == config.SourceEnv {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: %s is set and takes precedence over the keychain\n", config.EnvToken)
			}
			return nil
		},
	}
	c.Flags().StringVar(&tokenFlag, "token", "", "token value (otherwise you are prompted)")
	return c
}

func newAuthStatusCmd(build BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show token source, API host, and credit balance",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			token, source, err := config.Token()
			if err != nil {
				return err
			}
			cl, err := clientWith(build, token)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Host:    %s\n", cl.BaseURL())
			fmt.Fprintf(out, "Token:   configured (%s)\n", source)
			res, err := cl.Credits(cmd.Context())
			if err != nil {
				return fmt.Errorf("reading balance: %w", err)
			}
			fmt.Fprintf(out, "Balance: %s credits\n", formatCredits(res.Body))
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored token from the keychain",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.DeleteToken(); err != nil {
				return fmt.Errorf("removing token from keychain: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Token removed.")
			return nil
		},
	}
}

// promptToken reads a token from the terminal without echoing it, or one
// line from a pipe when stdin is not a terminal.
func promptToken(cmd *cobra.Command) (string, error) {
	in := cmd.InOrStdin()
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		fmt.Fprint(cmd.ErrOrStderr(), "API token: ")
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", fmt.Errorf("reading token: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	line, err := bufio.NewReader(io.LimitReader(in, maxTokenLength)).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading token: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// formatCredits extracts the balance from the /credits response, documented
// as {"total": n}; other shapes fall back to the raw JSON.
func formatCredits(body json.RawMessage) string {
	var obj struct {
		Total *json.Number `json:"total"`
	}
	if err := json.Unmarshal(body, &obj); err == nil && obj.Total != nil {
		return obj.Total.String()
	}
	return strings.TrimSpace(string(body))
}
