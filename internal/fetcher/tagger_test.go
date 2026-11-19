package fetcher

import (
	"testing"

	"finsec-term/internal/config"
	"finsec-term/internal/model"
)

func TestSanitizeHTML(t *testing.T) {
	raw := `<p>Breaking: <b>Nginx</b> update released!&nbsp;&amp;&nbsp;fixes RCE.</p><br/><a href="https://example.com">Read more</a>`
	expected := "Breaking: Nginx update released! & fixes RCE.\n\nRead more"

	got := SanitizeHTML(raw)
	if got != expected {
		t.Errorf("SanitizeHTML mismatch.\nExpected:\n%s\nGot:\n%s", expected, got)
	}
}

func TestExtractCVEsAndCWEs(t *testing.T) {
	text := "Advisory for CVE-2024-21413 and cve-2023-38606, mapped under CWE-79 and CWE-89."
	cves := ExtractCVEs(text)
	cwes := ExtractCWEs(text)

	if len(cves) != 2 {
		t.Fatalf("expected 2 CVEs, got %d: %v", len(cves), cves)
	}
	if cves[0] != "CVE-2024-21413" || cves[1] != "CVE-2023-38606" {
		t.Errorf("unexpected CVEs: %v", cves)
	}

	if len(cwes) != 2 {
		t.Fatalf("expected 2 CWEs, got %d: %v", len(cwes), cwes)
	}
	if cwes[0] != "CWE-79" || cwes[1] != "CWE-89" {
		t.Errorf("unexpected CWEs: %v", cwes)
	}
}

func TestTagger_TagAndScore(t *testing.T) {
	tagger := NewTagger(&config.DefaultConfig().Rules)

	// Test 1: Stack Critical
	item1 := model.FeedItem{
		Title:       "High severity vulnerability in Redis and RabbitMQ",
		Description: "Remote attacker can trigger denial of service or RCE. See CVE-2024-1111.",
	}
	tagger.TagAndScore(&item1)

	hasStackCrit := false
	for _, b := range item1.Badges {
		if b == model.BadgeStackCrit {
			hasStackCrit = true
		}
	}
	if !hasStackCrit {
		t.Errorf("expected STACK-CRIT badge, got %v", item1.Badges)
	}
	if item1.Severity != model.SeverityCritical {
		t.Errorf("expected SeverityCritical, got %s", item1.Severity)
	}
	if len(item1.CVEs) != 1 || item1.CVEs[0] != "CVE-2024-1111" {
		t.Errorf("expected CVE-2024-1111 extracted, got %v", item1.CVEs)
	}

	// Test 2: ID Macro & Pol
	item2 := model.FeedItem{
		Title:       "Bank Indonesia pertahankan BI Rate, Rupiah menguat terhadap USD",
		Description: "Presiden Prabowo menyambut baik kebijakan moneter dan stabilitas APBN.",
	}
	tagger.TagAndScore(&item2)

	hasIDMacro := false
	hasIDPol := false
	for _, b := range item2.Badges {
		if b == model.BadgeIDMacro {
			hasIDMacro = true
		}
		if b == model.BadgeIDPol {
			hasIDPol = true
		}
	}
	if !hasIDMacro {
		t.Errorf("expected ID-MACRO badge, got %v", item2.Badges)
	}
	if !hasIDPol {
		t.Errorf("expected ID-POL badge, got %v", item2.Badges)
	}

	// Test 3: Global Macro
	item3 := model.FeedItem{
		Title:       "The Fed hints at rate cut as FOMC evaluates CPI and Treasury yields",
		Description: "Jerome Powell delivers address on inflation outlook.",
	}
	tagger.TagAndScore(&item3)

	hasGlobal := false
	for _, b := range item3.Badges {
		if b == model.BadgeGlobalMkt {
			hasGlobal = true
		}
	}
	if !hasGlobal {
		t.Errorf("expected GLOBAL-MKT badge, got %v", item3.Badges)
	}

	// Test 4: False Positive Substring Regression (e.g. Haiti -> "ai", Software -> "war")
	item4 := model.FeedItem{
		Title:       "Geng Pendeta Ketahuan Bunuh Presiden, Tembak Mati di Kamar Tidur",
		Description: "Hakim Miami vonis Christian Emmanuel Sanon bersalah atas konspirasi pembunuhan Presiden Haiti Jovenel Moise.",
	}
	tagger.TagAndScore(&item4)

	for _, b := range item4.Badges {
		if b == model.BadgeTechAI {
			t.Errorf("False positive: 'Haiti' must NOT trigger TECH-AI badge! Badges: %v", item4.Badges)
		}
	}

	// Test 5: True Positive AI matching
	item5 := model.FeedItem{
		Title:       "OpenAI Mengumumkan Terobosan Baru Model AI Generatif",
		Description: "Perusahaan memperkenalkan sistem o3 yang berfokus pada penalaran mendalam.",
	}
	tagger.TagAndScore(&item5)

	hasTechAI := false
	for _, b := range item5.Badges {
		if b == model.BadgeTechAI {
			hasTechAI = true
		}
	}
	if !hasTechAI {
		t.Errorf("expected TECH-AI badge for genuine AI news, got %v", item5.Badges)
	}

	// Test 6: False Positive 'War' in 'Software'
	item6 := model.FeedItem{
		Title:       "New Software Architecture for Enterprise Scale",
		Description: "Engineers present new forward-looking benchmarks.",
	}
	tagger.TagAndScore(&item6)

	for _, b := range item6.Badges {
		if b == model.BadgeGlobalMkt {
			t.Errorf("False positive: 'Software' must NOT trigger GLOBAL-MKT ('War') badge! Badges: %v", item6.Badges)
		}
	}

	// Test 7: Social & X Account Badges
	item7 := model.FeedItem{
		Source:      "ZachXBT On-Chain Intel",
		Title:       "TeleSwap exploit investigation: $735K stolen",
		Description: "Tracing transaction hash to Tornado Cash.",
	}
	tagger.TagAndScore(&item7)

	hasZach := false
	for _, b := range item7.Badges {
		if b == model.BadgeZachXBT {
			hasZach = true
		}
	}
	if !hasZach {
		t.Errorf("expected %s badge, got %v", model.BadgeZachXBT, item7.Badges)
	}

	item8 := model.FeedItem{
		Source:      "ICW Antikorupsi",
		Title:       "Laporan Hasil Pemantauan Tren Vonis Korupsi",
		Description: "Kerugian negara capai triliunan rupiah.",
	}
	tagger.TagAndScore(&item8)

	hasICW := false
	for _, b := range item8.Badges {
		if b == model.BadgeICW {
			hasICW = true
		}
	}
	if !hasICW {
		t.Errorf("expected %s badge, got %v", model.BadgeICW, item8.Badges)
	}

	item9 := model.FeedItem{
		Source:      "Dark Web Informer",
		Title:       "France Travail Database Sale Claim: 2.6M records leaked",
		Description: "Actor cauchyx claims to be selling database.",
	}
	tagger.TagAndScore(&item9)

	hasDarkWeb := false
	for _, b := range item9.Badges {
		if b == model.BadgeDarkWeb {
			hasDarkWeb = true
		}
	}
	if !hasDarkWeb {
		t.Errorf("expected %s badge, got %v", model.BadgeDarkWeb, item9.Badges)
	}
}
