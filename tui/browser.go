package tui

import (
	"fmt"
	"math"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Gaurav-Gosain/scraped/scraper"
)

// --- Messages ---

type glamourRenderedMsg struct {
	idx      int
	rendered string
}

// --- Browser state ---

type browserState int

const (
	stateList browserState = iota
	statePager
)

type filterMode int

const (
	filterOff filterMode = iota
	filterOn
)

// --- Styles (Tokyo Night palette) ---

var (
	// Tokyo Night colors
	tnFg      = lipgloss.Color("#a9b1d6")
	tnBlue    = lipgloss.Color("#7aa2f7")
	tnPurple  = lipgloss.Color("#bb9af7")
	tnCyan    = lipgloss.Color("#7dcfff")
	tnGreen   = lipgloss.Color("#9ece6a")
	tnYellow  = lipgloss.Color("#e0af68")
	tnRed     = lipgloss.Color("#f7768e")
	tnComment = lipgloss.Color("#565f89")
	tnDark    = lipgloss.Color("#1a1b26")
	tnSurface = lipgloss.Color("#292e42")
	tnGutter  = lipgloss.Color("#3b4261")

	browserLogoStyle = lipgloss.NewStyle().
				Foreground(tnDark).
				Background(tnBlue).
				Bold(true).
				Padding(0, 1)

	browserCountStyle = lipgloss.NewStyle().
				Foreground(tnCyan).
				Bold(true)

	browserDimStyle = lipgloss.NewStyle().
			Foreground(tnComment)

	browserSubtleStyle = lipgloss.NewStyle().
				Foreground(tnGutter)

	selectedGutter = lipgloss.NewStyle().
			Foreground(tnBlue).
			SetString("│")

	normalGutter = lipgloss.NewStyle().
			Foreground(tnGutter).
			SetString(" ")

	selectedTitleStyle = lipgloss.NewStyle().
				Foreground(tnBlue).
				Bold(true)

	normalTitleStyle = lipgloss.NewStyle().
				Foreground(tnFg)

	selectedMetaStyle = lipgloss.NewStyle().
				Foreground(tnCyan)

	normalMetaStyle = lipgloss.NewStyle().
			Foreground(tnComment)

	errBadgeStyle = lipgloss.NewStyle().
			Foreground(tnRed)

	nativeBadgeStyle = lipgloss.NewStyle().
				Foreground(tnGreen)

	convertedBadgeStyle = lipgloss.NewStyle().
				Foreground(tnYellow)

	// Search match gutter markers
	matchGutterStr        = lipgloss.NewStyle().Foreground(tnPurple).Render("▍")
	currentMatchGutterStr = lipgloss.NewStyle().Foreground(tnBlue).Bold(true).Render("▍")

	// Status bar
	barBgStyle    = lipgloss.NewStyle().Background(tnDark)
	barNoteFg     = lipgloss.NewStyle().Foreground(tnFg).Background(tnDark)
	barHelpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5")).Background(tnSurface)
	barMatchStyle = lipgloss.NewStyle().Foreground(tnPurple).Background(tnDark).Bold(true)

	// Progress bar
	progressFilledStyle = lipgloss.NewStyle().Foreground(tnBlue)
	progressEmptyStyle  = lipgloss.NewStyle().Foreground(tnSurface)

	filterPromptStyle = lipgloss.NewStyle().
				Foreground(tnBlue)
)

const (
	listItemHeight    = 3 // title + meta + gap
	listTopPadding    = 5 // blank + logo + blank + blank(header-separator)
	listBottomPadding = 3 // blank + help + blank
	listHPad          = 4
	pagerBarHeight    = 2 // progress line + info line
)

// --- Model ---

type browserModel struct {
	state   browserState
	results []scraper.Result
	width   int
	height  int

	// List view
	filtered    []int
	cursor      int
	listOffset  int
	filterInput textinput.Model
	filterMode  filterMode

	// Pager view
	viewport    viewport.Model
	pagerInput  textinput.Model
	pagerSearch filterMode
	pagerQuery  string
	pagerIdx    int
	rendered    map[int]string
	renderIdx   int

	// Pager search state
	matchLines []int // line numbers where matches occur
	matchIdx   int   // current match index for n/N navigation
	matchTotal int   // total matches
}

