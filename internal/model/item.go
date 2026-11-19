package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type PaneID int

const (
	PaneCVE PaneID = iota + 1
	PaneGlobalMacro
	PaneIDMacro
	PaneInspector
)

func (p PaneID) String() string {
	switch p {
	case PaneCVE:
		return "TECH, AI & CYBER RADAR"
	case PaneGlobalMacro:
		return "GLOBAL MACRO & GEOPOLITICS"
	case PaneIDMacro:
		return "INDONESIA MARKET, POLITICS & CYBER"
	case PaneInspector:
		return "INSPECTOR / DETAIL PANE"
	default:
		return "UNKNOWN"
	}
}

type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MED"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRIT"
)

const (
	BadgeStackCrit = "STACK-CRIT"
	BadgeIDMacro   = "ID-MACRO"
	BadgeIDPol     = "ID-POL"
	BadgeGlobalMkt = "GLOBAL-MKT"
	BadgeCVE       = "CVE"
	BadgeTechAI    = "TECH-AI"

	// Social & Authority Intelligence Badges
	BadgeICW       = "X: @antikorupsi"
	BadgeZachXBT   = "X: @zachxbt"
	BadgeCVEDotNow = "X: @cvedotnow"
	BadgeCyberNews = "X: @The_Cyber_News"
	BadgeIntCyber  = "X: @IntCyberDigest"
	BadgeGithubPrj = "X: @GithubProjects"
	BadgeDarkWeb   = "X: @DarkWebInformer"
)

type FeedItem struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	URL           string    `json:"url"`
	Source        string    `json:"source"`
	PublishedAt   time.Time `json:"published_at"`
	Pane          PaneID    `json:"pane"`
	Severity      Severity  `json:"severity"`
	PriorityScore int       `json:"priority_score"`
	CVEs          []string  `json:"cves"`
	CWEs          []string  `json:"cwes"`
	Badges        []string  `json:"badges"`
	IsRead        bool      `json:"is_read"`
	IsCISA        bool      `json:"is_cisa"`
	VendorProject string    `json:"vendor_project,omitempty"`
	Product       string    `json:"product,omitempty"`
}

// GenerateID produces a deterministic hash based on URL or title+source
func GenerateID(url, title, source string) string {
	key := strings.TrimSpace(url)
	if key == "" {
		key = fmt.Sprintf("%s|%s", strings.TrimSpace(source), strings.TrimSpace(title))
	}
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:8])
}

// RelativeTime formats the duration between now and PublishedAt into human-readable compact time
func (item *FeedItem) RelativeTime() string {
	if item.PublishedAt.IsZero() {
		return "recent"
	}
	diff := time.Since(item.PublishedAt)
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	return item.PublishedAt.Format("02 Jan")
}

// FormattedWIB returns the time formatted in Asia/Jakarta timezone
func (item *FeedItem) FormattedWIB() string {
	if item.PublishedAt.IsZero() {
		return "N/A"
	}
	loc := time.FixedZone("WIB", 7*3600)
	return item.PublishedAt.In(loc).Format("2006-01-02 15:04:05 MST")
}

// MatchesFilter returns true if query is empty or matches title, description, tags, CVEs, or source
func (item *FeedItem) MatchesFilter(query string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(item.Title), q) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Description), q) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Source), q) {
		return true
	}
	for _, cve := range item.CVEs {
		if strings.Contains(strings.ToLower(cve), q) {
			return true
		}
	}
	for _, cwe := range item.CWEs {
		if strings.Contains(strings.ToLower(cwe), q) {
			return true
		}
	}
	for _, badge := range item.Badges {
		if strings.Contains(strings.ToLower(badge), q) {
			return true
		}
	}
	return false
}
