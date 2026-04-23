package handler

import (
	"sync"
	"time"
	"vote-system/internal/obs"
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

	// 连接限制配置
	maxConnectionsPerUser int
	connectionTTL         time.Duration
	cleanupInterval       time.Duration

	// 清理定时器
	cleanupTicker *time.Ticker
}

type sseConn struct {
	userID     string
	ch         chan []byte
	createdAt  time.Time
	lastActive time.Time
}

type ssePollConn struct {
	pollID     string
	ch         chan []byte
	createdAt  time.Time
	lastActive time.Time
}

func newSSEHub() *sseHub {
	return &sseHub{
		conns:                 make(map[string]map[*sseConn]struct{}),
		pollConns:             make(map[string]map[*ssePollConn]struct{}),
		maxConnectionsPerUser: 5,                 // 每用户最多5个连接
		connectionTTL:         30 * time.Minute, // 连接TTL 30分钟
		cleanupInterval:       5 * time.Minute,  // 清理间隔5分钟
		cleanupTicker:         time.NewTicker(5 * time.Minute),
	}
}

// NewSSEHubForRouter exposes hub creation to router package
// without exporting internal types.
func NewSSEHubForRouter() *sseHub {
	hub := newSSEHub()
	go hub.cleanupConnections() // 启动清理goroutine
	return hub
}

func (h *sseHub) cleanupConnections() {
	for range h.cleanupTicker.C {
		h.cleanupExpiredConnections()
	}
}

func (h *sseHub) cleanupExpiredConnections() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	// 清理用户连接
	for userID, connections := range h.conns {
		for conn := range connections {
			if now.Sub(conn.lastActive) > h.connectionTTL {
				delete(connections, conn)
				close(conn.ch)
				obs.RecordConnectionChange(conn.userID, -1)
			}
		}
		if len(connections) == 0 {
			delete(h.conns, userID)
		}
	}

	// 清理Poll连接
	for pollID, connections := range h.pollConns {
		for conn := range connections {
			if now.Sub(conn.lastActive) > h.connectionTTL {
				delete(connections, conn)
				close(conn.ch)
			}
		}
		if len(connections) == 0 {
			delete(h.pollConns, pollID)
		}
	}
}

func (h *sseHub) add(userID string) *sseConn {
	// 检查连接限制
	h.mu.RLock()
	userConnections := h.conns[userID]
	connectionCount := 0
	if userConnections != nil {
		connectionCount = len(userConnections)
	}
	h.mu.RUnlock()

	if connectionCount >= h.maxConnectionsPerUser {
		obs.RecordMessageDropped(userID)
		return nil
	}

	c := &sseConn{
		userID:     userID,
		ch:         make(chan []byte, 1024), // 缓冲区从16字节增加到1024字节
		createdAt:  time.Now(),
		lastActive: time.Now(),
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[userID] == nil {
		h.conns[userID] = make(map[*sseConn]struct{})
	}
	h.conns[userID][c] = struct{}{}
	obs.RecordConnectionChange(userID, 1)
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
	obs.RecordConnectionChange(c.userID, -1)
}

func (h *sseHub) broadcast(msg []byte) {
	start := time.Now()
	success := true

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, m := range h.conns {
		for c := range m {
			select {
			case c.ch <- msg:
				c.lastActive = time.Now()
				obs.RecordBufferUsage(c.userID, 1024, len(c.ch))
			default:
				obs.RecordMessageDropped(c.userID)
				success = false
			}
		}
	}

	obs.TrackSSEOp("", "broadcast", success, time.Since(start).Seconds())
}

func (h *sseHub) notifyUser(userID string, msg []byte) {
	start := time.Now()
	success := true

	h.mu.RLock()
	defer h.mu.RUnlock()
	m := h.conns[userID]
	for c := range m {
		select {
		case c.ch <- msg:
			c.lastActive = time.Now()
			obs.RecordBufferUsage(c.userID, 1024, len(c.ch))
		default:
			obs.RecordMessageDropped(userID)
			success = false
		}
	}

	obs.TrackSSEOp(userID, "notify_user", success, time.Since(start).Seconds())
}

func (h *sseHub) addPoll(pollID string) *ssePollConn {
	c := &ssePollConn{
		pollID:     pollID,
		ch:         make(chan []byte, 1024), // 缓冲区从16字节增加到1024字节
		createdAt:  time.Now(),
		lastActive: time.Now(),
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pollConns[pollID] == nil {
		h.pollConns[pollID] = make(map[*ssePollConn]struct{})
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
	start := time.Now()
	success := true

	h.mu.RLock()
	defer h.mu.RUnlock()
	m := h.pollConns[pollID]
	for c := range m {
		select {
		case c.ch <- msg:
			c.lastActive = time.Now()
			obs.RecordBufferUsage("poll_"+pollID, 1024, len(c.ch))
		default:
			success = false
		}
	}

	obs.TrackSSEOp("poll_"+pollID, "notify_poll", success, time.Since(start).Seconds())
}