func newBrowserModel(results []scraper.Result) browserModel {
	fi := textinput.New()
	fi.Prompt = "Find: "
	fis := fi.Styles()
	fis.Focused.Prompt = filterPromptStyle
	fis.Blurred.Prompt = filterPromptStyle
	fi.SetStyles(fis)

	pi := textinput.New()
	pi.Prompt = "Search: "
	pis := pi.Styles()
	pis.Focused.Prompt = filterPromptStyle
	pis.Blurred.Prompt = filterPromptStyle
	pi.SetStyles(pis)

	filtered := make([]int, len(results))
	for i := range results {
		filtered[i] = i
	}

	vp := viewport.New()

	return browserModel{
		state:       stateList,
		results:     results,
		filtered:    filtered,
		filterInput: fi,
		pagerInput:  pi,
		viewport:    vp,
		rendered:    make(map[int]string),
		renderIdx:   -1,
		pagerIdx:    -1,
	}
}

func (m browserModel) Init() tea.Cmd {
	return nil
}

func (m browserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - pagerBarHeight)
		m.rendered = make(map[int]string)
		m.renderIdx = -1
		if m.state == statePager {
			return m, m.rerenderPager()
		}
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case glamourRenderedMsg:
		m.rendered[msg.idx] = msg.rendered
		m.renderIdx = -1
		if m.state == statePager && m.pagerIdx == msg.idx {
			if m.pagerQuery != "" {
				m.findMatches()
				if len(m.matchLines) > 0 {
					m.viewport.SetYOffset(m.matchLines[0])
				}
			} else {
				m.viewport.SetContent(msg.rendered)
			}
			m.viewport.GotoTop()
		}
		return m, nil
	}

	switch m.state {
	case stateList:
		return m.updateList(msg)
	case statePager:
		return m.updatePager(msg)
	}

	return m, nil
}

// ══════════════════════════════════════════
// List view
// ══════════════════════════════════════════

func (m browserModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.filterMode == filterOn {
		return m.updateListFilter(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "j", "down":
			m.moveCursorDown()
		case "k", "up":
			m.moveCursorUp()
		case "g", "home":
			m.cursor = 0
			m.listOffset = 0
		case "G", "end":
			m.cursor = max(0, len(m.filtered)-1)
			m.ensureVisible()
		case "enter", "right", "l":
			if len(m.filtered) > 0 {
				return m, m.openDocument(m.filtered[m.cursor])
			}
		case "/":
			m.filterMode = filterOn
			m.filterInput.CursorEnd()
			return m, m.filterInput.Focus()
		}
	}

	return m, nil
}

func (m browserModel) updateListFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyPressMsg); ok {
		switch kmsg.String() {
		case "esc":
			m.filterMode = filterOff
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.rebuildFilter()
			return m, nil
		case "enter":
			m.filterMode = filterOff
			m.filterInput.Blur()
			if len(m.filtered) == 1 {
				return m, m.openDocument(m.filtered[0])
			}
			return m, nil
		case "down":
			m.moveCursorDown()
			return m, nil
		case "up":
			m.moveCursorUp()
			return m, nil
		}
	}

	var cmd tea.Cmd
	prev := m.filterInput.Value()
	m.filterInput, cmd = m.filterInput.Update(msg)
	if m.filterInput.Value() != prev {
		m.rebuildFilter()
	}
	return m, cmd
}

