package service

import (
	"context"
	"strings"
	"time"

	votingv1 "vote-system/internal/gen/voting/v1"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Store interface {
	CreatePoll(ctx context.Context, p Poll) (Poll, error)
	ClosePoll(ctx context.Context, pollID, userID, idempotencyKey string) (Poll, error)
	DeletePoll(ctx context.Context, pollID, userID, idempotencyKey string) (bool, error)
	Vote(ctx context.Context, pollID, userID, option, idempotencyKey string) (Poll, error)
	UndoVote(ctx context.Context, pollID, userID, idempotencyKey string) (Poll, error)
	GetPoll(ctx context.Context, pollID string) (Poll, error)
	SearchPolls(ctx context.Context, q SearchQuery) ([]PollSummary, string, error)
	GetPollStats(ctx context.Context, q StatsQuery) ([]Poll, string, error)
	ListPublicPolls(ctx context.Context, q StatsQuery) ([]PollSummary, string, error)
	ListPublicPollStats(ctx context.Context, q StatsQuery) ([]Poll, string, error)
	GetMyVotes(ctx context.Context, userID string, limit int32, cursor string) ([]MyVote, string, error)
	ListMyCreatedPollStats(ctx context.Context, userID string, q StatsQuery) ([]Poll, string, error)
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreatePoll(ctx context.Context, req *votingv1.CreatePollRequest) (*votingv1.Poll, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	if strings.TrimSpace(req.GetQuestion()) == "" {
		return nil, ErrInvalid
	}
	expiresAt := req.GetExpiresAt().AsTime()
	if req.GetExpiresAt() == nil || expiresAt.Before(time.Now().Add(30*time.Second)) {
		return nil, ErrInvalid
	}

	// 选项去空格 + 唯一性校验（大小写敏感），且至少 3 个唯一选项
	rawOpts := req.GetOptions()
	cleanOpts := make([]string, 0, len(rawOpts))
	seen := make(map[string]struct{}, len(rawOpts))
	for _, o := range rawOpts {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if _, ok := seen[o]; ok {
			// 前端会友好提示，这里统一返回 ErrInvalid
			return nil, ErrInvalid
		}
		seen[o] = struct{}{}
		cleanOpts = append(cleanOpts, o)
	}
	if len(cleanOpts) < 3 {
		return nil, ErrInvalid
	}

	p := Poll{
		ID:        uuid.NewString(),
		Question:  req.GetQuestion(),
		Options:   cleanOpts,
		Votes:     map[string]int64{},
		CreatedBy: userID,
		UpdatedBy: userID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt.UTC(),
		IsClosed:  false,
		IsPublic:  req.GetIsPublic(),
	}

	created, err := s.store.CreatePoll(ctx, p)
	if err != nil {
		return nil, err
	}
	return toProtoPoll(created), nil
}

func (s *Service) GetPoll(ctx context.Context, req *votingv1.GetPollRequest) (*votingv1.Poll, error) {
	pollID := strings.TrimSpace(req.GetPollId())
	if pollID == "" {
		return nil, ErrInvalid
	}
	p, err := s.store.GetPoll(ctx, pollID)
	if err != nil {
		return nil, err
	}
	return toProtoPoll(p), nil
}

func (s *Service) ClosePoll(ctx context.Context, req *votingv1.ClosePollRequest) (*votingv1.Poll, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	pollID := strings.TrimSpace(req.GetPollId())
	if pollID == "" {
		return nil, ErrInvalid
	}

	closed, err := s.store.ClosePoll(ctx, pollID, userID, req.GetIdempotencyKey())
	if err != nil {
		return nil, err
	}
	return toProtoPoll(closed), nil
}

func (s *Service) DeletePoll(ctx context.Context, req *votingv1.DeletePollRequest) (bool, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return false, ErrUnauthenticated
	}
	pollID := strings.TrimSpace(req.GetPollId())
	if pollID == "" {
		return false, ErrInvalid
	}
	return s.store.DeletePoll(ctx, pollID, userID, req.GetIdempotencyKey())
}

func (s *Service) Vote(ctx context.Context, req *votingv1.VoteRequest) (*votingv1.Poll, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	pollID := strings.TrimSpace(req.GetPollId())
	if pollID == "" {
		return nil, ErrInvalid
	}
	option := strings.TrimSpace(req.GetOption())
	if option == "" {
		return nil, ErrInvalid
	}

	p, err := s.store.Vote(ctx, pollID, userID, option, req.GetIdempotencyKey())
	if err != nil {
		return nil, err
	}
	return toProtoPoll(p), nil
}

func (s *Service) UndoVote(ctx context.Context, req *votingv1.UndoVoteRequest) (*votingv1.Poll, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	pollID := strings.TrimSpace(req.GetPollId())
	if pollID == "" {
		return nil, ErrInvalid
	}

	p, err := s.store.UndoVote(ctx, pollID, userID, req.GetIdempotencyKey())
	if err != nil {
		return nil, err
	}
	return toProtoPoll(p), nil
}

func (s *Service) SearchPolls(ctx context.Context, req *votingv1.SearchPollsRequest) ([]*votingv1.PollSummary, string, error) {
	q := SearchQuery{
		CreatedBy:     strings.TrimSpace(req.GetCreatedBy()),
		IncludeClosed: req.GetIncludeClosed(),
		CreatedAfter:  tsToTime(req.GetCreatedAfter()),
		CreatedBefore: tsToTime(req.GetCreatedBefore()),
		Limit:         req.GetLimit(),
		Cursor:        req.GetCursor(),
	}
	items, next, err := s.store.SearchPolls(ctx, q)
	if err != nil {
		return nil, "", err
	}
	out := make([]*votingv1.PollSummary, 0, len(items))
	for _, it := range items {
		out = append(out, &votingv1.PollSummary{
			Id:        it.ID,
			Question:  it.Question,
			CreatedAt: timestamppb.New(it.CreatedAt),
			ExpiresAt: timestamppb.New(it.ExpiresAt),
			IsClosed:  it.IsClosed,
			CreatedBy: it.CreatedBy,
		})
	}
	return out, next, nil
}

func (s *Service) GetPollStats(ctx context.Context, req *votingv1.GetPollStatsRequest) ([]*votingv1.Poll, string, error) {
	q := StatsQuery{
		IncludeClosed: req.GetIncludeClosed(),
		Limit:         req.GetLimit(),
		Cursor:        req.GetCursor(),
	}
	items, next, err := s.store.GetPollStats(ctx, q)
	if err != nil {
		return nil, "", err
	}
	out := make([]*votingv1.Poll, 0, len(items))
	for _, it := range items {
		out = append(out, toProtoPoll(it))
	}
	return out, next, nil
}

func (s *Service) ListPublicPolls(ctx context.Context, req *votingv1.ListPublicPollsRequest) ([]*votingv1.PollSummary, string, error) {
	q := StatsQuery{
		IncludeClosed: req.GetIncludeClosed(),
		Limit:         req.GetLimit(),
		Cursor:        req.GetCursor(),
	}
	items, next, err := s.store.ListPublicPolls(ctx, q)
	if err != nil {
		return nil, "", err
	}
	out := make([]*votingv1.PollSummary, 0, len(items))
	for _, it := range items {
		out = append(out, &votingv1.PollSummary{
			Id:        it.ID,
			Question:  it.Question,
			CreatedAt: timestamppb.New(it.CreatedAt),
			ExpiresAt: timestamppb.New(it.ExpiresAt),
			IsClosed:  it.IsClosed,
			CreatedBy: it.CreatedBy,
		})
	}
	return out, next, nil
}

func (s *Service) ListPublicPollStats(ctx context.Context, req *votingv1.ListPublicPollStatsRequest) ([]*votingv1.Poll, string, error) {
	q := StatsQuery{
		IncludeClosed: req.GetIncludeClosed(),
		Limit:         req.GetLimit(),
		Cursor:        req.GetCursor(),
	}
	items, next, err := s.store.ListPublicPollStats(ctx, q)
	if err != nil {
		return nil, "", err
	}
	out := make([]*votingv1.Poll, 0, len(items))
	for _, it := range items {
		out = append(out, toProtoPoll(it))
	}
	return out, next, nil
}

func (s *Service) GetMyVotes(ctx context.Context, req *votingv1.GetMyVotesRequest) ([]*votingv1.MyVote, string, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, "", ErrUnauthenticated
	}
	items, next, err := s.store.GetMyVotes(ctx, userID, req.GetLimit(), req.GetCursor())
	if err != nil {
		return nil, "", err
	}
	out := make([]*votingv1.MyVote, 0, len(items))
	for _, it := range items {
		out = append(out, &votingv1.MyVote{
			PollId: it.PollID,
			Option: it.Option,
			Poll:   toProtoPoll(it.Poll),
		})
	}
	return out, next, nil
}

