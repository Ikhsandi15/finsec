package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"finsec-term/internal/model"
)

var (
	nextDataRegex = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">([\s\S]*?)</script>`)
	scriptTagsRe  = regexp.MustCompile(`(?s)<script[^>]*>(.*?)</script>`)
	directURLRe   = regexp.MustCompile(`\\"url\\":\\"(https://t.co/[a-zA-Z0-9]+)\\".*?\\"expanded_url\\":\\"(.*?)\\"`)
	directTweetRe = regexp.MustCompile(`\\"created_at_ms\\":(\d+).*?\\"full_text\\":\\"(.*?)\\"(?:,\\"hashtag_entities\\"|.*?\\"rest_id\\":\\"(\d+)\\")`)
)

type TwitterFetcher struct {
	client  *http.Client
	tagger  *Tagger
	timeout time.Duration
}

func NewTwitterFetcher(timeout time.Duration, tagger *Tagger) *TwitterFetcher {
	return &TwitterFetcher{
		client: &http.Client{
			Timeout: timeout,
		},
		tagger:  tagger,
		timeout: timeout,
	}
}

// ExtractTwitterUsername parses username from https://x.com/username, twitter.com/username or @username
func ExtractTwitterUsername(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "www.")
	raw = strings.TrimPrefix(raw, "x.com/")
	raw = strings.TrimPrefix(raw, "twitter.com/")
	raw = strings.TrimPrefix(raw, "@")
	if idx := strings.Index(raw, "/"); idx != -1 {
		raw = raw[:idx]
	}
	if idx := strings.Index(raw, "?"); idx != -1 {
		raw = raw[:idx]
	}
	return strings.TrimSpace(raw)
}

type twitterNextData struct {
	Props struct {
		PageProps struct {
			Timeline struct {
				Entries []struct {
					Type    string `json:"type"`
					Content struct {
						Tweet struct {
							IDStr     string `json:"id_str"`
							CreatedAt string `json:"created_at"`
							FullText  string `json:"full_text"`
							Text      string `json:"text"`
							Permalink string `json:"permalink"`
							User      struct {
								Name       string `json:"name"`
								ScreenName string `json:"screen_name"`
							} `json:"user"`
							Entities struct {
								Urls []struct {
									DisplayUrl  string `json:"display_url"`
									ExpandedUrl string `json:"expanded_url"`
								} `json:"urls"`
							} `json:"entities"`
						} `json:"tweet"`
					} `json:"content"`
				} `json:"entries"`
			} `json:"timeline"`
		} `json:"pageProps"`
	} `json:"props"`
}

// Fetch attempts direct SSR scraping from x.com/{username} first, then falls back to syndication
func (f *TwitterFetcher) Fetch(ctx context.Context, sourceName, url string, pane model.PaneID, maxItems int) ([]model.FeedItem, error) {
	username := ExtractTwitterUsername(url)
	if username == "" {
		return nil, fmt.Errorf("empty twitter username in %s", url)
	}

	// 1. Try Direct SSR Extraction from x.com/{username}
	items, err := f.fetchDirectX(ctx, sourceName, username, pane, maxItems)
	if err == nil && len(items) > 0 {
		return items, nil
	}

	// 2. Fallback to Twitter Official Syndication Gateway
	return f.fetchSyndication(ctx, sourceName, username, pane, maxItems)
}

