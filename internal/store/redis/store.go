package redis

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"time"

	"vote-system/internal/service"

	goredis "github.com/redis/go-redis/v9"
)

type Store struct {
	rdb *goredis.Client
}

func New(rdb *goredis.Client) *Store {
	return &Store{rdb: rdb}
}

func (s *Store) CreatePoll(ctx context.Context, p service.Poll, idempotencyKey string) (service.Poll, error) {
	// Idempotent create: if Idempotency-Key is provided and already used, return the existing poll.
	if idempotencyKey != "" {
		args := make([]any, 0, 9+len(p.Options))
		args = append(args,
			p.CreatedBy,
			idempotencyKey,
			p.ID,
			p.Question,
			strconv.FormatInt(p.CreatedAt.Unix(), 10),
			strconv.FormatInt(p.ExpiresAt.Unix(), 10),
			boolToRedis(p.IsPublic),
			fmt.Sprintf("%0.6f", createdScore(p.CreatedAt, p.ID)),
			strconv.Itoa(len(p.Options)),
		)
		for _, opt := range p.Options {
			args = append(args, opt)
		}

		res, err := createPollScript.Run(ctx, s.rdb, []string{
			idemKey(p.CreatedBy, idempotencyKey),
			pollMetaKey(p.ID),
			pollOptionsKey(p.ID),
			pollVotesKey(p.ID),
			pollVotersKey(p.ID),
			pollsIndexKey(),
			pollsPublicIndexKey(),
			userCreatedPollsKey(p.CreatedBy),
		}, args...).Slice()
		if err != nil {
			return service.Poll{}, err
		}
		if len(res) >= 2 {
			code, _ := res[0].(int64)
			pollID, _ := res[1].(string)
			switch code {
			case createOK:
				return s.getPoll(ctx, pollID)
			case createNoop:
				return s.getPoll(ctx, pollID)
			default:
				return service.Poll{}, fmt.Errorf("unknown create result: %v", res)
			}
		}
		return service.Poll{}, fmt.Errorf("unexpected create result: %v", res)
	}

	pipe := s.rdb.TxPipeline()

	pollKey := pollMetaKey(p.ID)
	optionsKey := pollOptionsKey(p.ID)
	votesKey := pollVotesKey(p.ID)
	votersKey := pollVotersKey(p.ID)

	pipe.HSet(ctx, pollKey, map[string]any{
		"id":         p.ID,
		"question":   p.Question,
		"created_by": p.CreatedBy,
		"updated_by": p.UpdatedBy,
		"created_at": strconv.FormatInt(p.CreatedAt.Unix(), 10),
		"expires_at": strconv.FormatInt(p.ExpiresAt.Unix(), 10),
		"is_closed":  boolToRedis(p.IsClosed),
		"is_public":  boolToRedis(p.IsPublic),
	})

	for i, opt := range p.Options {
		pipe.HSet(ctx, optionsKey, strconv.Itoa(i), opt)
		pipe.HSet(ctx, votesKey, opt, 0)
	}

	pipe.SAdd(ctx, pollsIndexKey(), p.ID)
	if p.IsPublic {
		pipe.SAdd(ctx, pollsPublicIndexKey(), p.ID)
	}
	pipe.ZAdd(ctx, userCreatedPollsKey(p.CreatedBy), goredis.Z{
		Score:  createdScore(p.CreatedAt, p.ID),
		Member: p.ID,
	})
	pipe.Del(ctx, votersKey) // ensure empty

	if _, err := pipe.Exec(ctx); err != nil {
		return service.Poll{}, err
	}
	p.Votes = map[string]int64{}
	for _, opt := range p.Options {
		p.Votes[opt] = 0
	}
	return p, nil
}

