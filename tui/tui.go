package tui

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"

	"github.com/Gaurav-Gosain/scraped/scraper"
)

// Tokyo Night palette shared with browser.
var (
	subtle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
	title   = lipgloss.NewStyle().Foreground(lipgloss.Color("#1a1b26")).Background(lipgloss.Color("#7aa2f7")).Bold(true).Padding(0, 1)
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68"))
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	statNum = lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")).Bold(true)
	doneTag = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")).Bold(true)
)

// Raw ANSI for background fill effect (Tokyo Night storm bg #24283b = 36,40,59).
const (
	fillBgOn  = "\x1b[48;2;36;40;59m"
	fillBgOff = "\x1b[49m"
)

type (
	scrapeEventMsg scraper.Event
	scrapeDoneMsg  struct {
		results []scraper.Result
		err     error
	}
)

type (
	fillTickMsg struct{}
	finishMsg   struct{}
)

func fillTick() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(time.Time) tea.Msg {
		return fillTickMsg{}
	})
}

type model struct {
	spinner    spinner.Model
	progress   progress.Model
	logEntries []string
	fetched    int
	completed  int
	activeURLs []string
	activeSet  map[string]bool
	results    []scraper.Result
	err        error
	done       bool
	width      int
	height     int

	// Smooth fill animation
	fillTarget  float64 // target fill (actual progress)
	fillCurrent float64 // animated fill (smoothly approaches target)
	finishing   bool    // scraping done, letting fill animation complete
	holdTicks   int     // counter for post-completion hold
}

func newModel() model {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))),
	)

	p := progress.New(
		progress.WithColors(lipgloss.Color("#7aa2f7"), lipgloss.Color("#bb9af7")),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return model{
		spinner:   s,
		progress:  p,
		activeSet: make(map[string]bool),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fillTick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		barWidth := msg.Width - 20
		barWidth = max(barWidth, 20)
		barWidth = min(barWidth, 60)
		m.progress.SetWidth(barWidth)
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd

	case scrapeEventMsg:
		return m.handleScrapeEvent(scraper.Event(msg))

	case scrapeDoneMsg:
		m.results = msg.results
		m.err = msg.err
		m.finishing = true
		m.fillTarget = 1.0
		// Set the progress bar to 100% too.
		cmd := m.progress.SetPercent(1.0)
		// Don't quit yet — let fill animation complete + hold.
		return m, cmd

	case fillTickMsg:
		return m.handleFillTick()

	case finishMsg:
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m model) handleFillTick() (tea.Model, tea.Cmd) {
	diff := m.fillTarget - m.fillCurrent

	if math.Abs(diff) < 0.002 {
		// Snapped to target.
		m.fillCurrent = m.fillTarget

		if m.finishing {
			m.holdTicks++
			// Hold for ~600ms (37 ticks at 16ms) so the user sees the full fill.
			if m.holdTicks > 37 {
				return m, func() tea.Msg { return finishMsg{} }
			}
		}
	} else {
		// Smooth ease-out: move 10% of remaining distance each frame.
		m.fillCurrent += diff * 0.10
	}

	return m, fillTick()
}

func (m *model) addActive(url string) {
	if !m.activeSet[url] {
		m.activeSet[url] = true
		m.activeURLs = append(m.activeURLs, url)
	}
}

func (m *model) removeActive(url string) {
	if !m.activeSet[url] {
		return
	}
	delete(m.activeSet, url)
	for i, u := range m.activeURLs {
		if u == url {
			m.activeURLs = append(m.activeURLs[:i], m.activeURLs[i+1:]...)
			break
		}
	}
}

func (m model) handleScrapeEvent(e scraper.Event) (tea.Model, tea.Cmd) {
	truncW := max(20, m.width-20)

	switch e.Type {
	case "fetching":
		m.fetched++
		m.addActive(e.URL)

	case "done":
		m.removeActive(e.URL)
		if e.Source == "skipped" {
			break
		}
		m.completed++
		tag := green.Render("native")
		if e.Source == "converted" {
			tag = yellow.Render("converted")
		}
		entry := fmt.Sprintf("  %s %s [%s]", green.Render("✓"), truncateURL(e.URL, truncW), tag)
		m.logEntries = append(m.logEntries, entry)

	case "error":
		m.completed++
		m.removeActive(e.URL)
		errMsg := "unknown error"
		if e.Err != nil {
			errMsg = e.Err.Error()
			if len(errMsg) > 45 {
				errMsg = errMsg[:45] + "..."
			}
		}
		entry := fmt.Sprintf("  %s %s %s", red.Render("✗"), truncateURL(e.URL, max(20, truncW-50)), subtle.Render(errMsg))
		m.logEntries = append(m.logEntries, entry)
	}

	var pct float64
	if m.fetched > 0 {
		pct = float64(m.completed) / float64(m.fetched)
	}
	m.fillTarget = pct
	cmd := m.progress.SetPercent(pct)
	return m, cmd
}

func (m model) View() tea.View {
	if m.done {
		return tea.NewView("")
	}

	h := m.height
	w := m.width
	if h == 0 {
		h = 24
	}
	if w == 0 {
		w = 80
	}

	var lines []string

	// Header
	lines = append(lines, "")
	lines = append(lines, "  "+title.Render("scraped"))
	lines = append(lines, "")

	// Progress line
	var progLine string
	if m.finishing {
		progLine = "  " + doneTag.Render("✓ Done!") + " " + m.progress.View() + " " +
			statNum.Render(fmt.Sprintf("%d", m.completed)) +
			subtle.Render(fmt.Sprintf("/%d", m.fetched))
	} else {
		progLine = "  " + m.spinner.View() + " " + m.progress.View() + " " +
			statNum.Render(fmt.Sprintf("%d", m.completed)) +
			subtle.Render(fmt.Sprintf("/%d", m.fetched))
		if len(m.activeURLs) > 0 {
			progLine += subtle.Render(fmt.Sprintf(" (%d active)", len(m.activeURLs)))
		}
	}
	lines = append(lines, progLine)
	lines = append(lines, "")

	// Log entries — fill available space
	headerLines := len(lines)
	maxLogs := max(0, h-headerLines-1)

	if len(m.logEntries) > 0 {
		entries := m.logEntries
		if len(entries) > maxLogs {
			entries = entries[len(entries)-maxLogs:]
		}
		lines = append(lines, entries...)
	} else if len(m.activeURLs) > 0 {
		sorted := make([]string, len(m.activeURLs))
		copy(sorted, m.activeURLs)
		sort.Strings(sorted)
		shown := sorted
		if len(shown) > 3 {
			shown = shown[:3]
		}
		for _, u := range shown {
			lines = append(lines, subtle.Render(fmt.Sprintf("  → %s", truncateURL(u, max(20, w-10)))))
		}
		if len(sorted) > 3 {
			lines = append(lines, subtle.Render(fmt.Sprintf("  ...and %d more", len(sorted)-3)))
		}
	}

	// Pad to full height
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}

	// Apply background fill from bottom up using animated fillCurrent.
	fillRows := int(math.Round(m.fillCurrent * float64(h)))
	startFill := h - fillRows

	for i := startFill; i < len(lines); i++ {
		lineW := lipgloss.Width(lines[i])
		pad := max(0, w-lineW)
		line := fillBgOn + lines[i] + strings.Repeat(" ", pad) + fillBgOff
		// Re-apply fill bg after any SGR resets within styled content,
		// so the background persists through colored log entries.
		line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+fillBgOn)
		line = strings.ReplaceAll(line, "\x1b[m", "\x1b[m"+fillBgOn)
		lines[i] = line
	}

	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}

