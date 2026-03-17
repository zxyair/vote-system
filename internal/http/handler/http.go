package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	votingv1 "vote-system/internal/gen/voting/v1"
	"vote-system/internal/http/middleware"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type HTTP struct {
	Client votingv1.VotingServiceClient
	hub    *sseHub
}

func (h *HTTP) notifyPollInvalidate(pollID, reason string) {
	if h == nil || h.hub == nil || pollID == "" {
		return
	}
	msg, err := json.Marshal(pollInvalidateMsg{PollID: pollID, Reason: reason})
	if err != nil {
		return
	}
	h.hub.notifyPoll(pollID, msg)
}

func New(client votingv1.VotingServiceClient, hub *sseHub) *HTTP {
	return &HTTP{Client: client, hub: hub}
}

type createPollBody struct {
	Question  string   `json:"question"`
	Options   []string `json:"options"`
	ExpiresAt string   `json:"expires_at"` // RFC3339
	IsPublic  bool     `json:"is_public"`
}

func (h *HTTP) CreatePoll(c *gin.Context) {
	var body createPollBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	exp, err := time.Parse(time.RFC3339, body.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be RFC3339"})
		return
	}

	req := &votingv1.CreatePollRequest{
		UserId:         middleware.UserID(c),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Question:       body.Question,
		Options:        body.Options,
		ExpiresAt:      timestamppb.New(exp),
		IsPublic:       body.IsPublic,
	}

	resp, err := h.Client.CreatePoll(c.Request.Context(), req)
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	// notify creator + public stats if needed
	if h.hub != nil && resp.Poll != nil {
		h.notifyPollInvalidate(resp.Poll.GetId(), "create")
		if msg, err := json.Marshal(invalidateMsg{MyCreated: true}); err == nil {
			h.hub.notifyUser(resp.Poll.GetCreatedBy(), msg)
		}
		if resp.Poll.GetIsPublic() {
			if msg, err := json.Marshal(invalidateMsg{PublicStats: true}); err == nil {
				h.hub.broadcast(msg)
			}
		}
	}
	c.JSON(http.StatusOK, resp.Poll)
}

func (h *HTTP) GetPoll(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.Client.GetPoll(c.Request.Context(), &votingv1.GetPollRequest{PollId: id})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, resp.Poll)
}

