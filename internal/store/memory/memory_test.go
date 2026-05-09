package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"vote-system/internal/service"
)

func testPoll(id string) service.Poll {
	return service.Poll{
		ID:        id,
		Question:  "Best topic?",
		Options:   []string{"Go", "Redis", "Kubernetes"},
		Votes:     map[string]int64{"Go": 0, "Redis": 0, "Kubernetes": 0},
		CreatedBy: "creator_1",
		UpdatedBy: "creator_1",
		CreatedAt: time.Now().Add(-time.Minute),
		ExpiresAt: time.Now().Add(time.Hour),
		IsPublic:  true,
	}
}

func TestMemoryStoreVoteUndoAndDuplicate(t *testing.T) {
	ctx := context.Background()
	store := New()

	if _, err := store.CreatePoll(ctx, testPoll("poll_1")); err != nil {
		t.Fatalf("CreatePoll returned error: %v", err)
	}

	voted, err := store.Vote(ctx, "poll_1", "voter_1", "Go", "")
	if err != nil {
		t.Fatalf("Vote returned error: %v", err)
	}
	if got := voted.Votes["Go"]; got != 1 {
		t.Fatalf("Go votes = %d, want 1", got)
	}

	if _, err := store.Vote(ctx, "poll_1", "voter_1", "Redis", ""); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("duplicate Vote error = %v, want %v", err, service.ErrConflict)
	}

	undone, err := store.UndoVote(ctx, "poll_1", "voter_1", "")
	if err != nil {
		t.Fatalf("UndoVote returned error: %v", err)
	}
	if got := undone.Votes["Go"]; got != 0 {
		t.Fatalf("Go votes after undo = %d, want 0", got)
	}

	if _, err := store.UndoVote(ctx, "poll_1", "voter_1", ""); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("second UndoVote error = %v, want %v", err, service.ErrConflict)
	}
}

func TestMemoryStoreRejectsInvalidAndExpiredVotes(t *testing.T) {
	ctx := context.Background()
	store := New()

	p := testPoll("poll_1")
	if _, err := store.CreatePoll(ctx, p); err != nil {
		t.Fatalf("CreatePoll returned error: %v", err)
	}
	if _, err := store.Vote(ctx, "poll_1", "voter_1", "Rust", ""); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("invalid option Vote error = %v, want %v", err, service.ErrInvalid)
	}

	expired := testPoll("poll_2")
	expired.ExpiresAt = time.Now().Add(-time.Minute)
	if _, err := store.CreatePoll(ctx, expired); err != nil {
		t.Fatalf("CreatePoll expired returned error: %v", err)
	}
	if _, err := store.Vote(ctx, "poll_2", "voter_1", "Go", ""); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("expired Vote error = %v, want %v", err, service.ErrForbidden)
	}
}

func TestMemoryStoreListsAndDeletes(t *testing.T) {
	ctx := context.Background()
	store := New()

	publicPoll := testPoll("public")
	privatePoll := testPoll("private")
	privatePoll.IsPublic = false
	privatePoll.CreatedBy = "creator_2"

	if _, err := store.CreatePoll(ctx, publicPoll); err != nil {
		t.Fatalf("CreatePoll public returned error: %v", err)
	}
	if _, err := store.CreatePoll(ctx, privatePoll); err != nil {
		t.Fatalf("CreatePoll private returned error: %v", err)
	}

	public, _, err := store.ListPublicPolls(ctx, service.StatsQuery{IncludeClosed: true})
	if err != nil {
		t.Fatalf("ListPublicPolls returned error: %v", err)
	}
	if len(public) != 1 || public[0].ID != "public" {
		t.Fatalf("unexpected public polls: %#v", public)
	}

	created, _, err := store.ListMyCreatedPollStats(ctx, "creator_2", service.StatsQuery{IncludeClosed: true})
	if err != nil {
		t.Fatalf("ListMyCreatedPollStats returned error: %v", err)
	}
	if len(created) != 1 || created[0].ID != "private" {
		t.Fatalf("unexpected created stats: %#v", created)
	}

	deleted, err := store.DeletePoll(ctx, "private", "creator_2", "")
	if err != nil {
		t.Fatalf("DeletePoll returned error: %v", err)
	}
	if !deleted {
		t.Fatal("DeletePoll returned false, want true")
	}
	if _, err := store.GetPoll(ctx, "private"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetPoll after delete error = %v, want %v", err, service.ErrNotFound)
	}
}

