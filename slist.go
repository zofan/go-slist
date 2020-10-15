package slist

import (
	"bufio"
	"errors"
	"io"
	"math/rand"
	"net/http"
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

	restoreTime time.Duration
	maxFails    int
	mode        SelectMode
	n           int

	mu sync.Mutex
}

func New(mode SelectMode, maxFails int, restoreTime time.Duration) *List {
	l := &List{
		mode:        mode,
		maxFails:    maxFails,
		restoreTime: restoreTime,
		uniq:        make(map[string]struct{}),
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

func (l *List) LoadFromReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		l.Add(strings.TrimSpace(scanner.Text()))
	}

	return scanner.Err()
}

// 1.2.3.4, 1.2.3.4:8080, example.com, example.com:8000
func (l *List) Add(addr string) {
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

func (l *List) GoodList() []*Server {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.good
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
		s = l.good[rand.Intn(len(l.good))]
	case ModeRotate:
		addr := l.good[l.n]

		l.n++
		if int(l.n) > len(l.good)-1 {
			l.n = 0
		}

		s = addr
	default:
		return nil, ErrBadMode
	}

	s.lastUsage = time.Now()

	return
}

func (l *List) Restore() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i := 0; i < len(l.bad); i++ {
		s := l.bad[i]
		if time.Since(s.lastUsage) > l.restoreTime {
			l.bad = append(l.bad[:i], l.bad[i+1:]...)
			i--

			l.good = append(l.good, s)

			s.Good()
		}
	}
}

func (l *List) MarkBad(server *Server) {
	server.Bad()

	l.mu.Lock()
	defer l.mu.Unlock()

	if server.fails < l.maxFails {
		return
	}

	for i := 0; i < len(l.good); i++ {
		s := l.good[i]
		if s == server {
			l.good = append(l.good[:i], l.good[i+1:]...)
			i--

			l.bad = append(l.bad, s)

			l.n = 0
			break
		}
	}
}
