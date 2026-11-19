package fetcher

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"finsec-term/internal/config"
	"finsec-term/internal/model"
	"finsec-term/internal/storage"
)

type FetchResult struct {
	Pane1Items []model.FeedItem
	Pane2Items []model.FeedItem
	Pane3Items []model.FeedItem
	Errors     []string
	FetchedAt  time.Time
}

type Manager struct {
	cfg         *config.Config
	tagger      *Tagger
	cisaFetcher *CISAFetcher
	rssFetcher  *RSSFetcher
	cache       *storage.Cache
}

func NewManager(cfg *config.Config, cache *storage.Cache) *Manager {
	tagger := NewTagger(&cfg.Rules)
	return &Manager{
		cfg:         cfg,
		tagger:      tagger,
		cisaFetcher: NewCISAFetcher(cfg.RequestTimeout, tagger),
		rssFetcher:  NewRSSFetcher(cfg.RequestTimeout, tagger),
		cache:       cache,
	}
}

// FetchAll executes concurrent requests to all feeds across all panes
func (m *Manager) FetchAll(parentCtx context.Context) (*FetchResult, error) {
	ctx, cancel := context.WithTimeout(parentCtx, m.cfg.RequestTimeout+5*time.Second)
	defer cancel()

	var (
		mu         sync.Mutex
		pane1Items []model.FeedItem
		pane2Items []model.FeedItem
		pane3Items []model.FeedItem
		fetchErrs  []string
		seenIDs    = make(map[string]bool)
	)

	g, gCtx := errgroup.WithContext(ctx)

	// Fetch Pane 1 Feeds
	for _, feed := range m.cfg.Feeds.Pane1CVE {
		f := feed
		g.Go(func() error {
			var (
				items []model.FeedItem
				err   error
			)
			if f.Type == config.FeedTypeCISA {
				items, err = m.cisaFetcher.Fetch(gCtx, f.URL, m.cfg.MaxItemsPerFeed)
			} else {
				items, err = m.rssFetcher.Fetch(gCtx, f.Name, f.URL, model.PaneCVE, m.cfg.MaxItemsPerFeed)
			}

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fetchErrs = append(fetchErrs, fmt.Sprintf("%s: %v", f.Name, err))
				return nil // Don't abort other feeds
			}

			for _, it := range items {
				if !seenIDs[it.ID] {
					seenIDs[it.ID] = true
					if m.cache != nil {
						m.cache.RecordSeen(it.ID)
						it.IsRead = m.cache.IsRead(it.ID)
					}
					pane1Items = append(pane1Items, it)
				}
			}
			return nil
		})
	}

	// Fetch Pane 2 Feeds
	for _, feed := range m.cfg.Feeds.Pane2GlobalMacro {
		f := feed
		g.Go(func() error {
			items, err := m.rssFetcher.Fetch(gCtx, f.Name, f.URL, model.PaneGlobalMacro, m.cfg.MaxItemsPerFeed)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fetchErrs = append(fetchErrs, fmt.Sprintf("%s: %v", f.Name, err))
				return nil
			}

			for _, it := range items {
				if !seenIDs[it.ID] {
					seenIDs[it.ID] = true
					if m.cache != nil {
						m.cache.RecordSeen(it.ID)
						it.IsRead = m.cache.IsRead(it.ID)
					}
					pane2Items = append(pane2Items, it)
				}
			}
			return nil
		})
	}

	// Fetch Pane 3 Feeds
	for _, feed := range m.cfg.Feeds.Pane3IDMacro {
		f := feed
		g.Go(func() error {
			items, err := m.rssFetcher.Fetch(gCtx, f.Name, f.URL, model.PaneIDMacro, m.cfg.MaxItemsPerFeed)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fetchErrs = append(fetchErrs, fmt.Sprintf("%s: %v", f.Name, err))
				return nil
			}

			for _, it := range items {
				if !seenIDs[it.ID] {
					seenIDs[it.ID] = true
					if m.cache != nil {
						m.cache.RecordSeen(it.ID)
						it.IsRead = m.cache.IsRead(it.ID)
					}
					pane3Items = append(pane3Items, it)
				}
			}
			return nil
		})
	}

	_ = g.Wait()

	// Sort each pane items by PriorityScore desc, then PublishedAt desc
	sortItems(pane1Items)
	sortItems(pane2Items)
	sortItems(pane3Items)

	if m.cache != nil {
		_ = m.cache.Save()
	}

	return &FetchResult{
		Pane1Items: pane1Items,
		Pane2Items: pane2Items,
		Pane3Items: pane3Items,
		Errors:     fetchErrs,
		FetchedAt:  time.Now(),
	}, nil
}

func sortItems(items []model.FeedItem) {
	sort.Slice(items, func(i, j int) bool {
		if !items[i].PublishedAt.Equal(items[j].PublishedAt) {
			return items[i].PublishedAt.After(items[j].PublishedAt)
		}
		return items[i].PriorityScore > items[j].PriorityScore
	})
}
