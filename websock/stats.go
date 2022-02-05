package websock

import "time"

type Stats struct {
	Sent         int       `json:"sent"`
	LastSent     time.Time `json:"lastSent"`
	Received     int       `json:"received"`
	LastReceived time.Time `json:"lastReceived"`
}

func (s *Stats) IncrementSent() {
	s.LastSent = time.Now()
	s.Sent++
}

func (s *Stats) IncrementRecv() {
	s.LastReceived = time.Now()
	s.Received++
}
