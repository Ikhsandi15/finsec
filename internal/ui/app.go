package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"finsec-term/internal/config"
	"finsec-term/internal/fetcher"
	"finsec-term/internal/model"
	"finsec-term/internal/storage"
)

type tickClockMsg time.Time
type tickPollMsg time.Time
type fetchFinishedMsg struct {
	result *fetcher.FetchResult
	err    error
}
type statusClearMsg struct{}

type AppModel struct {
	cfg          *config.Config
	manager      *fetcher.Manager
	cache        *storage.Cache
	width        int
	height       int
	activePane   model.PaneID
	isFilterMode bool
	filterInput  textinput.Model
	filterQuery  string
	spinner      spinner.Model
	isFetching   bool
	statusMsg    string

	// Feed items
	rawPane1 []model.FeedItem
	rawPane2 []model.FeedItem
	rawPane3 []model.FeedItem

	// Filtered items
	filteredPane1 []model.FeedItem
	filteredPane2 []model.FeedItem
	filteredPane3 []model.FeedItem

	// Cursors per pane
	cursor1 int
	cursor2 int
	cursor3 int

	// Inspector viewport
	inspectorViewport viewport.Model
	inspectorReady    bool
	selectedItem      *model.FeedItem
}

func NewAppModel(cfg *config.Config, manager *fetcher.Manager, cache *storage.Cache) *AppModel {
	ti := textinput.New()
	ti.Placeholder = "Type keyword (e.g. PHP, Nginx, IDR, CVE-)..."
	ti.CharLimit = 100
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorAmber)

	return &AppModel{
		cfg:          cfg,
		manager:      manager,
		cache:        cache,
		activePane:   model.PaneCVE,
		filterInput:  ti,
		spinner:      s,
		isFetching:   true,
		statusMsg:    "Initial feed synchronization in progress...",
	}
}

func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.clockTickCmd(),
		m.fetchFeedsCmd(),
		m.pollTickCmd(),
	)
}

func (m *AppModel) clockTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickClockMsg(t)
	})
}

func (m *AppModel) pollTickCmd() tea.Cmd {
	interval := m.cfg.RefreshInterval
	if interval < time.Minute {
		interval = 15 * time.Minute
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickPollMsg(t)
	})
}

func (m *AppModel) fetchFeedsCmd() tea.Cmd {
	return func() tea.Msg {
		res, err := m.manager.FetchAll(context.Background())
		return fetchFinishedMsg{result: res, err: err}
	}
}

