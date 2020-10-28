package slist

import (
	"testing"
)

func TestGetServerRandom(t *testing.T) {
	r := New(ModeRandom, 10)

	err := r.LoadFromString("8.8.8.8\n#hello\n1.1.1.1\n8.8.8.4")
	if err != nil {
		t.Error(err)
	}

	srv1, err := r.Get()
	if err != nil {
		t.Error(err)
	}
	srv2, err := r.Get()
	if err != nil {
		t.Error(err)
	}
	if srv1 == srv2 {
		t.Error(`two servers are the same, expecting different`)
	}
}

func TestGetServerRoundRobin(t *testing.T) {
	r := New(ModeRotate, 10)

	err := r.LoadFromString("8.8.8.8\n1.1.1.1\n8.8.8.4")
	if err != nil {
		t.Error(err)
	}

	servers := r.All()

	srv1, err := r.Get()
	if err != nil {
		t.Error(err)
	}
	if srv1 != servers[0] {
		t.Error(`the first server expected ` + servers[0].Addr)
	}

	srv2, err := r.Get()
	if err != nil {
		t.Error(err)
	}
	if srv2 != servers[1] {
		t.Error(`the second server expected ` + servers[1].Addr)
	}

	srv3, err := r.Get()
	if err != nil {
		t.Error(err)
	}
	if srv3 != servers[2] {
		t.Error(`the third server expected ` + servers[2].Addr)
	}

	srv4, err := r.Get()
	if err != nil {
		t.Error(err)
	}
	if srv4 != servers[0] {
		t.Error(`the fourth server expected ` + servers[0].Addr)
	}
}

func TestBadServerList(t *testing.T) {
	r := New(ModeRotate, 10)

	err := r.LoadFromString("\n\n\r\n\t\n")
	if err != nil {
		t.Error(err)
	}

	if r.Count() != 0 {
		t.Error(`expected empty server list`)
	}
}