func (s *Store) ClosePoll(ctx context.Context, pollID, userID, idempotencyKey string) (service.Poll, error) {
	p, err := s.getPoll(ctx, pollID)
	if err != nil {
		return service.Poll{}, err
	}
	if p.CreatedBy != userID {
		return service.Poll{}, service.ErrForbidden
	}
	if p.IsClosed {
		return p, nil
	}

	if idempotencyKey != "" {
		ok, err := s.idemOK(ctx, userID, idempotencyKey)
		if err != nil {
			return service.Poll{}, err
		}
		if !ok {
			return s.getPoll(ctx, pollID)
		}
	}

	pollKey := pollMetaKey(pollID)
	if err := s.rdb.HSet(ctx, pollKey, "is_closed", "true", "updated_by", userID).Err(); err != nil {
		return service.Poll{}, err
	}
	if idempotencyKey != "" {
		_ = s.setIdem(ctx, userID, idempotencyKey)
	}
	return s.getPoll(ctx, pollID)
}

func (s *Store) DeletePoll(ctx context.Context, pollID, userID, idempotencyKey string) (bool, error) {
	p, err := s.getPoll(ctx, pollID)
	if err != nil {
		return false, err
	}
	if p.CreatedBy != userID {
		return false, service.ErrForbidden
	}

	if idempotencyKey != "" {
		ok, err := s.idemOK(ctx, userID, idempotencyKey)
		if err != nil {
			return false, err
		}
		if !ok {
			return true, nil
		}
	}

	keys := []string{
		pollMetaKey(pollID),
		pollOptionsKey(pollID),
		pollVotesKey(pollID),
		pollVotersKey(pollID),
	}
	if err := s.rdb.Del(ctx, keys...).Err(); err != nil {
		return false, err
	}
	if err := s.rdb.SRem(ctx, pollsIndexKey(), pollID).Err(); err != nil {
		return false, err
	}
	if err := s.rdb.SRem(ctx, pollsPublicIndexKey(), pollID).Err(); err != nil {
		return false, err
	}
	if err := s.rdb.ZRem(ctx, userCreatedPollsKey(userID), pollID).Err(); err != nil {
		return false, err
	}
	if idempotencyKey != "" {
		_ = s.setIdem(ctx, userID, idempotencyKey)
	}
	return true, nil
}

func (s *Store) GetPoll(ctx context.Context, pollID string) (service.Poll, error) {
	return s.getPoll(ctx, pollID)
}

func (s *Store) Vote(ctx context.Context, pollID, userID, option, idempotencyKey string) (service.Poll, error) {
	res, err := voteScript.Run(ctx, s.rdb, []string{
		pollMetaKey(pollID),
		pollVotesKey(pollID),
		pollVotersKey(pollID),
		userVotesKey(userID),
		idemKey(userID, idempotencyKey),
	}, pollID, userID, option, idempotencyKey).Int64()
	if err != nil {
		return service.Poll{}, err
	}
	switch res {
	case voteOK, voteNoop:
		return s.getPoll(ctx, pollID)
	case voteNotFound:
		return service.Poll{}, service.ErrNotFound
	case voteForbidden:
		return service.Poll{}, service.ErrForbidden
	case voteConflict:
		return service.Poll{}, service.ErrConflict
	case voteInvalid:
		return service.Poll{}, service.ErrInvalid
	default:
		return service.Poll{}, fmt.Errorf("unknown vote result: %d", res)
	}
}

func (s *Store) UndoVote(ctx context.Context, pollID, userID, idempotencyKey string) (service.Poll, error) {
	res, err := undoScript.Run(ctx, s.rdb, []string{
		pollMetaKey(pollID),
		pollVotesKey(pollID),
		pollVotersKey(pollID),
		userVotesKey(userID),
		idemKey(userID, idempotencyKey),
	}, pollID, userID, idempotencyKey).Int64()
	if err != nil {
		return service.Poll{}, err
	}
	switch res {
	case undoOK, undoNoop:
		return s.getPoll(ctx, pollID)
	case undoNotFound:
		return service.Poll{}, service.ErrNotFound
	case undoForbidden:
		return service.Poll{}, service.ErrForbidden
	case undoConflict:
		return service.Poll{}, service.ErrConflict
	default:
		return service.Poll{}, fmt.Errorf("unknown undo result: %d", res)
	}
}

