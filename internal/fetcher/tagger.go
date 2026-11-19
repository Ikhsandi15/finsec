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
	rules        *config.RulesConfig
	stackRegex   *regexp.Regexp
	techRegex    *regexp.Regexp
	idMacroRegex *regexp.Regexp
	idPolRegex   *regexp.Regexp
	globalRegex  *regexp.Regexp
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// buildRegex compiles a list of keywords into a single case-insensitive regex enforcing word boundaries
func buildRegex(keywords []string) *regexp.Regexp {
	if len(keywords) == 0 {
		return nil
	}
	var patterns []string
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		runes := []rune(kw)
		var p strings.Builder

		// If first character is an alphanumeric word character, require word boundary
		if isWordChar(runes[0]) {
			p.WriteString(`\b`)
		}
		p.WriteString(regexp.QuoteMeta(kw))
		// If last character is an alphanumeric word character, require word boundary
		if isWordChar(runes[len(runes)-1]) {
			p.WriteString(`\b`)
		}
		patterns = append(patterns, p.String())
	}
	if len(patterns) == 0 {
		return nil
	}
	fullPattern := "(?i)(?:" + strings.Join(patterns, "|") + ")"
	return regexp.MustCompile(fullPattern)
}

func NewTagger(rules *config.RulesConfig) *Tagger {
	if rules == nil {
		rules = &config.DefaultConfig().Rules
	}
	return &Tagger{
		rules:        rules,
		stackRegex:   buildRegex(rules.StackKeywords),
		techRegex:    buildRegex(rules.TechKeywords),
		idMacroRegex: buildRegex(rules.IDMacroKeywords),
		idPolRegex:   buildRegex(rules.IDPolKeywords),
		globalRegex:  buildRegex(rules.GlobalKeywords),
	}
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

// ExtractCVEs finds all unique CVE identifiers in the given text, capped at 10
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
			if len(result) >= 10 {
				break
			}
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

	// Check Stack Crit keywords with word boundary regex
	if t.stackRegex != nil && t.stackRegex.MatchString(combinedText) {
		badgeMap[model.BadgeStackCrit] = true
		score += 40
		if item.Severity == "" || item.Severity != model.SeverityCritical {
			item.Severity = model.SeverityCritical
		}
	}

	// Check ID Macro keywords with word boundary regex
	if t.idMacroRegex != nil && t.idMacroRegex.MatchString(combinedText) {
		badgeMap[model.BadgeIDMacro] = true
		score += 35
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityHigh
		}
	}

	// Check ID Pol keywords with word boundary regex
	if t.idPolRegex != nil && t.idPolRegex.MatchString(combinedText) {
		badgeMap[model.BadgeIDPol] = true
		score += 25
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityMedium
		}
	}

	// Check Global Mkt keywords with word boundary regex
	if t.globalRegex != nil && t.globalRegex.MatchString(combinedText) {
		badgeMap[model.BadgeGlobalMkt] = true
		score += 20
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityMedium
		}
	}

	// Check Tech & AI keywords with word boundary regex
	if t.techRegex != nil && t.techRegex.MatchString(combinedText) {
		badgeMap[model.BadgeTechAI] = true
		score += 30
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

	// Check Social & Authority Intelligence Sources
	srcLower := strings.ToLower(item.Source)
	textLower := strings.ToLower(combinedText)

	if strings.Contains(srcLower, "antikorupsi") || strings.Contains(srcLower, "icw") || strings.Contains(textLower, "@antikorupsi") {
		badgeMap[model.BadgeICW] = true
		score += 25
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityMedium
		}
	}

	if strings.Contains(srcLower, "zachxbt") || strings.Contains(textLower, "zachxbt") || strings.Contains(textLower, "@zachxbt") {
		badgeMap[model.BadgeZachXBT] = true
		score += 35
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityHigh
		}
	}

	if strings.Contains(srcLower, "cve project") || strings.Contains(textLower, "cvedotnow") || strings.Contains(textLower, "@cvedotnow") {
		badgeMap[model.BadgeCVEDotNow] = true
		score += 30
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityHigh
		}
	}

	if strings.Contains(srcLower, "dark reading") || strings.Contains(textLower, "the_cyber_news") || strings.Contains(textLower, "intcyberdigest") {
		badgeMap[model.BadgeCyberNews] = true
		score += 25
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityMedium
		}
	}

	if strings.Contains(srcLower, "github blog") || strings.Contains(textLower, "githubprojects") {
		badgeMap[model.BadgeGithubPrj] = true
		score += 20
		if item.Severity == "" || item.Severity == model.SeverityInfo {
			item.Severity = model.SeverityMedium
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
	if badgeMap[model.BadgeZachXBT] {
		badges = append(badges, model.BadgeZachXBT)
	}
	if badgeMap[model.BadgeICW] {
		badges = append(badges, model.BadgeICW)
	}
	if badgeMap[model.BadgeCVEDotNow] {
		badges = append(badges, model.BadgeCVEDotNow)
	}
	if badgeMap[model.BadgeCyberNews] {
		badges = append(badges, model.BadgeCyberNews)
	}
	if badgeMap[model.BadgeGithubPrj] {
		badges = append(badges, model.BadgeGithubPrj)
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
