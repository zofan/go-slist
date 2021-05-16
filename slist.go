package slist

import (
	"bufio"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	ModeRandom SelectMode = 1
	ModeRotate SelectMode = 2
	ModeTime   SelectMode = 3
)

var (
	ErrServerListEmpty = errors.New(`slist: server list is empty`)
	ErrBadMode         = errors.New(`slist: bad mode`)
)

type SelectMode int

type List struct {
	good []*Server
	bad  []*Server
	uniq map[string]struct{}

	maxKarma int
	mode     SelectMode
	n        int

	BanFunc func(s *Server) time.Time

	mu   sync.Mutex
	rand *rand.Rand
}

func New(mode SelectMode, maxKarma int) *List {
	l := &List{
		mode:     mode,
		maxKarma: maxKarma,
		uniq:     make(map[string]struct{}),
		BanFunc:  DefaultBan,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	go func() {
		for range time.Tick(time.Minute) {
			l.Restore()
		}
	}()

	return l
}

func (l *List) LoadFromString(servers string) error {
	return l.LoadFromReader(strings.NewReader(servers))
}

func (l *List) LoadFromURL(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return l.LoadFromReader(resp.Body)
}

func (l *List) LoadFromFile(file string) error {
	fh, err := os.OpenFile(file, os.O_RDONLY, 0664)
	if err != nil {
		return err
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		l.Add(scanner.Text())
	}

	return scanner.Err()
}

func (l *List) LoadFromReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		l.Add(scanner.Text())
	}

	return scanner.Err()
}

func (l *List) Add(addr string) {
	line := strings.TrimSpace(addr)

	if len(addr) == 0 || line[0] == '#' {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.uniq[addr]; ok {
		return
	}

	l.good = append(l.good, &Server{Addr: addr})
	l.uniq[addr] = struct{}{}
}

func (l *List) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.good)
}

func (l *List) All() []*Server {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.good
}

func (l *List) Shuffle() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.rand.Shuffle(len(l.good), func(i, j int) {
		l.good[i], l.good[j] = l.good[j], l.good[i]
	})
}

func (l *List) Get() (s *Server, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.good) == 0 {
		return nil, ErrServerListEmpty
	}

	switch l.mode {
	case ModeTime:
		s = l.good[time.Now().Unix()%int64(len(l.good))]
	case ModeRandom:
		s = l.good[l.rand.Intn(len(l.good))]
	case ModeRotate:
		if l.n == len(l.good) {
			l.n = 0
		}
		s = l.good[l.n]
		l.n++
	default:
		return nil, ErrBadMode
	}

	s.LastUsage = time.Now()

	return
}

func (l *List) Restore() {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	for i := 0; i < len(l.bad); i++ {
		s := l.bad[i]
		if now.After(s.BanExpires) {
			l.bad = append(l.bad[:i], l.bad[i+1:]...)
			l.good = append(l.good, s)
			l.markGood(s)
			i--
		}
	}
}

func (l *List) markGood(s *Server) {
	s.Karma = 0
	s.GoodCnt++
	s.BanExpires = time.Time{}
}

func (l *List) MarkGood(s *Server) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.markGood(s)
}

func (l *List) MarkBad(s *Server) {
	l.mu.Lock()
	defer l.mu.Unlock()

	s.BadCnt++
	s.Karma++

	if s.Karma < l.maxKarma {
		return
	}

	if l.BanFunc == nil {
		s.BanExpires = DefaultBan(s)
	} else {
		s.BanExpires = l.BanFunc(s)
	}

	for i := 0; i < len(l.good); i++ {
		if l.good[i] == s {
			l.good = append(l.good[:i], l.good[i+1:]...)
			l.bad = append(l.bad, l.good[i])
			l.n = 0
			break
		}
	}
}

func DefaultBan(s *Server) time.Time {
	return time.Now().Add(time.Minute)
}
