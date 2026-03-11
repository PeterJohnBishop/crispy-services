package manager

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/client"
)

type channelWriter struct{ logChan chan string }

func (cw channelWriter) Write(p []byte) (n int, err error) {
	cw.logChan <- string(p)
	return len(p), nil
}

type model struct {
	table      table.Model
	viewport   viewport.Model
	docker     *client.Client
	logChan    chan string
	logs       string
	cancel     context.CancelFunc
	prevCPU    map[string]uint64
	prevSystem map[string]uint64
	searchBar  textinput.Model
	searching  bool
	showHelp   bool
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchContainers(m.docker), doTick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.searching {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter", "esc":
				m.searching = false
				m.searchBar.Blur()
				return m, fetchContainers(m.docker)
			}
		}
		m.searchBar, cmd = m.searchBar.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(fetchContainers(m.docker), doTick())
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "/":
			m.searching = true
			m.searchBar.Focus()
			m.searchBar.SetValue("")
			return m, nil
		case "q", "ctrl+c":
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		case "r":
			return m, fetchContainers(m.docker)
		case "l":
			if m.cancel != nil {
				m.cancel()
			}
			id := m.table.SelectedRow()[0]
			m.logs = ""
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel
			go startLogging(ctx, m.docker, id, m.logChan)
			return m, m.waitForLogs()
		case "u":
			return m, m.controlContainer(m.table.SelectedRow()[0], "start")
		case "s":
			return m, m.controlContainer(m.table.SelectedRow()[0], "stop")
		case "x":
			return m, m.controlContainer(m.table.SelectedRow()[0], "remove")
		case "R": // RESTART (Shift+R)
			id := m.table.SelectedRow()[0]
			m.viewport.SetContent(fmt.Sprintf("Restarting %s...", id[:12]))
			return m, m.controlContainer(id, "restart")
		case "p": // PAUSE / UNPAUSE toggle
			id := m.table.SelectedRow()[0]
			status := m.table.SelectedRow()[2]
			action := "pause"
			if strings.Contains(strings.ToLower(status), "paused") {
				action = "unpause"
			}
			m.viewport.SetContent(fmt.Sprintf("Action: %s on %s...", action, id[:12]))
			return m, m.controlContainer(id, action)
		case "P": // PRUNE (Shift+P)
			m.viewport.SetContent("Pruning stopped containers...")
			return m, m.controlContainer("", "prune")
		}
	case logLineMsg:
		m.logs += string(msg)
		m.viewport.SetContent(m.logs)
		m.viewport.GotoBottom()
		return m, m.waitForLogs()
	case containerListMsg:
		rows := []table.Row{}
		filter := strings.ToLower(m.searchBar.Value())

		for _, c := range msg {
			name := "N/A"
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}

			if filter != "" && !strings.Contains(strings.ToLower(name), filter) && !strings.Contains(c.ID, filter) {
				continue
			}

			cpu, mem := m.getStats(c.ID)
			rows = append(rows, table.Row{c.ID[:12], name, c.Status, cpu, mem})
		}
		m.table.SetRows(rows)
	}
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)
	return m, cmd
}

func (m model) View() string {
	running := 0
	for _, row := range m.table.Rows() {
		if strings.Contains(strings.ToLower(row[2]), "up") {
			running++
		}
	}

	statusBar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(charmPink).
		Padding(0, 1).
		MarginLeft(1).
		Render(fmt.Sprintf("TOTAL: %d | RUNNING: %d", len(m.table.Rows()), running))

	keyStyle := lipgloss.NewStyle().Foreground(charmPink).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	shiftStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Bold(true)

	var footerContent string

	if m.searching {
		footerContent = lipgloss.NewStyle().MarginLeft(1).PaddingTop(1).
			Render(keyStyle.Render("SEARCHING: ") + m.searchBar.View() + descStyle.Render(" (esc to close)"))
	} else if !m.showHelp {
		footerContent = lipgloss.NewStyle().MarginLeft(1).PaddingTop(1).
			Render(descStyle.Render("Press ") + keyStyle.Render("?") + descStyle.Render(" for help menu"))
	} else {
		col1 := lipgloss.JoinVertical(lipgloss.Left,
			keyStyle.Render("  / ")+descStyle.Render("search"),
			keyStyle.Render("  r ")+descStyle.Render("refresh list"),
			keyStyle.Render("  l ")+descStyle.Render("stream logs"),
		)

		col2 := lipgloss.JoinVertical(lipgloss.Left,
			keyStyle.Render("  u/s ")+descStyle.Render("up / stop"),
			shiftStyle.Render(" SHIFT+R ")+descStyle.Render("restart"),
			keyStyle.Render("  p   ")+descStyle.Render("pause / unpause"),
		)

		col3 := lipgloss.JoinVertical(lipgloss.Left,
			keyStyle.Render("  x   ")+descStyle.Render("remove container"),
			shiftStyle.Render(" SHIFT+P ")+descStyle.Render("prune stopped"),
			keyStyle.Render("  q   ")+descStyle.Render("quit manager"),
		)

		helpGrid := lipgloss.JoinHorizontal(lipgloss.Top,
			col1, "     ", col2, "     ", col3,
		)

		footerContent = lipgloss.NewStyle().MarginLeft(1).PaddingTop(1).Render(helpGrid)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		tableStyle.Render(m.table.View()),
		logStyle.Render(m.viewport.View()),
		statusBar,
		footerContent,
	)
}
func ManageContainers() {
	cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "ID", Width: 12}, {Title: "Name", Width: 25},
			{Title: "Status", Width: 25}, {Title: "CPU", Width: 10}, {Title: "Mem", Width: 15},
		}),
		table.WithFocused(true), table.WithHeight(tableHeight),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(charmPink).BorderBottom(true).Foreground(charmPink).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(charmPink).Bold(true)
	t.SetStyles(s)

	ti := textinput.New()
	ti.Placeholder = "Search containers..."
	ti.CharLimit = 20
	ti.Width = 30

	m := model{
		table:      t,
		viewport:   viewport.New(windowWidth, logHeight),
		docker:     cli,
		logChan:    make(chan string),
		prevCPU:    make(map[string]uint64),
		prevSystem: make(map[string]uint64),
		searchBar:  ti,
		searching:  false,
	}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		os.Exit(1)
	}
}
