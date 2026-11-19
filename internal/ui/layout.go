package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"

	"finsec-term/internal/model"
)

type LayoutDimensions struct {
	Width        int
	Height       int
	HeaderHeight int
	StatusHeight int
	Col1Width    int
	Col2Width    int
	Row1Height   int
	Row2Height   int
}

// ComputeLayout calculates dimensions for the responsive 4-pane grid
func ComputeLayout(width, height int) LayoutDimensions {
	if width < 40 {
		width = 40
	}
	if height < 15 {
		height = 15
	}

	headerH := 1
	statusH := 1
	availableH := height - headerH - statusH
	if availableH < 6 {
		availableH = 6
	}

	col1W := width / 2
	col2W := width - col1W

	row1H := availableH / 2
	row2H := availableH - row1H

	return LayoutDimensions{
		Width:        width,
		Height:       height,
		HeaderHeight: headerH,
		StatusHeight: statusH,
		Col1Width:    col1W,
		Col2Width:    col2W,
		Row1Height:   row1H,
		Row2Height:   row2H,
	}
}

// RenderHeader renders the Bloomberg top bar with title, filter, unread count, spinner, and clocks
func RenderHeader(width int, filterQuery string, unreadCount int, isFetching bool, spin spinner.Model) string {
	now := time.Now()
	utcTime := now.UTC().Format("15:04:05 UTC")
	locWIB := time.FixedZone("WIB", 7*3600)
	wibTime := now.In(locWIB).Format("15:04:05 WIB")

	title := StyleHeaderTitle.Render("FINSEC-TERM // MACRO & THREAT INTEL")

	var statusBadges []string
	if isFetching {
		statusBadges = append(statusBadges, fmt.Sprintf("%s SYNCING", spin.View()))
	}
	if filterQuery != "" {
		statusBadges = append(statusBadges, StyleHeaderFilter.Render(fmt.Sprintf("[FILTER: %s]", filterQuery)))
	}
	if unreadCount > 0 {
		statusBadges = append(statusBadges, StyleHeaderUnread.Render(fmt.Sprintf("[%d UNREAD]", unreadCount)))
	}

	left := title
	if len(statusBadges) > 0 {
		left += " " + strings.Join(statusBadges, " ")
	}

	right := StyleHeaderClock.Render(fmt.Sprintf("%s | %s", utcTime, wibTime))

	spaceCount := width - lipgloss.Width(left) - lipgloss.Width(right)
	if spaceCount < 1 {
		spaceCount = 1
	}

	content := left + strings.Repeat(" ", spaceCount) + right
	return StyleHeaderBar.Width(width).Render(content)
}

// RenderStatusBar renders the bottom bar with keybindings or filter input
func RenderStatusBar(width int, isFilterMode bool, filterInputView string, statusMsg string) string {
	if isFilterMode {
		prompt := StyleFilterPrompt.Render("FILTER (Esc to clear/cancel, Enter to lock): ")
		content := prompt + filterInputView
		return StyleStatusBar.Width(width).Render(content)
	}

	var hints string
	if statusMsg != "" {
		hints = StyleStatusDesc.Render(statusMsg)
	} else {
		hints = fmt.Sprintf(
			"%s Switch Pane  %s Nav  %s Inspect  %s Filter  %s Open  %s Refresh  %s Mark Read  %s Quit",
			StyleStatusKey.Render("Tab/1-4"),
			StyleStatusKey.Render("j/k"),
			StyleStatusKey.Render("Enter"),
			StyleStatusKey.Render("/"),
			StyleStatusKey.Render("o"),
			StyleStatusKey.Render("r"),
			StyleStatusKey.Render("m"),
			StyleStatusKey.Render("q"),
		)
	}

	return StyleStatusBar.Width(width).Render(hints)
}

