package handler

import (
	"sync"
)

type invalidateMsg struct {
	MyVotes     bool `json:"myVotes"`
	PublicStats bool `json:"publicStats"`
	MyCreated   bool `json:"myCreated"`
}

type sseHub struct {
	mu sync.RWMutex
	// userID -> connections
	conns map[string]map[*sseConn]struct{}
	// pollID -> connections
	pollConns map[string]map[*ssePollConn]struct{}
}

type sseConn struct {
	userID string
	ch     chan []byte
}

type ssePollConn struct {
	pollID string
	ch     chan []byte
}

func newSSEHub() *sseHub {
	return &sseHub{
		conns:     map[string]map[*sseConn]struct{}{},
		pollConns: map[string]map[*ssePollConn]struct{}{},
	}
}

// NewSSEHubForRouter exposes hub creation to router package
// without exporting internal types.
func NewSSEHubForRouter() *sseHub {
	return newSSEHub()
}

func (h *sseHub) add(userID string) *sseConn {
	c := &sseConn{
		userID: userID,
		ch:     make(chan []byte, 16),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = map[*sseConn]struct{}{}
	}
	h.conns[userID][c] = struct{}{}
	return c
}

func (h *sseHub) remove(c *sseConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.conns[c.userID]
	if m == nil {
		return
	}
	delete(m, c)
	if len(m) == 0 {
		delete(h.conns, c.userID)
	}
	close(c.ch)
}

func (h *sseHub) broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, m := range h.conns {
		for c := range m {
			select {
			case c.ch <- msg:
			default:
				// drop if client is slow
			}
		}
	}
}

func (h *sseHub) notifyUser(userID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := h.conns[userID]
	for c := range m {
		select {
		case c.ch <- msg:
		default:
		}
	}
}

func (h *sseHub) addPoll(pollID string) *ssePollConn {
	c := &ssePollConn{
		pollID: pollID,
		ch:     make(chan []byte, 16),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pollConns[pollID] == nil {
		h.pollConns[pollID] = map[*ssePollConn]struct{}{}
	}
	h.pollConns[pollID][c] = struct{}{}
	return c
}

func (h *sseHub) removePoll(c *ssePollConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.pollConns[c.pollID]
	if m == nil {
		return
	}
	delete(m, c)
	if len(m) == 0 {
		delete(h.pollConns, c.pollID)
	}
	close(c.ch)
}

func (h *sseHub) notifyPoll(pollID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := h.pollConns[pollID]
	for c := range m {
		select {
		case c.ch <- msg:
		default:
		}
	}
}

