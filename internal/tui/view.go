package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"laxy-services/internal/models"
)

var (
	listKeys = [][2]string{
		{"↑↓/jk", "navigate"}, {"Enter/Tab", "focus logs"},
		{"S", "start"}, {"X", "stop"}, {"R", "restart"},
		{"Shift+R", "refresh"}, {"ESC", "clear search"}, {"Q", "quit"},
	}
	logKeys = [][2]string{
		{"↑↓/jk", "scroll"}, {"PgUp/PgDn", "page"}, {"g/G", "top/bottom"},
		{"f", "toggle live"}, {"/", "search"}, {"S/X/R", "start/stop/restart"},
		{"Tab/ESC", "back"}, {"Q", "quit"},
	}
	logSearchKeys = [][2]string{
		{"n/Enter", "next"}, {"N", "prev"}, {"ESC", "clear"},
	}
)

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.loading {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			styleTitleBox.Render("  LAXY-SERVICES  (TUI)  "),
		)
	}
	contentH := max(4, m.height-titleBarLines-statusBarLines)
	leftW := (m.width * leftPanelRatio) / 100
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderTitleBar(),
		lipgloss.JoinHorizontal(lipgloss.Top,
			m.renderServiceList(leftW, contentH),
			m.renderLogPanel(m.width-leftW, contentH),
		),
		m.renderStatusBar(),
	)
}

func (m Model) renderTitleBar() string {
	return styleTitleBox.Width(max(1, m.width-2)).Render("LAXY-SERVICES  (TUI)")
}

func (m Model) renderStatusBar() string {
	keys := listKeys
	if m.focus == panelLog && m.logSearchActive {
		keys = logSearchKeys
	} else if m.focus == panelLog {
		keys = logKeys
	}
	parts := make([]string, 0, len(keys))
	for _, s := range keys {
		parts = append(parts,
			styleStatusBracket.Render("[") +
				styleStatusKey.Render(s[0]) +
				styleStatusBracket.Render("]") +
				"  " + s[1],
		)
	}
	bar := strings.Join(parts, "   ")
	if m.statusMsg != "" {
		bar += "   " + styleMuted.Render("│  "+m.statusMsg)
	}
	return styleStatusBar.Width(m.width).Render(bar)
}

func (m Model) panelBorder(p panel) lipgloss.Style {
	if m.focus == p {
		return stylePanelFocused
	}
	return stylePanelBlur
}

