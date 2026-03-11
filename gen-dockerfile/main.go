package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Bold(true)
	subStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	doneStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
)

func getBinaryName(projectPath string) string {
	path := filepath.Clean(projectPath)
	modFile := filepath.Join(path, "go.mod")

	file, err := os.Open(modFile)
	if err != nil {
		return "main"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pathParts := strings.Split(parts[1], "/")
			return pathParts[len(pathParts)-1]
		}
	}
	return "main"
}

type model struct {
	inputs  []textinput.Model
	focused int
	done    bool
}

func initialModel() model {
	// 5 inputs now: Path, Base, Port, Entrypoint, Env
	m := model{
		inputs: make([]textinput.Model, 5),
	}

	for i := range m.inputs {
		t := textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

		switch i {
		case 0:
			t.Placeholder = "Project Root Path (./my-app or /Users/me/code/app)"
			t.Focus()
		case 1:
			t.Placeholder = "Base Image (golang:1.21-alpine)"
		case 2:
			t.Placeholder = "Port (8080)"
		case 3:
			t.Placeholder = "Entrypoint (auto-generated)"
		case 4:
			t.Placeholder = "Env Vars (KEY=VAL,...) or '.env'"
		}
		m.inputs[i] = t
	}
	return m
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			if m.focused == 0 {
				detected := getBinaryName(m.inputs[0].Value())
				m.inputs[3].SetValue("./" + detected)
			}

			if m.focused == len(m.inputs)-1 {
				m.done = true
				m.generateDockerfile()
				return m, tea.Quit
			}

			m.inputs[m.focused].Blur()
			m.focused++
			return m, m.inputs[m.focused].Focus()
		}
	}

	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.done {
		return doneStyle.Render(fmt.Sprintf("\nDockerfile created in: %s\n", m.inputs[0].Value()))
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("🐳 Docker-Gen") + "\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View() + "\n")
	}

	b.WriteString("\n(enter: next • esc: quit)\n")
	return b.String()
}

func (m model) generateDockerfile() {
	path := m.inputs[0].Value()
	base := m.inputs[1].Value()
	port := m.inputs[2].Value()
	entry := m.inputs[3].Value()
	envRaw := m.inputs[4].Value()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("FROM %s\nWORKDIR /app\nCOPY . .\n", base))

	if envRaw == ".env" {
		sb.WriteString("COPY .env .env\n")
	} else if envRaw != "" {
		for _, pair := range strings.Split(envRaw, ",") {
			sb.WriteString(fmt.Sprintf("ENV %s\n", strings.TrimSpace(pair)))
		}
	}

	binaryName := strings.TrimPrefix(entry, "./")
	sb.WriteString(fmt.Sprintf("RUN go build -o %s .\n", binaryName))

	if port != "" {
		sb.WriteString(fmt.Sprintf("EXPOSE %s\n", port))
	}
	sb.WriteString(fmt.Sprintf("CMD [\"%s\"]\n", entry))

	dockerfilePath := filepath.Join(path, "Dockerfile")
	_ = os.WriteFile(dockerfilePath, []byte(sb.String()), 0644)
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
