package model

import (
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	id1 := GenerateID("https://example.com/item1", "Title 1", "Source A")
	id2 := GenerateID("https://example.com/item1", "Different Title", "Source B")
	id3 := GenerateID("https://example.com/item2", "Title 1", "Source A")

	if id1 != id2 {
		t.Errorf("expected IDs with same URL to be identical, got %s and %s", id1, id2)
	}
	if id1 == id3 {
		t.Errorf("expected different URLs to generate different IDs, got %s", id1)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	item1 := FeedItem{PublishedAt: now.Add(-30 * time.Second)}
	if rel := item1.RelativeTime(); rel != "just now" {
		t.Errorf("expected 'just now', got '%s'", rel)
	}

	item2 := FeedItem{PublishedAt: now.Add(-15 * time.Minute)}
	if rel := item2.RelativeTime(); rel != "15m ago" {
		t.Errorf("expected '15m ago', got '%s'", rel)
	}

	item3 := FeedItem{PublishedAt: now.Add(-3 * time.Hour)}
	if rel := item3.RelativeTime(); rel != "3h ago" {
		t.Errorf("expected '3h ago', got '%s'", rel)
	}

	item4 := FeedItem{PublishedAt: now.Add(-2 * 24 * time.Hour)}
	if rel := item4.RelativeTime(); rel != "2d ago" {
		t.Errorf("expected '2d ago', got '%s'", rel)
	}
}

func TestMatchesFilter(t *testing.T) {
	item := FeedItem{
		Title:       "Critical Remote Code Execution in Nginx and PHP-FPM",
		Description: "Affects backend servers running on Linux and Windows",
		Source:      "BleepingComputer",
		CVEs:        []string{"CVE-2024-5555"},
		CWEs:        []string{"CWE-94"},
		Badges:      []string{BadgeStackCrit},
	}

	testCases := []struct {
		query    string
		expected bool
	}{
		{"", true},
		{"nginx", true},
		{"PHP", true},
		{"cve-2024-5555", true},
		{"cwe-94", true},
		{"bleeping", true},
		{"STACK-CRIT", true},
		{"golang", false},
		{"bitcoin", false},
	}

	for _, tc := range testCases {
		res := item.MatchesFilter(tc.query)
		if res != tc.expected {
			t.Errorf("query %q: expected %v, got %v", tc.query, tc.expected, res)
		}
	}
}
