# ark

`ark` is the official command line client for the AI-Ark API. It turns people
search, company search, job post search, email and phone finding, personality
analysis and credit checks into terminal commands that print JSON, a table or
CSV.

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
v=1.1.1
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
| `ark job-posts search` | Job Post Search | 0.1 per result |
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
`ark job-posts search` matches differently, see [Job posts](#job-posts).

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

# companies by their email stack (from the MX records of their domains)
ark company search --mail-platform google,microsoft --exclude-mail-security proofpoint,mimecast
ark people search --seniority c_suite --company-mail-platform google
```

`--dry-run` prints the JSON body the CLI would send, so you can copy it into
a direct API call or check a filter combination for free.

### Email stack

`--mail-platform` is the provider that hosts a company's email (`google` is
Google Workspace, `microsoft` is Microsoft 365, `self_hosted` is a mail
server of its own). `--mail-security` is the security gateway in front of
it (`proofpoint`, `mimecast`, ...); most companies have none. A company
with several domains matches when any of its domains does. People commands
take the same filters as `--company-mail-platform` and
`--company-mail-security`. Known values: `ark catalog mail-platforms` and
`ark catalog mail-security`. Each company in a result carries the detected
values per domain in its `mail` field (`ark people preview` leaves it out).

### Job posts

`ark job-posts search` searches job posts, newest first, to find companies
that are hiring for a role or ask for a tool right now. It costs 0.1
credits per result.

```bash
ark job-posts search --title "sales manager" --location Germany,Austria --posted 2026-10-01..
ark job-posts search --text salesforce --hiring-model direct --seniority mid_senior,director
ark job-posts search --technology Salesforce --salary 80k-150k --exclude-company-domain sap.com
ark job-posts search --company-domain sap.com --expires 2026-11-01..
```

| Flag | Meaning |
| --- | --- |
| `--title` | job title of the post |
| `--description` | words or phrases in the text of the post |
| `--text` | words or phrases in the title or in the text |
| `--location` | country, state, city, continent or metro area of the job |
| `--seniority`, `--employment-type` | `ark catalog job-seniority`, `ark catalog job-employment-types` |
| `--industry` | industry shown on the post: `ark catalog job-industries` |
| `--technology` | a technology named in the post, or a category: `ark catalog job-technologies`, `ark catalog job-technology-categories` |
| `--language` | language the post is written in: `ark catalog job-languages` |
| `--salary 80k-150k` | yearly salary in USD that the post's range overlaps; `any` keeps posts that show a salary |
| `--hiring-model direct\|agency` | posted by the hiring company itself, or by an agency for a client |
| `--posted`, `--expires`, `--collected` | day bands as `YYYY-MM-DD..YYYY-MM-DD`; either side may be empty |
| `--id` | job post ids, to fetch or `--exclude-id` posts you already have |
| `--company-id`, `--company-domain`, `--company-name`, `--company-linkedin` | the hiring company |

Three things differ from people and company search:

- `--match` defaults to `word` (the exact phrase, other words allowed around
  it). `smart` matches every word in any order, `strict` the whole value.
  `--description` and `--text` take `word` or `smart` only.
- Seniority, industries and technologies are the values shown on job posts,
  not the lists of people and company search. Use the `job-*` catalogs.
- There is no suppression list and no preview; a search pages through at
  most 10,000 posts.

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
| 5 | rate limited (HTTP 429); the error names the limit that is used up and when to retry |
| 6 | API key rejected (HTTP 401), or the plan has no API access (HTTP 403) |

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
  6) echo "check the API key and the plan"; exit 1 ;;
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

### Reach the people at companies that are hiring

Every job post carries the id of the hiring company. Search the posts, then
search the people at those companies. `--hiring-model direct` leaves out
agency posts, which name the agency as the company:

```bash
ark job-posts search --title "sales manager" --location Germany --posted 2026-10-01.. \
  --hiring-model direct --size 100 -o json | jq -r '.content[].company.id' | sort -u > companies.txt
ark people preview --current-company-id "$(paste -sd, companies.txt)" --seniority vp,director --department sales
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

How many requests you may send depends on your subscription plan. On a
credit plan every request uses credits from your balance and there is one
limit: requests per second. An Unlimited plan uses no credits and adds two
long windows, one of a few hours and one of seven days, counted per kind of
request. All API keys of a
workspace share the same limits.

With table and csv output, the summary on stderr shows what the call cost
and where the limits stand. On an Unlimited plan:

```text
7907 total matches, 0 credits used
limits: 340 of 1000 requests per 4 hours used (resets in 1h 2m); 910 of 5000 requests per 7 days used (resets in 5d 13h)
```

A call on an Unlimited plan is not charged and shows `0 credits used`. The
per-second window is left out: it starts over before your next command.

When the API answers 429 and asks for a wait of up to ten seconds, the
command waits and retries, up to three times. A longer wait fails at once
with exit code 5 and says when to retry:

```text
error: API error (HTTP 429): API rate limit exceeded for this endpoint. All 1000 requests per 4 hours are used. Retry in 1h 17m (at 12:00:00 UTC)
```

A request sent before that time is rejected again. A rejected request costs
no credits.

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
`cmd/testdata/search_paths.json` (people and company search) and
`cmd/testdata/job_post_paths.json` (job post search) hold every field of the
API request schemas; `TestFlagsReachEverySpecPath` and
`TestJobPostFlagsReachEverySpecPath` fail when one is not reachable from
flags. Regenerate the files from the OpenAPI spec
(<https://docs.ai-ark.com/reference>) when the API changes.

CI: lint, gitleaks, govulncheck, tests on Linux, macOS and Windows.

## Release

Push a `X.Y.Z` tag (no `v` prefix, e.g. `1.0.0`); GoReleaser builds all
platforms and attaches the binaries.

## Next

- [API Reference](https://docs.ai-ark.com/reference)
- [Filter reference for the CLI](https://github.com/ai-ark-com/ark-cli/blob/main/docs/FILTERS.md)
- [MCP Server](https://docs.ai-ark.com/reference/mcp)
- [Source and issues](https://github.com/ai-ark-com/ark-cli)
