package client

// Job post search (POST /v1/job-posts) takes two filter objects of its own:
// "job" describes the post and "company" the company that is hiring. It
// shares the any/all and include/exclude envelopes of the other searches.

// JobFilter holds the filters on the job post itself.
type JobFilter struct {
	ID                 *PlainFilter `json:"id,omitempty"`
	Title              *MatchFilter `json:"title,omitempty"`
	Description        *MatchFilter `json:"description,omitempty"`
	TitleOrDescription *MatchFilter `json:"titleOrDescription,omitempty"`
	Location           *MatchFilter `json:"location,omitempty"`
	EmploymentType     *PlainFilter `json:"employmentType,omitempty"`
	Seniority          *PlainFilter `json:"seniority,omitempty"`
	Industries         *PlainFilter `json:"industries,omitempty"`
	Technologies       *PlainFilter `json:"technologies,omitempty"`
	// YearlySalaryUSD matches posts whose salary range overlaps the band.
	YearlySalaryUSD *BandFilter  `json:"yearlySalaryUsd,omitempty"`
	Language        *JobLanguage `json:"language,omitempty"`
	// HiringModel is DIRECT (posted by the hiring company itself) or AGENCY
	// (posted by a recruiting or staffing agency).
	HiringModel string     `json:"hiringModel,omitempty"`
	PostedAt    *DayFilter `json:"postedAt,omitempty"`
	ExpiredAt   *DayFilter `json:"expiredAt,omitempty"`
	// Created is the day AI-Ark first collected the post, which can be later
	// than the day it was posted.
	Created *DayFilter `json:"created,omitempty"`
}

// JobLanguage filters by the language a post is written in.
type JobLanguage struct {
	Name *PlainFilter `json:"name,omitempty"`
}

// DayFilter matches a date field against a band of calendar days.
type DayFilter struct {
	Type  string    `json:"type"`
	Range *DayRange `json:"range,omitempty"`
}

// DayRange is an inclusive band of days as YYYY-MM-DD; an empty bound is
// open-ended.
type DayRange struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// HiringCompanyFilter identifies the company behind a job post.
type HiringCompanyFilter struct {
	ID       *PlainFilter `json:"id,omitempty"` // AI-Ark company IDs
	Domain   *PlainFilter `json:"domain,omitempty"`
	Name     *MatchFilter `json:"name,omitempty"`
	LinkedIn *PlainFilter `json:"linkedin,omitempty"`
}
