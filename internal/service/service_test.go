package service

import (
	"context"
	"errors"
	"testing"
	"time"

	votingv1 "vote-system/internal/gen/voting/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func futureTimestamp() *timestamppb.Timestamp {
	return timestamppb.New(time.Now().Add(time.Hour))
}

func newTestService() *Service {
	return New(newFakeStore())
}

type fakeStore struct {
	polls map[string]Poll
	votes map[string]map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		polls: map[string]Poll{},
		votes: map[string]map[string]string{},
	}
}

func (s *fakeStore) CreatePoll(_ context.Context, p Poll) (Poll, error) {
	if p.Votes == nil {
		p.Votes = map[string]int64{}
	}
	for _, opt := range p.Options {
		if _, ok := p.Votes[opt]; !ok {
			p.Votes[opt] = 0
		}
	}
	s.polls[p.ID] = cloneTestPoll(p)
	return cloneTestPoll(p), nil
}

func (s *fakeStore) ClosePoll(_ context.Context, pollID, userID, _ string) (Poll, error) {
	p, ok := s.polls[pollID]
	if !ok {
		return Poll{}, ErrNotFound
	}
	if p.CreatedBy != userID {
		return Poll{}, ErrForbidden
	}
	p.IsClosed = true
	p.UpdatedBy = userID
	s.polls[pollID] = p
	return cloneTestPoll(p), nil
}

func (s *fakeStore) DeletePoll(_ context.Context, pollID, _ string, _ string) (bool, error) {
	if _, ok := s.polls[pollID]; !ok {
		return false, ErrNotFound
	}
	delete(s.polls, pollID)
	delete(s.votes, pollID)
	return true, nil
}

func (s *fakeStore) Vote(_ context.Context, pollID, userID, option, _ string) (Poll, error) {
	p, ok := s.polls[pollID]
	if !ok {
		return Poll{}, ErrNotFound
	}
	if s.votes[pollID] == nil {
		s.votes[pollID] = map[string]string{}
	}
	if _, ok := s.votes[pollID][userID]; ok {
		return Poll{}, ErrConflict
	}
	valid := false
	for _, opt := range p.Options {
		if opt == option {
			valid = true
			break
		}
	}
	if !valid {
		return Poll{}, ErrInvalid
	}
	s.votes[pollID][userID] = option
	p.Votes[option]++
	p.UpdatedBy = userID
	s.polls[pollID] = p
	return cloneTestPoll(p), nil
}

func (s *fakeStore) UndoVote(_ context.Context, pollID, userID, _ string) (Poll, error) {
	p, ok := s.polls[pollID]
	if !ok {
		return Poll{}, ErrNotFound
	}
	opt, ok := s.votes[pollID][userID]
	if !ok {
		return Poll{}, ErrConflict
	}
	delete(s.votes[pollID], userID)
	p.Votes[opt]--
	s.polls[pollID] = p
	return cloneTestPoll(p), nil
}

func (s *fakeStore) GetPoll(_ context.Context, pollID string) (Poll, error) {
	p, ok := s.polls[pollID]
	if !ok {
		return Poll{}, ErrNotFound
	}
	return cloneTestPoll(p), nil
}

func (s *fakeStore) SearchPolls(_ context.Context, _ SearchQuery) ([]PollSummary, string, error) {
	out := make([]PollSummary, 0, len(s.polls))
	for _, p := range s.polls {
		out = append(out, PollSummary{
			ID:        p.ID,
			Question:  p.Question,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
			IsClosed:  p.IsClosed,
			CreatedBy: p.CreatedBy,
			IsPublic:  p.IsPublic,
		})
	}
	return out, "", nil
}

func (s *fakeStore) GetPollStats(_ context.Context, _ StatsQuery) ([]Poll, string, error) {
	out := make([]Poll, 0, len(s.polls))
	for _, p := range s.polls {
		out = append(out, cloneTestPoll(p))
	}
	return out, "", nil
}

