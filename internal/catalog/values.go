package catalog

// Embedded vocabularies, copied from the enumerations in the developer-portal
// OpenAPI specification (https://docs.ai-ark.com/reference).

// SeniorityLevels are the values of contact.seniority.
var SeniorityLevels = []string{
	"founder", "owner", "partner", "c_suite", "vp", "director", "head",
	"manager", "senior", "mid-level", "entry", "intern",
}

// CompanyTypeValues are the values of account.type.
var CompanyTypeValues = []string{
	"PRIVATELY_HELD", "PUBLIC_COMPANY", "SELF_OWNED", "SELF_EMPLOYED",
	"PARTNERSHIP", "NON_PROFIT", "EDUCATIONAL", "GOVERNMENT_AGENCY",
}

// LanguageValues are the values of contact.language and account.language.
var LanguageValues = []string{
	"english", "spanish", "french", "portuguese", "german", "dutch", "italian",
	"chinese", "turkish", "polish", "russian", "swedish", "arabic", "indonesian",
	"danish", "czech", "norwegian", "japanese", "korean", "romanian", "ukrainian",
	"thai", "hindi", "malay", "tagalog", "vietnamese", "finnish", "persian",
	"greek", "hungarian", "bengali", "marathi", "telugu", "panjabi", "serbian",
	"slovak", "croatian", "lithuanian", "latvian", "albanian", "icelandic",
	"armenian", "bosnian", "tamil", "javanese", "malayalam", "kannada", "burmese",
}

// BadgeValues are the values of contact.profileBadge.
var BadgeValues = []string{"VERIFIED", "PREMIUM", "OPEN_TO_WORK", "INFLUENCER", "CREATOR", "HIRING"}

// SocialMediaValues are the values of contact.socialMedia and account.socialMedia.
var SocialMediaValues = []string{"FACEBOOK", "INSTAGRAM", "TWITTER", "LINKEDIN"}

// FundingTypeValues are the values of account.funding.type.
var FundingTypeValues = []string{
	"PRE_SEED", "SEED", "SERIES_A", "SERIES_B", "SERIES_C", "SERIES_D", "SERIES_E",
	"SERIES_F", "SERIES_G", "SERIES_H", "SERIES_I", "SERIES_J", "VENTURE_ROUND",
	"ANGEL", "PRIVATE_EQUITY", "DEBT_FINANCING", "CONVERTIBLE_NOTE", "GRANT",
	"CORPORATE_ROUND", "EQUITY_CROWDFUNDING", "PRODUCT_CROWDFUNDING",
	"SECONDARY_MARKET", "POST_IPO_EQUITY", "POST_IPO_DEBT", "POST_IPO_SECONDARY",
	"NON_EQUITY_ASSISTANCE", "INITIAL_COIN_OFFERING", "UNDISCLOSED",
	"SERIES_UNKNOWN", "FUNDING_ROUND",
}

// PeopleKeywordSources are the profile sections contact.keyword can search
// (at most five per request).
var PeopleKeywordSources = []string{
	"HEADLINE", "SUMMARY", "SKILL", "CERTIFICATION", "COURSE", "PROJECTS",
	"PUBLICATION", "PATENT", "AWARD", "ORGANIZATION", "VOLUNTEERING",
	"TEST_SCORE", "WORK_HISTORY_DESCRIPTION", "EDUCATION_DESCRIPTION",
	"LANGUAGE_SKILL",
}

// CompanyKeywordSources are the company fields account.keyword can search.
var CompanyKeywordSources = []string{"NAME", "KEYWORD", "SEO", "DESCRIPTION", "INDUSTRY"}

// MetricFunctions are the departments accepted by account.metric.
var MetricFunctions = []string{
	"sales", "marketing", "engineering", "finance", "human_resources",
	"information_technology", "operations", "business_development",
	"customer_success_and_support", "product_management", "accounting", "legal",
	"consulting", "education", "research", "purchasing", "real_estate",
	"media_and_communication", "quality_assurance", "arts_and_design",
	"healthcare_services", "entrepreneurship", "community_and_social_services",
	"administrative", "military_and_protective_services",
	"program_and_project_management", "support",
}

// TimeFrameValues are the look-back windows (in months) of account.metric.growth.
var TimeFrameValues = []string{"ONE", "THREE", "SIX", "TWELVE", "TWENTY_FOUR"}