func TestMemoryStoreSearchStatsMyVotesAndClose(t *testing.T) {
	ctx := context.Background()
	store := New()

	p1 := testPoll("poll_1")
	p1.CreatedAt = time.Now().Add(-2 * time.Hour)
	p2 := testPoll("poll_2")
	p2.CreatedBy = "creator_2"
	p2.IsPublic = false

	if _, err := store.CreatePoll(ctx, p1); err != nil {
		t.Fatalf("CreatePoll p1 returned error: %v", err)
	}
	if _, err := store.CreatePoll(ctx, p2); err != nil {
		t.Fatalf("CreatePoll p2 returned error: %v", err)
	}

	found, _, err := store.SearchPolls(ctx, service.SearchQuery{
		CreatedBy:     "creator_1",
		IncludeClosed: true,
		CreatedAfter:  ptrTime(time.Now().Add(-3 * time.Hour)),
		CreatedBefore: ptrTime(time.Now()),
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("SearchPolls returned error: %v", err)
	}
	if len(found) != 1 || found[0].ID != "poll_1" {
		t.Fatalf("unexpected search result: %#v", found)
	}

	allStats, _, err := store.GetPollStats(ctx, service.StatsQuery{IncludeClosed: true})
	if err != nil {
		t.Fatalf("GetPollStats returned error: %v", err)
	}
	if len(allStats) != 2 {
		t.Fatalf("GetPollStats len = %d, want 2", len(allStats))
	}

	publicStats, _, err := store.ListPublicPollStats(ctx, service.StatsQuery{IncludeClosed: true})
	if err != nil {
		t.Fatalf("ListPublicPollStats returned error: %v", err)
	}
	if len(publicStats) != 1 || publicStats[0].ID != "poll_1" {
		t.Fatalf("unexpected public stats: %#v", publicStats)
	}

	if _, err := store.Vote(ctx, "poll_1", "voter_1", "Redis", ""); err != nil {
		t.Fatalf("Vote returned error: %v", err)
	}
	myVotes, _, err := store.GetMyVotes(ctx, "voter_1", 10, "")
	if err != nil {
		t.Fatalf("GetMyVotes returned error: %v", err)
	}
	if len(myVotes) != 1 || myVotes[0].PollID != "poll_1" || myVotes[0].Option != "Redis" {
		t.Fatalf("unexpected my votes: %#v", myVotes)
	}

	closed, err := store.ClosePoll(ctx, "poll_1", "creator_1", "")
	if err != nil {
		t.Fatalf("ClosePoll returned error: %v", err)
	}
	if !closed.IsClosed || closed.UpdatedBy != "creator_1" {
		t.Fatalf("unexpected closed poll: %#v", closed)
	}

	if _, err := store.ClosePoll(ctx, "missing", "creator_1", ""); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("ClosePoll missing error = %v, want %v", err, service.ErrNotFound)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func TestMemoryStoreClonesPolls(t *testing.T) {
	ctx := context.Background()
	store := New()

	created, err := store.CreatePoll(ctx, testPoll("poll_1"))
	if err != nil {
		t.Fatalf("CreatePoll returned error: %v", err)
	}
	created.Options[0] = "mutated"
	created.Votes["Go"] = 99

	got, err := store.GetPoll(ctx, "poll_1")
	if err != nil {
		t.Fatalf("GetPoll returned error: %v", err)
	}
	if got.Options[0] != "Go" || got.Votes["Go"] != 0 {
		t.Fatalf("store returned mutable state: options=%#v votes=%#v", got.Options, got.Votes)
	}
}
