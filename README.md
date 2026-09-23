# ark

`ark` is the official command line client for the AI-Ark API. It turns people
search, company search, email and phone finding, personality analysis and
credit checks into terminal commands that print JSON, a table or CSV.

It is a single static binary. Every call goes to `https://api.ai-ark.com` with
your own API key and is billed to your account like any other API call.

## Requirements

| | |
| --- | --- |
| Platforms | Linux, macOS, Windows (amd64 and arm64) |
| Command | `ark` |
| Source | [github.com/ai-ark-com/ark-cli](https://github.com/ai-ark-com/ark-cli) |
| Build from source | Go 1.26 or newer |

## Install

Download the archive for your platform from the
[releases page](https://github.com/ai-ark-com/ark-cli/releases), unpack it and
put `ark` on your `PATH`:

```bash
# Linux / macOS. Archives: ark_<version>_<linux|darwin>_<amd64|arm64>.tar.gz
v=1.0.0
curl -sLO https://github.com/ai-ark-com/ark-cli/releases/download/$v/ark_${v}_linux_amd64.tar.gz
curl -sLO https://github.com/ai-ark-com/ark-cli/releases/download/$v/checksums.txt
sha256sum --check --ignore-missing checksums.txt
tar xzf ark_${v}_linux_amd64.tar.gz && sudo mv ark /usr/local/bin/
```

On Windows unpack `ark_<version>_windows_amd64.zip` and add the folder to
`PATH`. Every release ships `checksums.txt` with the SHA-256 of each archive.

With a Go toolchain:

```bash
go install github.com/ai-ark-com/ark-cli@latest
# the binary is named ark-cli; rename or alias it to ark
```

Confirm the install:

```bash
ark version
```

## Authentication

Create an API key in the
[API Management Dashboard](https://app.ai-ark.com/settings/api-management/dashboard),
then store it:

```bash
ark auth login
```

The prompt does not echo. The key is checked against the API before it is
saved to the OS keychain (macOS Keychain, Windows Credential Manager, libsecret
on Linux). The key is never written to a file and never printed.

Check what the CLI will use:

```bash
ark auth status
```

```text
Host:    https://api.ai-ark.com
Token:   configured (keychain)
Balance: 1250 credits
```

Remove the stored key:

```bash
ark auth logout
```

### Credential precedence

1. `ARK_API_TOKEN` environment variable
2. OS keychain (`ark auth login`)

The environment variable always wins. Use it in CI, in containers and on
servers without a desktop session, where no keychain is available.

```bash
export ARK_API_TOKEN=your-key    # in CI, from a secret store
ark credits
```

> 📘 Piping the key
>
> `ark auth login` also reads the key from stdin, so a secret manager can feed
> it without the key ever appearing in shell history:
> `op read op://vault/ai-ark/token | ark auth login`

## Options

Flags belong to each command. These appear on most of them:

| Flag | Applies to | Meaning |
| --- | --- | --- |
| `-o, --output table\|json\|csv` | everything that prints data | render format (see Output and scripting) |
| `--columns a,b.c` | table, csv | dotted paths to show instead of the preset |
| `--page N`, `--size N` | searches, job results, submissions | page (0-based) and page size (1-100) |
| `--match smart\|word\|strict` | searches | how text filters match (default `smart`) |
| `--dry-run` | searches, export | print the request body, call nothing |
| `--body file.json` | searches, export | send a hand written body (`-` for stdin) |

| Variable | Purpose |
| --- | --- |
| `ARK_API_TOKEN` | API key; takes precedence over the keychain |
| `ARK_BASE_URL` | API host, https only (default `https://api.ai-ark.com`); the host is printed on stderr when set |

## First call

Preview a query before paying per result. A preview costs 1 credit per page,
whatever the page size:

```bash
ark people preview --location Germany --seniority c_suite,vp --industry "software development"
```

Run it for real (0.5 credits per result):

```bash
ark people search --location Germany --seniority c_suite,vp --industry "software development" --size 25
```

Look up allowed values and your balance (free):

```bash
ark catalog                        # the catalogs
ark catalog seniority              # values of one filter
ark catalog industries --search health
ark credits
```

## Commands

| Command | API endpoint | Credits |
| --- | --- | --- |
| `ark people preview` | People Preview | 1 per page |
| `ark people search` | People Search | 0.5 per result |
| `ark people export` | Export People with Email (async, up to 10,000) | 0.5 per person + 0.5 per found email |
| `ark people export status\|results\|list\|resend-webhook` | Export People Statistics, Results, Submissions, Resend Webhook | free |
| `ark people emails find <track-id>` | Find Emails by Track ID | 1 per found email |
| `ark people emails status\|results\|list\|resend-webhook` | Email Finder Statistics, Results, Submissions, Resend Webhook | free |
| `ark people enrich <id or LinkedIn URL>` | Export Single Person with Email (`--clay` for V2) | 1 if an email is found |
| `ark people reverse-lookup <email or phone>` | Reverse People Lookup | 0.5 |
| `ark people phone` | Mobile Phone Finder (`--clay` for V2) | 5 per found number |
| `ark people personality <LinkedIn URL>` | Personality Analysis | 4 |
| `ark company search` | Company Search (`--lookalike` included) | 0.1 per result |
| `ark lists create\|update` | Create or Update a List | free |
| `ark credits` | Fetch Your Credit | free |
| `ark catalog [name]` | filter values (industries, technologies, departments, ...) | free |
| `ark auth login\|status\|logout` | key management | free |
| `ark version` | version | free |

`ark <command> --help` lists every flag with its meaning.

## Filters

Every filter of the search request body has a flag; the full mapping is in
[FILTERS.md](https://github.com/ai-ark-com/ark-cli/blob/main/docs/FILTERS.md).
Flags are ANDed. Comma separated values inside one flag are ORed. Each list
filter comes in four forms:

| Flag | Meaning |
| --- | --- |
| `--x a,b` | a or b |
| `--exclude-x a,b` | neither a nor b |
| `--all-x a,b` | a and b |
| `--all-exclude-x a,b` | not (a and b) |

Text filters (`--title`, `--industry`, `--skill`, `--technology`, `--product`,
`--keyword`, `--company-name`, ...) honour `--match smart|word|strict`.

Ranges are written `51-200`, `1000+`, `-50`, with `k`, `m` and `b` suffixes
(`--revenue 1m-10m`). Tenure uses years and months (`--tenure-company 1y6m+`),
growth uses signed percentages (`--headcount-growth marketing:10..15:SIX`).

```bash
# people
ark people search --current-title "head of sales" --seniority head,vp,director
ark people search --department engineering_technical --skill kubernetes,terraform --match word
ark people search --keyword "carbon accounting" --keyword-source HEADLINE,SUMMARY
ark people search --badge HIRING --language german --exclude-location Berlin

# people filtered by employer: company flags take a company- prefix
ark people search --company-domain stripe.com,adyen.com --seniority c_suite
ark people search --industry "financial services" --employees 51-200,201-500 --founded 2015+

# companies
ark company search --industry "software development" --location Germany --revenue 10m-100m
ark company search --technology hubspot --exclude-technology salesforce --near 52.52,13.405 --radius 25
ark company search --lookalike raisin.com,https://www.linkedin.com/company/n26 --location Germany
```

`--dry-run` prints the JSON body the CLI would send, so you can copy it into
a direct API call or check a filter combination for free.

## Output and scripting

### Output formats

| `--output` | Result |
| --- | --- |
| `table` | default for searches and job results; a preset of columns, long cells truncated |
| `json` | default for single records and job submissions; the full API response, pretty printed |
| `csv` | same columns as the table, nothing truncated, safe for spreadsheets |

Table and CSV flatten each result into dotted paths (`profile.first_name`,
`company.summary.name`). Pick your own with `--columns`:

```bash
ark people search --location Germany --seniority c_suite -o csv \
  --columns id,profile.first_name,profile.last_name,profile.title,company.summary.name > leads.csv
```

Responses that are not a result page (a single person, a job submission) are
always printed as JSON.

### stdout and stderr

stdout carries data only. With `table` and `csv` the match count and the
credits used go to stderr, so stdout can be redirected. Errors are one line on
stderr, prefixed with `error:`:

```text
error: token rejected by API: 401 Unauthorized
```

### Exit codes

| Code | Meaning |
| --- | --- |
| 0 | success |
| 1 | any other error (network, server, keychain) |
| 2 | bad flags or arguments; a search without filters is refused locally |
| 3 | not found (HTTP 404): no email or phone for that person, unknown track id |
| 4 | out of credits (HTTP 402) |
| 5 | rate limited (HTTP 429) after three retries |
| 6 | API key rejected (HTTP 401/403) |

### Piping to jq

Use `-o json` and take what you need:

```bash
ark people search --location Germany --seniority c_suite -o json | jq -r '.content[].id'
ark people search --location Germany --seniority c_suite -o json | jq -r '.trackId'
ark credits -o json | jq .total
```

### Branching on exit codes

```bash
#!/usr/bin/env bash
ark people enrich https://www.linkedin.com/in/john-doe -o json > person.json
code=$?
case $code in
  0) echo "found: $(jq -r '.email.output[0].address' person.json)" ;;
  3) echo "no email for this person" ;;
  4) echo "out of credits"; exit 1 ;;
  6) echo "check the API key"; exit 1 ;;
  *) echo "failed with exit code $code"; exit 1 ;;
esac
```

> 🚧 `set -e`
>
> With `set -e` a non-zero exit aborts the script before your `case` runs.
> Capture the code as shown, or use `if ! ark ...; then`.

### Async jobs: webhook and polling

Bulk export and email finding are asynchronous. Submitting returns a
`trackId`; the API POSTs to your `--webhook` URL when the job is done. The
webhook must be a public HTTPS URL (loopback and http are rejected). You
don't have to serve it: poll `status` until the state is `DONE`, then read
`results`.

```bash
track=$(ark people export --industry "software development" --seniority c_suite \
  --size 500 --webhook https://example.com/hook | jq -r .trackId)

until ark people export status "$track" | jq -e '.state == "DONE"' >/dev/null; do
  sleep 30
done
ark people export results "$track" --size 100 -o csv > people.csv
```

`results` answers HTTP 409 (exit code 1) while the job is still running.

## Common workflows

### Size a market before spending credits

Preview costs 1 credit per page regardless of size and returns the total
match count. Narrow the query until the count is right, then search:

```bash
ark people preview --industry "software development" --location Germany
ark people preview --industry "software development" --location Germany --seniority c_suite,vp
ark people preview --industry "software development" --location Germany --seniority c_suite,vp --employees 51-200
ark people search  --industry "software development" --location Germany --seniority c_suite,vp --employees 51-200 --size 100
```

### Find emails for a search

Search, then submit its `trackId` to the email finder. The trackId works once
and expires after 6 hours. 1 credit per found email, nothing for misses.

```bash
ark people search --location Germany --seniority c_suite --size 100 -o json > search.json
ark people emails find "$(jq -r .trackId search.json)" --webhook https://example.com/hook
ark people emails status <track-id>                    # until "state": "DONE"
ark people emails results <track-id> -o csv > emails.csv
```

### Search and find emails in one job

Up to 10,000 people per job. 0.5 credits per person plus 0.5 per found email.

```bash
ark people export --industry retail --location "United States" --seniority director \
  --size 2000 --webhook https://example.com/hook
ark people export list                                 # your submissions and their refunds
```

### Enrich one person at a time

For CRMs, spreadsheets and Clay. `--clay` selects the V2 endpoints, which
answer HTTP 200 with `"data": null` instead of exit code 3 when nothing is
found:

```bash
ark people enrich https://www.linkedin.com/in/john-doe
ark people enrich 592439b8-13c7-e31c-296e-b6e2f7339aeb --clay
ark people phone --domain stripe.com --name "Patrick Collison"
ark people reverse-lookup ada@example.com
ark people reverse-lookup +14155550100
ark people personality https://www.linkedin.com/in/john-doe
```

### Leave out people you already contacted

Suppression lists hold ids of people or companies to exclude from a search.
They are free, hold up to 10,000 ids and expire after 24 hours.

```bash
ark people search --industry retail -o json | jq -r '.content[].id' \
  | ark lists create --type people --from-file -        # prints the list id
ark people search --industry retail --exclude-list <list-id>
ark lists update <list-id> --values <id>,<id>            # append; --replace overwrites
```

### Use a hand written request body

Anything the API accepts can be sent as is. The same filter and size checks
apply:

```bash
ark company search --dry-run --industry fintech --location Berlin > body.json
ark company search --body body.json
cat body.json | ark company search --body -
```

## Rate limits

The request rate depends on your subscription plan. When the API answers
429, the command retries up to three times, honouring `Retry-After`, then
exits with code 5.

## Shell completion

`ark` ships completion for bash, zsh, fish and PowerShell:

```bash
source <(ark completion bash)                 # current shell
ark completion zsh > "${fpath[1]}/_ark"       # zsh, then restart the shell
ark completion fish > ~/.config/fish/completions/ark.fish
ark completion powershell | Out-String | Invoke-Expression
```

`ark completion <shell> --help` shows where to install it permanently.

## Security notes

- The key lives only in the OS keychain or `ARK_API_TOKEN`; the CLI never
  logs, prints or stores it elsewhere.
- Redirects are refused, so the key cannot be forwarded to another host.
- `ARK_BASE_URL` must be https; when set, the host is shown on stderr with
  every call.
- CSV cells starting with `=`, `+`, `-` or `@` are quoted so spreadsheets do
  not run them as formulas.

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

## Next

- [API Reference](https://docs.ai-ark.com/reference)
- [Filter reference for the CLI](https://github.com/ai-ark-com/ark-cli/blob/main/docs/FILTERS.md)
- [MCP Server](https://docs.ai-ark.com/reference/mcp)
- [Source and issues](https://github.com/ai-ark-com/ark-cli)
