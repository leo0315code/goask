// Package hub 进程内的实时消息订阅/发布中心，供 SSE 推送使用。
// 单实例部署直接可用；多实例部署需要把 Publish 改为走 Redis pub/sub（见 docs/design.md）。
package hub

import "sync"

type Hub struct {
	mu   sync.RWMutex
	subs map[uint64]map[chan string]struct{}
}

func New() *Hub {
	return &Hub{subs: make(map[uint64]map[chan string]struct{})}
}

// Subscribe 订阅某用户的消息流，返回只读通道与取消函数。
func (h *Hub) Subscribe(userID uint64) (<-chan string, func()) {
	ch := make(chan string, 8)
	h.mu.Lock()
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[chan string]struct{})
	}
	h.subs[userID][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if set, ok := h.subs[userID]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(h.subs, userID)
			}
		}
		h.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

// Publish 向某用户的所有连接推送消息；订阅者积压时丢弃（客户端有兜底轮询）。
func (h *Hub) Publish(userID uint64, msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[userID] {
		select {
		case ch <- msg:
		default:
		}
	}
}
