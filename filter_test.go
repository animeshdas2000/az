package main

import (
	"testing"
)

func TestFilterEmpty(t *testing.T) {
	sessions := []session{
		{Path: "/Users/test/dokkai"},
		{Path: "/Users/test/scwroll"},
	}
	got := filterSessions("", sessions)
	if len(got) != len(sessions) {
		t.Errorf("empty query: want %d, got %d", len(sessions), len(got))
	}
}

func TestFilterMatchesBasename(t *testing.T) {
	sessions := []session{
		{Path: "/Users/test/dokkai"},
		{Path: "/Users/test/scwroll"},
		{Path: "/Users/test/boardroom"},
	}

	got := filterSessions("dok", sessions)
	if len(got) != 1 {
		t.Fatalf("want 1 match, got %d", len(got))
	}
	if got[0].Path != "/Users/test/dokkai" {
		t.Errorf("unexpected match: %q", got[0].Path)
	}
}

func TestFilterMatchesParentDir(t *testing.T) {
	sessions := []session{
		{Path: "/Users/test/realfast/boardroom"},
		{Path: "/Users/test/animesh/code"},
	}

	got := filterSessions("realfast", sessions)
	if len(got) != 1 || got[0].Path != "/Users/test/realfast/boardroom" {
		t.Errorf("expected realfast match, got %v", got)
	}
}

func TestFilterCaseInsensitive(t *testing.T) {
	sessions := []session{
		{Path: "/Users/test/MyProject"},
	}
	got := filterSessions("myproject", sessions)
	if len(got) != 1 {
		t.Errorf("want case-insensitive match, got %d", len(got))
	}
}

func TestFilterNoMatch(t *testing.T) {
	sessions := []session{
		{Path: "/Users/test/dokkai"},
	}
	got := filterSessions("zzznomatch", sessions)
	if len(got) != 0 {
		t.Errorf("want 0 matches, got %d", len(got))
	}
}

func TestDedup(t *testing.T) {
	sessions := []session{
		{Path: "/a/b", Mtime: 100},
		{Path: "/a/b", Mtime: 200}, // duplicate, more recent
		{Path: "/c/d", Mtime: 50},
	}
	got := dedup(sessions)
	if len(got) != 2 {
		t.Fatalf("want 2 after dedup, got %d", len(got))
	}
	// /a/b should have the more recent mtime
	for _, s := range got {
		if s.Path == "/a/b" && s.Mtime != 200 {
			t.Errorf("/a/b: want mtime 200, got %d", s.Mtime)
		}
	}
}
