package memory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"vote-system/internal/service"
)

type Store struct {
	mu sync.RWMutex

	polls map[string]service.Poll
	// pollID -> userID -> option
	votes map[string]map[string]string
	// userID -> set(pollID)
	created map[string]map[string]struct{}
	// userID -> idempotencyKey -> created pollID (expires at)
	createIdem map[string]map[string]createIdemEntry
}

type createIdemEntry struct {
	pollID    string
	expiresAt time.Time
}

func New() *Store {
	return &Store{
		polls:      map[string]service.Poll{},
		votes:      map[string]map[string]string{},
		created:    map[string]map[string]struct{}{},
		createIdem: map[string]map[string]createIdemEntry{},
	}
}

func (s *Store) CreatePoll(_ context.Context, p service.Poll, idempotencyKey string) (service.Poll, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idempotencyKey != "" {
		now := time.Now()
		m := s.createIdem[p.CreatedBy]
		if m == nil {
			m = map[string]createIdemEntry{}
			s.createIdem[p.CreatedBy] = m
		}
		if ent, ok := m[idempotencyKey]; ok {
			if now.Before(ent.expiresAt) {
				if existing, ok := s.polls[ent.pollID]; ok {
					return clonePoll(existing), nil
				}
			}
			delete(m, idempotencyKey)
		}
		// reserve idempotency for 5 minutes (same as Redis)
		m[idempotencyKey] = createIdemEntry{pollID: p.ID, expiresAt: now.Add(5 * time.Minute)}
	}

	s.polls[p.ID] = p
	if s.created[p.CreatedBy] == nil {
		s.created[p.CreatedBy] = map[string]struct{}{}
	}
	s.created[p.CreatedBy][p.ID] = struct{}{}
	return clonePoll(p), nil
}

func (s *Store) GetPoll(_ context.Context, pollID string) (service.Poll, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.polls[pollID]
	if !ok {
		return service.Poll{}, service.ErrNotFound
	}
	return clonePoll(p), nil
}

func (s *Store) ClosePoll(_ context.Context, pollID, userID, _ string) (service.Poll, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.polls[pollID]
	if !ok {
		return service.Poll{}, service.ErrNotFound
	}
	if p.IsClosed || time.Now().After(p.ExpiresAt) {
		p.IsClosed = true
		s.polls[pollID] = p
		return clonePoll(p), nil
	}
	p.IsClosed = true
	p.UpdatedBy = userID
	s.polls[pollID] = p
	return clonePoll(p), nil
}

func (s *Store) DeletePoll(_ context.Context, pollID, _ string, _ string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.polls[pollID]
	if !ok {
		return false, service.ErrNotFound
	}
	delete(s.polls, pollID)
	delete(s.votes, pollID)
	if s.created[p.CreatedBy] != nil {
		delete(s.created[p.CreatedBy], pollID)
	}
	return true, nil
}

func (s *Store) Vote(_ context.Context, pollID, userID, option, _ string) (service.Poll, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.polls[pollID]
	if !ok {
		return service.Poll{}, service.ErrNotFound
	}
	if p.IsClosed || time.Now().After(p.ExpiresAt) {
		return service.Poll{}, service.ErrForbidden
	}

	if s.votes[pollID] == nil {
		s.votes[pollID] = map[string]string{}
	}
	if _, already := s.votes[pollID][userID]; already {
		return service.Poll{}, service.ErrConflict
	}

	valid := false
	for _, o := range p.Options {
		if o == option {
			valid = true
			break
		}
	}
	if !valid {
		return service.Poll{}, service.ErrInvalid
	}

	s.votes[pollID][userID] = option
	if p.Votes == nil {
		p.Votes = map[string]int64{}
	}
	p.Votes[option]++
	p.UpdatedBy = userID
	s.polls[pollID] = p
	return clonePoll(p), nil
}

func (s *Store) UndoVote(_ context.Context, pollID, userID, _ string) (service.Poll, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.polls[pollID]
	if !ok {
		return service.Poll{}, service.ErrNotFound
	}
	if p.IsClosed || time.Now().After(p.ExpiresAt) {
		return service.Poll{}, service.ErrForbidden
	}
	opt, ok := s.votes[pollID][userID]
	if !ok {
		return service.Poll{}, service.ErrConflict
	}
	delete(s.votes[pollID], userID)
	if p.Votes != nil {
		p.Votes[opt]--
		if p.Votes[opt] <= 0 {
			delete(p.Votes, opt)
		}
	}
	p.UpdatedBy = userID
	s.polls[pollID] = p
	return clonePoll(p), nil
}

