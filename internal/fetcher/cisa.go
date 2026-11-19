package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"finsec-term/internal/model"
)

type CISAVulnerability struct {
	CveID                      string   `json:"cveID"`
	VendorProject              string   `json:"vendorProject"`
	Product                    string   `json:"product"`
	VulnerabilityName          string   `json:"vulnerabilityName"`
	DateAdded                  string   `json:"dateAdded"`
	ShortDescription           string   `json:"shortDescription"`
	RequiredAction             string   `json:"requiredAction"`
	DueDate                    string   `json:"dueDate"`
	KnownRansomwareCampaignUse string   `json:"knownRansomwareCampaignUse"`
	Notes                      string   `json:"notes"`
	CWEs                       []string `json:"cwes"`
}

type CISACatalog struct {
	Title           string              `json:"title"`
	CatalogVersion  string              `json:"catalogVersion"`
	DateReleased    string              `json:"dateReleased"`
	Count           int                 `json:"count"`
	Vulnerabilities []CISAVulnerability `json:"vulnerabilities"`
}

type CISAFetcher struct {
	client  *http.Client
	tagger  *Tagger
	timeout time.Duration
}

func NewCISAFetcher(timeout time.Duration, tagger *Tagger) *CISAFetcher {
	return &CISAFetcher{
		client: &http.Client{
			Timeout: timeout,
		},
		tagger:  tagger,
		timeout: timeout,
	}
}

// Fetch loads CISA KEV JSON catalog, sorts by dateAdded descending, and returns FeedItems
func (f *CISAFetcher) Fetch(ctx context.Context, url string, maxItems int) ([]model.FeedItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("cisa new request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 finsec-term/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cisa fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cisa http error: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cisa read body: %w", err)
	}

	var catalog CISACatalog
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("cisa json parse: %w", err)
	}

	vulns := catalog.Vulnerabilities
	// Sort by DateAdded descending
	sort.Slice(vulns, func(i, j int) bool {
		return vulns[i].DateAdded > vulns[j].DateAdded
	})

	if maxItems > 0 && len(vulns) > maxItems {
		vulns = vulns[:maxItems]
	}

	var items []model.FeedItem
	for _, v := range vulns {
		pubTime, _ := time.Parse("2006-01-02", v.DateAdded)
		if pubTime.IsZero() {
			pubTime = time.Now()
		}

		descParts := []string{v.ShortDescription}
		if v.RequiredAction != "" {
			descParts = append(descParts, fmt.Sprintf("Action: %s", v.RequiredAction))
		}
		if v.DueDate != "" {
			descParts = append(descParts, fmt.Sprintf("Due Date: %s", v.DueDate))
		}
		if v.KnownRansomwareCampaignUse != "" {
			descParts = append(descParts, fmt.Sprintf("Ransomware Campaign: %s", v.KnownRansomwareCampaignUse))
		}

		itemURL := fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", v.CveID)

		item := model.FeedItem{
			ID:            model.GenerateID(itemURL, v.VulnerabilityName, "CISA KEV"),
			Title:         fmt.Sprintf("[%s] %s - %s", v.CveID, v.VendorProject, v.VulnerabilityName),
			Description:   strings.Join(descParts, "\n\n"),
			URL:           itemURL,
			Source:        "CISA KEV",
			PublishedAt:   pubTime,
			Pane:          model.PaneCVE,
			Severity:      model.SeverityCritical,
			IsCISA:        true,
			VendorProject: v.VendorProject,
			Product:       v.Product,
			CVEs:          []string{v.CveID},
			CWEs:          v.CWEs,
		}

		f.tagger.TagAndScore(&item)
		items = append(items, item)
	}

	return items, nil
}