func (s *Store) SearchPolls(ctx context.Context, q service.SearchQuery) ([]service.PollSummary, string, error) {
	ids, err := s.rdb.SMembers(ctx, pollsIndexKey()).Result()
	if err != nil {
		return nil, "", err
	}

	items := make([]service.PollSummary, 0, len(ids))
	for _, id := range ids {
		p, err := s.getPoll(ctx, id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				continue
			}
			return nil, "", err
		}
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
		items = append(items, service.PollSummary{
			ID:        p.ID,
			Question:  p.Question,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
			IsClosed:  p.IsClosed,
			CreatedBy: p.CreatedBy,
		})
	}

	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, "", nil
}

func (s *Store) GetPollStats(ctx context.Context, q service.StatsQuery) ([]service.Poll, string, error) {
	ids, err := s.rdb.SMembers(ctx, pollsIndexKey()).Result()
	if err != nil {
		return nil, "", err
	}
	items := make([]service.Poll, 0, len(ids))
	for _, id := range ids {
		p, err := s.getPoll(ctx, id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				continue
			}
			return nil, "", err
		}
		if !q.IncludeClosed && p.IsClosed {
			continue
		}
		items = append(items, p)
	}
	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, "", nil
}

func (s *Store) ListPublicPolls(ctx context.Context, q service.StatsQuery) ([]service.PollSummary, string, error) {
	ids, err := s.rdb.SMembers(ctx, pollsPublicIndexKey()).Result()
	if err != nil {
		return nil, "", err
	}
	items := make([]service.PollSummary, 0, len(ids))
	for _, id := range ids {
		p, err := s.getPoll(ctx, id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				continue
			}
			return nil, "", err
		}
		if !q.IncludeClosed && p.IsClosed {
			continue
		}
		items = append(items, service.PollSummary{
			ID:        p.ID,
			Question:  p.Question,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
			IsClosed:  p.IsClosed,
			CreatedBy: p.CreatedBy,
			IsPublic:  true,
		})
	}
	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, "", nil
}

func (s *Store) ListPublicPollStats(ctx context.Context, q service.StatsQuery) ([]service.Poll, string, error) {
	ids, err := s.rdb.SMembers(ctx, pollsPublicIndexKey()).Result()
	if err != nil {
		return nil, "", err
	}
	items := make([]service.Poll, 0, len(ids))
	for _, id := range ids {
		p, err := s.getPoll(ctx, id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				continue
			}
			return nil, "", err
		}
		if !q.IncludeClosed && p.IsClosed {
			continue
		}
		items = append(items, p)
	}
	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, "", nil
}

func (s *Store) GetMyVotes(ctx context.Context, userID string, limit int32, _ string) ([]service.MyVote, string, error) {
	m, err := s.rdb.HGetAll(ctx, userVotesKey(userID)).Result()
	if err != nil {
		return nil, "", err
	}
	out := make([]service.MyVote, 0, len(m))
	for pollID, opt := range m {
		p, err := s.getPoll(ctx, pollID)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				continue
			}
			return nil, "", err
		}
		out = append(out, service.MyVote{
			PollID: pollID,
			Option: opt,
			Poll:   p,
		})
	}
	l := int(limit)
	if l <= 0 || l > 100 {
		l = 50
	}
	if len(out) > l {
		out = out[:l]
	}
	return out, "", nil
}

func (s *Store) ListMyCreatedPollStats(ctx context.Context, userID string, q service.StatsQuery) ([]service.Poll, string, error) {
	limit := int(q.Limit)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	max := "+inf"
	if q.Cursor != "" {
		if score, err := strconv.ParseFloat(q.Cursor, 64); err == nil {
			max = fmt.Sprintf("(%0.6f", score)
		}
	}

	rows, err := s.rdb.ZRevRangeByScoreWithScores(ctx, userCreatedPollsKey(userID), &goredis.ZRangeBy{
		Max:    max,
		Min:    "-inf",
		Offset: 0,
		Count:  int64(limit + 1),
	}).Result()
	if err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(rows) > limit {
		// Return cursor as the last item score in the current page.
		nextCursor = fmt.Sprintf("%0.6f", rows[limit-1].Score)
		rows = rows[:limit]
	}

	items := make([]service.Poll, 0, len(rows))
	for _, row := range rows {
		id, _ := row.Member.(string)
		p, err := s.getPoll(ctx, id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				continue
			}
			return nil, "", err
		}
		if !q.IncludeClosed && p.IsClosed {
			continue
		}
		// enforce ownership even if index is stale
		if p.CreatedBy != userID {
			continue
		}
		items = append(items, p)
	}
	return items, nextCursor, nil
}

