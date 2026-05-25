package main

import (
	"sort"
	"testing"
)

func TestHubUserListUpdatesOnJoinLeave(t *testing.T) {
	h := newHub()
	c1 := &client{name: "alice"}
	c2 := &client{name: "bob"}

	h.mu.Lock()
	h.clients[c1] = struct{}{}
	h.mu.Unlock()

	got := h.userList()
	sort.Strings(got)
	want := []string{"alice"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("after first join: got %v want %v", got, want)
	}

	h.mu.Lock()
	h.clients[c2] = struct{}{}
	h.mu.Unlock()

	got = h.userList()
	sort.Strings(got)
	want = []string{"alice", "bob"}
	if len(got) != len(want) {
		t.Fatalf("after second join: got %v want %v", got, want)
	}

	h.mu.Lock()
	delete(h.clients, c1)
	h.mu.Unlock()

	got = h.userList()
	want = []string{"bob"}
	if len(got) != 1 || got[0] != "bob" {
		t.Fatalf("after leave: got %v want %v", got, want)
	}
}
