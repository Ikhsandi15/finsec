package fetcher

import (
	"html"
	"regexp"
	"strings"

	"finsec-term/internal/config"
	"finsec-term/internal/model"
)

var (
	cveRegex  = regexp.MustCompile(`(?i)\bCVE-\d{4}-\d{4,7}\b`)
	cweRegex  = regexp.MustCompile(`(?i)\bCWE-\d+\b`)
	tagRegex  = regexp.MustCompile(`<[^>]*>`)
	wsRegex   = regexp.MustCompile(`[ \t]+`)
	lineRegex = regexp.MustCompile(`\n{3,}`)
)

type Tagger struct {
	rules *config.RulesConfig
}

func NewTagger(rules *config.RulesConfig) *Tagger {
	if rules == nil {
		rules = &config.DefaultConfig().Rules
	}
	return &Tagger{rules: rules}
}

// SanitizeHTML strips HTML tags, replaces breaks with newlines, decodes entities, and trims whitespace
func SanitizeHTML(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	// Normalize non-breaking spaces
	s := strings.ReplaceAll(input, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "\u00a0", " ")

	// Replace break tags and paragraphs with newlines
	s = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(s, "\n")
	s = regexp.MustCompile(`(?i)</p>`).ReplaceAllString(s, "\n\n")
	s = regexp.MustCompile(`(?i)</li>`).ReplaceAllString(s, "\n")

	// Strip remaining HTML tags
	s = tagRegex.ReplaceAllString(s, "")

	// Decode HTML entities
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")

	// Clean up whitespace & excessive newlines
	lines := strings.Split(s, "\n")
	var cleanedLines []string
	for _, line := range lines {
		cleaned := wsRegex.ReplaceAllString(strings.TrimSpace(line), " ")
		if cleaned != "" {
			cleanedLines = append(cleanedLines, cleaned)
		} else if len(cleanedLines) > 0 && cleanedLines[len(cleanedLines)-1] != "" {
			cleanedLines = append(cleanedLines, "")
		}
	}

	result := strings.Join(cleanedLines, "\n")
	result = lineRegex.ReplaceAllString(result, "\n\n")
	return strings.TrimSpace(result)
}

// ExtractCVEs finds all unique CVE identifiers in the given text
func ExtractCVEs(text string) []string {
	matches := cveRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		upper := strings.ToUpper(m)
		if !seen[upper] {
			seen[upper] = true
			result = append(result, upper)
		}
	}
	return result
}

// ExtractCWEs finds all unique CWE identifiers in the given text
func ExtractCWEs(text string) []string {
	matches := cweRegex.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		upper := strings.ToUpper(m)
		if !seen[upper] {
			seen[upper] = true
			result = append(result, upper)
		}
	}
	return result
}

func containsWord(target, keyword string) bool {
	// Case-insensitive boundary match or substring match
	t := strings.ToLower(target)
	k := strings.ToLower(keyword)
	if strings.Contains(t, k) {
		return true
	}
	return false
}

// TagAndScore inspects item content, extracts CVEs/CWEs, assigns badges, severity and calculates priority score
func (t *Tagger) TagAndScore(item *model.FeedItem) {
	combinedText := item.Title + " " + item.Description + " " + item.VendorProject + " " + item.Product

	// Extract CVEs & CWEs
	cves := ExtractCVEs(combinedText)
	if len(cves) > 0 {
		item.CVEs = append(item.CVEs, cves...)
	}
	cwes := ExtractCWEs(combinedText)
	if len(cwes) > 0 {
		item.CWEs = append(item.CWEs, cwes...)
	}

	score := 10
	badgeMap := make(map[string]bool)

	// Check CISA
	if item.IsCISA {
		badgeMap[model.BadgeStackCrit] = true
		badgeMap[model.BadgeCVE] = true
		score += 60
		item.Severity = model.SeverityCritical
	}

	// Check Stack Crit keywords
	stackMatched := false
	for _, kw := range t.rules.StackKeywords {
		if containsWord(combinedText, kw) {
			stackMatched = true
			score += 40
			break
		}
	}
	if stackMatched {
		badgeMap[model.BadgeStackCrit] = true
		if item.Severity == "" || item.Severity != model.SeverityCritical {
			item.Severity = model.SeverityCritical
		}
	}

	// Check ID Macro
	idMacroMatched := false
	for _, kw := range t.rules.IDMacroKeywords {
		if containsWord(combinedText, kw) {
			idMacroMatched = true
			score += 35
			break
		}
	}
	if idMacroMatched {
		badgeMap[model.BadgeIDMacro] = true
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityHigh
		}
	}

	// Check ID Pol
	idPolMatched := false
	for _, kw := range t.rules.IDPolKeywords {
		if containsWord(combinedText, kw) {
			idPolMatched = true
			score += 25
			break
		}
	}
	if idPolMatched {
		badgeMap[model.BadgeIDPol] = true
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityMedium
		}
	}

	// Check Global Mkt
	globalMatched := false
	for _, kw := range t.rules.GlobalKeywords {
		if containsWord(combinedText, kw) {
			globalMatched = true
			score += 20
			break
		}
	}
	if globalMatched {
		badgeMap[model.BadgeGlobalMkt] = true
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityMedium
		}
	}

	// Check Tech & AI
	techMatched := false
	for _, kw := range t.rules.TechKeywords {
		if containsWord(combinedText, kw) {
			techMatched = true
			score += 30
			break
		}
	}
	if techMatched {
		badgeMap[model.BadgeTechAI] = true
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityMedium
		}
	}

	if len(item.CVEs) > 0 {
		badgeMap[model.BadgeCVE] = true
		score += 20
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityHigh
		}
	}

	// Default severity if none assigned
	if item.Severity == "" {
		item.Severity = model.SeverityInfo
	}

	// Convert badges map to slice in consistent order
	var badges []string
	if badgeMap[model.BadgeStackCrit] {
		badges = append(badges, model.BadgeStackCrit)
	}
	if badgeMap[model.BadgeCVE] && !badgeMap[model.BadgeStackCrit] {
		badges = append(badges, model.BadgeCVE)
	}
	if badgeMap[model.BadgeIDMacro] {
		badges = append(badges, model.BadgeIDMacro)
	}
	if badgeMap[model.BadgeIDPol] {
		badges = append(badges, model.BadgeIDPol)
	}
	if badgeMap[model.BadgeGlobalMkt] {
		badges = append(badges, model.BadgeGlobalMkt)
	}
	if badgeMap[model.BadgeTechAI] {
		badges = append(badges, model.BadgeTechAI)
	}

	item.Badges = badges
	item.PriorityScore = score
}
