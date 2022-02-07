package websock

import (
	"encoding/json"
	"sync"
	"time"
)

type Stats struct {
	sent         int
	lastSent     time.Time
	received     int
	lastReceived time.Time
	lock         sync.RWMutex
}

func (s *Stats) MarshalJSON() ([]byte, error) {
	s.lock.RLock()
	s.lock.RUnlock()
	return json.Marshal(map[string]interface{}{
		"sent":         s.sent,
		"received":     s.received,
		"lastSent":     s.lastSent.Format("15:04:05 MST"),
		"lastReceived": s.lastReceived.Format("15:04:05 MST"),
	})
}

func (s *Stats) IncrementSent() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.lastSent = time.Now()
	s.sent++
}

func (s *Stats) IncrementRecv() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.lastReceived = time.Now()
	s.received++
}