func (m *browserModel) rebuildFilter() {
	query := strings.ToLower(m.filterInput.Value())
	m.filtered = m.filtered[:0]
	for i, r := range m.results {
		if query == "" || strings.Contains(strings.ToLower(r.URL), query) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor = 0
	m.listOffset = 0
}

func (m *browserModel) moveCursorDown() {
	if m.cursor < len(m.filtered)-1 {
		m.cursor++
		m.ensureVisible()
	}
}

func (m *browserModel) moveCursorUp() {
	if m.cursor > 0 {
		m.cursor--
		m.ensureVisible()
	}
}

func (m *browserModel) ensureVisible() {
	pp := m.perPage()
	if m.cursor < m.listOffset {
		m.listOffset = m.cursor
	} else if m.cursor >= m.listOffset+pp {
		m.listOffset = m.cursor - pp + 1
	}
}

func (m browserModel) perPage() int {
	avail := m.height - listTopPadding - listBottomPadding
	return max(1, avail/listItemHeight)
}

func (m *browserModel) openDocument(idx int) tea.Cmd {
	m.state = statePager
	m.pagerIdx = idx
	m.matchLines = nil
	m.matchIdx = 0
	m.matchTotal = 0

	r := m.results[idx]
	if r.Err != nil {
		m.viewport.SetContent(
			"\n " + errBadgeStyle.Render("Error scraping ") + r.URL + "\n\n " + r.Err.Error(),
		)
		return nil
	}

	if cached, ok := m.rendered[idx]; ok {
		m.viewport.SetContent(cached)
		m.viewport.GotoTop()
		return nil
	}

	m.viewport.SetContent(browserDimStyle.Render("\n  Rendering..."))
	m.renderIdx = idx
	return m.glamourCmd(idx)
}

func (m browserModel) glamourCmd(idx int) tea.Cmd {
	md := m.results[idx].Markdown
	w := m.width
	return func() tea.Msg {
		ww := max(20, w-4)
		r, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("tokyo-night"),
			glamour.WithWordWrap(ww),
		)
		if err != nil {
			return glamourRenderedMsg{idx: idx, rendered: md}
		}
		out, err := r.Render(md)
		if err != nil {
			return glamourRenderedMsg{idx: idx, rendered: md}
		}
		return glamourRenderedMsg{idx: idx, rendered: out}
	}
}

func (m *browserModel) rerenderPager() tea.Cmd {
	if m.pagerIdx < 0 || m.results[m.pagerIdx].Err != nil {
		return nil
	}
	m.viewport.SetContent(browserDimStyle.Render("\n  Rendering..."))
	m.renderIdx = m.pagerIdx
	return m.glamourCmd(m.pagerIdx)
}

// ══════════════════════════════════════════
// Pager view
// ══════════════════════════════════════════