func (s *Store) getPoll(ctx context.Context, pollID string) (service.Poll, error) {
	pollKey := pollMetaKey(pollID)
	m, err := s.rdb.HGetAll(ctx, pollKey).Result()
	if err != nil {
		return service.Poll{}, err
	}
	if len(m) == 0 {
		return service.Poll{}, service.ErrNotFound
	}
	createdAt, _ := strconv.ParseInt(m["created_at"], 10, 64)
	expiresAt, _ := strconv.ParseInt(m["expires_at"], 10, 64)

	optsMap, err := s.rdb.HGetAll(ctx, pollOptionsKey(pollID)).Result()
	if err != nil {
		return service.Poll{}, err
	}
	opts := make([]string, 0, len(optsMap))
	for i := 0; i < len(optsMap); i++ {
		v, ok := optsMap[strconv.Itoa(i)]
		if ok {
			opts = append(opts, v)
		}
	}

	votesStr, err := s.rdb.HGetAll(ctx, pollVotesKey(pollID)).Result()
	if err != nil {
		return service.Poll{}, err
	}
	votes := make(map[string]int64, len(votesStr))
	for k, sv := range votesStr {
		n, _ := strconv.ParseInt(sv, 10, 64)
		votes[k] = n
	}

	p := service.Poll{
		ID:        m["id"],
		Question:  m["question"],
		Options:   opts,
		Votes:     votes,
		CreatedBy: m["created_by"],
		UpdatedBy: m["updated_by"],
		CreatedAt: time.Unix(createdAt, 0).UTC(),
		ExpiresAt: time.Unix(expiresAt, 0).UTC(),
		IsClosed:  m["is_closed"] == "true",
		IsPublic:  m["is_public"] == "true",
	}
	if time.Now().After(p.ExpiresAt) {
		p.IsClosed = true
	}
	return p, nil
}

func (s *Store) idemOK(ctx context.Context, userID, key string) (bool, error) {
	if key == "" {
		return true, nil
	}
	v, err := s.rdb.Get(ctx, idemKey(userID, key)).Result()
	if err == nil && v != "" {
		return false, nil
	}
	if err != nil && err != goredis.Nil {
		return false, err
	}
	return true, nil
}

func (s *Store) setIdem(ctx context.Context, userID, key string) error {
	if key == "" {
		return nil
	}
	return s.rdb.Set(ctx, idemKey(userID, key), "1", 5*time.Minute).Err()
}

func boolToRedis(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func pollsIndexKey() string       { return "polls:index" }
func pollsPublicIndexKey() string { return "polls:public" }
func pollMetaKey(id string) string {
	return fmt.Sprintf("poll:%s", id)
}
func pollOptionsKey(id string) string { return fmt.Sprintf("poll:%s:options", id) }
func pollVotesKey(id string) string   { return fmt.Sprintf("poll:%s:votes", id) }
func pollVotersKey(id string) string  { return fmt.Sprintf("poll:%s:voters", id) }
func userVotesKey(userID string) string {
	return fmt.Sprintf("user:%s:votes", userID)
}
func userCreatedPollsKey(userID string) string {
	return fmt.Sprintf("user:%s:created_polls", userID)
}

func createdScore(t time.Time, id string) float64 {
	// score = unix_seconds + (crc32(id)%1e6)/1e6, to reduce collisions within same second
	frac := float64(crc32.ChecksumIEEE([]byte(id))%1_000_000) / 1_000_000
	return float64(t.Unix()) + frac
}
func idemKey(userID, key string) string {
	if key == "" {
		return "idem:disabled"
	}
	return fmt.Sprintf("idem:%s:%s", userID, key)
}
