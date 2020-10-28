package slist

import "time"

type Server struct {
	Addr string

	Karma   int
	GoodCnt int
	BadCnt  int

	LastUsage  time.Time
	BanExpires time.Time
}
