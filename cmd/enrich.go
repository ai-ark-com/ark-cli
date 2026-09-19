package cmd

import (
	"net/mail"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/client"
	"github.com/ai-ark-com/ark-cli/internal/output"
)

// clayFlag registers --clay, which selects the v2 variants of the single-
// person endpoints: they answer HTTP 200 with data: null on a miss instead
// of 404, as row-by-row enrichment tools expect.
func clayFlag(c *cobra.Command, clay *bool) {
	c.Flags().BoolVar(clay, "clay", false, "Clay-compatible endpoint: HTTP 200 with data: null instead of 404 on a miss")
}

func newPeopleEnrichCmd(build BuildInfo) *cobra.Command {
	var (
		clay bool
		out  outputFlags
	)
	c := &cobra.Command{
		Use:   "enrich <person-id | linkedin-url>",
		Short: "Export one person with a verified email (1 credit if found)",
		Long: `Full profile of one person plus a verified email. Give the AI-Ark id
(from a search or preview result) or the LinkedIn profile URL. 1 credit when
an email is found, 0 otherwise; a miss exits with code 3.`,
		Example: `  ark people enrich 592439b8-13c7-e31c-296e-b6e2f7339aeb
  ark people enrich https://www.linkedin.com/in/john-doe --clay`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &client.ExportSingleRequest{}
			switch {
			case isURL(args[0]):
				req.URL = args[0]
			case client.ValidateID("person id", args[0]) == nil:
				req.ID = args[0]
			default:
				return usageErrorf("%q is neither an AI-Ark id nor a LinkedIn URL", args[0])
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.ExportSingle(cmd.Context(), req, clay)
			})
		},
	}
	clayFlag(c, &clay)
	out.register(c.Flags(), output.JSON)
	return c
}

func newPeopleReverseLookupCmd(build BuildInfo) *cobra.Command {
	var out outputFlags
	c := &cobra.Command{
		Use:   "reverse-lookup <email | phone>",
		Short: "Find the person behind an email address or phone number (0.5 credits)",
		Example: `  ark people reverse-lookup ada@example.com
  ark people reverse-lookup +14155550100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contact, ok := parseContact(args[0])
			if !ok {
				return usageErrorf("%q is neither an email address nor a phone number like +14155550100", args[0])
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.ReverseLookup(cmd.Context(), contact)
			})
		},
	}
	out.register(c.Flags(), output.JSON)
	return c
}

func newPeoplePhoneCmd(build BuildInfo) *cobra.Command {
	var (
		req  client.PhoneFinderRequest
		clay bool
		out  outputFlags
	)
	c := &cobra.Command{
		Use:   "phone",
		Short: "Find a person's mobile number (5 credits if found)",
		Long: `Mobile number of a person, given the LinkedIn URL or the company domain
plus full name. 5 credits when a number is found, 0 otherwise.`,
		Example: `  ark people phone --linkedin https://www.linkedin.com/in/john-doe
  ark people phone --domain stripe.com --name "Patrick Collison"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch {
			case req.LinkedIn != "" && (req.Domain != "" || req.Name != ""):
				return usageErrorf("use either --linkedin or --domain with --name, not both")
			case req.LinkedIn != "" && !isURL(req.LinkedIn):
				return usageErrorf("--linkedin must be a full profile URL")
			case req.LinkedIn == "" && (req.Domain == "" || req.Name == ""):
				return usageErrorf("provide --linkedin, or both --domain and --name")
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.PhoneFinder(cmd.Context(), &req, clay)
			})
		},
	}
	f := c.Flags()
	f.StringVar(&req.LinkedIn, "linkedin", "", "LinkedIn profile URL")
	f.StringVar(&req.Domain, "domain", "", "company domain (use with --name)")
	f.StringVar(&req.Name, "name", "", "person's full name (use with --domain)")
	clayFlag(c, &clay)
	out.register(f, output.JSON)
	return c
}

func newPeoplePersonalityCmd(build BuildInfo) *cobra.Command {
	var out outputFlags
	c := &cobra.Command{
		Use:   "personality <linkedin-url>",
		Short: "DISC / OCEAN personality analysis with outreach guidance (4 credits)",
		Long: `DISC and OCEAN (Big Five) scores for a profile, with an archetype label and
outreach advice (email tone, what to say, what to avoid). 4 credits.`,
		Example: `  ark people personality https://www.linkedin.com/in/john-doe`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isURL(args[0]) {
				return usageErrorf("%q is not a LinkedIn profile URL", args[0])
			}
			return call(cmd, build, &out, func(cl *client.Client) (*client.Result, error) {
				return cl.Personality(cmd.Context(), args[0])
			})
		},
	}
	out.register(c.Flags(), output.JSON)
	return c
}

var phonePattern = regexp.MustCompile(`^\+?\d{7,15}$`)

// parseContact accepts an email address or a phone number (digits with an
// optional leading +; spaces, dashes and parentheses are ignored) and
// returns it in the form the API expects.
func parseContact(s string) (string, bool) {
	if addr, err := mail.ParseAddress(s); err == nil && addr.Name == "" {
		return addr.Address, true
	}
	digits := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(s))
	if phonePattern.MatchString(digits) {
		return digits, true
	}
	return "", false
}

func isURL(s string) bool {
	s = strings.ToLower(s)
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://")
}