// MatchModeValues are the text match modes.
var MatchModeValues = []string{"SMART", "WORD", "STRICT"}

// SubmissionStates are the values of the state query of the submissions endpoints.
var SubmissionStates = []string{"PENDING", "SETTLED"}

// MailPlatformValues are the known values of account.mail.platform. The list
// grows as new providers are detected, so the filter accepts other values too.
var MailPlatformValues = []string{
	"microsoft", "google", "self_hosted", "other", "ionos", "hostinger", "godaddy",
	"zoho", "strato", "ovh", "siteground", "allinkl", "opensrs", "cloudflare",
	"namecheap", "locaweb", "onecom", "tencent", "titan", "20i", "rackspace", "alibaba",
	"netease", "jimdo", "uolhost", "123reg", "infomaniak", "gandi", "network_solutions",
	"mailgun", "domainfactory", "register_it", "dogado", "hetzner", "amazon", "yandex",
	"proton", "yahoo", "mittwald", "kinghost", "united_domains", "telekom",
	"mijndomein", "combell", "hostpoint", "transip", "icloud", "namesco", "fastmail",
	"aruba", "intermedia", "aplus", "mailhostbox", "web_com", "lolipop", "hostnet",
	"domeneshop", "amen", "improvmx", "loopia", "webnode", "mxroute", "263",
	"dreamhost", "simply", "jouwweb", "dondominio", "skymail", "ocn", "nominalia",
	"openxchange", "lark", "forwardemail", "seeweb", "lcn", "goneo", "sakura",
	"krystal", "zenbox", "rediffmail", "home_pl", "umbler", "cpi", "nazwa",
	"namebright", "wordpress", "kagoya", "mailbox", "wedos", "mailru", "seznam",
	"naver", "xserver",
}

// MailSecurityValues are the known values of account.mail.security.
var MailSecurityValues = []string{
	"proofpoint", "mimecast", "other", "barracuda", "hornetsecurity", "mailchannels",
	"spamexperts", "appriver", "trendmicro", "cisco", "sophos", "mailinblack",
	"spamtitan", "vipre", "spamhero", "mesh", "mailprotector", "datev", "cloudflare",
	"symantec", "fortimail", "vade", "mailguard", "topsec", "securence", "libraesva",
	"forcepoint", "nospamproxy", "retarus", "heimdal", "trellix", "altospam",
	"mailroute", "seppmail", "gosecure",
}

// JobSeniorityLevels are the values of job.seniority (job post search).
var JobSeniorityLevels = []string{
	"internship", "entry", "associate", "mid_senior", "director", "executive",
}

// JobEmploymentTypeValues are the values of job.employmentType.
var JobEmploymentTypeValues = []string{
	"full_time", "part_time", "contract", "temporary", "internship", "volunteer",
}

// JobLanguageValues are the known values of job.language.name.
var JobLanguageValues = []string{
	"english", "german", "french", "dutch", "portuguese", "spanish", "chinese",
	"italian", "swedish", "korean", "polish", "japanese", "czech", "turkish", "danish",
	"vietnamese", "norwegian", "greek", "finnish", "indonesian", "hungarian",
	"romanian", "arabic", "russian", "catalan", "ukrainian", "hebrew", "slovak",
	"lithuanian", "thai", "croatian", "slovenian", "latvian", "bulgarian",
	"azerbaijani", "albanian", "serbian", "georgian", "estonian", "luxembourgish",
	"latin", "bengali", "armenian", "welsh", "hindi", "macedonian", "serbo-croatian",
	"persian", "burmese", "malay", "dhivehi", "marathi", "khmer", "malayalam",
	"mongolian", "basque", "icelandic", "afrikaans", "esperanto", "kazakh", "uzbek",
	"galician", "urdu", "irish", "tagalog", "tamil", "nepali", "oriya", "punjabi",
	"pashto", "telugu", "western frisian", "amharic", "egyptian arabic", "tatar",
	"cebuano", "central kurdish", "sinhala", "gujarati", "lao", "maltese", "assamese",
	"breton", "interlingua", "lojban", "low german", "mazanderani", "minangkabau",
	"scottish gaelic", "waray",
}

// HiringModelValues are the values of job.hiringModel.
var HiringModelValues = []string{"DIRECT", "AGENCY"}
