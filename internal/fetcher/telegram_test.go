package fetcher

import (
	"testing"
	"time"

	"finsec-term/internal/config"
	"finsec-term/internal/model"
)

func TestNormalizeTelegramURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://t.me/zachxbt", "https://t.me/s/zachxbt"},
		{"https://t.me/s/zachxbt", "https://t.me/s/zachxbt"},
		{"zachxbt", "https://t.me/s/zachxbt"},
		{"@zachxbt", "https://t.me/s/zachxbt"},
	}

	for _, tt := range tests {
		got := NormalizeTelegramURL(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeTelegramURL(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestTelegramHTMLParsing(t *testing.T) {
	sampleHTML := `
	<div class="tgme_widget_message_wrap js-widget_message_wrap">
		<div class="tgme_widget_message js-widget_message" data-post="zachxbt/100">
			<div class="tgme_widget_message_bubble">
				<div class="tgme_widget_message_text js-message_text" dir="auto">Critical exploit on TeleSwap bridge: &#036;735K drained to Tornado Cash.<br/>See tx details.</div>
				<time datetime="2024-07-20T16:04:12+00:00" class="time">16:04</time>
			</div>
		</div>
	</div>
	`

	tagger := NewTagger(&config.DefaultConfig().Rules)
	f := NewTelegramFetcher(5*time.Second, tagger)

	// Verify we can parse post, text and time
	chunks := []string{"", sampleHTML}
	if len(chunks) <= 1 {
		t.Fatal("expected chunks")
	}

	chunk := chunks[1]
	postMatch := tgPostRegex.FindStringSubmatch(chunk)
	if len(postMatch) < 2 || postMatch[1] != "zachxbt/100" {
		t.Errorf("postMatch failed: %v", postMatch)
	}

	textMatch := tgTextRegex.FindStringSubmatch(chunk)
	if len(textMatch) < 2 {
		t.Fatal("textMatch failed")
	}

	clean := SanitizeHTML(textMatch[1])
	if clean != "Critical exploit on TeleSwap bridge: $735K drained to Tornado Cash.\nSee tx details." {
		t.Errorf("unexpected clean text: %q", clean)
	}

	item := model.FeedItem{
		Title:       clean,
		Description: clean,
	}
	f.tagger.TagAndScore(&item)

	// TeleSwap / exploit text has base score
	if item.PriorityScore <= 0 {
		t.Errorf("expected positive priority score, got %d", item.PriorityScore)
	}
}