func (m Model) renderServiceList(width, height int) string {
	iW, iH := width-2, height-2
	if iW < 8 || iH < 4 {
		return m.panelBorder(panelList).Width(max(2, iW)).Height(max(2, iH)).Render("")
	}
	active := m.activeServices()
	nameW := max(6, iW-stateColW-subColW-4)

	title := stylePanelHeading.Render("─ SERVICES LIST ")
	if m.searchQuery != "" {
		title += styleMuted.Render(fmt.Sprintf("(%d/%d)", len(active), len(m.services))) +
			"   " + styleStatusBracket.Render("/") + " " +
			styleStateActive.Render(m.searchQuery+"▌")
	} else {
		title += styleMuted.Render(fmt.Sprintf("(%d)", len(m.services)))
	}

	hdr := styleColHeader.Render("  "+fmt.Sprintf("%-*s", nameW, "NAME")) +
		" " + styleColHeader.Render(fmt.Sprintf("%-*s", stateColW, "STATUS")) +
		" " + styleColHeader.Render(fmt.Sprintf("%-*s", subColW, "SUB"))

	listH := max(0, iH-listHeaderLines)
	rows := make([]string, 0, listH)
	for i := m.listOffset; i < len(active) && len(rows) < listH; i++ {
		rows = append(rows, m.renderServiceRow(i, active[i], iW, nameW))
	}
	if len(active) == 0 && m.searchQuery != "" {
		rows = append(rows, styleMuted.Render("  no results for \""+m.searchQuery+"\""))
	}
	for len(rows) < listH {
		rows = append(rows, "")
	}

	lines := append([]string{title, hdr, styleColHeader.Render(strings.Repeat("─", iW))}, rows...)
	return m.panelBorder(panelList).Width(iW).Height(iH).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderServiceRow(idx int, svc models.Service, width, nameW int) string {
	name := displayName(svc.Name)
	if runes := []rune(name); len(runes) > nameW {
		name = string(runes[:nameW-1]) + "…"
	}
	namePart := name + strings.Repeat(" ", max(0, nameW-len([]rune(name))))
	state := fmt.Sprintf("%-*s", stateColW, svc.ActiveState)
	sub := fmt.Sprintf("%-*s", subColW, svc.SubState)

	if idx == m.cursor {
		return styleSelectedRow.Width(width).Render("▶ " + namePart + " " + state + " " + sub)
	}
	return lipgloss.NewStyle().Width(width).Render(
		serviceIcon(svc.ActiveState) + " " + namePart +
			" " + serviceStateStyle(svc.ActiveState).Render(state) +
			" " + styleMuted.Render(sub),
	)
}

func (m Model) renderLogPanel(width, height int) string {
	iW, iH := width-2, height-2
	if iW < 8 || iH < 4 {
		return m.panelBorder(panelLog).Width(max(2, iW)).Height(max(2, iH)).Render("")
	}
	active := m.activeServices()
	svcName := ""
	if m.cursor < len(active) {
		svcName = displayName(active[m.cursor].Name)
	}

	title := stylePanelHeading.Render("─ LIVE LOGS ")
	if svcName != "" {
		title += styleMuted.Render("● ") + styleStateActive.Render(svcName) + "  " + m.renderScrollTag()
	}
	if m.logSearchActive || m.logSearchQuery != "" {
		title += "   " + styleStatusBracket.Render("/") + " " +
			styleStateActive.Render(m.logSearchQuery+"▌")
	}

	logH := iH - 1
	vis := m.visibleLogLines()
	rows := make([]string, 0, logH)

	if len(vis) == 0 {
		rows = append(rows, styleMuted.Render("  waiting for logs..."))
	} else {
		for _, line := range vis {
			if runes := []rune(line); len(runes) > iW {
				line = string(runes[:iW])
			}
			rows = append(rows, m.renderLogLine(line))
		}
	}
	for len(rows) < logH {
		rows = append(rows, "")
	}

	return m.panelBorder(panelLog).Width(iW).Height(iH).Render(
		lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, rows...)...),
	)
}

func (m Model) renderScrollTag() string {
	if m.logAnchor < 0 {
		return styleLiveTag.Render("[LIVE]")
	}
	behind := max(0, len(m.logLines)-m.logAnchor-m.logPanelHeight())
	return stylePausedTag.Render(fmt.Sprintf("[PAUSED  ↑+%d]", behind))
}

func (m Model) renderLogLine(line string) string {
	if m.logSearchQuery != "" {
		return highlightLogLine(line, m.logSearchQuery)
	}
	return colorizeLogLine(line)
}

func highlightLogLine(line, query string) string {
	lq := strings.ToLower(query)
	if !strings.Contains(strings.ToLower(line), lq) {
		return styleLogDimmed.Render(line)
	}
	var b strings.Builder
	for off := 0; off < len(line); {
		idx := strings.Index(strings.ToLower(line[off:]), lq)
		if idx < 0 {
			b.WriteString(styleLogDefault.Render(line[off:]))
			break
		}
		if idx > 0 {
			b.WriteString(styleLogDefault.Render(line[off : off+idx]))
		}
		b.WriteString(styleLogSearchMatch.Render(line[off+idx : off+idx+len(lq)]))
		off += idx + len(lq)
	}
	return b.String()
}

func colorizeLogLine(line string) string {
	l := strings.ToLower(line)
	switch {
	case strings.Contains(l, "error") || strings.Contains(l, "crit") ||
		strings.Contains(l, "emerg") || strings.Contains(l, "alert") ||
		strings.Contains(l, "failed"):
		return styleLogError.Render(line)
	case strings.Contains(l, "warn"):
		return styleLogWarn.Render(line)
	case strings.Contains(l, " info ") || strings.Contains(l, "[info]") || strings.Contains(l, ": info"):
		return styleLogInfo.Render(line)
	case strings.Contains(l, "systemd["):
		return styleLogSystem.Render(line)
	default:
		return styleLogDefault.Render(line)
	}
}