func (m *AppModel) clearStatusCmd() tea.Cmd {
	return tea.Tick(4*time.Second, func(t time.Time) tea.Msg {
		return statusClearMsg{}
	})
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		layout := ComputeLayout(m.width, m.height)

		inspW := layout.Col2Width - 4
		inspH := layout.Row2Height - 4
		if inspW < 10 {
			inspW = 10
		}
		if inspH < 4 {
			inspH = 4
		}

		if !m.inspectorReady {
			m.inspectorViewport = viewport.New(inspW, inspH)
			m.inspectorReady = true
		} else {
			m.inspectorViewport.Width = inspW
			m.inspectorViewport.Height = inspH
		}
		m.updateInspectorContent()

	case tickClockMsg:
		cmds = append(cmds, m.clockTickCmd())

	case tickPollMsg:
		m.isFetching = true
		m.statusMsg = "Auto-syncing feeds in background..."
		cmds = append(cmds, m.fetchFeedsCmd(), m.pollTickCmd())

	case fetchFinishedMsg:
		m.isFetching = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Fetch failed: %v", msg.err)
		} else if msg.result != nil {
			m.rawPane1 = msg.result.Pane1Items
			m.rawPane2 = msg.result.Pane2Items
			m.rawPane3 = msg.result.Pane3Items
			m.applyFilter()
			m.updateInspectorContent()

			errCount := len(msg.result.Errors)
			if errCount > 0 {
				m.statusMsg = fmt.Sprintf("Synced %d items (%d feed warnings)",
					len(m.rawPane1)+len(m.rawPane2)+len(m.rawPane3), errCount)
			} else {
				m.statusMsg = fmt.Sprintf("Successfully synced %d items across feeds",
					len(m.rawPane1)+len(m.rawPane2)+len(m.rawPane3))
			}
		}
		cmds = append(cmds, m.clearStatusCmd())

	case statusClearMsg:
		if !m.isFetching {
			m.statusMsg = ""
		}

	case spinner.TickMsg:
		if m.isFetching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		// When filter input is active
		if m.isFilterMode {
			switch msg.String() {
			case "enter":
				m.isFilterMode = false
				m.filterInput.Blur()
			case "esc":
				m.isFilterMode = false
				m.filterInput.SetValue("")
				m.filterInput.Blur()
				m.filterQuery = ""
				m.applyFilter()
				m.updateInspectorContent()
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				cmds = append(cmds, cmd)
				m.filterQuery = m.filterInput.Value()
				m.applyFilter()
				m.updateInspectorContent()
			}
			return m, tea.Batch(cmds...)
		}

		// Global normal mode keybindings
		switch msg.String() {
		case "ctrl+c", "q":
			if m.cache != nil {
				_ = m.cache.Save()
			}
			return m, tea.Quit

		case "tab":
			m.cyclePane(true)
			m.updateInspectorContent()

		case "shift+tab":
			m.cyclePane(false)
			m.updateInspectorContent()

		case "1":
			m.activePane = model.PaneCVE
			m.updateInspectorContent()
		case "2":
			m.activePane = model.PaneGlobalMacro
			m.updateInspectorContent()
		case "3":
			m.activePane = model.PaneIDMacro
			m.updateInspectorContent()
		case "4":
			m.activePane = model.PaneInspector

		case "/":
			m.isFilterMode = true
			m.filterInput.Focus()
			return m, textinput.Blink

		case "r":
			if !m.isFetching {
				m.isFetching = true
				m.statusMsg = "Forcing manual feeds refresh..."
				cmds = append(cmds, m.fetchFeedsCmd())
			}

		case "o":
			item := m.getCurrentSelectedItem()
			if item != nil && item.URL != "" {
				if err := OpenBrowser(item.URL); err != nil {
					m.statusMsg = fmt.Sprintf("Error opening browser: %v", err)
				} else {
					m.statusMsg = fmt.Sprintf("Opened in browser: %s", item.URL)
				}
				cmds = append(cmds, m.clearStatusCmd())
			}

		case "m":
			m.markActivePaneAsRead()
			m.statusMsg = fmt.Sprintf("Marked items in active pane as read")
			cmds = append(cmds, m.clearStatusCmd())

		case "enter":
			item := m.getCurrentSelectedItem()
			if item != nil {
				m.markItemAsRead(item)
				m.selectedItem = item
				m.activePane = model.PaneInspector
				m.updateInspectorContent()
			}

		case "j", "down":
			m.navigateDown()
			m.updateInspectorContent()

		case "k", "up":
			m.navigateUp()
			m.updateInspectorContent()

		default:
			if m.activePane == model.PaneInspector && m.inspectorReady {
				var cmd tea.Cmd
				m.inspectorViewport, cmd = m.inspectorViewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) cyclePane(forward bool) {
	if forward {
		switch m.activePane {
		case model.PaneCVE:
			m.activePane = model.PaneGlobalMacro
		case model.PaneGlobalMacro:
			m.activePane = model.PaneIDMacro
		case model.PaneIDMacro:
			m.activePane = model.PaneInspector
		case model.PaneInspector:
			m.activePane = model.PaneCVE
		}
	} else {
		switch m.activePane {
		case model.PaneCVE:
			m.activePane = model.PaneInspector
		case model.PaneGlobalMacro:
			m.activePane = model.PaneCVE
		case model.PaneIDMacro:
			m.activePane = model.PaneGlobalMacro
		case model.PaneInspector:
			m.activePane = model.PaneIDMacro
		}
	}
}

func (m *AppModel) navigateDown() {
	switch m.activePane {
	case model.PaneCVE:
		if m.cursor1 < len(m.filteredPane1)-1 {
			m.cursor1++
		}
	case model.PaneGlobalMacro:
		if m.cursor2 < len(m.filteredPane2)-1 {
			m.cursor2++
		}
	case model.PaneIDMacro:
		if m.cursor3 < len(m.filteredPane3)-1 {
			m.cursor3++
		}
	case model.PaneInspector:
		m.inspectorViewport.LineDown(1)
	}
}

func (m *AppModel) navigateUp() {
	switch m.activePane {
	case model.PaneCVE:
		if m.cursor1 > 0 {
			m.cursor1--
		}
	case model.PaneGlobalMacro:
		if m.cursor2 > 0 {
			m.cursor2--
		}
	case model.PaneIDMacro:
		if m.cursor3 > 0 {
			m.cursor3--
		}
	case model.PaneInspector:
		m.inspectorViewport.LineUp(1)
	}
}

func (m *AppModel) applyFilter() {
	q := m.filterQuery

	m.filteredPane1 = nil
	for _, it := range m.rawPane1 {
		if it.MatchesFilter(q) {
			m.filteredPane1 = append(m.filteredPane1, it)
		}
	}
	if m.cursor1 >= len(m.filteredPane1) {
		m.cursor1 = max(0, len(m.filteredPane1)-1)
	}

	m.filteredPane2 = nil
	for _, it := range m.rawPane2 {
		if it.MatchesFilter(q) {
			m.filteredPane2 = append(m.filteredPane2, it)
		}
	}
	if m.cursor2 >= len(m.filteredPane2) {
		m.cursor2 = max(0, len(m.filteredPane2)-1)
	}

	m.filteredPane3 = nil
	for _, it := range m.rawPane3 {
		if it.MatchesFilter(q) {
			m.filteredPane3 = append(m.filteredPane3, it)
		}
	}
	if m.cursor3 >= len(m.filteredPane3) {
		m.cursor3 = max(0, len(m.filteredPane3)-1)
	}
}

func (m *AppModel) getCurrentSelectedItem() *model.FeedItem {
	switch m.activePane {
	case model.PaneCVE:
		if len(m.filteredPane1) > 0 && m.cursor1 < len(m.filteredPane1) {
			return &m.filteredPane1[m.cursor1]
		}
	case model.PaneGlobalMacro:
		if len(m.filteredPane2) > 0 && m.cursor2 < len(m.filteredPane2) {
			return &m.filteredPane2[m.cursor2]
		}
	case model.PaneIDMacro:
		if len(m.filteredPane3) > 0 && m.cursor3 < len(m.filteredPane3) {
			return &m.filteredPane3[m.cursor3]
		}
	case model.PaneInspector:
		return m.selectedItem
	}
	return nil
}

func (m *AppModel) markItemAsRead(item *model.FeedItem) {
	if item == nil {
		return
	}
	item.IsRead = true
	if m.cache != nil {
		m.cache.MarkRead(item.ID)
	}
}

func (m *AppModel) markActivePaneAsRead() {
	var ids []string
	switch m.activePane {
	case model.PaneCVE:
		for i := range m.rawPane1 {
			m.rawPane1[i].IsRead = true
			ids = append(ids, m.rawPane1[i].ID)
		}
		for i := range m.filteredPane1 {
			m.filteredPane1[i].IsRead = true
		}
	case model.PaneGlobalMacro:
		for i := range m.rawPane2 {
			m.rawPane2[i].IsRead = true
			ids = append(ids, m.rawPane2[i].ID)
		}
		for i := range m.filteredPane2 {
			m.filteredPane2[i].IsRead = true
		}
	case model.PaneIDMacro:
		for i := range m.rawPane3 {
			m.rawPane3[i].IsRead = true
			ids = append(ids, m.rawPane3[i].ID)
		}
		for i := range m.filteredPane3 {
			m.filteredPane3[i].IsRead = true
		}
	}
	if m.cache != nil && len(ids) > 0 {
		m.cache.MarkMultipleRead(ids)
		_ = m.cache.Save()
	}
}

func (m *AppModel) countUnread() int {
	unread := 0
	for _, it := range m.rawPane1 {
		if !it.IsRead {
			unread++
		}
	}
	for _, it := range m.rawPane2 {
		if !it.IsRead {
			unread++
		}
	}
	for _, it := range m.rawPane3 {
		if !it.IsRead {
			unread++
		}
	}
	return unread
}

func (m *AppModel) updateInspectorContent() {
	if !m.inspectorReady {
		return
	}

	item := m.getCurrentSelectedItem()
	if item == nil {
		item = m.selectedItem
	}
	if item == nil {
		m.inspectorViewport.SetContent(StyleItemRead.Render("No item selected. Use Tab/1-3 to select a pane and j/k to navigate."))
		return
	}
	m.selectedItem = item

	wrapWidth := m.inspectorViewport.Width - 2
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	var sb strings.Builder

	// Title
	titleStyle := StyleInspectorTitle.Width(wrapWidth)
	sb.WriteString(titleStyle.Render(item.Title) + "\n\n")

	// Meta details
	sb.WriteString(StyleInspectorMetaKey.Render("SOURCE:     ") + StyleInspectorMetaVal.Render(item.Source) + "\n")
	sb.WriteString(StyleInspectorMetaKey.Render("PUBLISHED:  ") + StyleInspectorMetaVal.Render(item.FormattedWIB()+" ("+item.RelativeTime()+")") + "\n")
	sb.WriteString(StyleInspectorMetaKey.Render("SEVERITY:   ") + StyleInspectorMetaVal.Render(string(item.Severity)) + "\n")
	sb.WriteString(StyleInspectorMetaKey.Render("PRIORITY:   ") + StyleInspectorMetaVal.Render(fmt.Sprintf("%d", item.PriorityScore)) + "\n")

	if len(item.CVEs) > 0 {
		sb.WriteString(StyleInspectorMetaKey.Render("CVE IDENT:  ") + StyleBadgeCVE.Render(strings.Join(item.CVEs, ", ")) + "\n")
	}
	if len(item.CWEs) > 0 {
		sb.WriteString(StyleInspectorMetaKey.Render("CWE IDENT:  ") + StyleInspectorMetaVal.Render(strings.Join(item.CWEs, ", ")) + "\n")
	}

	if len(item.Badges) > 0 {
		var badgeStrs []string
		for _, b := range item.Badges {
			badgeStrs = append(badgeStrs, RenderBadge(b))
		}
		sb.WriteString(StyleInspectorMetaKey.Render("TAGS:       ") + strings.Join(badgeStrs, " ") + "\n")
	}

	if item.VendorProject != "" || item.Product != "" {
		sb.WriteString(StyleInspectorMetaKey.Render("AFFECTS:    ") + StyleInspectorMetaVal.Render(fmt.Sprintf("%s / %s", item.VendorProject, item.Product)) + "\n")
	}

	sb.WriteString(StyleInspectorMetaKey.Render("URL:        ") + StyleInspectorURL.Render(item.URL) + "\n\n")

	// Divider
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorBorderDim).Render(strings.Repeat("─", wrapWidth)) + "\n\n")

	// Description / Summary
	desc := item.Description
	if desc == "" {
		desc = "(No extended summary available for this feed entry)"
	}
	descWrapped := lipgloss.NewStyle().Width(wrapWidth).Render(desc)
	sb.WriteString(StyleInspectorDesc.Render(descWrapped) + "\n\n")

	sb.WriteString(lipgloss.NewStyle().Foreground(ColorTextMuted).Italic(true).Render("Press 'o' to open in browser | Press Enter to focus Inspector"))

	m.inspectorViewport.SetContent(sb.String())
}