func (s *fakeStore) ListPublicPolls(_ context.Context, _ StatsQuery) ([]PollSummary, string, error) {
	out := make([]PollSummary, 0)
	for _, p := range s.polls {
		if !p.IsPublic {
			continue
		}
		out = append(out, PollSummary{
			ID:        p.ID,
			Question:  p.Question,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
			IsClosed:  p.IsClosed,
			CreatedBy: p.CreatedBy,
			IsPublic:  p.IsPublic,
		})
	}
	return out, "", nil
}

func (s *fakeStore) ListPublicPollStats(_ context.Context, _ StatsQuery) ([]Poll, string, error) {
	out := make([]Poll, 0)
	for _, p := range s.polls {
		if p.IsPublic {
			out = append(out, cloneTestPoll(p))
		}
	}
	return out, "", nil
}

func (s *fakeStore) GetMyVotes(_ context.Context, userID string, _ int32, _ string) ([]MyVote, string, error) {
	out := make([]MyVote, 0)
	for pollID, byUser := range s.votes {
		opt, ok := byUser[userID]
		if !ok {
			continue
		}
		out = append(out, MyVote{
			PollID: pollID,
			Option: opt,
			Poll:   cloneTestPoll(s.polls[pollID]),
		})
	}
	return out, "", nil
}

func (s *fakeStore) ListMyCreatedPollStats(_ context.Context, userID string, _ StatsQuery) ([]Poll, string, error) {
	out := make([]Poll, 0)
	for _, p := range s.polls {
		if p.CreatedBy == userID {
			out = append(out, cloneTestPoll(p))
		}
	}
	return out, "", nil
}

func cloneTestPoll(p Poll) Poll {
	p.Options = append([]string(nil), p.Options...)
	votes := make(map[string]int64, len(p.Votes))
	for k, v := range p.Votes {
		votes[k] = v
	}
	p.Votes = votes
	return p
}

