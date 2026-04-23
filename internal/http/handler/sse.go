package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"vote-system/internal/http/middleware"
	"vote-system/internal/obs"

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
		obs.SSEErrors.WithLabelValues("setup", middleware.UserID(c)).Inc()
		return
	}

	userID := middleware.UserID(c)
	ctx := c.Request.Context()

	conn := h.hub.add(userID)
	if conn == nil {
		c.Status(429) // Too Many Requests
		obs.SSEErrors.WithLabelValues("connection_limit", userID).Inc()
		return
	}

	defer h.hub.remove(conn)

	// initial invalidate so client pulls once immediately
	initial, _ := json.Marshal(invalidateMsg{MyVotes: true, PublicStats: true, MyCreated: true})
	if _, err := fmt.Fprintf(c.Writer, "event: invalidate\ndata: %s\n\n", initial); err != nil {
		obs.SSEErrors.WithLabelValues("initial_send", userID).Inc()
		return
	}
	flusher.Flush()

	// keep-alive comments
	ka := time.NewTicker(20 * time.Second)
	defer ka.Stop()

	retryCount := 0
	maxRetries := 3

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-conn.ch:
			if _, err := fmt.Fprintf(c.Writer, "event: invalidate\ndata: %s\n\n", msg); err != nil {
				obs.SSEErrors.WithLabelValues("send", userID).Inc()
				retryCount++
				if retryCount >= maxRetries {
					return
				}
				time.Sleep(time.Second * time.Duration(retryCount))
				continue
			}
			flusher.Flush()
			retryCount = 0
		case <-ka.C:
			if _, err := io.WriteString(c.Writer, ": ping\n\n"); err != nil {
				obs.SSEErrors.WithLabelValues("ping", userID).Inc()
				return
			}
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
	userID := middleware.UserID(c)

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
	if _, err := fmt.Fprintf(c.Writer, "event: poll_invalidate\ndata: %s\n\n", initial); err != nil {
		obs.SSEErrors.WithLabelValues("poll_initial_send", userID).Inc()
		return
	}
	flusher.Flush()

	ka := time.NewTicker(20 * time.Second)
	defer ka.Stop()

	retryCount := 0
	maxRetries := 3

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-conn.ch:
			if _, err := fmt.Fprintf(c.Writer, "event: poll_invalidate\ndata: %s\n\n", msg); err != nil {
				obs.SSEErrors.WithLabelValues("poll_send", userID).Inc()
				retryCount++
				if retryCount >= maxRetries {
					return
				}
				time.Sleep(time.Second * time.Duration(retryCount))
				continue
			}
			flusher.Flush()
			retryCount = 0
		case <-ka.C:
			if _, err := io.WriteString(c.Writer, ": ping\n\n"); err != nil {
				obs.SSEErrors.WithLabelValues("poll_ping", userID).Inc()
				return
			}
			flusher.Flush()
		}
	}
}