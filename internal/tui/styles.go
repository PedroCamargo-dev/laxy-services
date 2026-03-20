package tui

import "github.com/charmbracelet/lipgloss"

var (
	clrGreen       = lipgloss.Color("#00ff41")
	clrGreenMid    = lipgloss.Color("#00cc33")
	clrGreenDark   = lipgloss.Color("#005c1a")
	clrGreenSelect = lipgloss.Color("#002d0e")
	clrRed         = lipgloss.Color("#ff4444")
	clrYellow      = lipgloss.Color("#ffd700")
	clrLightGray   = lipgloss.Color("#707070")
	clrWhite       = lipgloss.Color("#d0d0d0")
	clrSearchBg    = lipgloss.Color("#1a4d2e")

	styleTitleBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(clrGreenMid).
			Foreground(clrGreen).
			Bold(true).
			Align(lipgloss.Center)

	stylePanelFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(clrGreen)

	stylePanelBlur = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrGreenDark)

	stylePanelHeading = lipgloss.NewStyle().
				Foreground(clrGreenMid).
				Bold(true)

	styleColHeader = lipgloss.NewStyle().
			Foreground(clrGreenDark).
			Bold(true)

	styleSelectedRow = lipgloss.NewStyle().
				Background(clrGreenSelect).
				Foreground(clrGreen).
				Bold(true)

	styleIconActive   = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)
	styleIconFailed   = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	styleIconInactive = lipgloss.NewStyle().Foreground(clrLightGray)
	styleIconPending  = lipgloss.NewStyle().Foreground(clrYellow)

	styleStateActive   = lipgloss.NewStyle().Foreground(clrGreen)
	styleStateFailed   = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	styleStateInactive = lipgloss.NewStyle().Foreground(clrLightGray)
	styleStatePending  = lipgloss.NewStyle().Foreground(clrYellow)

	styleMuted = lipgloss.NewStyle().Foreground(clrLightGray)

	styleLogDefault     = lipgloss.NewStyle().Foreground(clrWhite)
	styleLogInfo        = lipgloss.NewStyle().Foreground(clrGreenMid)
	styleLogWarn        = lipgloss.NewStyle().Foreground(clrYellow)
	styleLogError       = lipgloss.NewStyle().Foreground(clrRed)
	styleLogSystem      = lipgloss.NewStyle().Foreground(clrGreenDark)
	styleLogDimmed      = lipgloss.NewStyle().Foreground(clrLightGray)
	styleLogSearchMatch = lipgloss.NewStyle().Background(clrSearchBg).Foreground(clrGreen).Bold(true)

	styleStatusBar     = lipgloss.NewStyle().Background(clrGreenSelect).Foreground(clrGreenMid).Padding(0, 1)
	styleStatusBracket = lipgloss.NewStyle().Foreground(clrGreenDark)
	styleStatusKey     = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)

	styleLiveTag   = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)
	stylePausedTag = lipgloss.NewStyle().Foreground(clrYellow).Bold(true)
)

func serviceIcon(activeState string) string {
	switch activeState {
	case "active":
		return styleIconActive.Render("✓")
	case "failed":
		return styleIconFailed.Render("✗")
	case "activating", "deactivating", "reloading":
		return styleIconPending.Render("◆")
	default:
		return styleIconInactive.Render("○")
	}
}

func serviceStateStyle(activeState string) lipgloss.Style {
	switch activeState {
	case "active":
		return styleStateActive
	case "failed":
		return styleStateFailed
	case "activating", "deactivating", "reloading":
		return styleStatePending
	default:
		return styleStateInactive
	}
}