func TestCreatePollValidation(t *testing.T) {
	tests := []struct {
		name string
		req  *votingv1.CreatePollRequest
		want error
	}{
		{
			name: "missing user",
			req: &votingv1.CreatePollRequest{
				Question:  "topic",
				Options:   []string{"Go", "Redis", "Kubernetes"},
				ExpiresAt: futureTimestamp(),
			},
			want: ErrUnauthenticated,
		},
		{
			name: "empty question",
			req: &votingv1.CreatePollRequest{
				UserId:    "user_1",
				Question:  " ",
				Options:   []string{"Go", "Redis", "Kubernetes"},
				ExpiresAt: futureTimestamp(),
			},
			want: ErrInvalid,
		},
		{
			name: "too few options",
			req: &votingv1.CreatePollRequest{
				UserId:    "user_1",
				Question:  "topic",
				Options:   []string{"Go", "Redis"},
				ExpiresAt: futureTimestamp(),
			},
			want: ErrInvalid,
		},
		{
			name: "duplicate option",
			req: &votingv1.CreatePollRequest{
				UserId:    "user_1",
				Question:  "topic",
				Options:   []string{"Go", "Redis", "Go"},
				ExpiresAt: futureTimestamp(),
			},
			want: ErrInvalid,
		},
		{
			name: "expires too soon",
			req: &votingv1.CreatePollRequest{
				UserId:    "user_1",
				Question:  "topic",
				Options:   []string{"Go", "Redis", "Kubernetes"},
				ExpiresAt: timestamppb.New(time.Now().Add(5 * time.Second)),
			},
			want: ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newTestService().CreatePoll(context.Background(), tt.req)
			if !errors.Is(err, tt.want) {
				t.Fatalf("CreatePoll error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCreatePollSuccess(t *testing.T) {
	got, err := newTestService().CreatePoll(context.Background(), &votingv1.CreatePollRequest{
		UserId:    "creator_1",
		Question:  "Best topic?",
		Options:   []string{" Go ", "Redis", "Kubernetes"},
		ExpiresAt: futureTimestamp(),
		IsPublic:  true,
	})
	if err != nil {
		t.Fatalf("CreatePoll returned error: %v", err)
	}
	if got.GetId() == "" {
		t.Fatal("CreatePoll returned empty id")
	}
	if got.GetCreatedBy() != "creator_1" || got.GetUpdatedBy() != "creator_1" {
		t.Fatalf("unexpected creator fields: created_by=%q updated_by=%q", got.GetCreatedBy(), got.GetUpdatedBy())
	}
	if len(got.GetOptions()) != 3 || got.GetOptions()[0] != "Go" {
		t.Fatalf("options were not normalized: %#v", got.GetOptions())
	}
	if !got.GetIsPublic() {
		t.Fatal("CreatePoll did not preserve is_public")
	}
}

func TestServiceVoteValidation(t *testing.T) {
	svc := newTestService()

	if _, err := svc.Vote(context.Background(), &votingv1.VoteRequest{PollId: "p", Option: "Go"}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Vote missing user error = %v, want %v", err, ErrUnauthenticated)
	}
	if _, err := svc.Vote(context.Background(), &votingv1.VoteRequest{UserId: "user_1", Option: "Go"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Vote missing poll error = %v, want %v", err, ErrInvalid)
	}
	if _, err := svc.Vote(context.Background(), &votingv1.VoteRequest{UserId: "user_1", PollId: "p"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Vote missing option error = %v, want %v", err, ErrInvalid)
	}
}

func TestServiceUndoDeleteAndListValidation(t *testing.T) {
	svc := newTestService()

	if _, err := svc.UndoVote(context.Background(), &votingv1.UndoVoteRequest{PollId: "p"}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("UndoVote missing user error = %v, want %v", err, ErrUnauthenticated)
	}
	if _, err := svc.UndoVote(context.Background(), &votingv1.UndoVoteRequest{UserId: "user_1"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("UndoVote missing poll error = %v, want %v", err, ErrInvalid)
	}
	if _, err := svc.DeletePoll(context.Background(), &votingv1.DeletePollRequest{PollId: "p"}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("DeletePoll missing user error = %v, want %v", err, ErrUnauthenticated)
	}
	if _, err := svc.DeletePoll(context.Background(), &votingv1.DeletePollRequest{UserId: "user_1"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("DeletePoll missing poll error = %v, want %v", err, ErrInvalid)
	}
	if _, _, err := svc.GetMyVotes(context.Background(), &votingv1.GetMyVotesRequest{}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("GetMyVotes missing user error = %v, want %v", err, ErrUnauthenticated)
	}
	if _, _, err := svc.ListMyCreatedPollStats(context.Background(), &votingv1.ListMyCreatedPollStatsRequest{}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("ListMyCreatedPollStats missing user error = %v, want %v", err, ErrUnauthenticated)
	}
}

func TestServiceEndToEndWithMemoryStore(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	poll, err := svc.CreatePoll(ctx, &votingv1.CreatePollRequest{
		UserId:    "creator_1",
		Question:  "Best topic?",
		Options:   []string{"Go", "Redis", "Kubernetes"},
		ExpiresAt: futureTimestamp(),
		IsPublic:  true,
	})
	if err != nil {
		t.Fatalf("CreatePoll returned error: %v", err)
	}

	voted, err := svc.Vote(ctx, &votingv1.VoteRequest{
		UserId: "voter_1",
		PollId: poll.GetId(),
		Option: "Redis",
	})
	if err != nil {
		t.Fatalf("Vote returned error: %v", err)
	}
	if got := voted.GetVotes()["Redis"]; got != 1 {
		t.Fatalf("Redis votes = %d, want 1", got)
	}

	myVotes, _, err := svc.GetMyVotes(ctx, &votingv1.GetMyVotesRequest{UserId: "voter_1"})
	if err != nil {
		t.Fatalf("GetMyVotes returned error: %v", err)
	}
	if len(myVotes) != 1 || myVotes[0].GetPollId() != poll.GetId() || myVotes[0].GetOption() != "Redis" {
		t.Fatalf("unexpected my votes: %#v", myVotes)
	}

	stats, _, err := svc.ListPublicPollStats(ctx, &votingv1.ListPublicPollStatsRequest{IncludeClosed: true})
	if err != nil {
		t.Fatalf("ListPublicPollStats returned error: %v", err)
	}
	if len(stats) != 1 || stats[0].GetId() != poll.GetId() {
		t.Fatalf("unexpected public stats: %#v", stats)
	}

	undone, err := svc.UndoVote(ctx, &votingv1.UndoVoteRequest{UserId: "voter_1", PollId: poll.GetId()})
	if err != nil {
		t.Fatalf("UndoVote returned error: %v", err)
	}
	if got := undone.GetVotes()["Redis"]; got != 0 {
		t.Fatalf("Redis votes after undo = %d, want 0", got)
	}
}

func TestServiceQueryAndCloseMethods(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	poll, err := svc.CreatePoll(ctx, &votingv1.CreatePollRequest{
		UserId:    "creator_1",
		Question:  "Best topic?",
		Options:   []string{"Go", "Redis", "Kubernetes"},
		ExpiresAt: futureTimestamp(),
		IsPublic:  true,
	})
	if err != nil {
		t.Fatalf("CreatePoll returned error: %v", err)
	}

	got, err := svc.GetPoll(ctx, &votingv1.GetPollRequest{PollId: poll.GetId()})
	if err != nil {
		t.Fatalf("GetPoll returned error: %v", err)
	}
	if got.GetId() != poll.GetId() {
		t.Fatalf("GetPoll id = %q, want %q", got.GetId(), poll.GetId())
	}

	if _, err := svc.GetPoll(ctx, &votingv1.GetPollRequest{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("GetPoll missing id error = %v, want %v", err, ErrInvalid)
	}

	search, _, err := svc.SearchPolls(ctx, &votingv1.SearchPollsRequest{
		CreatedBy:     "creator_1",
		IncludeClosed: true,
		CreatedAfter:  timestamppb.New(time.Now().Add(-time.Hour)),
		CreatedBefore: timestamppb.New(time.Now().Add(time.Hour)),
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("SearchPolls returned error: %v", err)
	}
	if len(search) != 1 || search[0].GetId() != poll.GetId() {
		t.Fatalf("unexpected search result: %#v", search)
	}

	allStats, _, err := svc.GetPollStats(ctx, &votingv1.GetPollStatsRequest{IncludeClosed: true, Limit: 10})
	if err != nil {
		t.Fatalf("GetPollStats returned error: %v", err)
	}
	if len(allStats) != 1 || allStats[0].GetId() != poll.GetId() {
		t.Fatalf("unexpected stats result: %#v", allStats)
	}

	publicPolls, _, err := svc.ListPublicPolls(ctx, &votingv1.ListPublicPollsRequest{IncludeClosed: true, Limit: 10})
	if err != nil {
		t.Fatalf("ListPublicPolls returned error: %v", err)
	}
	if len(publicPolls) != 1 || publicPolls[0].GetId() != poll.GetId() {
		t.Fatalf("unexpected public polls: %#v", publicPolls)
	}

	created, _, err := svc.ListMyCreatedPollStats(ctx, &votingv1.ListMyCreatedPollStatsRequest{
		UserId:        "creator_1",
		IncludeClosed: true,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListMyCreatedPollStats returned error: %v", err)
	}
	if len(created) != 1 || created[0].GetId() != poll.GetId() {
		t.Fatalf("unexpected created stats: %#v", created)
	}

	closed, err := svc.ClosePoll(ctx, &votingv1.ClosePollRequest{
		UserId: "creator_1",
		PollId: poll.GetId(),
	})
	if err != nil {
		t.Fatalf("ClosePoll returned error: %v", err)
	}
	if !closed.GetIsClosed() {
		t.Fatal("ClosePoll returned open poll")
	}

	if _, err := svc.ClosePoll(ctx, &votingv1.ClosePollRequest{PollId: poll.GetId()}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("ClosePoll missing user error = %v, want %v", err, ErrUnauthenticated)
	}
	if _, err := svc.ClosePoll(ctx, &votingv1.ClosePollRequest{UserId: "creator_1"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ClosePoll missing poll error = %v, want %v", err, ErrInvalid)
	}
}
