package cmd

import (
	"github.com/ai-ark-com/ark-cli/internal/catalog"
	"github.com/ai-ark-com/ark-cli/internal/client"
)

// facetsFor lists every list-valued filter of the API for a scope, in the
// order the flags appear in --help. The table is the single source of truth
// for flag names; cmd/filters_test.go checks that the flags together reach
// every path of the request schema.
func facetsFor(scope filterScope) []*facet {
	var facets []*facet
	if scope == peopleScope {
		facets = append(facets, contactFacets()...)
	}
	facets = append(facets, accountFacets(scope)...)
	if scope == companyScope {
		facets = append(facets, employeeFacets()...)
	}
	return facets
}

// sampleUUID is a well-formed id for tests and examples.
const sampleUUID = "931fc1a0-1ed5-baf8-0dd2-a068a0fd0430"

func contactFacets() []*facet {
	return []*facet{
		{
			name: "location", usage: "where the person lives: country, state, city, or region", sample: "Germany",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { contact(r).Location = f },
		},
		{
			name: "seniority", usage: "seniority level(s), see 'ark catalog seniority'",
			enum: catalog.SeniorityLevels, sample: "c_suite",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { contact(r).Seniority = f },
		},
		{
			name: "department", usage: "department or job function(s), see 'ark catalog departments'", sample: "sales",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { contact(r).DepartmentAndFunction = f },
		},
		{
			name: "title", usage: "job title(s) of the primary active role", kind: matchedFacet, sample: "founder",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) {
				experience(r).Latest = &client.Position{Title: f}
			},
		},
		{
			name: "current-title", usage: "job title(s) of any active role", kind: matchedFacet, sample: "advisor",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { currentPosition(r).Title = f },
		},
		{
			name: "previous-title", usage: "job title(s) of past roles", kind: matchedFacet, sample: "intern",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) {
				experience(r).Previous = &client.Position{Title: f}
			},
		},
		{
			name: "name", usage: "full name(s)", kind: matchedFacet, sample: "Jane Smith",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { contact(r).FullName = f },
		},
		{
			name: "skill", usage: "skill(s) on the profile", kind: matchedFacet, sample: "kubernetes",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { contact(r).Skill = f },
		},
		{
			name: "certification", usage: "professional certification(s)", kind: matchedFacet, sample: "pmp",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { contact(r).Certification = f },
		},
		{
			name: "linkedin", usage: "LinkedIn profile URL(s)", sample: "https://www.linkedin.com/in/jane-smith",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { contact(r).LinkedIn = f },
		},
		{
			name: "social-link", usage: "social profile URL(s) of the person", sample: "https://www.facebook.com/jane.smith",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { contact(r).SocialMediaLink = f },
		},
		{
			name: "company-id", usage: "AI-Ark company id(s) of the primary employer (from 'ark company search')",
			ids: true, sample: sampleUUID,
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { companyIDs(r).Latest = f },
		},
		{
			name: "current-company-id", usage: "company id(s) of any active role", ids: true, sample: sampleUUID,
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { companyIDs(r).Current = f },
		},
		{
			name: "previous-company-id", usage: "company id(s) of past employers", ids: true, sample: sampleUUID,
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { companyIDs(r).Previous = f },
		},
		{
			name: "language", usage: "language(s) the person speaks, see 'ark catalog languages'",
			kind: matchedFacet, enum: catalog.LanguageValues, sample: "english",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) {
				c := contact(r)
				if c.Language == nil {
					c.Language = &client.MatchedLanguageFilter{}
				}
				c.Language.Any, c.Language.All = f.Any, f.All
			},
		},
		{
			name: "badge", usage: "LinkedIn profile badge(s), see 'ark catalog badges'",
			enum: catalog.BadgeValues, sample: "HIRING",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { contact(r).ProfileBadge = f },
		},
		{
			name: "social-media", usage: "network(s) the person has a profile on, see 'ark catalog social-media'",
			enum: catalog.SocialMediaValues, sample: "LINKEDIN",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { contact(r).SocialMedia = f },
		},
		{
			name: "keyword", usage: "free-text term(s) searched in the --keyword-source sections",
			kind: keywordFacet, sample: "climate",
			sourceCatalog: catalog.PeopleSources, defaultSources: []string{"HEADLINE", "SUMMARY"}, maxSources: 5,
			keyword: func(r *client.SearchRequest, f *client.KeywordFilter) { contact(r).Keyword = f },
		},
		{
			name: "degree", usage: "education degree(s), e.g. mba", kind: matchedFacet, sample: "mba",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { education(r).Degree = f },
		},
		{
			name: "field-of-study", usage: "field(s) of study, e.g. \"computer science\"", kind: matchedFacet, sample: "economics",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { education(r).FieldOfStudy = f },
		},
		{
			name: "school", usage: "AI-Ark company id(s) of schools (find them with 'ark company search --domain harvard.edu')",
			ids: true, sample: sampleUUID,
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { education(r).School = f },
		},
	}
}