func (m browserModel) updatePager(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.pagerSearch == filterOn {
		return m.updatePagerSearch(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "esc", "left", "h":
			m.state = stateList
			m.clearSearch()
			return m, nil
		case "g", "home":
			m.viewport.GotoTop()
			return m, nil
		case "G", "end":
			m.viewport.GotoBottom()
			return m, nil
		case "/":
			m.pagerSearch = filterOn
			m.pagerInput.CursorEnd()
			return m, m.pagerInput.Focus()
		case "n":
			m.nextMatch()
			return m, nil
		case "N":
			m.prevMatch()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m browserModel) updatePagerSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyPressMsg); ok {
		switch kmsg.String() {
		case "esc":
			m.pagerSearch = filterOff
			m.pagerInput.Blur()
			m.pagerInput.SetValue("")
			m.clearSearch()
			return m, nil
		case "enter":
			m.pagerSearch = filterOff
			m.pagerInput.Blur()
			m.pagerQuery = m.pagerInput.Value()
			m.findMatches()
			if len(m.matchLines) > 0 {
				m.viewport.SetYOffset(m.matchLines[0])
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	prev := m.pagerInput.Value()
	m.pagerInput, cmd = m.pagerInput.Update(msg)
	if m.pagerInput.Value() != prev {
		m.pagerQuery = m.pagerInput.Value()
		m.findMatches()
	}
	return m, cmd
}

// ══════════════════════════════════════════
// Search highlighting
// ══════════════════════════════════════════

// refreshViewportContent re-applies gutter markers to matching lines.
// Always works from the original glamour-rendered content to avoid
// compounding markers on repeated calls.
func (m *browserModel) refreshViewportContent() {
	if m.pagerIdx < 0 {
		return
	}
	original, ok := m.rendered[m.pagerIdx]
	if !ok {
		return
	}

	if m.pagerQuery == "" || m.matchTotal == 0 {
		m.viewport.SetContent(original)
		return
	}

	lines := strings.Split(original, "\n")
	strippedLines := strings.Split(ansi.Strip(original), "\n")
	query := strings.ToLower(m.pagerQuery)

	currentLine := -1
	if m.matchIdx >= 0 && m.matchIdx < len(m.matchLines) {
		currentLine = m.matchLines[m.matchIdx]
	}

	for i := range lines {
		if i >= len(strippedLines) {
			break
		}
		if !strings.Contains(strings.ToLower(strippedLines[i]), query) {
			continue
		}
		if i == currentLine {
			lines[i] = currentMatchGutterStr + lines[i]
		} else {
			lines[i] = matchGutterStr + lines[i]
		}
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
}

// findMatches searches the ANSI-stripped original rendered content for the
// query and records matching line numbers, then refreshes the viewport with
// gutter markers.
func (m *browserModel) findMatches() {
	m.matchLines = nil
	m.matchIdx = 0
	m.matchTotal = 0

	if m.pagerQuery == "" {
		m.refreshViewportContent()
		return
	}

	original, ok := m.rendered[m.pagerIdx]
	if !ok {
		return
	}

	stripped := ansi.Strip(original)
	query := strings.ToLower(m.pagerQuery)
	lines := strings.Split(stripped, "\n")

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			m.matchLines = append(m.matchLines, i)
		}
	}
	m.matchTotal = len(m.matchLines)
	m.refreshViewportContent()
}

func (m *browserModel) nextMatch() {
	if len(m.matchLines) == 0 {
		return
	}
	m.matchIdx = (m.matchIdx + 1) % len(m.matchLines)
	m.refreshViewportContent()
	m.viewport.SetYOffset(m.matchLines[m.matchIdx])
}

func (m *browserModel) prevMatch() {
	if len(m.matchLines) == 0 {
		return
	}
	m.matchIdx = (m.matchIdx - 1 + len(m.matchLines)) % len(m.matchLines)
	m.refreshViewportContent()
	m.viewport.SetYOffset(m.matchLines[m.matchIdx])
}

func (m *browserModel) clearSearch() {
	m.pagerQuery = ""
	m.pagerInput.SetValue("")
	m.matchLines = nil
	m.matchIdx = 0
	m.matchTotal = 0
	m.refreshViewportContent()
}

// ══════════════════════════════════════════
// Views
// ══════════════════════════════════════════

func (m browserModel) View() tea.View {
	var s string
	switch m.state {
	case statePager:
		s = m.pagerView()
	default:
		s = m.listView()
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

// --- List view ---

func (m browserModel) listView() string {
	var b strings.Builder

	// Title area
	b.WriteString("\n")
	if m.filterMode == filterOn {
		b.WriteString("  ")
		b.WriteString(m.filterInput.View())
	} else {
		b.WriteString("  ")
		b.WriteString(browserLogoStyle.Render("scraped"))

		successCount := 0
		for _, r := range m.results {
			if r.Err == nil {
				successCount++
			}
		}
		b.WriteString("  ")
		b.WriteString(browserCountStyle.Render(fmt.Sprintf("%d", successCount)))
		b.WriteString(browserDimStyle.Render(" pages"))
		if errCount := len(m.results) - successCount; errCount > 0 {
			b.WriteString(browserDimStyle.Render(fmt.Sprintf("  •  %d errors", errCount)))
		}
		if m.filterInput.Value() != "" {
			b.WriteString(browserDimStyle.Render(fmt.Sprintf("  •  filtered: \"%s\"", m.filterInput.Value())))
		}
	}
	b.WriteString("\n\n")

	// Separator
	sep := browserSubtleStyle.Render(strings.Repeat("─", max(0, m.width-4)))
	b.WriteString("  ")
	b.WriteString(sep)
	b.WriteString("\n")

	// Items
	pp := m.perPage()
	end := min(m.listOffset+pp, len(m.filtered))
	truncTo := max(20, m.width-listHPad*2)

	if len(m.filtered) == 0 {
		b.WriteString("\n")
		b.WriteString("  ")
		b.WriteString(browserDimStyle.Render("No results match."))
		b.WriteString("\n")
	}

	for i := m.listOffset; i < end; i++ {
		idx := m.filtered[i]
		r := m.results[idx]
		isSel := i == m.cursor

		gut := normalGutter
		titStyle := normalTitleStyle
		metStyle := normalMetaStyle
		if isSel {
			gut = selectedGutter
			titStyle = selectedTitleStyle
			metStyle = selectedMetaStyle
		}

		// Badge
		var badge string
		switch {
		case r.Err != nil:
			badge = errBadgeStyle.Render("✗")
		case r.Source == "native":
			badge = nativeBadgeStyle.Render("●")
		default:
			badge = convertedBadgeStyle.Render("●")
		}

		urlStr := truncateURL(r.URL, truncTo)

		// Meta info
		var meta string
		if r.Err != nil {
			errMsg := r.Err.Error()
			if len(errMsg) > truncTo {
				errMsg = errMsg[:truncTo-3] + "..."
			}
			meta = metStyle.Render(errMsg)
		} else {
			meta = metStyle.Render(r.Source + " markdown")
		}

		b.WriteString("\n")
		fmt.Fprintf(&b, "  %s  %s  %s\n", gut, badge, titStyle.Render(urlStr))
		fmt.Fprintf(&b, "       %s\n", meta)
	}

	// Fill empty space
	itemLines := (end - m.listOffset) * listItemHeight
	if len(m.filtered) == 0 {
		itemLines = 2
	}
	avail := m.height - listTopPadding - listBottomPadding - itemLines
	if avail > 0 {
		b.WriteString(strings.Repeat("\n", avail))
	}

	// Help bar
	b.WriteString("\n")
	if m.filterMode == filterOn {
		b.WriteString(browserDimStyle.Render("  enter confirm  •  esc cancel  •  ↑↓ navigate"))
	} else {
		b.WriteString(browserDimStyle.Render("  ↑↓/jk navigate  •  enter open  •  / filter  •  q quit"))
	}

	return b.String()
}

// --- Pager view ---

func (m browserModel) pagerView() string {
	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	m.progressBarLine(&b)
	m.pagerInfoBar(&b)
	return b.String()
}

func (m browserModel) progressBarLine(b *strings.Builder) {
	pct := m.viewport.ScrollPercent()
	filled := min(int(math.Round(pct*float64(m.width))), m.width)
	empty := max(0, m.width-filled)
	b.WriteString(progressFilledStyle.Render(strings.Repeat("━", filled)))
	b.WriteString(progressEmptyStyle.Render(strings.Repeat("─", empty)))
	b.WriteString("\n")
}

func (m browserModel) pagerInfoBar(b *strings.Builder) {
	if m.pagerSearch == filterOn {
		bar := "  " + m.pagerInput.View()
		if m.matchTotal > 0 {
			bar += browserDimStyle.Render(fmt.Sprintf("  %d matches", m.matchTotal))
		} else if m.pagerQuery != "" {
			bar += browserDimStyle.Render("  no matches")
		}
		pad := max(0, m.width-lipgloss.Width(bar))
		b.WriteString(barBgStyle.Render(bar + strings.Repeat(" ", pad)))
		return
	}

	logo := browserLogoStyle.Render("scraped")

	// URL
	note := ""
	if m.pagerIdx >= 0 && m.pagerIdx < len(m.results) {
		note = " " + m.results[m.pagerIdx].URL + " "
	}

	// Match indicator
	var matchInfo string
	if m.matchTotal > 0 {
		matchInfo = barMatchStyle.Render(fmt.Sprintf(" %d/%d ", m.matchIdx+1, m.matchTotal))
	}

	// Help buttons
	helpParts := []string{"esc back"}
	if m.matchTotal > 0 {
		helpParts = append(helpParts, "n/N match")
	}
	helpParts = append(helpParts, "/ search")
	help := barHelpStyle.Render(" " + strings.Join(helpParts, "  ") + " ")

	// Truncate note to available space
	fixedW := lipgloss.Width(logo) + lipgloss.Width(matchInfo) + lipgloss.Width(help)
	noteMax := max(0, m.width-fixedW)
	if lipgloss.Width(note) > noteMax {
		if noteMax > 3 {
			note = note[:noteMax-1] + "…"
		} else {
			note = ""
		}
	}
	note = barNoteFg.Render(note)

	// Padding
	usedW := lipgloss.Width(logo) + lipgloss.Width(note) + lipgloss.Width(matchInfo) + lipgloss.Width(help)
	pad := max(0, m.width-usedW)
	padding := barNoteFg.Render(strings.Repeat(" ", pad))

	b.WriteString(logo)
	b.WriteString(note)
	b.WriteString(padding)
	b.WriteString(matchInfo)
	b.WriteString(help)
}

// RunBrowser launches the interactive results browser TUI.
func RunBrowser(results []scraper.Result) error {
	m := newBrowserModel(results)
	p := tea.NewProgram(m)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("browser TUI error: %w", err)
	}
	return nil
}
