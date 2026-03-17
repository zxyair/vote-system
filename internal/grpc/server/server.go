package server

import (
	"context"

	votingv1 "vote-system/internal/gen/voting/v1"
	"vote-system/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	votingv1.UnimplementedVotingServiceServer
	svc *service.Service
}

func New(svc *service.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreatePoll(ctx context.Context, req *votingv1.CreatePollRequest) (*votingv1.CreatePollResponse, error) {
	poll, err := s.svc.CreatePoll(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.CreatePollResponse{Poll: poll}, nil
}

func (s *Server) ClosePoll(ctx context.Context, req *votingv1.ClosePollRequest) (*votingv1.ClosePollResponse, error) {
	poll, err := s.svc.ClosePoll(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.ClosePollResponse{Poll: poll}, nil
}

func (s *Server) DeletePoll(ctx context.Context, req *votingv1.DeletePollRequest) (*votingv1.DeletePollResponse, error) {
	deleted, err := s.svc.DeletePoll(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.DeletePollResponse{Deleted: deleted}, nil
}

func (s *Server) Vote(ctx context.Context, req *votingv1.VoteRequest) (*votingv1.VoteResponse, error) {
	poll, err := s.svc.Vote(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.VoteResponse{Poll: poll}, nil
}

func (s *Server) UndoVote(ctx context.Context, req *votingv1.UndoVoteRequest) (*votingv1.UndoVoteResponse, error) {
	poll, err := s.svc.UndoVote(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.UndoVoteResponse{Poll: poll}, nil
}

func (s *Server) GetPoll(ctx context.Context, req *votingv1.GetPollRequest) (*votingv1.GetPollResponse, error) {
	poll, err := s.svc.GetPoll(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.GetPollResponse{Poll: poll}, nil
}

func (s *Server) SearchPolls(ctx context.Context, req *votingv1.SearchPollsRequest) (*votingv1.SearchPollsResponse, error) {
	polls, next, err := s.svc.SearchPolls(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.SearchPollsResponse{Polls: polls, NextCursor: next}, nil
}

func (s *Server) GetPollStats(ctx context.Context, req *votingv1.GetPollStatsRequest) (*votingv1.GetPollStatsResponse, error) {
	polls, next, err := s.svc.GetPollStats(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.GetPollStatsResponse{Polls: polls, NextCursor: next}, nil
}

func (s *Server) ListPublicPolls(ctx context.Context, req *votingv1.ListPublicPollsRequest) (*votingv1.ListPublicPollsResponse, error) {
	polls, next, err := s.svc.ListPublicPolls(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.ListPublicPollsResponse{Polls: polls, NextCursor: next}, nil
}

func (s *Server) ListPublicPollStats(ctx context.Context, req *votingv1.ListPublicPollStatsRequest) (*votingv1.ListPublicPollStatsResponse, error) {
	polls, next, err := s.svc.ListPublicPollStats(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.ListPublicPollStatsResponse{Polls: polls, NextCursor: next}, nil
}

func (s *Server) GetMyVotes(ctx context.Context, req *votingv1.GetMyVotesRequest) (*votingv1.GetMyVotesResponse, error) {
	votes, next, err := s.svc.GetMyVotes(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.GetMyVotesResponse{Votes: votes, NextCursor: next}, nil
}

func (s *Server) ListMyCreatedPollStats(ctx context.Context, req *votingv1.ListMyCreatedPollStatsRequest) (*votingv1.ListMyCreatedPollStatsResponse, error) {
	polls, next, err := s.svc.ListMyCreatedPollStats(ctx, req)
	if err != nil {
		return nil, toStatus(err)
	}
	return &votingv1.ListMyCreatedPollStatsResponse{Polls: polls, NextCursor: next}, nil
}

func toStatus(err error) error {
	switch {
	case service.IsNotFound(err):
		return status.Error(codes.NotFound, err.Error())
	case service.IsUnauthenticated(err):
		return status.Error(codes.Unauthenticated, err.Error())
	case service.IsForbidden(err):
		return status.Error(codes.PermissionDenied, err.Error())
	case service.IsConflict(err):
		return status.Error(codes.AlreadyExists, err.Error())
	case service.IsInvalid(err):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