func accountFacets(scope filterScope) []*facet {
	acct := accountFlagName(scope)
	return []*facet{
		{
			name: acct("domain"), usage: "company website domain(s), e.g. stripe.com", sample: "stripe.com",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { account(r).Domain = f },
		},
		{
			name: acct("name"), usage: "company name(s)", kind: matchedFacet, sample: "Stripe",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { account(r).Name = f },
		},
		{
			name: acct("linkedin"), usage: "company LinkedIn URL(s)", sample: "https://www.linkedin.com/company/stripe",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { account(r).LinkedIn = f },
		},
		{
			name: acct("url"), usage: "company URL(s) in any form: domain, www, full URL, or LinkedIn URL",
			kind: matchedFacet, sample: "www.stripe.com",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { account(r).URL = f },
		},
		{
			name: acct("social-link"), usage: "social profile URL(s) of the company", sample: "https://www.facebook.com/stripe",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { account(r).SocialMediaLink = f },
		},
		{
			name: acct("phone"), usage: "company phone number(s) in E.164 format, e.g. +18885335659", sample: "+18885335659",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { account(r).PhoneNumber = f },
		},
		{
			name: "industry", usage: "company industry(ies), see 'ark catalog industries'",
			kind: matchedFacet, sample: "software development",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { account(r).Industries = f },
		},
		{
			name: acct("location"), usage: "company office location(s): country, state, city, or region", sample: "Germany",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { account(r).Location = f },
		},
		{
			name: "product", usage: "products or services the company offers", kind: matchedFacet, sample: "payroll software",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { account(r).ProductAndServices = f },
		},
		{
			name: acct("social-media"), usage: "network(s) the company has a profile on, see 'ark catalog social-media'",
			enum: catalog.SocialMediaValues, sample: "LINKEDIN",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { account(r).SocialMedia = f },
		},
		{
			name: acct("type"), usage: "company type(s), see 'ark catalog company-types'",
			enum: catalog.CompanyTypeValues, sample: "PRIVATELY_HELD",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { account(r).Type = f },
		},
		{
			name: acct("language"), usage: "language(s) the company operates in, see 'ark catalog languages'",
			enum: catalog.LanguageValues, sample: "german",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) {
				a := account(r)
				if a.Language == nil {
					a.Language = &client.LanguageFilter{}
				}
				a.Language.Any, a.Language.All = f.Any, f.All
			},
		},
		{
			name: acct("keyword"), usage: "free-text term(s) searched in company fields",
			kind: keywordFacet, sample: "carbon accounting",
			sourceCatalog: catalog.CompanySources, defaultSources: []string{"NAME", "DESCRIPTION", "KEYWORD"},
			keyword: func(r *client.SearchRequest, f *client.KeywordFilter) { account(r).Keyword = f },
		},
		{
			name: "technology", usage: "technology(ies) the company uses, see 'ark catalog technologies'",
			kind: matchedFacet, sample: "hubspot",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { account(r).Technologies = f },
		},
		{
			name: "technology-exact", usage: "technology(ies) as exact catalog values (the API's original filter, no match modes)",
			sample: "AWS",
			plain:  func(r *client.SearchRequest, f *client.PlainFilter) { account(r).Technology = f },
		},
		{
			name: "naics", usage: "NAICS code(s), e.g. 541511", sample: "541511",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { account(r).NAICS = f },
		},
	}
}

func employeeFacets() []*facet {
	return []*facet{
		{
			name: "employee-title", usage: "companies employing people with these job titles",
			kind: matchedFacet, sample: "head of sales",
			matched: func(r *client.SearchRequest, f *client.MatchFilter) { employee(r).Title = f },
		},
		{
			name: "employee-seniority", usage: "companies employing people at these seniority levels",
			enum: catalog.SeniorityLevels, sample: "vp",
			plain: func(r *client.SearchRequest, f *client.PlainFilter) { employee(r).Seniority = f },
		},
		{
			name: "employee-department", usage: "companies employing people in these departments, see 'ark catalog departments'",
			sample: "software_development",
			plain:  func(r *client.SearchRequest, f *client.PlainFilter) { employee(r).DepartmentAndFunction = f },
		},
	}
}