// WrapOrTruncate shortens string with ellipsis to fit target length
func Truncate(s string, maxLen int) string {
	if maxLen <= 3 {
		return "..."
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// RenderPaneContainer wraps content with bordered box and title
func RenderPaneContainer(title string, content string, width, height int, isActive bool) string {
	// Account for borders (2 horizontal, 2 vertical)
	innerW := width - 2
	innerH := height - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	var titleStyled string
	if isActive {
		titleStyled = StylePaneTitleActive.Render(fmt.Sprintf(" %s ", title))
	} else {
		titleStyled = StylePaneTitleInactive.Render(fmt.Sprintf(" %s ", title))
	}

	style := StylePaneInactive
	if isActive {
		style = StylePaneActive
	}

	box := style.Width(innerW).Height(innerH).Render(content)

	// Replace top border title
	lines := strings.Split(box, "\n")
	if len(lines) > 0 {
		topLine := lines[0]
		topWidth := lipgloss.Width(topLine)
		titleWidth := lipgloss.Width(titleStyled)

		if topWidth >= titleWidth+4 {
			// Place title near left corner
			prefix := topLine[:2]
			remaining := topWidth - 2 - titleWidth
			if remaining > 0 {
				suffix := topLine[topWidth-remaining:]
				lines[0] = prefix + titleStyled + suffix
			}
		}
		return strings.Join(lines, "\n")
	}

	return box
}

// RenderFeedItemList renders the virtual scrollable list of items for Panes 1, 2, 3
func RenderFeedItemList(items []model.FeedItem, cursor int, width, height int) string {
	if len(items) == 0 {
		emptyMsg := lipgloss.NewStyle().Foreground(ColorTextMuted).Italic(true).Render("No items matching criteria")
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, emptyMsg)
	}

	contentW := width
	visibleLines := height

	// Calculate scroll offset to keep cursor in view
	scrollOffset := 0
	if cursor >= visibleLines {
		scrollOffset = cursor - visibleLines + 1
	}

	var rows []string
	for i := scrollOffset; i < len(items) && len(rows) < visibleLines; i++ {
		item := items[i]
		isSelected := (i == cursor)

		// Badges rendering
		var badgesStr string
		for _, b := range item.Badges {
			badgesStr += RenderBadge(b) + " "
		}
		badgesStr = strings.TrimSpace(badgesStr)

		// Time indicator
		timeStr := StyleItemTime.Render(item.RelativeTime())

		// Read indicator: unread items have a bright dot, read items dim
		indicator := "  "
		if !item.IsRead {
			indicator = lipgloss.NewStyle().Foreground(ColorAmber).Bold(true).Render("• ")
		}

		// Source indicator
		sourceStr := StyleItemSource.Render("[" + Truncate(item.Source, 12) + "]")

		// Calculate available width for title
		fixedW := lipgloss.Width(indicator) + lipgloss.Width(sourceStr) + 1 + lipgloss.Width(timeStr) + 2
		if badgesStr != "" {
			fixedW += lipgloss.Width(badgesStr) + 1
		}

		titleW := contentW - fixedW
		if titleW < 10 {
			titleW = 10
		}

		cleanTitle := Truncate(item.Title, titleW)

		var titleStyled string
		if isSelected {
			titleStyled = StyleItemSelected.Render(cleanTitle)
		} else if item.IsRead {
			titleStyled = StyleItemRead.Render(cleanTitle)
		} else {
			titleStyled = StyleItemNormal.Render(cleanTitle)
		}

		var lineParts []string
		lineParts = append(lineParts, indicator)
		lineParts = append(lineParts, sourceStr)
		if badgesStr != "" {
			lineParts = append(lineParts, badgesStr)
		}
		lineParts = append(lineParts, titleStyled)

		leftContent := strings.Join(lineParts, " ")
		leftW := lipgloss.Width(leftContent)
		spacerLen := contentW - leftW - lipgloss.Width(timeStr)
		if spacerLen < 1 {
			spacerLen = 1
		}

		line := leftContent + strings.Repeat(" ", spacerLen) + timeStr
		if isSelected {
			line = lipgloss.NewStyle().Background(ColorAmberDark).Foreground(ColorBlack).Render(line)
		}
		rows = append(rows, line)
	}

	return strings.Join(rows, "\n")
}