// IsTTY reports whether stderr is connected to a terminal.
func IsTTY() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// RunWithProgress runs the scraper with a TUI progress display.
// Falls back to log-based output when no TTY is available.
func RunWithProgress(ctx context.Context, opts scraper.Options) ([]scraper.Result, error) {
	if !IsTTY() {
		return runWithLogs(ctx, opts)
	}

	m := newModel()
	prog := tea.NewProgram(m)

	// Mute Go's standard logger and stderr during TUI to prevent
	// library output (colly, html-to-markdown, etc.) from corrupting
	// the Bubble Tea alt screen. Restore after.
	origStdlogOutput := stdlog.Writer()
	stdlog.SetOutput(io.Discard)
	origStderr := os.Stderr
	devNull, _ := os.Open(os.DevNull)
	if devNull != nil {
		os.Stderr = devNull
	}

	go func() {
		opts.OnEvent = func(e scraper.Event) {
			prog.Send(scrapeEventMsg(e))
		}
		results, err := scraper.Run(ctx, opts)
		prog.Send(scrapeDoneMsg{results: results, err: err})
	}()

	finalModel, err := prog.Run()

	// Restore stderr and standard logger.
	os.Stderr = origStderr
	stdlog.SetOutput(origStdlogOutput)
	if devNull != nil {
		_ = devNull.Close()
	}

	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	fm := finalModel.(model)
	return fm.results, fm.err
}

func runWithLogs(ctx context.Context, opts scraper.Options) ([]scraper.Result, error) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.InfoLevel)

	logger.Info("Starting scrape", "urls", len(opts.URLs), "depth", opts.Depth, "parallelism", opts.Parallelism)

	opts.OnEvent = func(e scraper.Event) {
		switch e.Type {
		case "fetching":
			logger.Info("Fetching", "url", e.URL)
		case "done":
			if e.Source != "skipped" {
				logger.Info("Done", "url", e.URL, "source", e.Source)
			}
		case "error":
			logger.Error("Failed", "url", e.URL, "err", e.Err)
		}
	}

	results, err := scraper.Run(ctx, opts)
	if err != nil {
		return nil, err
	}

	logger.Info("Scraping complete", "total", len(results))
	return results, nil
}

func truncateURL(u string, maxLen int) string {
	if len(u) <= maxLen {
		return u
	}
	return u[:maxLen-3] + "..."
}
