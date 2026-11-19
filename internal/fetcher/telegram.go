package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"finsec-term/internal/model"
)

var (
	tgPostRegex = regexp.MustCompile(`data-post="([^"]+)"`)
	tgTextRegex = regexp.MustCompile(`(?s)<div class="tgme_widget_message_text[^"]*"[^>]*>([\s\S]*?)</div>`)
	tgTimeRegex = regexp.MustCompile(`<time[^>]*datetime="([^"]+)"`)
)

type TelegramFetcher struct {
	client  *http.Client
	tagger  *Tagger
	timeout time.Duration
}

func NewTelegramFetcher(timeout time.Duration, tagger *Tagger) *TelegramFetcher {
	return &TelegramFetcher{
		client: &http.Client{
			Timeout: timeout,
		},
		tagger:  tagger,
		timeout: timeout,
	}
}

// NormalizeTelegramURL converts https://t.me/username to https://t.me/s/username
func NormalizeTelegramURL(url string) string {
	url = strings.TrimSpace(url)
	if strings.Contains(url, "t.me/s/") {
		return url
	}
	if strings.Contains(url, "t.me/") {
		return strings.Replace(url, "t.me/", "t.me/s/", 1)
	}
	// If only username provided (e.g. "zachxbt")
	if !strings.HasPrefix(url, "http") {
		return "https://t.me/s/" + strings.TrimPrefix(url, "@")
	}
	return url
}

// Fetch pulls messages from a public Telegram channel web view
func (f *TelegramFetcher) Fetch(ctx context.Context, sourceName, url string, pane model.PaneID, maxItems int) ([]model.FeedItem, error) {
	fetchURL := NormalizeTelegramURL(url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("tg new request for %s: %w", sourceName, err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tg fetch %s: %w", sourceName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tg status %d for %s", resp.StatusCode, sourceName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tg read body %s: %w", sourceName, err)
	}

	htmlContent := string(body)
	chunks := strings.Split(htmlContent, "tgme_widget_message_wrap")
	if len(chunks) <= 1 {
		return nil, nil
	}

	var items []model.FeedItem

	// Telegram messages are ordered chronological (oldest to newest).
	// Iterate backwards so newest messages are parsed first.
	for i := len(chunks) - 1; i >= 1; i-- {
		chunk := chunks[i]

		postMatch := tgPostRegex.FindStringSubmatch(chunk)
		if len(postMatch) < 2 {
			continue
		}
		postID := postMatch[1]
		link := "https://t.me/" + postID

		textMatch := tgTextRegex.FindStringSubmatch(chunk)
		if len(textMatch) < 2 {
			continue // Media-only post without caption
		}

		cleanText := SanitizeHTML(textMatch[1])
		if strings.TrimSpace(cleanText) == "" {
			continue
		}

		// Title: First line or first 120 chars
		title := cleanText
		if idx := strings.Index(title, "\n"); idx != -1 {
			title = title[:idx]
		}
		titleRunes := []rune(title)
		if len(titleRunes) > 120 {
			title = string(titleRunes[:117]) + "..."
		}

		// Published time
		pubTime := time.Now()
		timeMatch := tgTimeRegex.FindStringSubmatch(chunk)
		if len(timeMatch) >= 2 {
			if t, err := time.Parse(time.RFC3339, timeMatch[1]); err == nil {
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
		if maxItems > 0 && len(items) >= maxItems {
			break
		}
	}

	return items, nil
}
