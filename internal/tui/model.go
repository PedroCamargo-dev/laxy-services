package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"laxy-services/internal/journal"
	"laxy-services/internal/models"
	"laxy-services/internal/systemd"
)

const (
	maxStoredLogLines = 2000
	leftPanelRatio    = 35
	titleBarLines     = 3
	statusBarLines    = 1
	panelBorderLines  = 2
	listHeaderLines   = 3
	stateColW         = 10
	subColW           = 9
)

type panel int

const (
	panelList panel = iota
	panelLog
)

type (
	initDoneMsg        struct{ client *systemd.Client; services []models.Service; err error }
	servicesRefreshedMsg struct{ services []models.Service; err error }
	logLineMsg           struct{ line string }
	actionDoneMsg        struct{ service, action string; err error }
)

type Model struct {
	services          []models.Service
	cursor, listOffset int
	logLines          []string
	width, height     int
	systemd           *systemd.Client
	follower          *journal.Follower
	followerLines     <-chan string
	statusMsg         string
	searchQuery       string
	focus             panel
	logAnchor         int
	logSearchQuery    string
	logSearchActive   bool
	logSearchMatchIdx int
	loading           bool
}

func New() Model { return Model{loading: true, logAnchor: -1} }

func (m Model) Init() tea.Cmd { return connectSystemd() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case initDoneMsg:
		if msg.err != nil {
			m.statusMsg, m.loading = fmt.Sprintf("dbus error: %v", msg.err), false
			return m, nil
		}
		m.systemd, m.services, m.loading = msg.client, sortServices(msg.services), false
		if len(m.services) > 0 {
			return m, m.startFollower(m.services[0].Name)
		}
	case servicesRefreshedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("refresh error: %v", msg.err)
			return m, nil
		}
		prev := ""
		if active := m.activeServices(); m.cursor < len(active) {
			prev = active[m.cursor].Name
		}
		m.services, m.cursor = sortServices(msg.services), 0
		for i, s := range m.activeServices() {
			if s.Name == prev {
				m.cursor = i
				break
			}
		}
		m.adjustListOffset()
		m.statusMsg = fmt.Sprintf("refreshed  %d services", len(m.services))
	case logLineMsg:
		m.logLines = append(m.logLines, msg.line)
		if excess := len(m.logLines) - maxStoredLogLines; excess > 0 {
			m.logLines = m.logLines[excess:]
			if m.logAnchor >= 0 {
				m.logAnchor = max(0, m.logAnchor-excess)
			}
		}
		return m, waitForLogLine(m.followerLines)
	case actionDoneMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("%s %s: error: %v", msg.action, displayName(msg.service), msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("%s %s: ok", msg.action, displayName(msg.service))
		}
		return m, refreshServices(m.systemd)
	case tea.KeyMsg:
		if m.focus == panelLog {
			return m.handleLogKey(msg)
		}
		return m.handleListKey(msg)
	}
	return m, nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading || m.systemd == nil {
		if k := msg.String(); k == "ctrl+c" || k == "q" {
			m.stopFollower()
			return m, tea.Quit
		}
		return m, nil
	}
	switch msg.String() {
	case "ctrl+c":
		return m.quit()
	case "q":
		if m.searchQuery == "" {
			return m.quit()
		}
		m.searchQuery += "q"
		return m, m.applyServiceSearch()
	case "tab", "enter":
		m.focus = panelLog
	case "esc":
		if m.searchQuery != "" {
			m.searchQuery, m.cursor, m.listOffset = "", 0, 0
			return m, m.switchFollower()
		}
	case "backspace":
		if runes := []rune(m.searchQuery); len(runes) > 0 {
			m.searchQuery = string(runes[:len(runes)-1])
			return m, m.applyServiceSearch()
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--; m.adjustListOffset(); return m, m.switchFollower()
		}
	case "down", "j":
		if m.cursor < len(m.activeServices())-1 {
			m.cursor++; m.adjustListOffset(); return m, m.switchFollower()
		}
	case "s", "x", "r":
		if name, ok := m.selectedSvcName(); ok {
			return m, unitAction(m.systemd, keyToAction(msg.String()), name)
		}
	case "R":
		return m, refreshServices(m.systemd)
	default:
		if k := msg.String(); len(k) == 1 {
			m.searchQuery += k
			return m, m.applyServiceSearch()
		}
	}
	return m, nil
}

