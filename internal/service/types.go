package service

import "time"

type Poll struct {
	ID        string
	Question  string
	Options   []string
	Votes     map[string]int64
	CreatedBy string
	UpdatedBy string
	CreatedAt time.Time
	ExpiresAt time.Time
	IsClosed  bool
	IsPublic  bool
}

type PollSummary struct {
	ID        string
	Question  string
	CreatedAt time.Time
	ExpiresAt time.Time
	IsClosed  bool
	CreatedBy string
	IsPublic  bool
}

type MyVote struct {
	PollID string
	Option string
	Poll   Poll
}

type SearchQuery struct {
	CreatedBy     string
	IncludeClosed bool
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Limit         int32
	Cursor        string
}

type StatsQuery struct {
	IncludeClosed bool
	Limit         int32
	Cursor        string
}