func (m *AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing FINSEC-TERM..."
	}

	layout := ComputeLayout(m.width, m.height)

	// 1. Top Header
	header := RenderHeader(m.width, m.filterQuery, m.countUnread(), m.isFetching, m.spinner)

	// 2. Panes
	// Pane 1: Top-Left (CVE & Stack Radar)
	pane1List := RenderFeedItemList(m.filteredPane1, m.cursor1, layout.Col1Width-4, layout.Row1Height-2)
	pane1View := RenderPaneContainer(
		fmt.Sprintf("[1] %s (%d)", model.PaneCVE.String(), len(m.filteredPane1)),
		pane1List,
		layout.Col1Width,
		layout.Row1Height,
		m.activePane == model.PaneCVE,
	)

	// Pane 2: Top-Right (Global Macro & Geopolitics)
	pane2List := RenderFeedItemList(m.filteredPane2, m.cursor2, layout.Col2Width-4, layout.Row1Height-2)
	pane2View := RenderPaneContainer(
		fmt.Sprintf("[2] %s (%d)", model.PaneGlobalMacro.String(), len(m.filteredPane2)),
		pane2List,
		layout.Col2Width,
		layout.Row1Height,
		m.activePane == model.PaneGlobalMacro,
	)

	// Pane 3: Bottom-Left (Indonesia Market, Politics & Cyber)
	pane3List := RenderFeedItemList(m.filteredPane3, m.cursor3, layout.Col1Width-4, layout.Row2Height-2)
	pane3View := RenderPaneContainer(
		fmt.Sprintf("[3] %s (%d)", model.PaneIDMacro.String(), len(m.filteredPane3)),
		pane3List,
		layout.Col1Width,
		layout.Row2Height,
		m.activePane == model.PaneIDMacro,
	)

	// Pane 4: Bottom-Right (Inspector / Detail Pane)
	var pane4Content string
	if m.inspectorReady {
		pane4Content = m.inspectorViewport.View()
	} else {
		pane4Content = "Loading inspector..."
	}
	pane4View := RenderPaneContainer(
		fmt.Sprintf("[4] %s", model.PaneInspector.String()),
		pane4Content,
		layout.Col2Width,
		layout.Row2Height,
		m.activePane == model.PaneInspector,
	)

	// Top Row: Pane 1 + Pane 2
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, pane1View, pane2View)

	// Bottom Row: Pane 3 + Pane 4
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, pane3View, pane4View)

	// 3. Status Bar
	statusBar := RenderStatusBar(m.width, m.isFilterMode, m.filterInput.View(), m.statusMsg)

	// Combine all rows
	return lipgloss.JoinVertical(lipgloss.Left, header, topRow, bottomRow, statusBar)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
