# Filter checklist

Every field of the search request body (people search, preview, export and
company search) and the flag that sets it. Taken from the OpenAPI spec at
<https://docs.ai-ark.com/reference>. `cmd/testdata/search_paths.json` holds
the 297 leaf paths; `TestFlagsReachEverySpecPath` fails if a flag stops
covering one.

## How list filters map to flags

Each list filter has four flags:

| Flag | Wire path | Meaning |
| --- | --- | --- |
| `--<facet> a,b` | `<facet>.any.include` | match a OR b |
| `--exclude-<facet> a,b` | `<facet>.any.exclude` | exclude a OR b |
| `--all-<facet> a,b` | `<facet>.all.include` | match a AND b |
| `--all-exclude-<facet> a,b` | `<facet>.all.exclude` | exclude only when a AND b match |

Text filters (marked *matched*) send `{mode, content}`; `mode` comes from
`--match smart|word|strict`. Keyword filters add `sources` from the
`--<facet>-source` flag. Values are comma separated and flags can be
repeated. The `--all-*` flags are hidden from `--help`.

## Person filters (`ark people preview|search|export`)

| Wire path | Flag | Kind / values |
| --- | --- | --- |
| `contact.location` | `--location` | plain |
| `contact.seniority` | `--seniority` | enum: `ark catalog seniority` |
| `contact.departmentAndFunction` | `--department` | `ark catalog departments` |
| `contact.experience.latest.title` | `--title` | matched |
| `contact.experience.current.title` | `--current-title` | matched |
| `contact.experience.previous.title` | `--previous-title` | matched |
| `contact.experience.current.duration.currentCompany` | `--tenure-company 2y-5y` | band in years/months |
| `contact.experience.current.duration.currentJob` | `--tenure-job 1y6m+` | band in years/months |
| `contact.experience.current.duration.total` | `--tenure-total -10y` | band in years/months |
| `contact.fullName` | `--name` | matched |
| `contact.skill` | `--skill` | matched |
| `contact.certification` | `--certification` | matched |
| `contact.linkedin` | `--linkedin` | plain (full profile URLs) |
| `contact.socialMediaLink` | `--social-link` | plain (full profile URLs) |
| `contact.company.latest` | `--company-id` | AI-Ark company UUIDs |
| `contact.company.current` | `--current-company-id` | AI-Ark company UUIDs |
| `contact.company.previous` | `--previous-company-id` | AI-Ark company UUIDs |
| `contact.language.any/all` | `--language` | matched, enum: `ark catalog languages` |
| `contact.language.range` | `--language-count 3+` | band |
| `contact.profileBadge` | `--badge` | enum: `ark catalog badges` |
| `contact.socialMedia` | `--social-media` | enum: `ark catalog social-media` |
| `contact.keyword` | `--keyword` + `--keyword-source` | keyword; sources from `ark catalog keyword-sources`, max 5 |
| `contact.education.school` | `--school` | AI-Ark company UUIDs |
| `contact.education.degree` | `--degree` | matched |
| `contact.education.fieldOfStudy` | `--field-of-study` | matched |
| `contact.education.date.start/end` | `--education-date 2015-2025` | year band |
| `contact.education.date.present` | `--studying` | boolean |
| `contact.socialMediaFollower.linkedin.followers` | `--followers 1k-5k` | bands, repeatable |
| `contact.socialMediaFollower.linkedin.connections` | `--connections 500+` | bands, repeatable |
| `lists.people_id.exclude` | `--exclude-list` | list UUIDs from `ark lists create` |

## Company filters

Used as `account.*` by every people command (flags prefixed `company-`
where the short name means the person) and by `ark company search` under
the short names shown in the second column.

| Wire path | People flag | Company flag | Kind / values |
| --- | --- | --- | --- |
| `account.domain` | `--company-domain` | `--domain` | plain |
| `account.name` | `--company-name` | `--name` | matched |
| `account.linkedin` | `--company-linkedin` | `--linkedin` | plain (full URLs) |
| `account.url` | `--company-url` | `--url` | matched (domain, www, URL, or LinkedIn URL) |
| `account.socialMediaLink` | `--company-social-link` | `--social-link` | plain |
| `account.phoneNumber` | `--company-phone` | `--phone` | plain, E.164 |
| `account.industries` | `--industry` | `--industry` | matched: `ark catalog industries` |
| `account.location` | `--company-location` | `--location` | plain |
| `account.productAndServices` | `--product` | `--product` | matched |
| `account.socialMedia` | `--company-social-media` | `--social-media` | enum |
| `account.type` | `--company-type` | `--type` | enum: `ark catalog company-types` |
| `account.language.any/all` | `--company-language` | `--language` | enum: `ark catalog languages` |
| `account.language.range` | `--company-language-count 2+` | `--language-count 2+` | band |
| `account.keyword` | `--company-keyword` + `--company-keyword-source` | `--keyword` + `--keyword-source` | keyword: `ark catalog company-keyword-sources` |
| `account.technologies` | `--technology` | `--technology` | matched: `ark catalog technologies` |
| `account.technology` (legacy, exact values) | `--technology-exact` | `--technology-exact` | plain |
| `account.naics` | `--naics` | `--naics` | plain |
| `account.foundedYear` | `--founded 2015-2022` / `any` / `none` | same | year band or ALL / NONE |
| `account.employeeSize` | `--employees 51-200` | same | bands, repeatable |
| `account.retailSize` | `--retail-locations 10-100` | same | bands, repeatable |
| `account.revenue` | `--revenue 1m-10m` / `any` / `none` | same | bands or ALL / NONE |
| `account.geoLocation` | `--near lat,lng --radius 50 --radius-unit km` | same | radius search |
| `account.funding.type` | `--funding-type` | same | enum: `ark catalog funding-types` |
| `account.funding.totalAmount` | `--funding-total 1m-5m` | same | band |
| `account.funding.lastAmount` | `--funding-last 500k-2m` | same | band |
| `account.funding.duration` | `--funding-date 2021-05-01..2025-05-01` | same | date band (epoch ms on the wire) |
| `account.metric.employee[]` | `--headcount sales,marketing:10-100` | same | repeatable; departments from `ark catalog metric-functions` |
| `account.metric.growth[]` | `--headcount-growth marketing:10..15:SIX` | same | repeatable, signed `from..to` percentages; windows from `ark catalog time-frames` |
| `account.employee.title` | - | `--employee-title` | matched (company search only) |
| `account.employee.seniority` | - | `--employee-seniority` | enum (company search only) |
| `account.employee.departmentAndFunction` | - | `--employee-department` | plain (company search only) |
| `lookalikeDomains` | - | `--lookalike` | up to 5 domains or LinkedIn company URLs |
| `lists.company_id.exclude` | - | `--exclude-list` | list UUIDs |

## Request envelope

| Wire path | Flag |
| --- | --- |
| `page` | `--page` |
| `size` | `--size` (1–100; 1–10,000 for `people export`) |
| `webhook` | `--webhook` (`people export` only, public HTTPS) |

## Bands

`a-b` (inclusive), `a+` or `a-` (open end), `-b` (open start), `a` (exact).
Numbers accept `k`, `m`, `b` suffixes. Tenure bands use `NyMm` (months
beyond 11 roll into years), e.g. `1y6m-3y`. Growth percentages use
`from..to` so they can be negative, e.g. `-20..-5`. `--revenue` and
`--founded` also accept `any` (a value exists) and `none` (no value).
