package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"vote-system/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

type pollInvalidateMsg struct {
	PollID string `json:"pollId"`
	Reason string `json:"reason"`
}

func (h *HTTP) ResultsSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		c.Status(500)
		return
	}

	userID := middleware.UserID(c)
	ctx := c.Request.Context()

	conn := h.hub.add(userID)
	defer h.hub.remove(conn)

	// initial invalidate so client pulls once immediately
	initial, _ := json.Marshal(invalidateMsg{MyVotes: true, PublicStats: true, MyCreated: true})
	_, _ = fmt.Fprintf(c.Writer, "event: invalidate\ndata: %s\n\n", initial)
	flusher.Flush()

	// keep-alive comments
	ka := time.NewTicker(20 * time.Second)
	defer ka.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-conn.ch:
			_, _ = fmt.Fprintf(c.Writer, "event: invalidate\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-ka.C:
			_, _ = io.WriteString(c.Writer, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (h *HTTP) PollSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		c.Status(500)
		return
	}

	// ensure auth middleware ran (also keeps parity with /events/results)
	_ = middleware.UserID(c)

	pollID := c.Param("poll_id")
	if pollID == "" {
		c.Status(400)
		return
	}

	ctx := c.Request.Context()
	conn := h.hub.addPoll(pollID)
	defer h.hub.removePoll(conn)

	// initial invalidate so client pulls once immediately
	initial, _ := json.Marshal(pollInvalidateMsg{PollID: pollID, Reason: "init"})
	_, _ = fmt.Fprintf(c.Writer, "event: poll_invalidate\ndata: %s\n\n", initial)
	flusher.Flush()

	ka := time.NewTicker(20 * time.Second)
	defer ka.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-conn.ch:
			_, _ = fmt.Fprintf(c.Writer, "event: poll_invalidate\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-ka.C:
			_, _ = io.WriteString(c.Writer, ": ping\n\n")
			flusher.Flush()
		}
	}
}

