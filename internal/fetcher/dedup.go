package fetcher

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"finsec-term/internal/model"
)

var (
	sourceSuffixRegex = regexp.MustCompile(`(?i)\s*[-–—|]\s*(bbc news|bbc|cnbc indonesia|cnbc|investing\.com|reuters|bloomberg|antara news|antara|bleepingcomputer|the hacker news|dark reading|kontan|bisnis\.com|cointelegraph|coindesk)\s*$`)
	punctRegex        = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	spaceRegex        = regexp.MustCompile(`\s+`)
)

// NormalizeURL strips tracking parameters, fragments, and standardizes host/scheme
func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(raw)
	}

	// Lowercase host and strip port if standard
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	if strings.HasSuffix(host, ":80") {
		host = strings.TrimSuffix(host, ":80")
	} else if strings.HasSuffix(host, ":443") {
		host = strings.TrimSuffix(host, ":443")
	}

	// Filter tracking query params
	if u.RawQuery != "" {
		q := u.Query()
		trackingKeys := []string{
			"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
			"ref", "source", "fbclid", "gclid", "ncid", "_ga", "from", "spm",
		}
		for _, k := range trackingKeys {
			q.Del(k)
		}
		u.RawQuery = q.Encode()
	}

	u.Host = host
	u.Fragment = "" // Strip #fragment
	u.Scheme = "https"

	res := u.String()
	res = strings.TrimSuffix(res, "/")
	return res
}

// NormalizeTitle produces a canonical clean token string for duplicate detection
func NormalizeTitle(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	// Strip common trailing source branding (e.g. " - CNBC Indonesia")
	s = sourceSuffixRegex.ReplaceAllString(s, "")
	// Strip punctuation and special chars
	s = punctRegex.ReplaceAllString(s, " ")
	// Normalize multiple spaces
	s = spaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// wordSet creates a set of lowercase words with min length 2
func wordSet(text string) map[string]struct{} {
	words := strings.Fields(text)
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		if len(w) >= 2 {
			m[w] = struct{}{}
		}
	}
	return m
}

// jaccardSimilarity calculates word overlap between two texts
func jaccardSimilarity(w1, w2 map[string]struct{}) float64 {
	if len(w1) == 0 || len(w2) == 0 {
		return 0
	}
	intersection := 0
	for k := range w1 {
		if _, ok := w2[k]; ok {
			intersection++
		}
	}
	union := len(w1) + len(w2) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// IsDuplicate checks whether two items are essentially the same story
func IsDuplicate(a, b *model.FeedItem) bool {
	if a == nil || b == nil {
		return false
	}

	// 1. Same normalized URL
	normA := NormalizeURL(a.URL)
	normB := NormalizeURL(b.URL)
	if normA != "" && normB != "" && normA == normB {
		return true
	}

	// 2. Exact same normalized title
	titleA := NormalizeTitle(a.Title)
	titleB := NormalizeTitle(b.Title)
	if titleA != "" && titleA == titleB {
		return true
	}

	// 3. High word overlap on titles (>= 80% Jaccard) if published within 48 hours
	if !a.PublishedAt.IsZero() && !b.PublishedAt.IsZero() {
		diff := a.PublishedAt.Sub(b.PublishedAt)
		if diff < 0 {
			diff = -diff
		}
		if diff <= 48*time.Hour {
			wordsA := wordSet(titleA)
			wordsB := wordSet(titleB)
			if len(wordsA) >= 4 && len(wordsB) >= 4 {
				if sim := jaccardSimilarity(wordsA, wordsB); sim >= 0.80 {
					return true
				}
			}
		}
	}

	return false
}

// DeduplicateItems removes duplicate articles, retaining the richer / higher-scored item
func DeduplicateItems(items []model.FeedItem) []model.FeedItem {
	if len(items) <= 1 {
		return items
	}

	var result []model.FeedItem
	seenNormURLs := make(map[string]int)   // normURL -> index in result
	seenNormTitles := make(map[string]int) // normTitle -> index in result

	for _, it := range items {
		normURL := NormalizeURL(it.URL)
		normTitle := NormalizeTitle(it.Title)

		isDup := false
		targetIdx := -1

		if normURL != "" {
			if idx, ok := seenNormURLs[normURL]; ok {
				isDup = true
				targetIdx = idx
			}
		}

		if !isDup && normTitle != "" {
			if idx, ok := seenNormTitles[normTitle]; ok {
				isDup = true
				targetIdx = idx
			}
		}

		// Fuzzy check against existing items published within 48h
		if !isDup && len(strings.Fields(normTitle)) >= 4 {
			wordsIt := wordSet(normTitle)
			for idx := range result {
				existing := &result[idx]
				diff := it.PublishedAt.Sub(existing.PublishedAt)
				if diff < 0 {
					diff = -diff
				}
				if diff <= 48*time.Hour {
					wordsEx := wordSet(NormalizeTitle(existing.Title))
					if len(wordsEx) >= 4 && jaccardSimilarity(wordsIt, wordsEx) >= 0.80 {
						isDup = true
						targetIdx = idx
						break
					}
				}
			}
		}

		if isDup && targetIdx >= 0 {
			// Merge intelligence metadata into the existing item
			existing := &result[targetIdx]
			mergeIntelligence(existing, &it)
			continue
		}

		// New unique item
		idx := len(result)
		result = append(result, it)
		if normURL != "" {
			seenNormURLs[normURL] = idx
		}
		if normTitle != "" {
			seenNormTitles[normTitle] = idx
		}
	}

	return result
}

func mergeIntelligence(dest, src *model.FeedItem) {
	// Keep highest priority score
	if src.PriorityScore > dest.PriorityScore {
		dest.PriorityScore = src.PriorityScore
	}

	// Escalate severity if src is higher
	if dest.Severity == "" || (src.Severity == model.SeverityCritical && dest.Severity != model.SeverityCritical) {
		dest.Severity = src.Severity
	} else if src.Severity == model.SeverityHigh && dest.Severity == model.SeverityLow {
		dest.Severity = model.SeverityHigh
	}

	// Merge CVEs
	existingCVEs := make(map[string]bool)
	for _, cve := range dest.CVEs {
		existingCVEs[cve] = true
	}
	for _, cve := range src.CVEs {
		if !existingCVEs[cve] {
			dest.CVEs = append(dest.CVEs, cve)
			existingCVEs[cve] = true
		}
	}

	// Merge Badges
	existingBadges := make(map[string]bool)
	for _, b := range dest.Badges {
		existingBadges[b] = true
	}
	for _, b := range src.Badges {
		if !existingBadges[b] {
			dest.Badges = append(dest.Badges, b)
			existingBadges[b] = true
		}
	}
}
