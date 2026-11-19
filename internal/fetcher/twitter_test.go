package fetcher

import (
	"encoding/json"
	"testing"
	"time"

	"finsec-term/internal/config"
	"finsec-term/internal/model"
)

func TestExtractTwitterUsername(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://x.com/antikorupsi", "antikorupsi"},
		{"https://twitter.com/antikorupsi?lang=en", "antikorupsi"},
		{"@antikorupsi", "antikorupsi"},
		{"antikorupsi", "antikorupsi"},
		{"https://x.com/cvedotnow/", "cvedotnow"},
		{"https://x.com/DarkWebInformer", "DarkWebInformer"},
	}

	for _, tt := range tests {
		got := ExtractTwitterUsername(tt.input)
		if got != tt.expected {
			t.Errorf("ExtractTwitterUsername(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestTwitterJSONParsing(t *testing.T) {
	sampleJSON := `{
		"props": {
			"pageProps": {
				"timeline": {
					"entries": [
						{
							"type": "tweet",
							"content": {
								"tweet": {
									"id_str": "1856582242639683697",
									"created_at": "Sat Feb 22 02:33:53 +0000 2025",
									"full_text": "Catatan Kritis ICW atas RUU Perampasan Aset: unduh dokumen di https://cloud.antikorupsi.id/s/NeBWJyCbkRBi2qP",
									"permalink": "/antikorupsi/status/1856582242639683697",
									"user": {
										"name": "Indonesia Corruption Watch",
										"screen_name": "antikorupsi"
									},
									"entities": {
										"urls": []
									}
								}
							}
						}
					]
				}
			}
		}
	}`

	var data twitterNextData
	if err := json.Unmarshal([]byte(sampleJSON), &data); err != nil {
		t.Fatalf("failed to unmarshal twitter json: %v", err)
	}

	if len(data.Props.PageProps.Timeline.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(data.Props.PageProps.Timeline.Entries))
	}

	tweet := data.Props.PageProps.Timeline.Entries[0].Content.Tweet
	if tweet.User.ScreenName != "antikorupsi" {
		t.Errorf("unexpected screen name: %s", tweet.User.ScreenName)
	}

	tagger := NewTagger(&config.DefaultConfig().Rules)
	f := NewTwitterFetcher(5*time.Second, tagger)

	item := model.FeedItem{
		Source:      "X: @antikorupsi",
		Title:       tweet.FullText,
		Description: tweet.FullText,
	}
	f.tagger.TagAndScore(&item)

	hasICW := false
	for _, b := range item.Badges {
		if b == model.BadgeICW {
			hasICW = true
		}
	}
	if !hasICW {
		t.Errorf("expected %s badge, got %v", model.BadgeICW, item.Badges)
	}
}