func (m Model) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.logSearchActive {
		switch msg.String() {
		case "ctrl+c":
			return m.quit()
		case "esc":
			m.logSearchQuery, m.logSearchActive, m.logSearchMatchIdx = "", false, 0
		case "backspace":
			if runes := []rune(m.logSearchQuery); len(runes) > 0 {
				m.logSearchQuery = string(runes[:len(runes)-1])
				m.logSearchMatchIdx = 0
				m.jumpToLogMatch(0)
			}
		case "enter", "n":
			m.navigateLogMatch(1)
		case "N":
			m.navigateLogMatch(-1)
		default:
			if k := msg.String(); len(k) == 1 {
				m.logSearchQuery += k
				m.logSearchMatchIdx = 0
				m.jumpToLogMatch(0)
			}
		}
		return m, nil
	}
	switch msg.String() {
	case "ctrl+c", "q":
		return m.quit()
	case "tab", "esc":
		m.focus = panelList
	case "up", "k":
		m.scrollLogUp(1)
	case "down", "j":
		m.scrollLogDown(1)
	case "pgup":
		m.scrollLogUp(m.logPanelHeight() - 2)
	case "pgdown":
		m.scrollLogDown(m.logPanelHeight() - 2)
	case "g":
		m.logAnchor = 0
	case "G":
		m.logAnchor = -1
	case "f":
		if m.logAnchor < 0 {
			m.logAnchor = max(0, len(m.logLines)-m.logPanelHeight())
		} else {
			m.logAnchor = -1
		}
	case "/":
		m.logSearchActive = true
	case "s", "x", "r":
		if name, ok := m.selectedSvcName(); ok {
			return m, unitAction(m.systemd, keyToAction(msg.String()), name)
		}
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	leftW := (m.width * leftPanelRatio) / 100
	onLog := msg.X >= leftW

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if onLog {
			m.scrollLogUp(3)
		} else if m.cursor > 0 {
			m.cursor--; m.adjustListOffset(); return m, m.switchFollower()
		}
	case tea.MouseButtonWheelDown:
		if onLog {
			m.scrollLogDown(3)
		} else if m.cursor < len(m.activeServices())-1 {
			m.cursor++; m.adjustListOffset(); return m, m.switchFollower()
		}
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			break
		}
		if !onLog {
			startY := titleBarLines + 1 + listHeaderLines
			if msg.Y >= startY {
				if idx := (msg.Y - startY) + m.listOffset; idx < len(m.activeServices()) {
					m.cursor = idx; m.adjustListOffset(); m.focus = panelList
					return m, m.switchFollower()
				}
			}
		} else {
			m.focus = panelLog
		}
	}
	return m, nil
}

func (m *Model) scrollLogUp(n int) {
	if m.logAnchor < 0 {
		m.logAnchor = max(0, len(m.logLines)-m.logPanelHeight()-n)
	} else {
		m.logAnchor = max(0, m.logAnchor-n)
	}
}

func (m *Model) scrollLogDown(n int) {
	if m.logAnchor >= 0 {
		m.logAnchor += n
		if m.logAnchor+m.logPanelHeight() >= len(m.logLines) {
			m.logAnchor = -1
		}
	}
}

func (m *Model) navigateLogMatch(dir int) {
	matches := m.logMatchIndices()
	if len(matches) == 0 {
		return
	}
	m.logSearchMatchIdx = (m.logSearchMatchIdx + dir + len(matches)) % len(matches)
	m.setLogAnchorToMatch(matches[m.logSearchMatchIdx])
}

func (m *Model) jumpToLogMatch(ordinal int) {
	if matches := m.logMatchIndices(); len(matches) > 0 {
		m.setLogAnchorToMatch(matches[ordinal%len(matches)])
	}
}

func (m *Model) setLogAnchorToMatch(lineIdx int) {
	if a := lineIdx - m.logPanelHeight()/3; a < 0 {
		m.logAnchor = 0
	} else {
		m.logAnchor = a
	}
}

func (m Model) logMatchIndices() []int {
	if m.logSearchQuery == "" {
		return nil
	}
	q := strings.ToLower(m.logSearchQuery)
	var out []int
	for i, line := range m.logLines {
		if strings.Contains(strings.ToLower(line), q) {
			out = append(out, i)
		}
	}
	return out
}

func (m Model) logPanelHeight() int {
	if h := m.height - titleBarLines - statusBarLines - panelBorderLines - 1; h > 0 {
		return h
	}
	return 1
}

func (m Model) visibleLogLines() []string {
	if len(m.logLines) == 0 {
		return nil
	}
	logH := m.logPanelHeight()
	if m.logAnchor < 0 {
		if len(m.logLines) <= logH {
			return m.logLines
		}
		return m.logLines[len(m.logLines)-logH:]
	}
	if m.logAnchor >= len(m.logLines) {
		return nil
	}
	return m.logLines[m.logAnchor:min(m.logAnchor+logH, len(m.logLines))]
}