func (s *Service) ListMyCreatedPollStats(ctx context.Context, req *votingv1.ListMyCreatedPollStatsRequest) ([]*votingv1.Poll, string, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, "", ErrUnauthenticated
	}
	q := StatsQuery{
		IncludeClosed: req.GetIncludeClosed(),
		Limit:         req.GetLimit(),
		Cursor:        req.GetCursor(),
	}
	items, next, err := s.store.ListMyCreatedPollStats(ctx, userID, q)
	if err != nil {
		return nil, "", err
	}
	out := make([]*votingv1.Poll, 0, len(items))
	for _, it := range items {
		out = append(out, toProtoPoll(it))
	}
	return out, next, nil
}

func toProtoPoll(p Poll) *votingv1.Poll {
	votes := make(map[string]int64, len(p.Votes))
	for k, v := range p.Votes {
		votes[k] = v
	}
	return &votingv1.Poll{
		Id:        p.ID,
		Question:  p.Question,
		Options:   append([]string(nil), p.Options...),
		Votes:     votes,
		CreatedBy: p.CreatedBy,
		UpdatedBy: p.UpdatedBy,
		CreatedAt: timestamppb.New(p.CreatedAt),
		ExpiresAt: timestamppb.New(p.ExpiresAt),
		IsClosed:  p.IsClosed,
		IsPublic:  p.IsPublic,
	}
}

func normalizeOptions(opts []string) []string {
	out := make([]string, 0, len(opts))
	seen := make(map[string]struct{}, len(opts))
	for _, o := range opts {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		key := strings.ToLower(o)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, o)
	}
	return out
}

func tsToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime().UTC()
	return &t
}
