package slist

import "time"

type Server struct {
	Addr string

	bad       bool
	fails     int
	lastUsage time.Time
}

func (s *Server) Bad() {
	s.bad = true
	s.fails++
}

func (s *Server) Good() {
	s.bad = false
}