func (m Model) activeServices() []models.Service {
	if m.searchQuery == "" {
		return m.services
	}
	q := strings.ToLower(m.searchQuery)
	out := make([]models.Service, 0, len(m.services))
	for _, s := range m.services {
		if strings.Contains(strings.ToLower(displayName(s.Name)), q) ||
			strings.Contains(strings.ToLower(s.Description), q) {
			out = append(out, s)
		}
	}
	return out
}

func (m Model) selectedSvcName() (string, bool) {
	if active := m.activeServices(); m.cursor < len(active) {
		return active[m.cursor].Name, true
	}
	return "", false
}

func (m Model) quit() (tea.Model, tea.Cmd) {
	m.stopFollower()
	if m.systemd != nil {
		m.systemd.Close()
	}
	return m, tea.Quit
}

func (m *Model) applyServiceSearch() tea.Cmd {
	if active := m.activeServices(); m.cursor >= len(active) {
		m.cursor = max(0, len(active)-1)
	}
	m.listOffset = 0
	return m.switchFollower()
}

func (m *Model) adjustListOffset() {
	h := m.listPanelRowHeight()
	if h <= 0 {
		return
	}
	if active := m.activeServices(); m.cursor >= len(active) {
		m.cursor = max(0, len(active)-1)
	}
	if m.cursor < m.listOffset {
		m.listOffset = m.cursor
	} else if m.cursor >= m.listOffset+h {
		m.listOffset = m.cursor - h + 1
	}
}

func (m Model) listPanelRowHeight() int {
	if h := m.height - titleBarLines - statusBarLines - panelBorderLines - listHeaderLines; h > 0 {
		return h
	}
	return 1
}

func (m *Model) startFollower(name string) tea.Cmd {
	m.logLines, m.logAnchor = nil, -1
	m.logSearchQuery, m.logSearchActive, m.logSearchMatchIdx = "", false, 0
	f := journal.NewFollower(name)
	m.follower, m.followerLines = f, f.Lines
	return waitForLogLine(f.Lines)
}

func (m *Model) switchFollower() tea.Cmd {
	m.stopFollower()
	if name, ok := m.selectedSvcName(); ok {
		return m.startFollower(name)
	}
	return nil
}

func (m *Model) stopFollower() {
	if m.follower != nil {
		m.follower.Stop()
		m.follower, m.followerLines = nil, nil
	}
}

func connectSystemd() tea.Cmd {
	return func() tea.Msg {
		c, err := systemd.NewClient()
		if err != nil {
			return initDoneMsg{err: err}
		}
		svcs, err := c.ListServices()
		return initDoneMsg{client: c, services: svcs, err: err}
	}
}

func refreshServices(c *systemd.Client) tea.Cmd {
	return func() tea.Msg {
		svcs, err := c.ListServices()
		return servicesRefreshedMsg{services: svcs, err: err}
	}
}

func waitForLogLine(ch <-chan string) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return nil
		}
		return logLineMsg{line: line}
	}
}

func unitAction(c *systemd.Client, action, name string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch action {
		case "start":
			err = c.StartUnit(name)
		case "stop":
			err = c.StopUnit(name)
		case "restart":
			err = c.RestartUnit(name)
		}
		return actionDoneMsg{service: name, action: action, err: err}
	}
}

func keyToAction(key string) string {
	return map[string]string{"s": "start", "x": "stop", "r": "restart"}[key]
}

func sortServices(services []models.Service) []models.Service {
	out := make([]models.Service, len(services))
	copy(out, services)
	sort.Slice(out, func(i, j int) bool {
		pi, pj := statePriority(out[i].ActiveState), statePriority(out[j].ActiveState)
		if pi != pj {
			return pi < pj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func statePriority(state string) int {
	switch state {
	case "active":
		return 0
	case "activating", "reloading":
		return 1
	case "failed":
		return 2
	case "inactive":
		return 3
	}
	return 4
}

func displayName(raw string) string {
	return decodeSystemdEscape(strings.TrimSuffix(raw, ".service"))
}

func decodeSystemdEscape(s string) string {
	if !strings.Contains(s, `\x`) {
		return s
	}
	var buf strings.Builder
	buf.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if i+3 < len(s) && s[i] == '\\' && s[i+1] == 'x' {
			if b, err := strconv.ParseUint(s[i+2:i+4], 16, 8); err == nil {
				buf.WriteByte(byte(b))
				i += 3
				continue
			}
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}
