package main

import (
	"os"
	"strings"
)

// filterSessions returns sessions where any path component contains query
// (case-insensitive). Returns all sessions when query is empty.
func filterSessions(query string, sessions []session) []session {
	if query == "" {
		return sessions
	}
	q := strings.ToLower(query)
	var out []session
	for _, s := range sessions {
		parts := strings.Split(s.Path, string(os.PathSeparator))
		for _, part := range parts {
			if strings.Contains(strings.ToLower(part), q) {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// dedup removes duplicate paths, keeping the entry with the highest Mtime.
func dedup(sessions []session) []session {
	best := map[string]session{}
	for _, s := range sessions {
		if existing, ok := best[s.Path]; !ok || s.Mtime > existing.Mtime {
			best[s.Path] = s
		}
	}
	out := make([]session, 0, len(best))
	for _, s := range best {
		out = append(out, s)
	}
	return out
}
