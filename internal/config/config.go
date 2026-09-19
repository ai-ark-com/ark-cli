// Package config resolves the API token and base URL for the CLI.
//
// Token resolution order (first hit wins):
//  1. ARK_API_TOKEN environment variable (CI and scripts).
//  2. OS keychain (macOS Keychain, Windows Credential Manager, libsecret),
//     written by `ark auth login`. The token never lands in a plaintext file.
//
// Base URL defaults to production and can be overridden with ARK_BASE_URL,
// e.g. to point at a staging host. Only HTTPS is accepted (plain HTTP for
// loopback addresses only) so the token is never sent in clear text.
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	// DefaultBaseURL is the production developer-portal host.
	DefaultBaseURL = "https://api.ai-ark.com"

	// EnvToken is the environment variable that overrides the keychain token.
	EnvToken = "ARK_API_TOKEN" //nolint:gosec // the variable's name, not a credential
	// EnvBaseURL is the environment variable that overrides the API host.
	EnvBaseURL = "ARK_BASE_URL"

	keyringService = "ark-cli"
	keyringUser    = "api-token"
)

// Token sources, as reported by Token.
const (
	SourceEnv      = "environment (" + EnvToken + ")"
	SourceKeychain = "keychain"
)

// ErrNoToken is returned when no token can be found in the environment or keychain.
var ErrNoToken = errors.New("no API token found: run `ark auth login`, or set " + EnvToken)

// BaseURL returns the configured API host without a trailing slash.
func BaseURL() (string, error) {
	raw := strings.TrimSpace(os.Getenv(EnvBaseURL))
	if raw == "" {
		return DefaultBaseURL, nil
	}
	return validateBaseURL(raw)
}

// validateBaseURL accepts "scheme://host[:port][/path]" and rejects anything
// that could leak the token (plain HTTP to a remote host, embedded
// credentials) or silently break request paths (query strings, fragments).
func validateBaseURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%s: %w", EnvBaseURL, err)
	}
	switch {
	case u.Host == "" || u.Opaque != "":
		return "", fmt.Errorf("%s must look like https://host[:port]", EnvBaseURL)
	case u.User != nil:
		return "", fmt.Errorf("%s must not contain credentials", EnvBaseURL)
	case u.RawQuery != "" || u.Fragment != "":
		return "", fmt.Errorf("%s must not contain a query string or fragment", EnvBaseURL)
	case u.Scheme == "https":
	case u.Scheme == "http" && isLoopback(u.Hostname()):
	default:
		return "", fmt.Errorf("%s must use https (http is allowed for localhost only)", EnvBaseURL)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Token returns the API token and where it came from (SourceEnv or
// SourceKeychain), or ErrNoToken if none is configured.
func Token() (token, source string, err error) {
	if v := strings.TrimSpace(os.Getenv(EnvToken)); v != "" {
		return v, SourceEnv, nil
	}
	secret, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", "", ErrNoToken
		}
		return "", "", fmt.Errorf("reading keychain: %w", err)
	}
	if secret = strings.TrimSpace(secret); secret == "" {
		return "", "", ErrNoToken
	}
	return secret, SourceKeychain, nil
}

// SaveToken stores the token in the OS keychain.
func SaveToken(token string) error {
	return keyring.Set(keyringService, keyringUser, strings.TrimSpace(token))
}

// DeleteToken removes the token from the OS keychain. Removing an absent
// token is not an error.
func DeleteToken() error {
	err := keyring.Delete(keyringService, keyringUser)
	if err != nil && errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