func (h *HTTP) ClosePoll(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.Client.ClosePoll(c.Request.Context(), &votingv1.ClosePollRequest{
		UserId:         middleware.UserID(c),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		PollId:         id,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	if h.hub != nil && resp.Poll != nil {
		h.notifyPollInvalidate(resp.Poll.GetId(), "close")
		if msg, err := json.Marshal(invalidateMsg{MyCreated: true}); err == nil {
			h.hub.notifyUser(resp.Poll.GetCreatedBy(), msg)
		}
		if resp.Poll.GetIsPublic() {
			if msg, err := json.Marshal(invalidateMsg{PublicStats: true}); err == nil {
				h.hub.broadcast(msg)
			}
		}
	}
	c.JSON(http.StatusOK, resp.Poll)
}

func (h *HTTP) DeletePoll(c *gin.Context) {
	id := c.Param("id")
	// prefetch poll for notifications (delete response doesn't contain poll)
	var pref *votingv1.Poll
	if h.hub != nil {
		if got, err := h.Client.GetPoll(c.Request.Context(), &votingv1.GetPollRequest{PollId: id}); err == nil {
			pref = got.Poll
		}
	}
	resp, err := h.Client.DeletePoll(c.Request.Context(), &votingv1.DeletePollRequest{
		UserId:         middleware.UserID(c),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		PollId:         id,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	if h.hub != nil && pref != nil {
		h.notifyPollInvalidate(pref.GetId(), "delete")
		if msg, err := json.Marshal(invalidateMsg{MyCreated: true}); err == nil {
			h.hub.notifyUser(pref.GetCreatedBy(), msg)
		}
		if pref.GetIsPublic() {
			if msg, err := json.Marshal(invalidateMsg{PublicStats: true}); err == nil {
				h.hub.broadcast(msg)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"deleted": resp.Deleted})
}

type voteBody struct {
	Option string `json:"option"`
}

func (h *HTTP) Vote(c *gin.Context) {
	pollID := c.Param("poll_id")
	var body voteBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Option == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "option required"})
		return
	}

	resp, err := h.Client.Vote(c.Request.Context(), &votingv1.VoteRequest{
		UserId:         middleware.UserID(c),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		PollId:         pollID,
		Option:         body.Option,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	if h.hub != nil && resp.Poll != nil {
		h.notifyPollInvalidate(resp.Poll.GetId(), "vote")
		// voter
		if msg, err := json.Marshal(invalidateMsg{MyVotes: true}); err == nil {
			h.hub.notifyUser(middleware.UserID(c), msg)
		}
		// creator's created list stats
		if msg, err := json.Marshal(invalidateMsg{MyCreated: true}); err == nil {
			h.hub.notifyUser(resp.Poll.GetCreatedBy(), msg)
		}
		// public stats if public poll
		if resp.Poll.GetIsPublic() {
			if msg, err := json.Marshal(invalidateMsg{PublicStats: true}); err == nil {
				h.hub.broadcast(msg)
			}
		}
	}
	c.JSON(http.StatusOK, resp.Poll)
}

func (h *HTTP) UndoVote(c *gin.Context) {
	pollID := c.Param("poll_id")
	resp, err := h.Client.UndoVote(c.Request.Context(), &votingv1.UndoVoteRequest{
		UserId:         middleware.UserID(c),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		PollId:         pollID,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	if h.hub != nil && resp.Poll != nil {
		h.notifyPollInvalidate(resp.Poll.GetId(), "undo")
		if msg, err := json.Marshal(invalidateMsg{MyVotes: true}); err == nil {
			h.hub.notifyUser(middleware.UserID(c), msg)
		}
		if msg, err := json.Marshal(invalidateMsg{MyCreated: true}); err == nil {
			h.hub.notifyUser(resp.Poll.GetCreatedBy(), msg)
		}
		if resp.Poll.GetIsPublic() {
			if msg, err := json.Marshal(invalidateMsg{PublicStats: true}); err == nil {
				h.hub.broadcast(msg)
			}
		}
	}
	c.JSON(http.StatusOK, resp.Poll)
}

func (h *HTTP) SearchPolls(c *gin.Context) {
	createdBy := c.Query("created_by")
	includeClosed := c.Query("include_closed") == "true"

	resp, err := h.Client.SearchPolls(c.Request.Context(), &votingv1.SearchPollsRequest{
		CreatedBy:     createdBy,
		IncludeClosed: includeClosed,
		Limit:         50,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HTTP) GetPollStats(c *gin.Context) {
	includeClosed := c.Query("include_closed") == "true"
	resp, err := h.Client.GetPollStats(c.Request.Context(), &votingv1.GetPollStatsRequest{
		IncludeClosed: includeClosed,
		Limit:         50,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HTTP) ListPublicPolls(c *gin.Context) {
	includeClosed := c.Query("include_closed") == "true"
	resp, err := h.Client.ListPublicPolls(c.Request.Context(), &votingv1.ListPublicPollsRequest{
		IncludeClosed: includeClosed,
		Limit:         50,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HTTP) ListPublicPollStats(c *gin.Context) {
	includeClosed := c.Query("include_closed") == "true"
	resp, err := h.Client.ListPublicPollStats(c.Request.Context(), &votingv1.ListPublicPollStatsRequest{
		IncludeClosed: includeClosed,
		Limit:         50,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HTTP) GetMyVotes(c *gin.Context) {
	resp, err := h.Client.GetMyVotes(c.Request.Context(), &votingv1.GetMyVotesRequest{
		UserId: middleware.UserID(c),
		Limit:  50,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HTTP) ListMyCreatedPollStats(c *gin.Context) {
	includeClosed := c.Query("include_closed") == "true"
	limit := int32(50)
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 && n <= 100 {
			limit = int32(n)
		}
	}
	cursor := c.Query("cursor")
	resp, err := h.Client.ListMyCreatedPollStats(c.Request.Context(), &votingv1.ListMyCreatedPollStatsRequest{
		UserId:        middleware.UserID(c),
		IncludeClosed: includeClosed,
		Limit:         limit,
		Cursor:        cursor,
	})
	if err != nil {
		code, msg := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, resp)
}
