// Package captcha 实现内置的一次性算术验证码（内存存储，带过期清理）。
package captcha

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type item struct {
	answer  string
	expires time.Time
}

type Store struct {
	mu    sync.Mutex
	items map[string]item
	ttl   time.Duration
}

func New(ttl time.Duration) *Store {
	s := &Store{items: make(map[string]item), ttl: ttl}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.cleanup()
		}
	}()
	return s
}

// New 生成一道算术题，返回验证码 ID 与题目文本（如 "7 + 5 = ?"）。
func (s *Store) New() (id, question string) {
	n1, n2 := randInt(9)+1, randInt(9)+1
	var answer int
	if randInt(2) == 0 {
		answer = n1 + n2
		question = fmt.Sprintf("%d + %d = ?", n1, n2)
	} else {
		if n1 < n2 {
			n1, n2 = n2, n1
		}
		answer = n1 - n2
		question = fmt.Sprintf("%d - %d = ?", n1, n2)
	}

	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	id = hex.EncodeToString(buf)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[id] = item{answer: fmt.Sprintf("%d", answer), expires: time.Now().Add(s.ttl)}
	return id, question
}

// Verify 校验并销毁验证码（一次性），任何不匹配都失败。
func (s *Store) Verify(id, answer string) bool {
	if id == "" || answer == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	if !ok {
		return false
	}
	delete(s.items, id)
	return time.Now().Before(it.expires) && it.answer == trim(answer)
}

func (s *Store) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, it := range s.items {
		if now.After(it.expires) {
			delete(s.items, id)
		}
	}
}

func randInt(n int) int {
	buf := make([]byte, 1)
	_, _ = rand.Read(buf)
	return int(buf[0]) % n
}

func trim(s string) string {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r != ' ' && r != '　' {
			out = append(out, byte(r))
		}
	}
	return string(out)
}
