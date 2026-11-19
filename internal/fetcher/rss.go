package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"finsec-term/internal/model"
)

type RSSFetcher struct {
	client  *http.Client
	parser  *gofeed.Parser
	tagger  *Tagger
	timeout time.Duration
}

func NewRSSFetcher(timeout time.Duration, tagger *Tagger) *RSSFetcher {
	httpClient := &http.Client{
		Timeout: timeout,
	}

	parser := gofeed.NewParser()
	parser.Client = httpClient

	return &RSSFetcher{
		client:  httpClient,
		parser:  parser,
		tagger:  tagger,
		timeout: timeout,
	}
}

// Fetch parses an RSS/Atom feed from the URL and maps entries to FeedItem slice
func (f *RSSFetcher) Fetch(ctx context.Context, sourceName, url string, pane model.PaneID, maxItems int) ([]model.FeedItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("rss new request for %s: %w", sourceName, err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 finsec-term/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9, */*;q=0.8")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rss fetch %s: %w", sourceName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rss status %d for %s", resp.StatusCode, sourceName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rss read %s: %w", sourceName, err)
	}

	feed, err := f.parser.ParseString(string(body))
	if err != nil {
		return nil, fmt.Errorf("rss parse %s: %w", sourceName, err)
	}

	var items []model.FeedItem
	feedItems := feed.Items
	if maxItems > 0 && len(feedItems) > maxItems {
		feedItems = feedItems[:maxItems]
	}

	for _, item := range feedItems {
		if item == nil {
			continue
		}

		cleanTitle := SanitizeHTML(item.Title)
		if cleanTitle == "" {
			continue
		}

		// Choose richest content: content > description
		descRaw := item.Description
		if item.Content != "" && len(item.Content) > len(descRaw) {
			descRaw = item.Content
		}
		cleanDesc := SanitizeHTML(descRaw)

		pubTime := time.Now()
		if item.PublishedParsed != nil {
			pubTime = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			pubTime = *item.UpdatedParsed
		}

		link := strings.TrimSpace(item.Link)
		id := model.GenerateID(link, cleanTitle, sourceName)

		feedItem := model.FeedItem{
			ID:          id,
			Title:       cleanTitle,
			Description: cleanDesc,
			URL:         link,
			Source:      sourceName,
			PublishedAt: pubTime,
			Pane:        pane,
			IsCISA:      false,
		}

		f.tagger.TagAndScore(&feedItem)
		items = append(items, feedItem)
	}

	return items, nil
}
