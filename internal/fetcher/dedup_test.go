package fetcher

import (
	"testing"
	"time"

	"finsec-term/internal/model"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "https://www.cnbcindonesia.com/market/20261119-lps-bank?utm_source=twitter&utm_medium=social#read",
			expected: "https://cnbcindonesia.com/market/20261119-lps-bank",
		},
		{
			input:    "http://bbc.co.uk/news/technology-12345/?ref=rss&fbclid=abcdef",
			expected: "https://bbc.co.uk/news/technology-12345",
		},
	}

	for _, tc := range tests {
		got := NormalizeURL(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizeURL(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "LPS Ungkap Penyebab Bank Gagal, Masalah Utama Gara-Gara Ini! - CNBC Indonesia",
			expected: "lps ungkap penyebab bank gagal masalah utama gara gara ini",
		},
		{
			input:    "Fed Cuts Interest Rates by 25bps - BBC News",
			expected: "fed cuts interest rates by 25bps",
		},
	}

	for _, tc := range tests {
		got := NormalizeTitle(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizeTitle(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestDeduplicateItems(t *testing.T) {
	now := time.Now()
	items := []model.FeedItem{
		{
			ID:            "1",
			Title:         "LPS Ungkap Penyebab Bank Gagal, Masalah Utama Gara-Gara Ini!",
			URL:           "https://www.cnbcindonesia.com/market/bank-gagal-1?utm_source=rss",
			Source:        "CNBC ID Market",
			PublishedAt:   now,
			PriorityScore: 20,
			Badges:        []string{"ID-MACRO"},
		},
		{
			ID:            "2",
			Title:         "LPS Ungkap Penyebab Bank Gagal, Masalah Utama Gara-Gara Ini! - CNBC Indonesia",
			URL:           "https://www.cnbcindonesia.com/news/bank-gagal-1",
			Source:        "CNBC ID News",
			PublishedAt:   now.Add(10 * time.Minute),
			PriorityScore: 35,
			Badges:        []string{"ID-MACRO", "ID-POL"},
		},
		{
			ID:            "3",
			Title:         "Total Unrelated News About Apple Stock",
			URL:           "https://www.cnbc.com/apple-stock",
			Source:        "CNBC Markets",
			PublishedAt:   now,
			PriorityScore: 10,
		},
	}

	deduped := DeduplicateItems(items)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 items after dedup, got %d", len(deduped))
	}

	// First item should have merged score 35 and merged badges
	if deduped[0].PriorityScore != 35 {
		t.Errorf("expected merged priority score 35, got %d", deduped[0].PriorityScore)
	}
	if len(deduped[0].Badges) != 2 {
		t.Errorf("expected 2 merged badges, got %v", deduped[0].Badges)
	}
}