func (s *Store) SearchPolls(_ context.Context, q service.SearchQuery) ([]service.PollSummary, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type row struct {
		id string
		p  service.Poll
	}
	rows := make([]row, 0, len(s.polls))
	for id, p := range s.polls {
		if q.CreatedBy != "" && p.CreatedBy != q.CreatedBy {
			continue
		}
		if !q.IncludeClosed && p.IsClosed {
			continue
		}
		if q.CreatedAfter != nil && !p.CreatedAt.After(*q.CreatedAfter) {
			continue
		}
		if q.CreatedBefore != nil && !p.CreatedAt.Before(*q.CreatedBefore) {
			continue
		}
		rows = append(rows, row{id: id, p: p})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].p.CreatedAt.After(rows[j].p.CreatedAt) })
	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}

	out := make([]service.PollSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, service.PollSummary{
			ID:        r.p.ID,
			Question:  r.p.Question,
			CreatedAt: r.p.CreatedAt,
			ExpiresAt: r.p.ExpiresAt,
			IsClosed:  r.p.IsClosed,
			CreatedBy: r.p.CreatedBy,
			IsPublic:  r.p.IsPublic,
		})
	}
	return out, "", nil
}

func (s *Store) GetPollStats(_ context.Context, q service.StatsQuery) ([]service.Poll, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := make([]service.Poll, 0, len(s.polls))
	for _, p := range s.polls {
		if !q.IncludeClosed && p.IsClosed {
			continue
		}
		rows = append(rows, clonePoll(p))
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.After(rows[j].CreatedAt) })
	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, "", nil
}

func (s *Store) ListPublicPolls(_ context.Context, q service.StatsQuery) ([]service.PollSummary, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := make([]service.PollSummary, 0, len(s.polls))
	for _, p := range s.polls {
		if !p.IsPublic {
			continue
		}
		if !q.IncludeClosed && p.IsClosed {
			continue
		}
		rows = append(rows, service.PollSummary{
			ID:        p.ID,
			Question:  p.Question,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
			IsClosed:  p.IsClosed,
			CreatedBy: p.CreatedBy,
			IsPublic:  p.IsPublic,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.After(rows[j].CreatedAt) })
	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, "", nil
}

func (s *Store) ListPublicPollStats(_ context.Context, q service.StatsQuery) ([]service.Poll, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := make([]service.Poll, 0, len(s.polls))
	for _, p := range s.polls {
		if !p.IsPublic {
			continue
		}
		if !q.IncludeClosed && p.IsClosed {
			continue
		}
		rows = append(rows, clonePoll(p))
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.After(rows[j].CreatedAt) })
	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, "", nil
}

func (s *Store) GetMyVotes(_ context.Context, userID string, limit int32, _ string) ([]service.MyVote, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]service.MyVote, 0)
	for pollID, byUser := range s.votes {
		opt, ok := byUser[userID]
		if !ok {
			continue
		}
		p, ok := s.polls[pollID]
		if !ok {
			continue
		}
		out = append(out, service.MyVote{PollID: pollID, Option: opt, Poll: clonePoll(p)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Poll.CreatedAt.After(out[j].Poll.CreatedAt) })
	l := int(limit)
	if l <= 0 || l > 100 {
		l = 50
	}
	if len(out) > l {
		out = out[:l]
	}
	return out, "", nil
}

func (s *Store) ListMyCreatedPollStats(_ context.Context, userID string, q service.StatsQuery) ([]service.Poll, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set := s.created[userID]
	if set == nil {
		return []service.Poll{}, "", nil
	}
	out := make([]service.Poll, 0, len(set))
	for id := range set {
		p, ok := s.polls[id]
		if !ok {
			continue
		}
		if !q.IncludeClosed && p.IsClosed {
			continue
		}
		out = append(out, clonePoll(p))
	}
	sort.Slice(out, func(i, j int) bool {
		si := createdScore(out[i].CreatedAt, out[i].ID)
		sj := createdScore(out[j].CreatedAt, out[j].ID)
		return si > sj
	})
	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	cursorScore := 0.0
	hasCursor := false
	if q.Cursor != "" {
		if v, err := strconv.ParseFloat(q.Cursor, 64); err == nil {
			cursorScore = v
			hasCursor = true
		}
	}

	filtered := make([]service.Poll, 0, len(out))
	for _, p := range out {
		s := createdScore(p.CreatedAt, p.ID)
		if hasCursor && s >= cursorScore {
			continue
		}
		filtered = append(filtered, p)
	}

	next := ""
	if len(filtered) > limit {
		// Cursor is the last item score in the current page.
		next = fmt.Sprintf("%0.6f", createdScore(filtered[limit-1].CreatedAt, filtered[limit-1].ID))
		filtered = filtered[:limit]
	}
	return filtered, next, nil
}

func clonePoll(p service.Poll) service.Poll {
	opts := append([]string(nil), p.Options...)
	votes := make(map[string]int64, len(p.Votes))
	for k, v := range p.Votes {
		votes[k] = v
	}
	p.Options = opts
	p.Votes = votes
	return p
}