// fetchDirectX pulls live tweets rendered in Relay SSR hydration scripts on x.com/{username}
func (f *TwitterFetcher) fetchDirectX(ctx context.Context, sourceName, username string, pane model.PaneID, maxItems int) ([]model.FeedItem, error) {
	directURL := fmt.Sprintf("https://x.com/%s", username)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, directURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("x.com returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	htmlStr := string(body)

	// Harvest expanded URLs across the page
	urlMap := make(map[string]string)
	urlMatches := directURLRe.FindAllStringSubmatch(htmlStr, -1)
	for _, um := range urlMatches {
		shortURL := um[1]
		var expURL string
		_ = json.Unmarshal([]byte(`"`+um[2]+`"`), &expURL)
		if expURL != "" {
			urlMap[shortURL] = expURL
		}
	}

	scripts := scriptTagsRe.FindAllStringSubmatch(htmlStr, -1)
	var items []model.FeedItem
	seenIDs := make(map[string]bool)

	for _, s := range scripts {
		content := s[1]
		if !strings.Contains(content, `\"TimelineTimelineItem\"`) && !strings.Contains(content, `\"full_text\"`) {
			continue
		}

		matches := directTweetRe.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			ms, err := strconv.ParseInt(m[1], 10, 64)
			if err != nil || ms == 0 {
				continue
			}
			pubDate := time.UnixMilli(ms)

			var unescaped string
			_ = json.Unmarshal([]byte(`"`+m[2]+`"`), &unescaped)
			cleanText := SanitizeHTML(unescaped)
			if strings.TrimSpace(cleanText) == "" {
				continue
			}

			// Replace short links with expanded target URLs (e.g. cloud.antikorupsi.id/s/...)
			for shortU, longU := range urlMap {
				cleanText = strings.ReplaceAll(cleanText, shortU, longU)
			}

			tweetID := m[3]
			link := fmt.Sprintf("https://x.com/%s/status/%s", username, tweetID)
			if tweetID == "" {
				link = fmt.Sprintf("https://x.com/%s", username)
			}

			if seenIDs[link] {
				continue
			}
			seenIDs[link] = true

			title := cleanText
			if idx := strings.Index(title, "\n"); idx != -1 {
				title = title[:idx]
			}
			titleRunes := []rune(title)
			if len(titleRunes) > 120 {
				title = string(titleRunes[:117]) + "..."
			}

			feedItem := model.FeedItem{
				ID:          model.GenerateID(link, title, sourceName),
				Title:       title,
				URL:         link,
				Description: cleanText,
				PublishedAt: pubDate,
				Source:      sourceName,
				Pane:        pane,
				IsRead:      false,
			}

			if f.tagger != nil {
				f.tagger.TagAndScore(&feedItem)
			}

			items = append(items, feedItem)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})

	if maxItems > 0 && len(items) > maxItems {
		items = items[:maxItems]
	}

	return items, nil
}

// fetchSyndication pulls tweets from Twitter Syndication Gateway
func (f *TwitterFetcher) fetchSyndication(ctx context.Context, sourceName, username string, pane model.PaneID, maxItems int) ([]model.FeedItem, error) {
	apiURL := fmt.Sprintf("https://syndication.twitter.com/srv/timeline-profile/screen-name/%s", username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("twitter new req %s: %w", username, err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("twitter fetch %s: %w", username, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twitter status %d for %s", resp.StatusCode, username)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("twitter read %s: %w", username, err)
	}

	match := nextDataRegex.FindSubmatch(body)
	if len(match) < 2 {
		return nil, fmt.Errorf("twitter next_data not found for %s", username)
	}

	var data twitterNextData
	if err := json.Unmarshal(match[1], &data); err != nil {
		return nil, fmt.Errorf("twitter unmarshal %s: %w", username, err)
	}

	entries := data.Props.PageProps.Timeline.Entries
	var items []model.FeedItem

	for _, entry := range entries {
		tweet := entry.Content.Tweet
		if tweet.IDStr == "" && tweet.Permalink == "" {
			continue
		}

		rawText := tweet.FullText
		if rawText == "" {
			rawText = tweet.Text
		}
		cleanText := SanitizeHTML(rawText)
		if cleanText == "" {
			continue
		}

		// Replace short display links with expanded URLs if available
		for _, u := range tweet.Entities.Urls {
			if u.ExpandedUrl != "" && u.DisplayUrl != "" {
				cleanText = strings.ReplaceAll(cleanText, u.DisplayUrl, u.ExpandedUrl)
			}
		}

		title := cleanText
		if idx := strings.Index(title, "\n"); idx != -1 {
			title = title[:idx]
		}
		titleRunes := []rune(title)
		if len(titleRunes) > 120 {
			title = string(titleRunes[:117]) + "..."
		}

		link := "https://x.com" + tweet.Permalink
		if tweet.Permalink == "" {
			link = fmt.Sprintf("https://x.com/%s/status/%s", username, tweet.IDStr)
		}

		pubTime := time.Now()
		if tweet.CreatedAt != "" {
			if t, err := time.Parse("Mon Jan 02 15:04:05 -0700 2006", tweet.CreatedAt); err == nil {
				pubTime = t
			}
		}

		id := model.GenerateID(link, title, sourceName)

		feedItem := model.FeedItem{
			ID:          id,
			Title:       title,
			URL:         link,
			Description: cleanText,
			PublishedAt: pubTime,
			Source:      sourceName,
			Pane:        pane,
			IsRead:      false,
		}

		if f.tagger != nil {
			f.tagger.TagAndScore(&feedItem)
		}

		items = append(items, feedItem)
	}

	// Sort tweets strictly chronological descending (newest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})

	if maxItems > 0 && len(items) > maxItems {
		items = items[:maxItems]
	}

	return items, nil
}
