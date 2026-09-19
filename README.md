# ark

Command line client for the [AI-Ark API](https://docs.ai-ark.com/): people and
company search, verified emails, mobile numbers, personality analysis.

Single static binary, no server side. It uses your own API token and every call
is billed to your account like any other API call. The token lives in the OS
keychain; nothing else is stored locally.

## Install

Download the binary for your platform from the [releases](../../releases) page
and put it on your `PATH`, or, with Go 1.26+ installed:

```bash
go install github.com/ai-ark-com/ark-cli@latest   # installs $(go env GOPATH)/bin/ark-cli
```

Build from a checkout:

```bash
git clone https://github.com/ai-ark-com/ark-cli.git && cd ark-cli
go build -o ark .        # or: make docker-build (no Go toolchain needed)
```

## Quick start

```bash
# Create a token at https://app.ai-ark.com/settings/api-management/dashboard
ark auth login              # prompts for the token, checks it, stores it in the keychain
                            # (in CI or on a server without a keychain, set ARK_API_TOKEN instead)

# Check what a query matches. 1 credit per page, whatever the page size.
ark people preview --location Germany --seniority c_suite,vp --industry "software development"

# Run it for real. 0.5 credits per result.
ark people search --location Germany --seniority c_suite,vp --industry "software development" --size 25

ark catalog seniority                   # allowed values of a filter
ark catalog industries --search health
ark credits                             # balance
```

## Commands

| Command | Endpoint | Cost |
| --- | --- | --- |
| `ark people preview` | People Preview | 1 / page |
| `ark people search` | People Search | 0.5 / result |
| `ark people export` | Export People with Email (async, up to 10,000) | 0.5 / person + 0.5 / found email |
| `ark people export status\|results\|list\|resend-webhook` | export statistics, results, submissions, notify | free |
| `ark people emails find <track-id>` | Find Emails by Track ID | 1 / found email |
| `ark people emails status\|results\|list\|resend-webhook` | email finder statistics, results, submissions, notify | free |
| `ark people enrich <id or LinkedIn URL>` | Export Single Person with Email (`--clay` for v2) | 1 if an email is found, else 0 |
| `ark people reverse-lookup <email or phone>` | Reverse People Lookup | 0.5 |
| `ark people phone` | Mobile Phone Finder (`--clay` for v2) | 5 / found number |
| `ark people personality <LinkedIn URL>` | Personality Analysis | 4 |
| `ark company search` | Company Search, `--lookalike` included | 0.1 / result |
| `ark lists create\|update` | Create or Update a List | free |
| `ark credits` | Fetch Your Credit | free |
| `ark catalog [name]` | filter values (industries, technologies, departments, ...) | free |
| `ark auth login\|status\|logout` | token management | - |
| `ark version` | version | - |

`ark <command> --help` lists the flags.

## Filters

Every filter of the API has a flag. [docs/FILTERS.md](docs/FILTERS.md) maps each
field of the request body to its flag; a test fails if a field is left out.

Flags are ANDed. Comma separated values inside one flag are ORed. List flags
come in four forms:

| Flag | Meaning |
| --- | --- |
| `--x a,b` | a or b |
| `--exclude-x a,b` | neither a nor b |
| `--all-x a,b` | a and b |
| `--all-exclude-x a,b` | not (a and b) |

The `--all-*` forms are hidden from `--help` to keep it short. Text filters
(`--title`, `--industry`, `--skill`, `--technology`, `--product`, `--keyword`,
`--company-name`, ...) take `--match smart|word|strict`. `smart` is the default
and also matches related terms.

```bash
# people
ark people search --current-title "head of sales" --seniority head,vp,director
ark people search --department engineering_technical --skill kubernetes,terraform --match word
ark people search --keyword "carbon accounting" --keyword-source HEADLINE,SUMMARY,WORK_HISTORY_DESCRIPTION
ark people search --badge HIRING --language german --exclude-location Berlin
ark people search --all-skill python,"machine learning" --tenure-company 2y+ --followers 5k+ --studying

# people, filtered by employer (company-level flags carry a company- prefix here)
ark people search --company-domain stripe.com,adyen.com --seniority c_suite
ark people search --industry "financial services" --employees 51-200,201-500 --founded 2015+ --funding-type SERIES_A
ark people search --company-id 931fc1a0-1ed5-baf8-0dd2-a068a0fd0430   # id from ark company search

# companies
ark company search --industry "software development" --location Germany --revenue 10m-100m
ark company search --technology hubspot --exclude-technology salesforce --near 52.52,13.405 --radius 25
ark company search --lookalike raisin.com,https://www.linkedin.com/company/n26 --location Germany
ark company search --employee-title "head of sales" --employee-seniority head,vp
ark company search --funding-type SERIES_A --funding-total 5m+ --funding-date 2024-01-01.. --headcount-growth engineering:20..:TWELVE
```

Ranges: `51-200`, `1000+`, `-50`; `k`, `m`, `b` suffixes work (`--revenue 1m-10m`).
`--revenue` and `--founded` also take `any` or `none`. Tenure, education dates,
follower counts, headcount and growth per department, funding amounts and
dates, and the radius search are described in
[docs/FILTERS.md](docs/FILTERS.md).

`ark catalog <name>` lists the allowed values for `--seniority`,
`--department`, `--industry`, `--technology`, `--company-type`, `--language`,
`--badge`, `--funding-type` and the keyword sources. Locations take countries,
states, cities and regions such as `Europe`.

`--dry-run` prints the request body without calling the API. `--body file.json`
(or `--body -` for stdin) sends a hand written body instead; the same filter
and size checks apply. Request format: <https://docs.ai-ark.com/reference>.

## Emails and exports

Email finding and bulk export are asynchronous. The API wants a public HTTPS
`--webhook` URL to call when the job is done (loopback and http URLs are
rejected). You don't have to serve it: poll `status` until it says `DONE`, then
read `results`. Each job has a `trackId`.

```bash
# (examples use jq to pull ids out of JSON)
# search, then find emails for exactly that result set.
# The trackId from a search works once and expires after 6 hours.
ark people search --location Germany --seniority c_suite -o json > search.json
ark people emails find "$(jq -r .trackId search.json)" --webhook https://example.com/hook
ark people emails status <track-id>                  # until "state": "DONE"
ark people emails results <track-id> -o csv > emails.csv

# search + emails in one job, up to 10,000 people
ark people export --industry "software development" --seniority c_suite --size 1000 --webhook https://example.com/hook
ark people export status <track-id>                  # results return 409 until DONE
ark people export results <track-id> --page 0 --size 100

# one person at a time (CRM, spreadsheets, Clay). --clay returns HTTP 200
# with "data": null instead of exit code 3 when nothing is found.
ark people enrich https://www.linkedin.com/in/john-doe
ark people phone --domain stripe.com --name "Patrick Collison"
```

## Suppression lists

Lists hold ids of people or companies to leave out of a search, e.g. people you
already contacted.

```bash
ark people search --industry retail -o json | jq -r '.content[].id' \
  | ark lists create --type people --from-file -            # prints the list id
ark people search --industry retail --exclude-list <list-id>
ark lists update <list-id> --values <person-id>,<person-id>  # append; --replace overwrites
```

Lists are free, hold up to 10,000 ids and expire after 24 hours.

## Output

`-o table` (default for searches and job results), `-o json` (default for single
records and job submissions) or `-o csv`. Table and CSV flatten each result
into dotted paths and show a default set of columns; pick your own with
`--columns id,profile.title,company.summary.name`. JSON is the full document,
pipe it to `jq`. With table and csv the match count and credits used go to
stderr, so stdout can be redirected. CSV cells starting with `=`, `+`, `-` or
`@` get a leading quote so spreadsheets don't run them as formulas.

Exit codes:

| Code | Meaning |
| --- | --- |
| 0 | ok |
| 1 | error |
| 2 | bad flags or arguments |
| 3 | not found (HTTP 404, e.g. no email or phone for that person) |
| 4 | out of credits (HTTP 402) |
| 5 | rate limited (HTTP 429) after retries |
| 6 | token rejected (HTTP 401/403) |

## Configuration

| Variable | Purpose |
| --- | --- |
| `ARK_API_TOKEN` | token; takes precedence over the keychain (CI, scripts) |
| `ARK_BASE_URL` | API host, https only (default `https://api.ai-ark.com`) |

`ark auth status` shows which token source is in use. The token is never
written to a file, never printed and never sent on redirects. When
`ARK_BASE_URL` is set the host is printed on stderr with every call.

On Linux the keychain is libsecret over D-Bus (a desktop session). On servers
and in containers use `ARK_API_TOKEN`.

## Rate limits

5 requests per second per token. A 429 is retried up to three times (honouring
`Retry-After`). A search without any filter is refused locally.

## Development

```bash
make check          # vet, golangci-lint, tests (Go 1.26+, golangci-lint v2)
make docker-check   # same, inside Docker
make docker-fmt     # gofumpt + goimports
make docker-build   # build inside Docker
```

Filter flags are generated from the table in `cmd/facets.go`.
`cmd/testdata/search_paths.json` holds every field of the API request schema
and `TestFlagsReachEverySpecPath` fails when one is not reachable from flags.
Regenerate the file from the OpenAPI spec (<https://docs.ai-ark.com/reference>)
when the API changes.

CI: lint, gitleaks, govulncheck, tests on Linux, macOS and Windows.

## Release

Push a `X.Y.Z` tag (no `v` prefix, e.g. `1.0.0`); GoReleaser builds all
platforms and attaches the binaries.
